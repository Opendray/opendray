// LiveBackup is the runtime lifecycle manager for the backup
// feature. It owns:
//
//	- The current *Service (or nil when feature is off)
//	- The Scheduler goroutine + its cancel function
//
// Both are swapped atomically by Arm/Disarm so other parts of the
// process can keep a stable *LiveBackup reference and ask "is the
// feature on right now?" via Service() — a single atomic load,
// safe to call on every HTTP request.
//
// The point of this struct is to let /backup-setup turn the
// feature on without restarting opendray: a chi.Mux that's already
// running can't have new routes added thread-safely, but a wrapper
// handler whose internals are swappable via atomic.Pointer is.
// Handlers (the HTTP routes) is mounted ONCE at startup against a
// LiveBackup; each route reads the live Service via Service() and
// 503s if it returns nil (see requireArmed middleware in
// handler.go).
//
// Concurrency model: a sync.Mutex serializes Arm and Disarm calls
// to keep the "stop old scheduler, start new scheduler, swap
// router state" sequence consistent. Reads (Service()) are
// lock-free.

package backup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LiveBackup is the lifecycle handle exposed to handlers and the
// rest of the app. Construct one at startup; pass the same
// pointer to Handlers, SetupHandlers, and the summarizer cipher
// adapter — none of them need to know whether the feature is
// currently armed.
type LiveBackup struct {
	cfg  Config
	deps liveDeps
	log  *slog.Logger

	state atomic.Pointer[liveState]

	// Serializes Arm / Disarm. Reads via Service() are lock-free.
	armMu sync.Mutex
}

// liveDeps captures everything the Service constructor needs
// except the passphrase, which arrives at Arm() time.
type liveDeps struct {
	pool       *pgxpool.Pool
	dsn        string
	configPath string
}

// liveState bundles everything that must swap together when the
// feature is (re)armed: the Service itself plus the scheduler's
// cancel func. atomic.Pointer[liveState] is the single source of
// truth.
type liveState struct {
	svc         *Service
	cancelSched context.CancelFunc
}

// NewLiveBackup constructs a lifecycle handle. The returned
// LiveBackup is initially DISARMED — call Arm to turn the feature
// on (either at startup, when a passphrase is found via
// LoadPassphrase, or later when the operator hits /backup-setup
// from the UI).
//
// pool / dsn / configPath are captured here because the Service
// constructor needs them every time we Arm. log is reused.
func NewLiveBackup(
	cfg Config,
	pool *pgxpool.Pool,
	dsn string,
	configPath string,
	log *slog.Logger,
) *LiveBackup {
	return &LiveBackup{
		cfg: cfg,
		deps: liveDeps{
			pool:       pool,
			dsn:        dsn,
			configPath: configPath,
		},
		log: log,
	}
}

// Service returns the live backup Service, or nil when the feature
// is off. Cheap atomic load — safe on every request path.
func (l *LiveBackup) Service() *Service {
	st := l.state.Load()
	if st == nil {
		return nil
	}
	return st.svc
}

// IsArmed is a thin shortcut for callers that only need the
// boolean. Equivalent to `Service() != nil`.
func (l *LiveBackup) IsArmed() bool { return l.Service() != nil }

// Arm builds a fresh Service from the given passphrase, bootstraps
// it (runs migrations / pg_dump probe), starts a scheduler under a
// cancellable context, and atomically swaps the live state.
//
// If the feature was already armed, the old scheduler is cancelled
// after the new state is in place — order matters so requests
// in-flight against the old Service complete instead of seeing nil
// mid-request.
//
// ctx is used as the scheduler's parent context; pass the app's
// long-lived context (the one that closes on opendray shutdown)
// so the scheduler stops cleanly on process exit.
func (l *LiveBackup) Arm(ctx context.Context, passphrase string) error {
	if passphrase == "" {
		return errors.New("Arm called with empty passphrase")
	}
	svc, err := NewService(l.cfg, ServiceDeps{
		Pool:       l.deps.pool,
		Passphrase: passphrase,
		DSN:        l.deps.dsn,
		ConfigPath: l.deps.configPath,
		Log:        l.log,
	})
	if err != nil {
		return fmt.Errorf("init service: %w", err)
	}
	if err := svc.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	return l.armWithService(ctx, svc)
}

// ArmWithService is the same as Arm but takes a pre-built Service
// (used at app startup where we want to construct + Bootstrap
// outside of the LiveBackup so init errors short-circuit boot
// rather than getting buried inside Arm).
func (l *LiveBackup) ArmWithService(ctx context.Context, svc *Service) error {
	if svc == nil {
		return errors.New("ArmWithService called with nil svc")
	}
	return l.armWithService(ctx, svc)
}

func (l *LiveBackup) armWithService(ctx context.Context, svc *Service) error {
	l.armMu.Lock()
	defer l.armMu.Unlock()

	// Start the scheduler goroutine under a cancellable context.
	// Each Arm gets its own context; the cancel func is stored in
	// liveState so a subsequent Disarm or re-Arm can stop only the
	// goroutine it spawned.
	schedCtx, cancel := context.WithCancel(ctx)
	sched := NewScheduler(svc, 0)
	go sched.Run(schedCtx)

	newState := &liveState{svc: svc, cancelSched: cancel}
	old := l.state.Swap(newState)
	if old != nil {
		// Cancel old scheduler AFTER the swap, so any handler that
		// loaded the old state mid-request keeps a valid Service
		// until its request completes.
		old.cancelSched()
	}
	if l.log != nil {
		l.log.Info("backup armed",
			"key_fingerprint", svc.CipherFingerprint())
	}
	return nil
}

// Disarm stops the scheduler and clears the live state. Idempotent
// — calling Disarm on an already-off LiveBackup is a no-op.
// Returns the previously-active Service (or nil) so callers can
// release any external state they hold; in practice nothing else
// does, so the return is usually ignored.
func (l *LiveBackup) Disarm() *Service {
	l.armMu.Lock()
	defer l.armMu.Unlock()
	old := l.state.Swap(nil)
	if old == nil {
		return nil
	}
	old.cancelSched()
	if l.log != nil {
		l.log.Info("backup disarmed")
	}
	return old.svc
}

// liveCipher adapts a LiveBackup to the summarizer.Cipher
// interface (EncryptField + DecryptField). It re-reads the live
// Service on every call so summarizer code can encrypt API keys
// even when the operator armed backup via UI mid-runtime — no
// restart required for anthropic provider setup. When the feature
// is off, both methods return ErrCipherNotArmed.
type liveCipher struct {
	live *LiveBackup
}

// ErrCipherNotArmed signals to the summarizer that no backup
// cipher is available right now. Surfaced as a 400 when an
// operator tries to add an anthropic provider before backups are
// enabled.
var ErrCipherNotArmed = errors.New("backup cipher not configured — enable backups first")

func (c *liveCipher) EncryptField(plain string) (string, error) {
	svc := c.live.Service()
	if svc == nil {
		return "", ErrCipherNotArmed
	}
	return svc.cipher.EncryptField(plain)
}

func (c *liveCipher) DecryptField(envelope string) (string, error) {
	svc := c.live.Service()
	if svc == nil {
		return "", ErrCipherNotArmed
	}
	return svc.cipher.DecryptField(envelope)
}

// NewLiveCipher returns a summarizer-compatible cipher backed by
// the given LiveBackup. The wrapper survives Arm/Disarm calls —
// each EncryptField/DecryptField re-reads the live Service.
func NewLiveCipher(l *LiveBackup) interface {
	EncryptField(plain string) (string, error)
	DecryptField(envelope string) (string, error)
} {
	return &liveCipher{live: l}
}
