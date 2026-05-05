package summarizer

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProviderRow is one memory_summarizer_providers row.
//
// APIKeyCiphertext holds the AES-GCM envelope; APIKeyPlaintext is
// only set in-flight (Insert / Update input, decoded for Build()
// output) and never persists. Callers must zero it after use.
type ProviderRow struct {
	ID                 string
	Name               string
	Kind               string // "anthropic" | "ollama"
	Model              string
	BaseURL            string
	APIKeyCiphertext   string  // empty for ollama
	APIKeyPlaintext    string  // never persisted; populated on input or after decryption
	APIKeyFingerprint  string  // first 16 hex of SHA-256(plaintext); shown in UI
	ExtraConfig        map[string]any
	Enabled            bool
	IsDefault          bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CallLogRow is one memory_summarizer_calls row written after every
// invocation. INSERT-only — aggregated by SUM in /cost endpoints.
type CallLogRow struct {
	ID                    string
	RuleID                string // nullable
	ProviderID            string // nullable
	SessionID             string // nullable
	StartedAt             time.Time
	FinishedAt            time.Time
	DurationMs            int
	InputTokens           int
	OutputTokens          int
	EstimatedUSD          float64
	FactsExtracted        int
	FactsStored           int
	FactsSkippedDedup     int
	Status                string // 'succeeded' | 'failed' | 'timeout' | 'provider_unavailable'
	Error                 string
	RawResponseTruncated  string
}

// CostSummary aggregates call-log rows for the /cost endpoint.
type CostSummary struct {
	ProviderID    string
	PeriodStart   time.Time
	PeriodEnd     time.Time
	Calls         int
	InputTokens   int
	OutputTokens  int
	EstimatedUSD  float64
}

// Cipher is the encrypt/decrypt contract we accept from outside —
// duck-typed to whatever backup.Cipher provides. Defined locally
// so this package doesn't import internal/backup directly (avoids
// import cycles + clarifies the dependency surface).
type Cipher interface {
	EncryptField(plain string) (string, error)
	DecryptField(envelope string) (string, error)
}

// Store wraps pgxpool with summarizer-specific CRUD.
type Store struct {
	pool   *pgxpool.Pool
	cipher Cipher // may be nil → ollama-only deployments
}

func NewStore(pool *pgxpool.Pool, cipher Cipher) *Store {
	return &Store{pool: pool, cipher: cipher}
}

// Sentinel errors.
var (
	ErrProviderNotFound      = errors.New("summarizer store: provider not found")
	ErrCallNotFound          = errors.New("summarizer store: call not found")
	ErrCipherRequired        = errors.New("summarizer store: backup cipher required for non-ollama providers (set OPENDRAY_BACKUP_KEY)")
	ErrDuplicateName         = errors.New("summarizer store: provider name already in use")
)

// ─── providers ───────────────────────────────────────────────────

// InsertProvider validates + encrypts the api_key (when present)
// and writes the row. Returns the new row with ciphertext populated
// and plaintext zeroed.
//
// Behaviour around is_default: when row.IsDefault is true, this
// call also clears IsDefault on every existing row before inserting,
// honouring the partial unique index. This lives in a single
// transaction so observers never see a state with two defaults or
// none.
func (s *Store) InsertProvider(ctx context.Context, row ProviderRow) (ProviderRow, error) {
	if row.Kind != "anthropic" && row.Kind != "ollama" {
		return ProviderRow{}, fmt.Errorf("summarizer store: unknown kind %q", row.Kind)
	}
	if row.Name == "" {
		return ProviderRow{}, errors.New("summarizer store: name required")
	}
	if row.Model == "" {
		return ProviderRow{}, errors.New("summarizer store: model required")
	}
	if row.Kind == "ollama" && row.BaseURL == "" {
		return ProviderRow{}, errors.New("summarizer store: ollama provider needs base_url")
	}
	if row.ID == "" {
		row.ID = newProviderID()
	}

	// Encrypt + fingerprint api_key (anthropic only — ollama has none).
	if row.Kind == "anthropic" {
		if row.APIKeyPlaintext == "" {
			return ProviderRow{}, errors.New("summarizer store: anthropic provider needs api_key")
		}
		if s.cipher == nil {
			return ProviderRow{}, ErrCipherRequired
		}
		envelope, err := s.cipher.EncryptField(row.APIKeyPlaintext)
		if err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: encrypt api_key: %w", err)
		}
		row.APIKeyCiphertext = envelope
		row.APIKeyFingerprint = fingerprintAPIKey(row.APIKeyPlaintext)
	}

	extraJSON, err := json.Marshal(row.ExtraConfig)
	if err != nil {
		return ProviderRow{}, fmt.Errorf("summarizer store: marshal extra_config: %w", err)
	}
	if extraJSON == nil || string(extraJSON) == "null" {
		extraJSON = []byte("{}")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProviderRow{}, fmt.Errorf("summarizer store: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if row.IsDefault {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET is_default = FALSE, updated_at = NOW() WHERE is_default = TRUE`); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: clear prior default: %w", err)
		}
	}

	now := time.Now().UTC()
	row.CreatedAt = now
	row.UpdatedAt = now

	_, err = tx.Exec(ctx, `
		INSERT INTO memory_summarizer_providers
			(id, name, kind, model, base_url,
			 api_key_ciphertext, api_key_fingerprint,
			 extra_config, enabled, is_default, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$11)`,
		row.ID, row.Name, row.Kind, row.Model, row.BaseURL,
		nullIfEmpty(row.APIKeyCiphertext), nullIfEmpty(row.APIKeyFingerprint),
		extraJSON, row.Enabled, row.IsDefault, now,
	)
	if err != nil {
		if isUniqueViolationOnName(err) {
			return ProviderRow{}, ErrDuplicateName
		}
		return ProviderRow{}, fmt.Errorf("summarizer store: insert provider: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ProviderRow{}, fmt.Errorf("summarizer store: commit: %w", err)
	}

	row.APIKeyPlaintext = ""
	return row, nil
}

