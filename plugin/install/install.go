package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/opendray/opendray/kernel/store"
	"github.com/opendray/opendray/plugin"
	"github.com/opendray/opendray/plugin/bridge"
)

// Default TTL for a pending install — user has 10 minutes to confirm
// after staging before the token expires.
const defaultPendingTTL = 10 * time.Minute

// Default reap interval for the janitor goroutine.
const defaultReapInterval = 1 * time.Minute

// Sentinel errors callers can switch on.
var (
	// ErrTokenNotFound is returned by Confirm when the token is absent
	// or expired. The janitor may have already cleaned up the staged dir
	// by the time this fires — Confirm does not try to resurrect it.
	ErrTokenNotFound = errors.New("install: token not found or expired")

	// ErrInvalidManifest is returned by Stage when the manifest fails v1
	// validation or when a legacy (non-v1) manifest is submitted. The
	// install flow is v1-only; legacy plugins are handled by the compat
	// synthesiser (T12) at startup, not through Installer.
	ErrInvalidManifest = errors.New("install: manifest failed v1 validation")

	// ErrLocalDisabled is returned by Stage when the source is a local-scheme
	// source (local:<abs> or bare absolute path) and AllowLocal is false.
	// Callers should surface this as HTTP 403 EFORBIDDEN. Set AllowLocal=true
	// (via OPENDRAY_ALLOW_LOCAL_PLUGINS=1) to permit local installs.
	ErrLocalDisabled = errors.New("install: local plugin installs disabled")
)

// Installer orchestrates the plugin install flow.
//
// Lifecycle: Stage → Confirm (or let TTL expire → janitor reaps). Uninstall
// is idempotent. The installer owns one background janitor goroutine; call
// Stop() when shutting down to avoid leaking it.
type Installer struct {
	DataDir string
	DB      *store.DB
	Runtime *plugin.Runtime
	Gate    *bridge.Gate
	Log     *slog.Logger

	// AllowLocal gates local-scheme sources in Stage. When false (the default),
	// Stage returns ErrLocalDisabled for any LocalSource. Set to true when
	// OPENDRAY_ALLOW_LOCAL_PLUGINS=1 (or equivalent) is configured.
	// This field is safe to set before the first Stage call; concurrent
	// mutation is not supported.
	AllowLocal bool // T25: controlled by kernel/config.Config.AllowLocalPlugins

	pending *pendingStore

	// janitor lifecycle
	jctx    context.Context
	jcancel context.CancelFunc
	jwg     sync.WaitGroup
	once    sync.Once
}

// NewInstaller constructs an Installer with the default 10-minute TTL and
// 1-minute reap interval. A background janitor starts immediately; call
// Stop() to clean it up.
func NewInstaller(dataDir string, db *store.DB, rt *plugin.Runtime, gate *bridge.Gate, log *slog.Logger) *Installer {
	return NewInstallerWithTTL(dataDir, db, rt, gate, log, defaultPendingTTL, defaultReapInterval)
}

// NewInstallerWithTTL is like NewInstaller but with caller-supplied TTL
// and reap interval. Test-only convenience that keeps production
// construction site from having to specify timing.
func NewInstallerWithTTL(dataDir string, db *store.DB, rt *plugin.Runtime, gate *bridge.Gate, log *slog.Logger, ttl, reapEvery time.Duration) *Installer {
	if log == nil {
		log = slog.Default()
	}
	if ttl <= 0 {
		ttl = defaultPendingTTL
	}
	if reapEvery <= 0 {
		reapEvery = defaultReapInterval
	}
	jctx, jcancel := context.WithCancel(context.Background())
	i := &Installer{
		DataDir: dataDir,
		DB:      db,
		Runtime: rt,
		Gate:    gate,
		Log:     log,
		pending: newPendingStore(ttl, time.Now),
		jctx:    jctx,
		jcancel: jcancel,
	}
	i.jwg.Add(1)
	go i.janitor(reapEvery)
	return i
}

// Stop terminates the janitor. Idempotent — subsequent calls are no-ops.
func (i *Installer) Stop() {
	i.once.Do(func() {
		i.jcancel()
		i.jwg.Wait()
	})
}

