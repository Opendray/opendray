package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/memory/summarizer"
)

// Registry wires the per-task config + dependencies the two
// Worker implementations need. The four memory-system touchpoints
// (gatekeeper / cleaner / gitactivity / transcript) ask the
// registry for a Worker by TaskKind; the registry reads the
// current memory_workers row, constructs the right impl, returns
// it. Subsystems don't cache — config changes via UI take effect
// on the next call.
//
// Metrics are recorded inline around the Run() call: every
// invocation gets a memory_worker_calls row whether it succeeded
// or not.
type Registry struct {
	store         *store
	summarizerReg *summarizer.Registry
	accounts      AccountReader
	agyAccounts   AgyAccountReader
	log           *slog.Logger
}

// NewRegistry wires the resolver. summarizerReg is required (so
// the summarizer worker can pick providers); accounts / agyAccounts are
// optional (only needed for multi-account agent workers — Claude via
// accounts, antigravity via agyAccounts. When nil, agent workers fall back
// to whatever the CLI's host config resolves on its own).
func NewRegistry(
	pool *pgxpool.Pool,
	summarizerReg *summarizer.Registry,
	accounts AccountReader,
	agyAccounts AgyAccountReader,
	log *slog.Logger,
) *Registry {
	if log == nil {
		log = slog.Default()
	}
	return &Registry{
		store:         newStore(pool),
		summarizerReg: summarizerReg,
		accounts:      accounts,
		agyAccounts:   agyAccounts,
		log:           log.With("component", "memory.worker.registry"),
	}
}

// WorkerFor reads the config for the given task and returns a
// constructed Worker. Returns ErrNoWorkerConfigured if the row
// is missing or disabled — callers should degrade gracefully.
func (r *Registry) WorkerFor(ctx context.Context, task TaskKind) (Worker, error) {
	cfg, err := r.store.Get(ctx, task)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrNoWorkerConfigured
	}
	return r.buildWorker(cfg)
}

// buildWorker constructs the Worker impl for an explicit config. Shared
// by WorkerFor (stored config) and RunWith (caller-supplied override).
func (r *Registry) buildWorker(cfg Config) (Worker, error) {
	switch cfg.Kind {
	case WorkerSummarizer:
		if r.summarizerReg == nil {
			return nil, errors.New("memory worker: summarizer registry not wired")
		}
		return NewSummarizerWorker(r.summarizerReg, cfg), nil
	case WorkerAgent:
		return NewAgentWorker(r.accounts, r.agyAccounts, cfg, r.log), nil
	default:
		return nil, fmt.Errorf("memory worker: unknown kind %q", cfg.Kind)
	}
}

// Run is the convenience entry point most callers use. It
// resolves the worker from the task's stored config, calls Run,
// and records a metrics row. On error returns the worker error
// verbatim (caller decides how to degrade).
func (r *Registry) Run(ctx context.Context, req Request) (Response, error) {
	worker, err := r.WorkerFor(ctx, req.Task)
	if err != nil {
		return Response{}, err
	}
	return r.runWorker(ctx, worker, req)
}

// RunWith runs a request against a CALLER-SUPPLIED config instead of the
// task's stored row — used for per-call overrides (e.g. a curation
// conversation that pins its own cloud-agent provider + model). Metrics
// are still recorded under req.Task. The override must be self-contained
// (Kind + ProviderID/Model as appropriate); Enabled is ignored.
func (r *Registry) RunWith(ctx context.Context, override Config, req Request) (Response, error) {
	worker, err := r.buildWorker(override)
	if err != nil {
		return Response{}, err
	}
	return r.runWorker(ctx, worker, req)
}

// runWorker invokes a constructed worker and records a best-effort
// metrics row around the call.
func (r *Registry) runWorker(ctx context.Context, worker Worker, req Request) (Response, error) {
	t0 := time.Now()
	resp, runErr := worker.Run(ctx, req)
	dur := time.Since(t0).Milliseconds()

	// Best-effort metrics — use a detached context so a cancelled
	// caller doesn't kill the INSERT.
	bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.store.RecordCall(bg, CallSummary{
		Task:         req.Task,
		WorkerKind:   worker.Kind(),
		ProviderID:   resp.ProviderID,
		AccountID:    resp.AccountID,
		DurationMS:   dur,
		Success:      runErr == nil,
		ErrorMessage: errString(runErr),
		InputBytes:   len(req.UserInput),
		OutputBytes:  len(resp.Content),
		TokensIn:     resp.TokensIn,
		TokensOut:    resp.TokensOut,
	})
	return resp, runErr
}

// Store exposes the underlying store for handlers that need
// config CRUD + metrics queries.
func (r *Registry) Store() *store { return r.store }

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
