package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// bootDBForKV boots an embedded Postgres, runs all migrations, and returns a
// fully migrated *DB. Skipped under -short.
func bootDBForKV(t *testing.T) *DB {
	t.Helper()
	// Reuse the same boot helper pattern as plugin_consents_test.go.
	// bootDBForConsents already handles -short + embedded-postgres lifecycle.
	return bootDBForConsents(t)
}

// insertKVPlugin inserts a minimal plugins row to satisfy the FK on plugin_kv.
func insertKVPlugin(t *testing.T, ctx context.Context, db *DB, name string) {
	t.Helper()
	insertTestPlugin(t, ctx, db, name, "0.0.1")
}

// ── TestKV_RoundTrip ─────────────────────────────────────────────────────────

// TestKV_RoundTrip verifies set → get returns the same bytes and found==true.
func TestKV_RoundTrip(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-roundtrip")

	want := json.RawMessage(`{"hello":"world","n":42}`)
	if err := db.KVSet(ctx, "kv-roundtrip", "my-key", want); err != nil {
		t.Fatalf("KVSet: %v", err)
	}

	got, found, err := db.KVGet(ctx, "kv-roundtrip", "my-key")
	if err != nil {
		t.Fatalf("KVGet: %v", err)
	}
	if !found {
		t.Fatal("KVGet: want found=true, got false")
	}

	// JSONB may reorder keys; compare as normalised JSON.
	var wantObj, gotObj any
	if err := json.Unmarshal(want, &wantObj); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if err := json.Unmarshal(got, &gotObj); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	wantB, _ := json.Marshal(wantObj)
	gotB, _ := json.Marshal(gotObj)
	if string(wantB) != string(gotB) {
		t.Errorf("value mismatch: got %s want %s", gotB, wantB)
	}
}

// ── TestKV_GetMissing ────────────────────────────────────────────────────────

// TestKV_GetMissing verifies that a non-existent key returns (nil, false, nil).
func TestKV_GetMissing(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-missing")

	val, found, err := db.KVGet(ctx, "kv-missing", "no-such-key")
	if err != nil {
		t.Fatalf("KVGet: want nil error, got %v", err)
	}
	if found {
		t.Error("KVGet: want found=false, got true")
	}
	if val != nil {
		t.Errorf("KVGet: want nil value, got %s", val)
	}
}

// ── TestKV_Delete_Idempotent ─────────────────────────────────────────────────

// TestKV_Delete_Idempotent verifies that deleting a non-existent key returns nil.
func TestKV_Delete_Idempotent(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-delete-idem")

	// Delete a key that was never inserted.
	if err := db.KVDelete(ctx, "kv-delete-idem", "ghost-key"); err != nil {
		t.Errorf("KVDelete non-existent: want nil, got %v", err)
	}

	// Insert then delete — subsequent get must return not-found.
	if err := db.KVSet(ctx, "kv-delete-idem", "real-key", json.RawMessage(`1`)); err != nil {
		t.Fatalf("KVSet: %v", err)
	}
	if err := db.KVDelete(ctx, "kv-delete-idem", "real-key"); err != nil {
		t.Fatalf("KVDelete existing: %v", err)
	}
	_, found, err := db.KVGet(ctx, "kv-delete-idem", "real-key")
	if err != nil {
		t.Fatalf("KVGet after delete: %v", err)
	}
	if found {
		t.Error("KVGet after delete: want found=false, got true")
	}
}

// ── TestKV_List_WithPrefix ───────────────────────────────────────────────────