// GetProvider reads the row and (when configured for an anthropic
// row + cipher available) decrypts api_key into APIKeyPlaintext.
// Caller is responsible for zeroing the plaintext after use.
func (s *Store) GetProvider(ctx context.Context, id string) (ProviderRow, error) {
	row := s.pool.QueryRow(ctx, providerSelectStmt+` WHERE id = $1`, id)
	got, err := s.scanProvider(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderRow{}, ErrProviderNotFound
	}
	return got, err
}

// ListProviders returns all rows in stable order (created_at ASC).
// Plaintext is NOT decrypted in list responses; callers wanting to
// instantiate one provider call GetProvider for that single row.
func (s *Store) ListProviders(ctx context.Context) ([]ProviderRow, error) {
	rows, err := s.pool.Query(ctx, providerSelectStmt+` ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("summarizer store: list providers: %w", err)
	}
	defer rows.Close()
	var out []ProviderRow
	for rows.Next() {
		r, err := s.scanProviderListRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetDefaultProvider returns the row marked is_default=TRUE.
// Returns ErrProviderNotFound when no row is marked.
func (s *Store) GetDefaultProvider(ctx context.Context) (ProviderRow, error) {
	row := s.pool.QueryRow(ctx, providerSelectStmt+` WHERE is_default = TRUE LIMIT 1`)
	got, err := s.scanProvider(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderRow{}, ErrProviderNotFound
	}
	return got, err
}

// ProviderPatch carries optional updates.
type ProviderPatch struct {
	Name              *string
	Model             *string
	BaseURL           *string
	APIKeyPlaintext   *string // when set, re-encrypt + bump fingerprint
	ExtraConfig       map[string]any
	Enabled           *bool
	IsDefault         *bool
}

func (s *Store) UpdateProvider(ctx context.Context, id string, p ProviderPatch) (ProviderRow, error) {
	cur, err := s.GetProvider(ctx, id)
	if err != nil {
		return ProviderRow{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProviderRow{}, fmt.Errorf("summarizer store: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// If IsDefault flipping to true, clear other defaults first.
	if p.IsDefault != nil && *p.IsDefault && !cur.IsDefault {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET is_default = FALSE, updated_at = NOW()
			   WHERE is_default = TRUE AND id <> $1`, id); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: clear prior default: %w", err)
		}
	}

	if p.Name != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET name=$1, updated_at=NOW() WHERE id=$2`, *p.Name, id); err != nil {
			if isUniqueViolationOnName(err) {
				return ProviderRow{}, ErrDuplicateName
			}
			return ProviderRow{}, fmt.Errorf("summarizer store: update name: %w", err)
		}
	}
	if p.Model != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET model=$1, updated_at=NOW() WHERE id=$2`, *p.Model, id); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: update model: %w", err)
		}
	}
	if p.BaseURL != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET base_url=$1, updated_at=NOW() WHERE id=$2`, *p.BaseURL, id); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: update base_url: %w", err)
		}
	}
	if p.APIKeyPlaintext != nil {
		if cur.Kind == "ollama" {
			return ProviderRow{}, errors.New("summarizer store: ollama provider has no api_key")
		}
		if s.cipher == nil {
			return ProviderRow{}, ErrCipherRequired
		}
		envelope, err := s.cipher.EncryptField(*p.APIKeyPlaintext)
		if err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: encrypt api_key: %w", err)
		}
		fp := fingerprintAPIKey(*p.APIKeyPlaintext)
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers
			    SET api_key_ciphertext=$1, api_key_fingerprint=$2, updated_at=NOW()
			  WHERE id=$3`, envelope, fp, id); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: update api_key: %w", err)
		}
	}
	if p.ExtraConfig != nil {
		raw, err := json.Marshal(p.ExtraConfig)
		if err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: marshal extra_config: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET extra_config=$1::jsonb, updated_at=NOW() WHERE id=$2`,
			raw, id); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: update extra_config: %w", err)
		}
	}
	if p.Enabled != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET enabled=$1, updated_at=NOW() WHERE id=$2`, *p.Enabled, id); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: update enabled: %w", err)
		}
	}
	if p.IsDefault != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE memory_summarizer_providers SET is_default=$1, updated_at=NOW() WHERE id=$2`, *p.IsDefault, id); err != nil {
			return ProviderRow{}, fmt.Errorf("summarizer store: update is_default: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ProviderRow{}, fmt.Errorf("summarizer store: commit: %w", err)
	}
	return s.GetProvider(ctx, id)
}

func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	cmd, err := s.pool.Exec(ctx, `DELETE FROM memory_summarizer_providers WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("summarizer store: delete provider: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrProviderNotFound
	}
	return nil
}