// Stage fetches the bundle, validates the v1 manifest, records a
// PendingInstall, and returns the token the caller must echo back to
// Confirm. The staged directory lives inside DataDir so Confirm's
// os.Rename is guaranteed to be intra-filesystem.
func (i *Installer) Stage(ctx context.Context, src Source) (*PendingInstall, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 0) T25 local-scheme gate. Check before any I/O so we never create
	// staging artefacts for a denied request.
	if _, isLocal := src.(LocalSource); isLocal && !i.AllowLocal {
		return nil, fmt.Errorf("%w: set OPENDRAY_ALLOW_LOCAL_PLUGINS=1 to enable", ErrLocalDisabled)
	}

	// 1) Fetch the source into a temp dir (external to DataDir).
	fetchedPath, cleanupFetched, err := src.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("install: fetch: %w", err)
	}
	// Fetched path is moved into DataDir below; if Stage fails we still
	// want the fetched copy gone.
	defer func() {
		if cleanupFetched != nil {
			cleanupFetched()
		}
	}()

	// 2) Parse + validate the manifest.
	prov, err := plugin.LoadManifest(fetchedPath)
	if err != nil {
		return nil, fmt.Errorf("install: load manifest: %w", err)
	}
	if !prov.IsV1() {
		// Legacy manifests are rejected by Installer on purpose — they
		// flow through the compat path (T12), not through here.
		// Spec: plugin/install is v1-only; legacy panels remain via
		// compat-mode synthesis (see docs/plugin-platform/07-lifecycle.md
		// §Compat mode).
		return nil, fmt.Errorf("%w: manifest is not v1 (missing publisher or engines.opendray)", ErrInvalidManifest)
	}
	if ve := plugin.ValidateV1(prov); len(ve) > 0 {
		msgs := make([]string, 0, len(ve))
		for _, e := range ve {
			msgs = append(msgs, e.Error())
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, msgs)
	}

	// 3) Compute the canonical manifest hash. We pin this into
	// plugin_consents so later loads / updates can detect drift.
	manifestHash, err := SHA256CanonicalManifest(prov)
	if err != nil {
		return nil, fmt.Errorf("install: hash manifest: %w", err)
	}

	// 4) Move the fetched dir into a staging-* subdir of DataDir. This
	// guarantees os.Rename on Confirm is within the same filesystem.
	if err := os.MkdirAll(i.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("install: ensure data dir: %w", err)
	}
	stagingParent, err := os.MkdirTemp(i.DataDir, "staging-*")
	if err != nil {
		return nil, fmt.Errorf("install: mkdir staging: %w", err)
	}
	stagedPath := filepath.Join(stagingParent, "bundle")
	// Rename, not copy, avoids a double-copy now that Fetch already made
	// the bundle local. Cross-filesystem will fail here — but MkdirTemp
	// under DataDir guarantees intra-fs.
	if err := os.Rename(fetchedPath, stagedPath); err != nil {
		// Fall back to copy if Rename fails (e.g. fetchedPath still on
		// os.TempDir and DataDir is on a different volume).
		if copyErr := copyTree(ctx, fetchedPath, stagedPath); copyErr != nil {
			_ = os.RemoveAll(stagingParent)
			return nil, fmt.Errorf("install: stage bundle: %w", copyErr)
		}
	} else {
		// Rename succeeded — the fetchedPath no longer exists, so the
		// cleanup func would be a no-op but it's cheap to skip.
		cleanupFetched = nil
	}

	// 5) Build + register the pending entry.
	perms := plugin.PermissionsV1{}
	if prov.Permissions != nil {
		perms = *prov.Permissions
	}
	now := i.pending.now()
	pend := &PendingInstall{
		Token:        newToken(),
		Name:         prov.Name,
		Version:      prov.Version,
		ManifestHash: manifestHash,
		Perms:        perms,
		StagedPath:   stagedPath,
		ExpiresAt:    now.Add(i.pending.ttl),
	}
	i.pending.put(pend)

	// 6) Audit row for the stage event. Do NOT hash raw paths; use the
	// source's Describe() string so audit never leaks user filesystem
	// layout. argsHash is truncated to 16 hex chars for readability.
	i.writeAudit(ctx, store.AuditEntry{
		PluginName: prov.Name,
		Ns:         "install",
		Method:     "stage",
		Result:     "ok",
		ArgsHash:   shortHash(src.Describe()),
	})

	return pend, nil
}

