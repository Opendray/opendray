package summarizer

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// devDB returns a pgxpool against the dev database addressed by the
// OPENDRAY_DEV_DB_URL env var. Tests that depend on a real Postgres
// skip cleanly when the env is unset (CI default) or when the DB is
// unreachable.
//
// Set OPENDRAY_DEV_DB_URL to a writable Postgres DSN to run these
// tests against your own database. The DSN must NOT be hard-coded
// here — committed credentials are forever.
func devDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("OPENDRAY_DEV_DB_URL")
	if url == "" {
		t.Skip("OPENDRAY_DEV_DB_URL not set; export a writable Postgres DSN to run this test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Skipf("dev DB unreachable, skipping: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("dev DB ping failed, skipping: %v", err)
	}
	return pool
}

// fakeCipher is a minimal Cipher impl for unit tests — XOR with a
// fixed key, prefixed so we know it round-trips. Never use in prod.
type fakeCipher struct{}

func (fakeCipher) EncryptField(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	return "fake:" + plain, nil
}
func (fakeCipher) DecryptField(envelope string) (string, error) {
	if envelope == "" {
		return "", nil
	}
	if !strings.HasPrefix(envelope, "fake:") {
		return "", errors.New("fakeCipher: bad envelope")
	}
	return strings.TrimPrefix(envelope, "fake:"), nil
}

func resetSummarizerTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, _ = pool.Exec(context.Background(), `DELETE FROM memory_summarizer_calls`)
	_, _ = pool.Exec(context.Background(), `DELETE FROM memory_summarizer_providers`)
}

func TestStore_InsertProvider_AnthropicNeedsCipher(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	storeNoKey := NewStore(pool, nil)
	_, err := storeNoKey.InsertProvider(context.Background(), ProviderRow{
		Name: "haiku-test", Kind: "anthropic", Model: "claude-haiku-4-5",
		APIKeyPlaintext: "test-key", Enabled: true,
	})
	if !errors.Is(err, ErrCipherRequired) {
		t.Errorf("got %v, want ErrCipherRequired", err)
	}
}

