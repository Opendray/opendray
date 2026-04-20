package auth

// rotate_test.go — integration test for RotateCredentialsAndKEK (M5 D3).
//
// Exercises the full rotation path against an embedded Postgres:
//
//	1. Fresh install (no admin row yet) → method saves creds, no walk.
//	2. Seed plugin_secret_kek rows with DEKs wrapped under KEK_old.
//	3. Rotate to a new password. Verify:
//	   - admin_auth hash updated,
//	   - plugin_secret_kek.kek_kid matches the rotation's new kid,
//	   - wrapped_dek cannot be unwrapped with the old KEK,
//	   - wrapped_dek CAN be unwrapped with the new KEK (new hash),
//	   - the unwrapped plaintext DEK is byte-equal to the original.
//	4. Failure rollback: simulate an unwrap error by seeding a bogus
//	   wrapped_dek → rotation errors and admin_auth stays on the old hash.

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/opendray/opendray/kernel/store"
)

// bootRotatePG boots embedded-postgres + migrates. Separate from other
// test harnesses' cache dirs so the suite parallelises cleanly.
func bootRotatePG(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping embedded-postgres integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)

	port := freeRotatePort(t)
	dataDir := t.TempDir()
	cacheDir := filepath.Join(os.TempDir(), "opendray-pg-cache")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("opendray").
			Password("testpw").
			Database("opendray").
			Port(uint32(port)).
			DataPath(dataDir).
			RuntimePath(filepath.Join(cacheDir, "runtime")).
			BinariesPath(cacheDir).
			StartTimeout(2 * time.Minute),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("pg start: %v", err)
	}
	stop := func() { _ = pg.Stop() }
	t.Cleanup(stop)

	db, err := store.New(ctx, store.Config{
		Host: "127.0.0.1", Port: fmt.Sprintf("%d", port),
		User: "opendray", Password: "testpw", DBName: "opendray",
	})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(db.Close)
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db.Pool, stop
}

func freeRotatePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestRotate_FreshInstall_Saves verifies rotation on an empty admin_auth
// just writes the new row (no walk).
func TestRotate_FreshInstall_Saves(t *testing.T) {
	pool, _ := bootRotatePG(t)
	cs := NewCredentialStore(pool)

	ctx := context.Background()
	if err := cs.RotateCredentialsAndKEK(ctx, "admin", "passw0rd!"); err != nil {
		t.Fatalf("fresh rotate: %v", err)
	}
	got, err := cs.Load(ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got == nil || got.Username != "admin" {
		t.Errorf("load: got %+v", got)
	}
	if !VerifyPassword(got.PasswordHash, "passw0rd!") {
		t.Error("new password does not verify")
	}
}

// TestRotate_RewrapsExistingDEKs is the core acceptance case: after
// rotation, plaintext encrypted under the original DEK must still
// decrypt after the DEK is unwrapped with the new KEK.
func TestRotate_RewrapsExistingDEKs(t *testing.T) {
	pool, _ := bootRotatePG(t)
	cs := NewCredentialStore(pool)
	ctx := context.Background()

	// Seed admin with password A.
	if err := cs.Save(ctx, "admin", "old-password-A"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	// Seed a plugin row (FK target).
	if _, err := pool.Exec(ctx,
		`INSERT INTO plugins (name, version, enabled) VALUES ('svc', '1.0.0', true)`,
	); err != nil {
		t.Fatalf("seed plugin: %v", err)
	}

	// Derive KEK_old from the actual stored hash + seed kid "v1".
	creds, _ := cs.Load(ctx)
	oldKEK, err := deriveKEKFromHash(creds.PasswordHash, "v1")
	if err != nil {
		t.Fatalf("derive old KEK: %v", err)
	}

	// Generate a DEK + wrap it under the old KEK.
	dek := make([]byte, DEKSize)
	if _, err := rand.Read(dek); err != nil {
		t.Fatalf("rand dek: %v", err)
	}
	origDEK := append([]byte(nil), dek...) // keep a copy before Wrap zeroes the input
	wrapped, err := WrapDEK(append([]byte(nil), oldKEK...), dek)
	if err != nil {
		t.Fatalf("wrap: %v", err)
	}

	// Encrypt some plaintext with the DEK — this simulates what
	// plugin_secret rows hold.
	ciphertext, nonce := gcmSeal(t, origDEK, []byte("api-key-plaintext"))

	// Insert the wrapped DEK row + a fake secret ciphertext.
	if _, err := pool.Exec(ctx,
		`INSERT INTO plugin_secret_kek (plugin_name, wrapped_dek, kek_kid)
		 VALUES ('svc', $1, 'v1')`,
		wrapped,
	); err != nil {
		t.Fatalf("seed kek row: %v", err)
	}

	// Rotate to a new password.
	if err := cs.RotateCredentialsAndKEK(ctx, "admin", "new-password-B"); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	// Assertion 1: admin_auth hash changed (verifies new password).
	after, _ := cs.Load(ctx)
	if !VerifyPassword(after.PasswordHash, "new-password-B") {
		t.Error("rotation left old hash — VerifyPassword fails for new password")
	}
	if after.PasswordHash == creds.PasswordHash {
		t.Error("rotation did not change password_hash")
	}

	// Assertion 2: plugin_secret_kek row has a fresh kid.
	var newWrapped []byte
	var newKid string
	if err := pool.QueryRow(ctx,
		`SELECT wrapped_dek, kek_kid FROM plugin_secret_kek WHERE plugin_name='svc'`,
	).Scan(&newWrapped, &newKid); err != nil {
		t.Fatalf("load rewrapped row: %v", err)
	}
	if newKid == "v1" {
		t.Errorf("kek_kid not bumped: %q", newKid)
	}
	if bytes.Equal(newWrapped, wrapped) {
		t.Error("wrapped_dek byte-identical — rewrap did not actually run")
	}

	// Assertion 3: NEW KEK unwraps to the ORIGINAL DEK bytes.
	newKEK, err := deriveKEKFromHash(after.PasswordHash, newKid)
	if err != nil {
		t.Fatalf("derive new KEK: %v", err)
	}
	recovered, err := UnwrapDEK(append([]byte(nil), newKEK...), newWrapped)
	if err != nil {
		t.Fatalf("unwrap with new KEK: %v", err)
	}
	if !bytes.Equal(recovered, origDEK) {
		t.Error("rewrap corrupted the DEK")
	}

	// Assertion 4: the DEK still decrypts our pre-rotation ciphertext.
	plaintext := gcmOpen(t, recovered, ciphertext, nonce)
	if string(plaintext) != "api-key-plaintext" {
		t.Errorf("post-rotation decrypt mismatch: %q", plaintext)
	}

	// Assertion 5: old KEK (re-derived) no longer unwraps the new blob.
	oldKEKRedo, _ := deriveKEKFromHash(creds.PasswordHash, "v1")
	if _, err := UnwrapDEK(append([]byte(nil), oldKEKRedo...), newWrapped); err == nil {
		t.Error("old KEK still unwraps rewrapped DEK — rotation ineffective")
	}
}

// TestRotate_RollsBackOnBadWrap simulates a torn-wrap (corruption) and
// checks the rotation tx rolls back completely: admin_auth stays on the
// pre-rotation hash, and the corrupted wrapped_dek row is untouched.
func TestRotate_RollsBackOnBadWrap(t *testing.T) {
	pool, _ := bootRotatePG(t)
	cs := NewCredentialStore(pool)
	ctx := context.Background()

	if err := cs.Save(ctx, "admin", "pre-rotation-pw"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	preHash, _ := cs.Load(ctx)

	if _, err := pool.Exec(ctx,
		`INSERT INTO plugins (name, version, enabled) VALUES ('svc', '1.0.0', true)`,
	); err != nil {
		t.Fatalf("seed plugin: %v", err)
	}

	// Intentionally bogus wrapped_dek — too short for a GCM payload.
	if _, err := pool.Exec(ctx,
		`INSERT INTO plugin_secret_kek (plugin_name, wrapped_dek, kek_kid)
		 VALUES ('svc', $1, 'v1')`,
		[]byte{0x01, 0x02, 0x03},
	); err != nil {
		t.Fatalf("seed bad kek row: %v", err)
	}

	if err := cs.RotateCredentialsAndKEK(ctx, "admin", "should-not-land"); err == nil {
		t.Fatal("expected rotation to fail on malformed wrapped_dek")
	}

	// Admin hash unchanged.
	after, _ := cs.Load(ctx)
	if after.PasswordHash != preHash.PasswordHash {
		t.Errorf("rotation rolled forward despite failure: hash changed")
	}
	if !VerifyPassword(after.PasswordHash, "pre-rotation-pw") {
		t.Error("pre-rotation password no longer verifies")
	}

	// KEK row still bogus (not replaced by a new wrap).
	var kid string
	var wrapped []byte
	_ = pool.QueryRow(ctx,
		`SELECT wrapped_dek, kek_kid FROM plugin_secret_kek WHERE plugin_name='svc'`,
	).Scan(&wrapped, &kid)
	if kid != "v1" {
		t.Errorf("kek_kid changed despite rollback: %q", kid)
	}
	if !bytes.Equal(wrapped, []byte{0x01, 0x02, 0x03}) {
		t.Error("wrapped_dek replaced despite rollback")
	}
}

// ─── crypto helpers (mirror what plugin/bridge/api_secret does) ────────────

// gcmSeal encrypts plaintext with dek using AES-256-GCM. Returns (ciphertext, nonce).
func gcmSeal(t *testing.T, dek, plaintext []byte) ([]byte, []byte) {
	t.Helper()
	block, err := aes.NewCipher(dek)
	if err != nil {
		t.Fatalf("aes cipher: %v", err)
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("rand nonce: %v", err)
	}
	return g.Seal(nil, nonce, plaintext, nil), nonce
}

func gcmOpen(t *testing.T, dek, ciphertext, nonce []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(dek)
	if err != nil {
		t.Fatalf("aes cipher: %v", err)
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	pt, err := g.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("gcm open: %v", err)
	}
	return pt
}
