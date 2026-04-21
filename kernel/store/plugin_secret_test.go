package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
)

// bootDBForSecret re-uses the same embedded-postgres harness as plugin_kv_test.
func bootDBForSecret(t *testing.T) *DB {
	t.Helper()
	return bootDBForConsents(t)
}

// TestSecret_RoundTrip: set then get returns the same ciphertext + nonce.
func TestSecret_RoundTrip(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-rt", "0.0.1")

	ct := []byte{0x01, 0x02, 0x03, 0x04, 0xFF, 0xAA}
	nonce := bytes.Repeat([]byte{0x7A}, 12)

	if err := db.SecretSet(ctx, "sec-rt", "mykey", ct, nonce); err != nil {
		t.Fatalf("SecretSet: %v", err)
	}

	gotCt, gotNonce, found, err := db.SecretGet(ctx, "sec-rt", "mykey")
	if err != nil {
		t.Fatalf("SecretGet: %v", err)
	}
	if !found {
		t.Fatal("want found=true")
	}
	if !bytes.Equal(gotCt, ct) {
		t.Errorf("ciphertext mismatch: got %x want %x", gotCt, ct)
	}
	if !bytes.Equal(gotNonce, nonce) {
		t.Errorf("nonce mismatch: got %x want %x", gotNonce, nonce)
	}
}

// TestSecret_Get_Missing: absent key returns (nil, nil, false, nil).
func TestSecret_Get_Missing(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-miss", "0.0.1")

	ct, nonce, found, err := db.SecretGet(ctx, "sec-miss", "never-written")
	if err != nil {
		t.Fatalf("SecretGet: %v", err)
	}
	if found {
		t.Error("want found=false")
	}
	if ct != nil || nonce != nil {
		t.Errorf("want nil, nil; got %v, %v", ct, nonce)
	}
}

// TestSecret_Set_RejectsEmptyNonce: nonce is mandatory.
func TestSecret_Set_RejectsEmptyNonce(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-nonce", "0.0.1")

	err := db.SecretSet(ctx, "sec-nonce", "k", []byte{0x01}, []byte{})
	if err == nil {
		t.Error("want error on empty nonce")
	}
}

// TestSecret_Set_RejectsEmptyCiphertext: ciphertext is mandatory.
func TestSecret_Set_RejectsEmptyCiphertext(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-ct", "0.0.1")

	err := db.SecretSet(ctx, "sec-ct", "k", []byte{}, bytes.Repeat([]byte{0x01}, 12))
	if err == nil {
		t.Error("want error on empty ciphertext")
	}
}

// TestSecret_Delete_Idempotent: deleting a missing key is a no-op.
func TestSecret_Delete_Idempotent(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-del", "0.0.1")

	// Delete without a prior set.
	if err := db.SecretDelete(ctx, "sec-del", "ghost"); err != nil {
		t.Errorf("SecretDelete missing: %v", err)
	}
	// Set, delete, get → not found.
	if err := db.SecretSet(ctx, "sec-del", "real", []byte{0x01}, bytes.Repeat([]byte{0x02}, 12)); err != nil {
		t.Fatalf("SecretSet: %v", err)
	}
	if err := db.SecretDelete(ctx, "sec-del", "real"); err != nil {
		t.Fatalf("SecretDelete: %v", err)
	}
	_, _, found, err := db.SecretGet(ctx, "sec-del", "real")
	if err != nil {
		t.Fatalf("SecretGet after delete: %v", err)
	}
	if found {
		t.Error("want found=false after delete")
	}
}

// TestSecret_List_Sorted: list returns all keys ascending; empty plugin
// returns an empty slice.
func TestSecret_List_Sorted(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-list", "0.0.1")
	insertTestPlugin(t, ctx, db, "sec-empty", "0.0.1")

	for _, k := range []string{"zeta", "alpha", "mu"} {
		if err := db.SecretSet(ctx, "sec-list", k, []byte{0xFF}, bytes.Repeat([]byte{0xAB}, 12)); err != nil {
			t.Fatalf("SecretSet %q: %v", k, err)
		}
	}

	keys, err := db.SecretList(ctx, "sec-list")
	if err != nil {
		t.Fatalf("SecretList: %v", err)
	}
	want := []string{"alpha", "mu", "zeta"}
	if len(keys) != len(want) {
		t.Fatalf("len: got %d want %d (%v)", len(keys), len(want), keys)
	}
	for i, v := range want {
		if keys[i] != v {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], v)
		}
	}

	empty, err := db.SecretList(ctx, "sec-empty")
	if err != nil {
		t.Fatalf("SecretList empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("empty plugin keys: got %v", empty)
	}
}