func TestStore_OllamaNoCipherRequired(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	storeNoKey := NewStore(pool, nil)
	row, err := storeNoKey.InsertProvider(context.Background(), ProviderRow{
		Name: "ollama-test", Kind: "ollama",
		Model:   "qwen2.5:7b",
		BaseURL: "http://localhost:11434",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("ollama insert without cipher should succeed: %v", err)
	}
	if row.ID == "" {
		t.Error("ID should be auto-generated")
	}
}

func TestStore_RoundTrip_Anthropic(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	store := NewStore(pool, fakeCipher{})

	row, err := store.InsertProvider(context.Background(), ProviderRow{
		Name: "haiku-rt", Kind: "anthropic", Model: "claude-haiku-4-5",
		APIKeyPlaintext: "the-real-secret", Enabled: true, IsDefault: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.APIKeyPlaintext != "" {
		t.Error("InsertProvider should zero plaintext on return")
	}
	if !strings.HasPrefix(row.APIKeyCiphertext, "fake:") {
		t.Errorf("ciphertext should be prefixed: %q", row.APIKeyCiphertext)
	}
	if row.APIKeyFingerprint == "" {
		t.Error("fingerprint should be set")
	}

	// Fetch + decrypt
	got, err := store.GetProvider(context.Background(), row.ID)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if got.APIKeyPlaintext != "the-real-secret" {
		t.Errorf("decrypted plaintext = %q, want 'the-real-secret'", got.APIKeyPlaintext)
	}
	if !got.IsDefault {
		t.Error("IsDefault should persist")
	}
}

func TestStore_DefaultExclusivity(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	store := NewStore(pool, fakeCipher{})
	a, _ := store.InsertProvider(context.Background(), ProviderRow{
		Name: "p1", Kind: "anthropic", Model: "claude-haiku-4-5",
		APIKeyPlaintext: "k1", Enabled: true, IsDefault: true,
	})
	b, _ := store.InsertProvider(context.Background(), ProviderRow{
		Name: "p2", Kind: "anthropic", Model: "claude-haiku-4-5",
		APIKeyPlaintext: "k2", Enabled: true, IsDefault: true,
	})

	got1, _ := store.GetProvider(context.Background(), a.ID)
	got2, _ := store.GetProvider(context.Background(), b.ID)
	if got1.IsDefault {
		t.Error("a should have lost default when b inserted with IsDefault=true")
	}
	if !got2.IsDefault {
		t.Error("b should be the default")
	}
}

func TestStore_DuplicateName(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	store := NewStore(pool, fakeCipher{})
	_, err := store.InsertProvider(context.Background(), ProviderRow{
		Name: "dup", Kind: "ollama", Model: "x", BaseURL: "http://localhost",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.InsertProvider(context.Background(), ProviderRow{
		Name: "dup", Kind: "ollama", Model: "y", BaseURL: "http://localhost",
	})
	if !errors.Is(err, ErrDuplicateName) {
		t.Errorf("got %v, want ErrDuplicateName", err)
	}
}

func TestStore_LogCall_AndAggregate(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	store := NewStore(pool, fakeCipher{})
	prov, _ := store.InsertProvider(context.Background(), ProviderRow{
		Name: "haiku-stats", Kind: "anthropic", Model: "claude-haiku-4-5",
		APIKeyPlaintext: "k", Enabled: true,
	})

	if err := store.LogCall(context.Background(), CallLogRow{
		ProviderID:     prov.ID,
		Status:         "succeeded",
		InputTokens:    100,
		OutputTokens:   25,
		EstimatedUSD:   0.000225,
		FactsExtracted: 3,
		FactsStored:    2,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.LogCall(context.Background(), CallLogRow{
		ProviderID:   prov.ID,
		Status:       "succeeded",
		InputTokens:  200,
		OutputTokens: 50,
		EstimatedUSD: 0.00045,
		FactsStored:  4,
	}); err != nil {
		t.Fatal(err)
	}

	cs, err := store.ProviderCostSince(context.Background(), prov.ID, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if cs.Calls != 2 {
		t.Errorf("Calls = %d, want 2", cs.Calls)
	}
	if cs.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", cs.InputTokens)
	}
	if cs.OutputTokens != 75 {
		t.Errorf("OutputTokens = %d, want 75", cs.OutputTokens)
	}
	// 0.000225 + 0.00045 = 0.000675
	if cs.EstimatedUSD < 0.000674 || cs.EstimatedUSD > 0.000676 {
		t.Errorf("EstimatedUSD = %g, want ~0.000675", cs.EstimatedUSD)
	}
}

func TestStore_Update_ReencryptsAPIKey(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	store := NewStore(pool, fakeCipher{})
	row, _ := store.InsertProvider(context.Background(), ProviderRow{
		Name: "haiku-upd", Kind: "anthropic", Model: "claude-haiku-4-5",
		APIKeyPlaintext: "old-key",
	})
	originalFp := row.APIKeyFingerprint

	newKey := "new-rotated-key"
	if _, err := store.UpdateProvider(context.Background(), row.ID, ProviderPatch{
		APIKeyPlaintext: &newKey,
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := store.GetProvider(context.Background(), row.ID)
	if got.APIKeyPlaintext != newKey {
		t.Errorf("updated plaintext = %q, want %q", got.APIKeyPlaintext, newKey)
	}
	if got.APIKeyFingerprint == originalFp {
		t.Error("fingerprint should change on rotation")
	}
}

func TestStore_DeleteProvider(t *testing.T) {
	pool := devDB(t)
	defer pool.Close()
	resetSummarizerTables(t, pool)

	store := NewStore(pool, fakeCipher{})
	row, _ := store.InsertProvider(context.Background(), ProviderRow{
		Name: "to-del", Kind: "ollama", Model: "x", BaseURL: "http://localhost",
	})
	if err := store.DeleteProvider(context.Background(), row.ID); err != nil {
		t.Fatal(err)
	}
	_, err := store.GetProvider(context.Background(), row.ID)
	if !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("got %v, want ErrProviderNotFound", err)
	}
	// Idempotent? No — second delete returns ErrProviderNotFound.
	if err := store.DeleteProvider(context.Background(), row.ID); !errors.Is(err, ErrProviderNotFound) {
		t.Errorf("second delete should return ErrProviderNotFound, got %v", err)
	}
}