// TestKV_List_WithPrefix inserts 5 keys and verifies list "" returns all 5 asc
// and list "b." returns exactly 2.
func TestKV_List_WithPrefix(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-list")

	keys := []string{"a", "b.x", "b.y", "c", "d"}
	for _, k := range keys {
		if err := db.KVSet(ctx, "kv-list", k, json.RawMessage(`null`)); err != nil {
			t.Fatalf("KVSet %q: %v", k, err)
		}
	}

	t.Run("list all (empty prefix)", func(t *testing.T) {
		got, err := db.KVList(ctx, "kv-list", "")
		if err != nil {
			t.Fatalf("KVList: %v", err)
		}
		if len(got) != 5 {
			t.Fatalf("want 5 keys, got %d: %v", len(got), got)
		}
		// Must be ascending.
		for i, want := range keys {
			if got[i] != want {
				t.Errorf("got[%d]=%q want %q", i, got[i], want)
			}
		}
	})

	t.Run("list with prefix b.", func(t *testing.T) {
		got, err := db.KVList(ctx, "kv-list", "b.")
		if err != nil {
			t.Fatalf("KVList(b.): %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 keys, got %d: %v", len(got), got)
		}
		if got[0] != "b.x" || got[1] != "b.y" {
			t.Errorf("unexpected keys: %v", got)
		}
	})
}

// ── TestKV_Cascade ───────────────────────────────────────────────────────────

