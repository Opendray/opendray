package checkpoint

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionInfo is the minimal view of a session the checkpoint service needs
// to capture context on demand (the manual trigger). The session manager
// satisfies it without checkpoint importing the session package's types.
type SessionInfo interface {
	// CheckpointContext returns the working directory and the operator
	// input-history tail for a session id, or ok=false if unknown.
	CheckpointContext(id string) (cwd string, input []byte, ok bool)
}

// Service captures, lists, reads and reaps session checkpoints. It is safe
// for concurrent use.
type Service struct {
	store       *store
	sessions    SessionInfo
	storageRoot string
	log         *slog.Logger
	now         func() time.Time
}

// NewService builds the checkpoint service. storageRoot is the base dir the
// per-checkpoint payload dirs live under (e.g. ~/.opendray/checkpoints);
// it is created on first capture. sessions may be nil (manual capture then
// requires the caller to supply context explicitly via CaptureFor).
func NewService(pool *pgxpool.Pool, sessions SessionInfo, storageRoot string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		store:       newStore(pool),
		sessions:    sessions,
		storageRoot: storageRoot,
		log:         log.With("component", "checkpoint"),
		now:         time.Now,
	}
}

// SetSessions wires the session source after construction, breaking the
// startup cycle where the session manager needs this service as its
// checkpoint recorder while this service needs the manager for manual
// captures. Call once at startup before serving traffic.
func (s *Service) SetSessions(sessions SessionInfo) {
	s.sessions = sessions
}

// Capture snapshots a session's current context. cwd and input are supplied
// by the caller (the session manager already holds them), so Capture has no
// dependency on session internals. The metadata row is inserted on success.
func (s *Service) Capture(ctx context.Context, sessionID, cwd string, trigger Trigger, input []byte, note string) (Checkpoint, error) {
	if s.storageRoot == "" {
		return Checkpoint{}, ErrNoStorageDir
	}
	if !trigger.Valid() {
		return Checkpoint{}, ErrInvalidTrigger
	}
	cp, err := Capture(ctx, s.storageRoot, CaptureRequest{
		CheckpointID: newID(),
		SessionID:    sessionID,
		Cwd:          cwd,
		Trigger:      trigger,
		InputHistory: input,
		Note:         note,
		Now:          s.now(),
	})
	if err != nil {
		return Checkpoint{}, err
	}
	if err := s.store.insert(ctx, cp); err != nil {
		// Don't leave an orphaned on-disk dir if the row didn't persist.
		_ = os.RemoveAll(cp.StoragePath)
		return Checkpoint{}, err
	}
	s.log.Info("captured session checkpoint",
		"session_id", sessionID, "checkpoint_id", cp.ID, "trigger", trigger,
		"is_git", cp.IsGit, "diff_bytes", cp.DiffBytes,
		"untracked_files", cp.UntrackedFiles, "truncated", cp.Truncated)
	s.pruneForSession(ctx, sessionID)
	return cp, nil
}

// RetainPerSession bounds how many checkpoints a session keeps; older ones
// are reaped (metadata + on-disk payload) after each capture.
const RetainPerSession = 10

// pruneForSession enforces RetainPerSession, best-effort: a prune failure is
// logged, never surfaced to the capture caller (the checkpoint succeeded).
func (s *Service) pruneForSession(ctx context.Context, sessionID string) {
	paths, err := s.store.pruneForSession(ctx, sessionID, RetainPerSession)
	if err != nil {
		s.log.Warn("checkpoint prune failed", "session_id", sessionID, "err", err)
		return
	}
	for _, p := range paths {
		if p != "" {
			if err := os.RemoveAll(p); err != nil {
				s.log.Warn("checkpoint prune reap failed", "path", p, "err", err)
			}
		}
	}
}

// CaptureManual captures via the SessionInfo lookup (the manual API path).
func (s *Service) CaptureManual(ctx context.Context, sessionID, note string) (Checkpoint, error) {
	if s.sessions == nil {
		return Checkpoint{}, fmt.Errorf("checkpoint: no session source wired")
	}
	cwd, input, ok := s.sessions.CheckpointContext(sessionID)
	if !ok {
		return Checkpoint{}, ErrNotFound
	}
	return s.Capture(ctx, sessionID, cwd, TriggerManual, input, note)
}

// CaptureInterrupt is the session manager's hook: it snapshots a session
// that is about to be interrupted by a gateway shutdown. Best-effort and
// self-contained (its own timeout) so it can never wedge shutdown; errors
// are logged, not returned.
func (s *Service) CaptureInterrupt(sessionID, cwd string, input []byte) {
	if s.storageRoot == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := s.Capture(ctx, sessionID, cwd, TriggerInterrupted, input, ""); err != nil {
		s.log.Warn("interrupt checkpoint failed", "session_id", sessionID, "err", err)
	}
}

// List returns a session's checkpoints, newest first.
func (s *Service) List(ctx context.Context, sessionID string) ([]Checkpoint, error) {
	return s.store.listForSession(ctx, sessionID)
}

// Get returns one checkpoint's manifest.
func (s *Service) Get(ctx context.Context, id string) (Checkpoint, error) {
	return s.store.get(ctx, id)
}

// ReadDiff returns the stored uncommitted diff for a checkpoint (empty when
// the working tree had no tracked changes).
func (s *Service) ReadDiff(ctx context.Context, id string) ([]byte, error) {
	cp, err := s.store.get(ctx, id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(cp.StoragePath, fileDiff))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("checkpoint: read diff: %w", err)
	}
	return data, nil
}

// Delete removes a checkpoint's metadata row and its on-disk payload.
func (s *Service) Delete(ctx context.Context, id string) error {
	path, err := s.store.delete(ctx, id)
	if err != nil {
		return err
	}
	if path != "" {
		if err := os.RemoveAll(path); err != nil {
			s.log.Warn("checkpoint payload reap failed", "checkpoint_id", id, "path", path, "err", err)
		}
	}
	return nil
}
