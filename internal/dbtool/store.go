package dbtool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a connection id doesn't exist.
var ErrNotFound = errors.New("dbtool: connection not found")

// ErrDuplicateName is returned when (cwd, name) already exists.
var ErrDuplicateName = errors.New("dbtool: a connection with this name already exists for the project")

// FieldCipher wraps a short secret at rest — same shape the channel and
// git-host stores use (the backup cipher satisfies it). Declared here,
// where it is consumed. nil / unarmed cipher means passwords stay
// plaintext, exactly like channel bot tokens before backups are armed.
type FieldCipher interface {
	EncryptField(plain string) (string, error)
	DecryptField(envelope string) (string, error)
}

// encryptedFieldPrefix marks a value wrapped by FieldCipher.EncryptField.
const encryptedFieldPrefix = "v1:"

// Store persists db_connections rows in opendray's own database. The
// password column holds either a "v1:" AES-GCM envelope (backups armed)
// or legacy plaintext. Round-trip safe: an envelope that can't be
// decrypted (rotated key) is surfaced as an empty password but kept
// intact on partial updates, never blanked.
type Store struct {
	pool   *pgxpool.Pool
	cipher FieldCipher
}

// NewStore builds the store. cipher may be nil (plaintext at rest).
func NewStore(pool *pgxpool.Pool, cipher FieldCipher) *Store {
	return &Store{pool: pool, cipher: cipher}
}

const connColumns = `id, cwd, name, driver, host, port, db_name, username,
	password_enc, ssl_mode, read_only, options, created_at, updated_at`

// Insert stores a new connection. c.Password is plaintext; it is
// encrypted here when the cipher is armed.
func (s *Store) Insert(ctx context.Context, c Connection) error {
	opts, err := marshalOptions(c.Options)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO db_connections
			(id, cwd, name, driver, host, port, db_name, username,
			 password_enc, ssl_mode, read_only, options, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14)`,
		c.ID, c.Cwd, c.Name, c.Driver, c.Host, c.Port, c.DBName, c.Username,
		s.encrypt(c.Password), c.SSLMode, c.ReadOnly, opts, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateName
		}
		return fmt.Errorf("dbtool: insert connection: %w", err)
	}
	return nil
}

// Get returns one connection with its password decrypted (empty when the
// envelope can't be opened) and HasPassword set.
func (s *Store) Get(ctx context.Context, id string) (Connection, error) {
	return s.scan(s.pool.QueryRow(ctx,
		`SELECT `+connColumns+` FROM db_connections WHERE id = $1`, id))
}

// List returns the connections for one cwd (or all when cwd is empty),
// passwords decrypted.
func (s *Store) List(ctx context.Context, cwd string) ([]Connection, error) {
	q := `SELECT ` + connColumns + ` FROM db_connections`
	var args []any
	if cwd != "" {
		q += ` WHERE cwd = $1`
		args = append(args, cwd)
	}
	q += ` ORDER BY name`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("dbtool: list connections: %w", err)
	}
	defer rows.Close()
	var out []Connection
	for rows.Next() {
		c, err := s.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdatePatch carries partial changes; nil fields are left alone.
// Password semantics: nil = keep the stored secret (even an unreadable
// envelope), pointer to "" = clear it, anything else = replace.
type UpdatePatch struct {
	Name     *string
	Host     *string
	Port     *int
	DBName   *string
	Username *string
	Password *string
	SSLMode  *string
	ReadOnly *bool
	Options  *map[string]any
}

// Update applies patch to id and returns the updated row.
func (s *Store) Update(ctx context.Context, id string, patch UpdatePatch) (Connection, error) {
	sets := []string{"updated_at = $1"}
	args := []any{time.Now().UTC()}
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if patch.Name != nil {
		add("name", *patch.Name)
	}
	if patch.Host != nil {
		add("host", *patch.Host)
	}
	if patch.Port != nil {
		add("port", *patch.Port)
	}
	if patch.DBName != nil {
		add("db_name", *patch.DBName)
	}
	if patch.Username != nil {
		add("username", *patch.Username)
	}
	if patch.Password != nil {
		add("password_enc", s.encrypt(*patch.Password))
	}
	if patch.SSLMode != nil {
		add("ssl_mode", *patch.SSLMode)
	}
	if patch.ReadOnly != nil {
		add("read_only", *patch.ReadOnly)
	}
	if patch.Options != nil {
		opts, err := marshalOptions(*patch.Options)
		if err != nil {
			return Connection{}, err
		}
		args = append(args, opts)
		sets = append(sets, fmt.Sprintf("options = $%d::jsonb", len(args)))
	}
	args = append(args, id)
	tag, err := s.pool.Exec(ctx,
		fmt.Sprintf(`UPDATE db_connections SET %s WHERE id = $%d`,
			strings.Join(sets, ", "), len(args)), args...)
	if err != nil {
		if isUniqueViolation(err) {
			return Connection{}, ErrDuplicateName
		}
		return Connection{}, fmt.Errorf("dbtool: update connection: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Connection{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

// Delete removes id. Missing rows return ErrNotFound.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM db_connections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("dbtool: delete connection: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) scan(row pgx.Row) (Connection, error) {
	var c Connection
	var stored string
	var opts []byte
	err := row.Scan(&c.ID, &c.Cwd, &c.Name, &c.Driver, &c.Host, &c.Port,
		&c.DBName, &c.Username, &stored, &c.SSLMode, &c.ReadOnly, &opts,
		&c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("dbtool: scan connection: %w", err)
	}
	c.Password = s.decrypt(stored)
	c.HasPassword = stored != ""
	if len(opts) > 0 {
		_ = json.Unmarshal(opts, &c.Options)
	}
	if c.Options == nil {
		c.Options = map[string]any{}
	}
	return c, nil
}

// encrypt wraps plain when a cipher is armed; empty input, already-
// wrapped input, or an unarmed cipher pass through unchanged.
func (s *Store) encrypt(plain string) string {
	if plain == "" || s.cipher == nil || strings.HasPrefix(plain, encryptedFieldPrefix) {
		return plain
	}
	enc, err := s.cipher.EncryptField(plain)
	if err != nil || enc == "" {
		return plain
	}
	return enc
}

// decrypt unwraps an envelope; plaintext passes through, an unreadable
// envelope (no cipher / rotated key) yields "" — the stored ciphertext
// stays intact in the DB, the caller just can't connect until the
// password is re-entered.
func (s *Store) decrypt(stored string) string {
	if !strings.HasPrefix(stored, encryptedFieldPrefix) {
		return stored
	}
	if s.cipher == nil {
		return ""
	}
	plain, err := s.cipher.DecryptField(stored)
	if err != nil {
		return ""
	}
	return plain
}

func marshalOptions(m map[string]any) ([]byte, error) {
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("dbtool: marshal options: %w", err)
	}
	return raw, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