// Confirm looks up token, moves the staged dir to its final home, writes
// plugins + plugin_consents in a transaction, and registers the provider
// with the runtime. On any failure Confirm is all-or-nothing: either the
// DB rows + filesystem move + in-memory registration all happen, or none
// of them do. The staged dir survives a failure so the caller can retry
// after fixing whatever blocked the install (e.g. cleaning out the final
// path).
//
// NOTE — rollback semantics: The spec calls out that the DB transaction
// must roll back cleanly when os.Rename fails. We sequence the operations
// as [rename → tx.commit → Runtime.Register] so that the costly
// filesystem move happens BEFORE any DB commit. If Rename fails, the tx
// is rolled back via a deferred rollback call (pgx treats rollback as a
// no-op after commit, so this is safe either way). If Runtime.Register
// fails (which is effectively impossible given it only writes to an
// in-memory map and the plugins row we just committed), we log and
// surface the error but the DB state is left untouched — M1's runtime
// supervisor will reconcile on next restart.
func (i *Installer) Confirm(ctx context.Context, token string) error {
	pend, ok := i.pending.take(token)
	if !ok {
		return ErrTokenNotFound
	}

	finalDir := filepath.Join(i.DataDir, pend.Name, pend.Version)

	// Serialize manifest for plugins row.
	provJSON, err := i.readStagedManifestJSON(pend.StagedPath)
	if err != nil {
		// Re-insert the pending entry so the caller can retry after
		// inspecting the staged dir. We lose the original TTL here, but
		// a read-manifest failure on our own staged dir is unexpected
		// enough that returning a clear error is more important.
		i.pending.put(pend)
		return fmt.Errorf("install: read staged manifest: %w", err)
	}

	// Serialize permissions for plugin_consents row.
	permsJSON, err := json.Marshal(pend.Perms)
	if err != nil {
		i.pending.put(pend)
		return fmt.Errorf("install: marshal perms: %w", err)
	}

	tx, err := i.DB.Pool.Begin(ctx)
	if err != nil {
		i.pending.put(pend)
		return fmt.Errorf("install: begin tx: %w", err)
	}
	// Rollback is a no-op after successful commit, so we always defer it.
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Write plugins row.
	if _, err := tx.Exec(ctx,
		`INSERT INTO plugins (name, version, manifest, enabled)
		 VALUES ($1, $2, $3, true)
		 ON CONFLICT (name) DO UPDATE SET
		     version = EXCLUDED.version,
		     manifest = EXCLUDED.manifest,
		     enabled  = true,
		     updated_at = now()`,
		pend.Name, pend.Version, provJSON,
	); err != nil {
		i.pending.put(pend)
		return fmt.Errorf("install: upsert plugin row: %w", err)
	}

	// Write plugin_consents row.
	if _, err := tx.Exec(ctx,
		`INSERT INTO plugin_consents (plugin_name, manifest_hash, perms_json)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (plugin_name) DO UPDATE SET
		     manifest_hash = EXCLUDED.manifest_hash,
		     perms_json    = EXCLUDED.perms_json,
		     updated_at    = now()`,
		pend.Name, pend.ManifestHash, permsJSON,
	); err != nil {
		i.pending.put(pend)
		return fmt.Errorf("install: upsert consent row: %w", err)
	}

	// Ensure the final path's parent exists, then move the staged dir.
	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		i.pending.put(pend)
		return fmt.Errorf("install: mkdir final parent: %w", err)
	}

	// If the final path already exists, relocate it to .trash so we can
	// still Rename atomically. Trash cleanup is out of scope for M1.
	if _, err := os.Stat(finalDir); err == nil {
		trashName := fmt.Sprintf("%s-%s-%d", pend.Name, pend.Version, time.Now().UnixNano())
		trashDir := filepath.Join(i.DataDir, ".trash")
		if err := os.MkdirAll(trashDir, 0o755); err != nil {
			i.pending.put(pend)
			return fmt.Errorf("install: mkdir trash: %w", err)
		}
		if err := os.Rename(finalDir, filepath.Join(trashDir, trashName)); err != nil {
			// Rollback semantics documented above: tx rollback + keep
			// staged dir for retry. Re-register the pending entry so
			// the caller can attempt Confirm again after resolving the
			// conflict.
			i.pending.put(pend)
			return fmt.Errorf("install: trash existing final: %w", err)
		}
	}

	if err := os.Rename(pend.StagedPath, finalDir); err != nil {
		// Rename failed → tx rolls back via the deferred closure; staged
		// dir is preserved for retry. Re-insert pending entry with the
		// same token so the caller can retry after fixing the target.
		i.pending.put(pend)
		return fmt.Errorf("install: move staged → final: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		// We already moved the dir. Best-effort: put it back. If that
		// fails too, the filesystem state is ahead of the DB — log and
		// surface; runtime supervisor will reconcile.
		if rbErr := os.Rename(finalDir, pend.StagedPath); rbErr != nil {
			i.Log.Error("install: confirm rollback filesystem failed", "err", rbErr)
		}
		i.pending.put(pend)
		return fmt.Errorf("install: commit tx: %w", err)
	}
	committed = true

	// DB + FS committed. Now register with the runtime — in-memory only,
	// can't fail the install at this point.
	prov, err := plugin.LoadManifest(finalDir)
	if err != nil {
		i.Log.Error("install: reload manifest after confirm", "err", err, "plugin", pend.Name)
	} else {
		if err := i.Runtime.Register(ctx, prov); err != nil {
			i.Log.Error("install: runtime register", "err", err, "plugin", pend.Name)
		}
	}

	// Remove the now-empty staging-* parent dir so it doesn't linger.
	_ = os.Remove(filepath.Dir(pend.StagedPath))

	// Audit.
	i.writeAudit(ctx, store.AuditEntry{
		PluginName: pend.Name,
		Ns:         "install",
		Method:     "confirm",
		Result:     "ok",
		ArgsHash:   shortHash(pend.Token[:16]), // token is already a secret; short-hashing it protects the audit log
	})

	return nil
}