// ─── call log ────────────────────────────────────────────────────

// LogCall records one finished provider call. Insert-only.
func (s *Store) LogCall(ctx context.Context, row CallLogRow) error {
	if row.ID == "" {
		row.ID = newCallLogID()
	}
	if row.FinishedAt.IsZero() {
		row.FinishedAt = time.Now().UTC()
	}
	if row.DurationMs == 0 && !row.StartedAt.IsZero() {
		row.DurationMs = int(row.FinishedAt.Sub(row.StartedAt).Milliseconds())
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO memory_summarizer_calls
			(id, rule_id, provider_id, session_id,
			 started_at, finished_at, duration_ms,
			 input_tokens, output_tokens, estimated_usd,
			 facts_extracted, facts_stored, facts_skipped_dedup,
			 status, error, raw_response_truncated)
		VALUES ($1, $2, $3, $4,
		        $5, $6, $7,
		        $8, $9, $10,
		        $11, $12, $13,
		        $14, $15, $16)`,
		row.ID, nullIfEmpty(row.RuleID), nullIfEmpty(row.ProviderID), nullIfEmpty(row.SessionID),
		row.StartedAt, row.FinishedAt, row.DurationMs,
		row.InputTokens, row.OutputTokens, row.EstimatedUSD,
		row.FactsExtracted, row.FactsStored, row.FactsSkippedDedup,
		row.Status, nullIfEmpty(row.Error), nullIfEmpty(row.RawResponseTruncated),
	)
	if err != nil {
		return fmt.Errorf("summarizer store: insert call log: %w", err)
	}
	return nil
}

// ProviderCostSince aggregates call_log rows for one provider since
// `since`. Drives the /cost admin endpoint.
func (s *Store) ProviderCostSince(ctx context.Context, providerID string, since time.Time) (CostSummary, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT
		  COUNT(*),
		  COALESCE(SUM(input_tokens), 0),
		  COALESCE(SUM(output_tokens), 0),
		  COALESCE(SUM(estimated_usd), 0)
		FROM memory_summarizer_calls
		WHERE provider_id = $1
		  AND started_at >= $2`,
		providerID, since,
	)
	var cs CostSummary
	if err := row.Scan(&cs.Calls, &cs.InputTokens, &cs.OutputTokens, &cs.EstimatedUSD); err != nil {
		return CostSummary{}, fmt.Errorf("summarizer store: cost since: %w", err)
	}
	cs.ProviderID = providerID
	cs.PeriodStart = since
	cs.PeriodEnd = time.Now().UTC()
	return cs, nil
}

