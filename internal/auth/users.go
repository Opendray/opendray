package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Role is an operator's permission level. Admins reach every route;
// members reach everything except the admin-only group (user
// management, integrations admin, settings, MCP/skills vault, backups,
// self-update, audit). Roles gate the API only — they are not an
// isolation boundary, since every session runs as the same OS user.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// Valid reports whether r is a known role.
func (r Role) Valid() bool { return r == RoleAdmin || r == RoleMember }

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("username already exists")
)

// User is one additional operator login stored in the users table.
// The primary admin (keyfile / config / env) is not a row here.
type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        Role       `json:"role"`
	Disabled    bool       `json:"disabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	passwordHash string
}

// UserPatch is a partial update; nil fields are left unchanged.
type UserPatch struct {
	DisplayName *string
	Role        *Role
	Disabled    *bool
	Password    *string
}

// UserStore persists additional operator logins.
type UserStore interface {
	List(ctx context.Context) ([]User, error)
	Get(ctx context.Context, id string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Create(ctx context.Context, username, displayName, password string, role Role) (User, error)
	Update(ctx context.Context, id string, p UserPatch) (User, error)
	Delete(ctx context.Context, id string) error
	TouchLogin(ctx context.Context, id string) error
}

// PGUserStore is the Postgres-backed UserStore.
type PGUserStore struct{ pool *pgxpool.Pool }

func NewPGUserStore(pool *pgxpool.Pool) *PGUserStore { return &PGUserStore{pool: pool} }

const userSelect = `
    SELECT id, username, display_name, password_hash, role, disabled,
           created_at, updated_at, last_login_at
    FROM users`

func scanUser(row pgx.Row) (User, error) {
	var u User
	var role string
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.passwordHash, &role,
		&u.Disabled, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	u.Role = Role(role)
	return u, nil
}

func (s *PGUserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, userSelect+` ORDER BY lower(username)`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *PGUserStore) Get(ctx context.Context, id string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, userSelect+` WHERE id=$1`, id))
}

func (s *PGUserStore) GetByUsername(ctx context.Context, username string) (User, error) {
	return scanUser(s.pool.QueryRow(ctx, userSelect+` WHERE lower(username)=lower($1)`, username))
}

func (s *PGUserStore) Create(ctx context.Context, username, displayName, password string, role Role) (User, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}
	u, err := scanUser(s.pool.QueryRow(ctx, `
        INSERT INTO users (username, display_name, password_hash, role)
        VALUES ($1, $2, $3, $4)
        RETURNING id, username, display_name, password_hash, role, disabled,
                  created_at, updated_at, last_login_at`,
		username, displayName, hash, string(role)))
	if isUniqueViolation(err) {
		return User{}, ErrUserExists
	}
	return u, err
}

func (s *PGUserStore) Update(ctx context.Context, id string, p UserPatch) (User, error) {
	var hash *string
	if p.Password != nil {
		h, err := hashPassword(*p.Password)
		if err != nil {
			return User{}, err
		}
		hash = &h
	}
	var role *string
	if p.Role != nil {
		r := string(*p.Role)
		role = &r
	}
	return scanUser(s.pool.QueryRow(ctx, `
        UPDATE users SET
            display_name  = COALESCE($2, display_name),
            role          = COALESCE($3, role),
            disabled      = COALESCE($4, disabled),
            password_hash = COALESCE($5, password_hash),
            updated_at    = NOW()
        WHERE id=$1
        RETURNING id, username, display_name, password_hash, role, disabled,
                  created_at, updated_at, last_login_at`,
		id, p.DisplayName, role, p.Disabled, hash))
}

func (s *PGUserStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *PGUserStore) TouchLogin(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET last_login_at=NOW() WHERE id=$1`, id)
	return err
}

func hashPassword(password string) (string, error) {
	if len(password) < MinPasswordLen {
		return "", fmt.Errorf("password must be at least %d characters", MinPasswordLen)
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// validUsername keeps usernames to a simple, display-safe shape.
func validUsername(u string) bool {
	if len(u) < 2 || len(u) > 64 {
		return false
	}
	for _, c := range u {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._-@", c)
		if !ok {
			return false
		}
	}
	return true
}