// Uninstall removes a plugin end-to-end: runtime deregistration, DB
// cascade via DeletePlugin (which cascades plugin_consents), and
// filesystem deletion of the extracted bundle. Audit rows are
// PRESERVED — they are historical and not keyed to the plugin FK.
//
// Uninstall is idempotent: calling it on an unknown plugin is not an
// error. The audit row is still written with Result="ok" so the
// historical trail records that an uninstall command was issued.
func (i *Installer) Uninstall(ctx context.Context, name string) error {
	// Best-effort runtime remove. It errors on unknown plugins, which we
	// treat as non-fatal for idempotency.
	_ = i.Runtime.Remove(ctx, name)

	// DeleteConsent first so a FK cascade on plugins doesn't surprise us.
	// DeleteConsent is idempotent.
	if err := i.DB.DeleteConsent(ctx, name); err != nil {
		return fmt.Errorf("install: delete consent: %w", err)
	}

	// DeletePlugin — the plugins table row. Cascades plugin_consents too
	// (the row is already gone, but the cascade costs nothing).
	if err := i.DB.DeletePlugin(ctx, name); err != nil {
		return fmt.Errorf("install: delete plugin: %w", err)
	}

	// Remove the filesystem tree for every version of this plugin. We
	// delete the whole parent dir (${DataDir}/${name}) because M1 only
	// keeps a single version per plugin.
	pluginDir := filepath.Join(i.DataDir, name)
	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("install: remove plugin dir: %w", err)
	}

	i.writeAudit(ctx, store.AuditEntry{
		PluginName: name,
		Ns:         "install",
		Method:     "uninstall",
		Result:     "ok",
		ArgsHash:   shortHash(name),
	})

	return nil
}

// janitor reaps expired pending installs on a fixed interval until
// Installer.Stop() cancels its context.
func (i *Installer) janitor(reapEvery time.Duration) {
	defer i.jwg.Done()
	ticker := time.NewTicker(reapEvery)
	defer ticker.Stop()
	for {
		select {
		case <-i.jctx.Done():
			return
		case <-ticker.C:
			paths := i.pending.reap()
			for _, p := range paths {
				// Delete the staging-* parent directory so the whole
				// per-stage temp space is gone.
				_ = os.RemoveAll(filepath.Dir(p))
			}
		}
	}
}

// PeekName returns the plugin name for a pending token without consuming it.
// Returns ("", false) when the token is absent or expired. This is used by
// the HTTP confirm handler to include the name in the response body, since
// Confirm itself only returns an error.
func (i *Installer) PeekName(token string) (string, bool) {
	i.pending.mu.Lock()
	defer i.pending.mu.Unlock()
	p, ok := i.pending.byToken[token]
	if !ok {
		return "", false
	}
	if i.pending.now().After(p.ExpiresAt) {
		return "", false
	}
	return p.Name, true
}

// pendingCount exposes the in-memory count for tests — keeps tests from
// having to poke at unexported fields.
func (i *Installer) pendingCount() int {
	return i.pending.count()
}

// writeAudit best-effort logs an audit row. Audit failures must not break
// the primary install flow — we log them at warn level and move on.
func (i *Installer) writeAudit(ctx context.Context, e store.AuditEntry) {
	if i.DB == nil {
		return
	}
	if err := i.DB.AppendAudit(ctx, e); err != nil {
		i.Log.Warn("install: audit append failed", "err", err, "plugin", e.PluginName, "method", e.Method)
	}
}

// readStagedManifestJSON re-reads manifest.json from the staged bundle and
// returns its raw bytes so we can write it verbatim into the plugins row.
// Using the raw bytes (not a re-marshal of Provider) preserves whatever
// v2Reserved/unknown fields the author shipped — forward compat.
func (i *Installer) readStagedManifestJSON(bundleDir string) (json.RawMessage, error) {
	b, err := os.ReadFile(filepath.Join(bundleDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// shortHash returns the first 16 hex chars of sha256(s). Matches the
// convention used by the capability gate audit rows: never log raw
// arguments, always log a short hash so correlation is possible without
// leaking secrets.
func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:16]
}