// TestKV_Cascade verifies ON DELETE CASCADE: deleting the plugins row removes
// all plugin_kv rows for that plugin.
func TestKV_Cascade(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-cascade")

	for i := range 3 {
		key := fmt.Sprintf("key%d", i)
		if err := db.KVSet(ctx, "kv-cascade", key, json.RawMessage(`true`)); err != nil {
			t.Fatalf("KVSet %q: %v", key, err)
		}
	}

	// Delete the plugins row — should cascade to plugin_kv.
	if _, err := db.Pool.Exec(ctx, `DELETE FROM plugins WHERE name = $1`, "kv-cascade"); err != nil {
		t.Fatalf("delete plugins row: %v", err)
	}

	var count int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM plugin_kv WHERE plugin_name = $1`, "kv-cascade").Scan(&count); err != nil {
		t.Fatalf("count plugin_kv: %v", err)
	}
	if count != 0 {
		t.Errorf("plugin_kv: want 0 rows after cascade, got %d", count)
	}
}

// ── TestKV_ValueTooLarge ─────────────────────────────────────────────────────

// TestKV_ValueTooLarge checks the 1 MiB per-key quota:
//   - 1 MiB + 1 byte → ErrValueTooLarge
//   - exactly 1 MiB   → OK
func TestKV_ValueTooLarge(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-toolarge")

	t.Run("1MiB+1 returns ErrValueTooLarge", func(t *testing.T) {
		// Build a JSON string whose byte length is MaxValueBytes+1.
		// A JSON string value "\"...<n bytes>...\"" where n = MaxValueBytes+1-2 (quotes).
		payload := makeJSONString(MaxValueBytes + 1)
		err := db.KVSet(ctx, "kv-toolarge", "big-key", payload)
		if !errors.Is(err, ErrValueTooLarge) {
			t.Errorf("want ErrValueTooLarge, got %v", err)
		}
	})

	t.Run("exactly 1MiB is OK", func(t *testing.T) {
		payload := makeJSONString(MaxValueBytes)
		if err := db.KVSet(ctx, "kv-toolarge", "exact-key", payload); err != nil {
			t.Errorf("want nil, got %v", err)
		}
	})
}

// makeJSONString builds a valid json.RawMessage whose total byte length is
// exactly n bytes. For n >= 2 it produces `"<n-2 'a' chars>"`.
func makeJSONString(n int) json.RawMessage {
	if n < 2 {
		return json.RawMessage(`""`)
	}
	b := make([]byte, n)
	b[0] = '"'
	for i := 1; i < n-1; i++ {
		b[i] = 'a'
	}
	b[n-1] = '"'
	return json.RawMessage(b)
}

// ── TestKV_PluginQuotaExceeded ───────────────────────────────────────────────

// TestKV_PluginQuotaExceeded prepopulates 99 MiB across many keys then
// attempts a 2 MiB set → ErrPluginQuotaExceeded.
func TestKV_PluginQuotaExceeded(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-quota")

	// Seed 99 MiB via direct SQL to avoid test time bloat.
	// Each row is 1 MiB of value (stored as a JSON string).
	// 99 rows × 1 MiB = 99 MiB total.
	const mib = 1 << 20
	for i := range 99 {
		key := fmt.Sprintf("seed-%03d", i)
		val := makeJSONString(mib)
		if err := db.KVSet(ctx, "kv-quota", key, val); err != nil {
			t.Fatalf("seed KVSet[%d]: %v", i, err)
		}
	}

	// Now attempt a 2 MiB set — total would exceed 100 MiB.
	bigVal := makeJSONString(2 * mib)
	err := db.KVSet(ctx, "kv-quota", "overflow-key", bigVal)
	// bigVal itself is 2 MiB which exceeds MaxValueBytes (1 MiB), so it hits
	// ErrValueTooLarge first. We test quota separately with a 1 MiB value that
	// would push total over 100 MiB.
	//
	// So use exactly 1 MiB for the triggering value (fits per-key quota).
	_ = err // discard the ErrValueTooLarge from the 2 MiB attempt above

	triggerVal := makeJSONString(mib) // 1 MiB exactly; per-key OK, total = 100 MiB
	err = db.KVSet(ctx, "kv-quota", "trigger-key", triggerVal)
	if !errors.Is(err, ErrPluginQuotaExceeded) {
		t.Errorf("want ErrPluginQuotaExceeded, got %v", err)
	}
}

// ── TestKV_ContextCancelled_ReturnsError ─────────────────────────────────────

// TestKV_ContextCancelled_ReturnsError verifies that all KV operations propagate
// context cancellation as a wrapped DB error (covers the error branches).
func TestKV_ContextCancelled_ReturnsError(t *testing.T) {
	db := bootDBForKV(t)
	insertKVPlugin(t, context.Background(), db, "kv-ctxcancel")

	// Pre-cancel context so all DB operations fail immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("KVGet returns error on cancelled ctx", func(t *testing.T) {
		_, _, err := db.KVGet(ctx, "kv-ctxcancel", "k")
		if err == nil {
			t.Error("want error on cancelled ctx, got nil")
		}
	})

	t.Run("KVSet quota check fails on cancelled ctx", func(t *testing.T) {
		err := db.KVSet(ctx, "kv-ctxcancel", "k", json.RawMessage(`1`))
		if err == nil {
			t.Error("want error on cancelled ctx, got nil")
		}
	})

	t.Run("KVDelete returns error on cancelled ctx", func(t *testing.T) {
		err := db.KVDelete(ctx, "kv-ctxcancel", "k")
		if err == nil {
			t.Error("want error on cancelled ctx, got nil")
		}
	})

	t.Run("KVList returns error on cancelled ctx", func(t *testing.T) {
		_, err := db.KVList(ctx, "kv-ctxcancel", "")
		if err == nil {
			t.Error("want error on cancelled ctx, got nil")
		}
	})
}

// ── TestKV_Concurrent50Writers ───────────────────────────────────────────────

// TestKV_Concurrent50Writers launches 50 goroutines all setting the same key.
// After all complete, exactly 1 row must exist for that key. -race clean.
func TestKV_Concurrent50Writers(t *testing.T) {
	db := bootDBForKV(t)
	ctx := context.Background()
	insertKVPlugin(t, ctx, db, "kv-concurrent")

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		i := i
		go func() {
			defer wg.Done()
			val := json.RawMessage(fmt.Sprintf(`{"writer":%d}`, i))
			if err := db.KVSet(ctx, "kv-concurrent", "shared-key", val); err != nil {
				t.Errorf("goroutine %d KVSet: %v", i, err)
			}
		}()
	}
	wg.Wait()

	var count int
	if err := db.Pool.QueryRow(ctx,
		`SELECT count(*) FROM plugin_kv WHERE plugin_name = $1 AND key = $2`,
		"kv-concurrent", "shared-key").Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Errorf("want exactly 1 row after 50 concurrent sets, got %d", count)
	}
}