// ─── helpers ─────────────────────────────────────────────────────

const providerSelectStmt = `
	SELECT id, name, kind, model, base_url,
	       COALESCE(api_key_ciphertext, ''),
	       COALESCE(api_key_fingerprint, ''),
	       COALESCE(extra_config, '{}'::jsonb),
	       enabled, is_default,
	       created_at, updated_at
	  FROM memory_summarizer_providers`

type rowScanner interface {
	Scan(dest ...any) error
}

// scanProvider reads a row and decrypts api_key if cipher is
// available (anthropic provider). Used for single-row fetches
// where the caller wants to instantiate a Provider.
func (s *Store) scanProvider(row rowScanner) (ProviderRow, error) {
	r, err := s.scanProviderRow(row)
	if err != nil {
		return ProviderRow{}, err
	}
	if r.APIKeyCiphertext != "" && s.cipher != nil {
		plain, derr := s.cipher.DecryptField(r.APIKeyCiphertext)
		if derr != nil {
			return r, fmt.Errorf("summarizer store: decrypt api_key for %q: %w", r.ID, derr)
		}
		r.APIKeyPlaintext = plain
	}
	return r, nil
}

// scanProviderListRow is the same as scanProvider minus decryption —
// callers iterating multiple rows shouldn't pull plaintext into RAM.
func (s *Store) scanProviderListRow(row rowScanner) (ProviderRow, error) {
	return s.scanProviderRow(row)
}

func (s *Store) scanProviderRow(row rowScanner) (ProviderRow, error) {
	var (
		r        ProviderRow
		extraRaw []byte
	)
	err := row.Scan(
		&r.ID, &r.Name, &r.Kind, &r.Model, &r.BaseURL,
		&r.APIKeyCiphertext, &r.APIKeyFingerprint,
		&extraRaw, &r.Enabled, &r.IsDefault,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return ProviderRow{}, err
	}
	if len(extraRaw) > 0 {
		_ = json.Unmarshal(extraRaw, &r.ExtraConfig)
	}
	if r.ExtraConfig == nil {
		r.ExtraConfig = map[string]any{}
	}
	return r, nil
}

// nullIfEmpty turns "" into pgx's nil so the column stores NULL.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// fingerprintAPIKey returns the first 16 hex chars of SHA-256(plain) —
// shown in the UI so operators can verify "the saved key matches
// the one in my password manager" without exposing the secret.
func fingerprintAPIKey(plain string) string {
	if plain == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:8])
}

// newProviderID is "msp_" + 22-char base32 (~110 bits entropy).
func newProviderID() string { return idWithPrefix("msp_") }

// newCallLogID is "msc_" + 22-char base32.
func newCallLogID() string { return idWithPrefix("msc_") }

func idWithPrefix(prefix string) string {
	var b [14]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("summarizer store: rand.Read: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return prefix + strings.ToLower(enc)
}

func isUniqueViolationOnName(err error) bool {
	if err == nil {
		return false
	}
	// pgconn.PgError SQLSTATE 23505 is unique_violation; the
	// constraint we care about is memory_summarizer_providers_name_key.
	if strings.Contains(err.Error(), "23505") &&
		strings.Contains(err.Error(), "name") {
		return true
	}
	return false
}

// silence imports we don't use inside hot paths but want available.
var _ = sql.NullString{}
