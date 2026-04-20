package auth

// rotate.go — atomic admin-password rotation + KEK rewrap walk (M5 D3).
//
// The KEK is derived from the admin bcrypt hash (see secret_kek.go). Rotating
// the admin password therefore invalidates every wrapped DEK stored in
// plugin_secret_kek — the new hash produces a different KEK, and the old
// wrap cannot be opened anymore.
//
// RotateCredentialsAndKEK solves this by performing both operations inside
// a single transaction:
//
//  1. SELECT the current (old) admin hash FOR UPDATE.
//  2. For every plugin_secret_kek row: derive the *old* KEK from the old
//     hash + the row's stored kid, unwrap the DEK, then derive the *new*
//     KEK from the newly-bcrypted password + the fresh kid, rewrap.
//  3. UPDATE admin_auth with the new hash.
//  4. Commit.
//
// A crash mid-flight rolls back the tx — admin_auth stays on the old hash
// and the wrapped DEKs remain intact. The worst case is "rotate again" on
// next login, which is safe.

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/hkdf"
)

// RotateCredentialsAndKEK atomically:
//  1. bcrypt-hashes newPassword,
//  2. unwraps every plugin_secret_kek row with the old KEK and rewraps
//     under the new KEK (new kid),
//  3. upserts the admin_auth row with the new hash.
//
// All three happen inside one tx. On fresh installs (no prior admin row),
// the walk is skipped and the method behaves like Save — so callers can
// route both bootstrap and rotation through this single entry point.
//
// Returns nil on success. Any failure rolls the tx back: old hash + old
// wrapped DEKs remain on disk unchanged.
func (s *CredentialStore) RotateCredentialsAndKEK(ctx context.Context, username, newPassword string) error {
	if username == "" || newPassword == "" {
		return fmt.Errorf("auth: username and password required")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: hash new password: %w", err)
	}
	newKid := rotationKid(time.Now())

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("auth: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after Commit

	// Lock the admin row (or note its absence on fresh install).
	var oldHash string
	err = tx.QueryRow(ctx,
		`SELECT password_hash FROM admin_auth WHERE id = 1 FOR UPDATE`,
	).Scan(&oldHash)
	fresh := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !fresh {
		return fmt.Errorf("auth: select current admin hash: %w", err)
	}

	if !fresh && oldHash != "" {
		if err := rewrapAllDEKsTx(ctx, tx, oldHash, string(newHash), newKid); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO admin_auth (id, username, password_hash, updated_at)
		 VALUES (1, $1, $2, NOW())
		 ON CONFLICT (id) DO UPDATE
		   SET username = EXCLUDED.username,
		       password_hash = EXCLUDED.password_hash,
		       updated_at = NOW()`,
		username, string(newHash),
	); err != nil {
		return fmt.Errorf("auth: upsert admin_auth: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: commit rotation: %w", err)
	}
	return nil
}

// rewrapAllDEKsTx walks plugin_secret_kek inside an open tx, unwraps each
// row with the old KEK (derived from oldHash + that row's stored kid),
// and rewraps with the new KEK (derived from newHash + newKid). Rows are
// locked FOR UPDATE to block any concurrent SecretSet from racing with
// the rewrap.
func rewrapAllDEKsTx(ctx context.Context, tx pgx.Tx, oldHash, newHash, newKid string) error {
	rows, err := tx.Query(ctx,
		`SELECT plugin_name, wrapped_dek, kek_kid FROM plugin_secret_kek FOR UPDATE`,
	)
	if err != nil {
		return fmt.Errorf("auth: select kek rows: %w", err)
	}
	type kekRow struct {
		plugin, kid string
		wrapped     []byte
	}
	var list []kekRow
	for rows.Next() {
		var r kekRow
		if sErr := rows.Scan(&r.plugin, &r.wrapped, &r.kid); sErr != nil {
			rows.Close()
			return fmt.Errorf("auth: scan kek row: %w", sErr)
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("auth: iter kek rows: %w", err)
	}

	for _, r := range list {
		oldKEK, err := deriveKEKFromHash(oldHash, r.kid)
		if err != nil {
			return fmt.Errorf("auth: derive old KEK for %s: %w", r.plugin, err)
		}
		dek, err := UnwrapDEK(oldKEK, r.wrapped) // zeroes oldKEK via defer
		if err != nil {
			return fmt.Errorf("auth: unwrap DEK for %s: %w", r.plugin, err)
		}
		newKEK, err := deriveKEKFromHash(newHash, newKid)
		if err != nil {
			zeroBytes(dek)
			return fmt.Errorf("auth: derive new KEK for %s: %w", r.plugin, err)
		}
		wrapped, err := WrapDEK(newKEK, dek) // zeroes newKEK and we zero dek
		zeroBytes(dek)
		if err != nil {
			return fmt.Errorf("auth: rewrap DEK for %s: %w", r.plugin, err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE plugin_secret_kek
			   SET wrapped_dek = $1, kek_kid = $2, updated_at = now()
			 WHERE plugin_name = $3`,
			wrapped, newKid, r.plugin,
		); err != nil {
			return fmt.Errorf("auth: update kek row for %s: %w", r.plugin, err)
		}
	}
	return nil
}

// deriveKEKFromHash mirrors adminAuthKEKProvider.DeriveKEK but takes the
// hash string directly so the rotation tx can derive a KEK from the OLD
// admin hash it just SELECTed. Kept unexported — the public KEKProvider
// stays the only entry point for other packages.
func deriveKEKFromHash(passwordHash, kid string) ([]byte, error) {
	if passwordHash == "" {
		return nil, ErrKEKNotReady
	}
	ikm := []byte(passwordHash)
	salt := []byte("opendray-plugin-kek")
	info := []byte("opendray-plugin-kek/" + kid)

	reader := hkdf.New(sha256.New, ikm, salt, info)
	kek := make([]byte, KEKSize)
	if _, err := io.ReadFull(reader, kek); err != nil {
		return nil, fmt.Errorf("auth: hkdf expand KEK: %w", err)
	}
	return kek, nil
}

// rotationKid returns a monotonic, human-readable key-id for a rotation
// event. Isolated into a function so tests can pin time.
func rotationKid(now time.Time) string {
	return "rot-" + now.UTC().Format("20060102T150405Z")
}
