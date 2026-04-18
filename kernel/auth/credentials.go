package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Credentials represents the currently-stored admin credentials.
type Credentials struct {
	Username     string
	PasswordHash string
}

// CredentialStore reads and writes the single-row admin_auth record.
// Returns (nil, nil) when no row exists so callers can fall back to
// bootstrap env credentials on a fresh install.
type CredentialStore struct {
	pool *pgxpool.Pool
}

// NewCredentialStore wires a store to the given pool.
func NewCredentialStore(pool *pgxpool.Pool) *CredentialStore {
	return &CredentialStore{pool: pool}
}

// Load returns the stored credentials, or (nil, nil) if none have been
// written yet.
func (s *CredentialStore) Load(ctx context.Context) (*Credentials, error) {
	var c Credentials
	err := s.pool.QueryRow(ctx,
		`SELECT username, password_hash FROM admin_auth WHERE id = 1`,
	).Scan(&c.Username, &c.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("auth: load credentials: %w", err)
	}
	return &c, nil
}

// Save upserts the admin credentials row with the given username and
// plaintext password (hashed here so callers never see or log plain
// text alongside the write).
func (s *CredentialStore) Save(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("auth: username and password required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth: hash password: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO admin_auth (id, username, password_hash, updated_at)
		 VALUES (1, $1, $2, NOW())
		 ON CONFLICT (id) DO UPDATE
		   SET username = EXCLUDED.username,
		       password_hash = EXCLUDED.password_hash,
		       updated_at = NOW()`,
		username, string(hash),
	)
	if err != nil {
		return fmt.Errorf("auth: save credentials: %w", err)
	}
	return nil
}

// VerifyPassword checks a plaintext password against the stored hash
// using bcrypt. Returns true iff the password matches.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