// TestSecret_Cascade_OnPluginDelete: ON DELETE CASCADE on plugins removes
// all plugin_secret rows. Same property for plugin_secret_kek.
func TestSecret_Cascade_OnPluginDelete(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-cascade", "0.0.1")

	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("k%d", i)
		if err := db.SecretSet(ctx, "sec-cascade", key, []byte{0x01}, bytes.Repeat([]byte{0x02}, 12)); err != nil {
			t.Fatalf("SecretSet: %v", err)
		}
	}
	if err := db.EnsureKEKRow(ctx, "sec-cascade", []byte("wrapped-blob"), "kid-v1"); err != nil {
		t.Fatalf("EnsureKEKRow: %v", err)
	}

	if _, err := db.Pool.Exec(ctx, `DELETE FROM plugins WHERE name = $1`, "sec-cascade"); err != nil {
		t.Fatalf("delete plugin: %v", err)
	}

	var secretCount, kekCount int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM plugin_secret WHERE plugin_name = $1`, "sec-cascade").Scan(&secretCount)
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM plugin_secret_kek WHERE plugin_name = $1`, "sec-cascade").Scan(&kekCount)
	if secretCount != 0 {
		t.Errorf("plugin_secret rows: want 0, got %d", secretCount)
	}
	if kekCount != 0 {
		t.Errorf("plugin_secret_kek rows: want 0, got %d", kekCount)
	}
}

// TestSecret_KEKRow_RoundTrip: ensure + get returns the same wrapped blob + kid.
func TestSecret_KEKRow_RoundTrip(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-kek", "0.0.1")

	wrapped := []byte("wrapped-dek-bytes-opaque")
	if err := db.EnsureKEKRow(ctx, "sec-kek", wrapped, "kid-2026"); err != nil {
		t.Fatalf("EnsureKEKRow: %v", err)
	}

	got, kid, err := db.GetWrappedDEK(ctx, "sec-kek")
	if err != nil {
		t.Fatalf("GetWrappedDEK: %v", err)
	}
	if !bytes.Equal(got, wrapped) {
		t.Errorf("wrapped: got %q want %q", got, wrapped)
	}
	if kid != "kid-2026" {
		t.Errorf("kid: got %q", kid)
	}
}

// TestSecret_KEKRow_UpsertRotation: re-calling EnsureKEKRow with a new kid
// replaces the existing row (rotation support).
func TestSecret_KEKRow_UpsertRotation(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-rot", "0.0.1")

	if err := db.EnsureKEKRow(ctx, "sec-rot", []byte("v1-blob"), "kid-v1"); err != nil {
		t.Fatalf("EnsureKEKRow v1: %v", err)
	}
	if err := db.EnsureKEKRow(ctx, "sec-rot", []byte("v2-blob"), "kid-v2"); err != nil {
		t.Fatalf("EnsureKEKRow v2: %v", err)
	}

	wrapped, kid, err := db.GetWrappedDEK(ctx, "sec-rot")
	if err != nil {
		t.Fatalf("GetWrappedDEK: %v", err)
	}
	if string(wrapped) != "v2-blob" {
		t.Errorf("want v2-blob, got %q", wrapped)
	}
	if kid != "kid-v2" {
		t.Errorf("kid: want kid-v2, got %q", kid)
	}
}

// TestSecret_GetWrappedDEK_NotFound: absent plugin returns pgx.ErrNoRows so
// the bridge layer can branch into first-write DEK generation.
func TestSecret_GetWrappedDEK_NotFound(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-no-kek", "0.0.1")

	_, _, err := db.GetWrappedDEK(ctx, "sec-no-kek")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("want pgx.ErrNoRows, got %v", err)
	}
}

// TestSecret_EnsureKEKRow_RejectsEmpty: empty wrapped blob / empty kid rejected.
func TestSecret_EnsureKEKRow_RejectsEmpty(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-empty-kek", "0.0.1")

	if err := db.EnsureKEKRow(ctx, "sec-empty-kek", nil, "kid"); err == nil {
		t.Error("want error on empty wrappedDEK")
	}
	if err := db.EnsureKEKRow(ctx, "sec-empty-kek", []byte("blob"), ""); err == nil {
		t.Error("want error on empty kid")
	}
}

// TestSecret_SeparatePlugins_Isolated: secrets for one plugin are invisible
// to another plugin — a cross-plugin get of the same key returns not-found.
func TestSecret_SeparatePlugins_Isolated(t *testing.T) {
	db := bootDBForSecret(t)
	ctx := context.Background()
	insertTestPlugin(t, ctx, db, "sec-a", "0.0.1")
	insertTestPlugin(t, ctx, db, "sec-b", "0.0.1")

	if err := db.SecretSet(ctx, "sec-a", "shared", []byte("a-secret"), bytes.Repeat([]byte{0x01}, 12)); err != nil {
		t.Fatalf("Set a: %v", err)
	}

	_, _, found, err := db.SecretGet(ctx, "sec-b", "shared")
	if err != nil {
		t.Fatalf("Get b: %v", err)
	}
	if found {
		t.Error("secrets leaked across plugins")
	}
}
