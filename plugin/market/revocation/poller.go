package revocation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/opendray/opendray/plugin/market"
)

// DefaultPollInterval is the spec-default revocation poll cadence
// (docs/plugin-platform/09-marketplace.md §Kill-switch).
const DefaultPollInterval = 6 * time.Hour

// MinPollInterval and MaxPollInterval bound the
// OPENDRAY_REVOCATION_POLL_HOURS env override. The ceiling prevents
// "effectively never" configurations — revocation is security
// infrastructure, it must poll.
const (
	MinPollInterval = 1 * time.Hour
	MaxPollInterval = 168 * time.Hour
)

// InstalledPlugin describes one currently-installed plugin the
// poller should match against. The caller (wired in T11) supplies
// a snapshot on each sweep so the poller doesn't need to couple
// to plugin.Runtime directly.
type InstalledPlugin struct {
	Publisher string
	Name      string
	Version   string
}

// ActionHandler is invoked once per (Entry, InstalledPlugin) match.
// The handler runs synchronously on the poller goroutine; a T9
// implementation dispatches to uninstall / disable / warn UIs.
// Errors are logged; they don't abort the sweep.
type ActionHandler func(ctx context.Context, entry Entry, target InstalledPlugin) error

// Config configures a Poller. RevocationCatalog is required; every
// other field has a reasonable default so the caller can leave
// them zero during bring-up.
type Config struct {
	// Catalog supplies FetchRevocations (and the cache). Required.
	Catalog market.Catalog

	// Interval is how often to poll. Zero uses DefaultPollInterval.
	// Values outside [MinPollInterval, MaxPollInterval] are clamped.
	Interval time.Duration

	// Installed returns the snapshot of installed plugins on each
	// sweep. Called under the poller goroutine; the caller is
	// responsible for making this cheap + thread-safe. Required.
	Installed func() []InstalledPlugin

	// OnAction is called for every matching revocation entry.
	// Required.
	OnAction ActionHandler

	// Logger is used for diagnostics. Zero uses slog.Default.
	Logger *slog.Logger
}

// Poller runs a background goroutine that pulls revocations.json
// from the catalog on Config.Interval and invokes Config.OnAction
// for every matching entry against the current installed set.
//
// Safe to stop via the ctx passed to Run. One poller per process
// is the expected deployment.
type Poller struct {
	cfg Config
}

// New constructs a Poller. Returns an error when required fields
// are missing so misconfiguration fails at wire-time rather than
// after a silent no-op deploy.
func New(cfg Config) (*Poller, error) {
	if cfg.Catalog == nil {
		return nil, fmt.Errorf("revocation: Config.Catalog is required")
	}
	if cfg.Installed == nil {
		return nil, fmt.Errorf("revocation: Config.Installed is required")
	}
	if cfg.OnAction == nil {
		return nil, fmt.Errorf("revocation: Config.OnAction is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	cfg.Interval = clampInterval(cfg.Interval)
	return &Poller{cfg: cfg}, nil
}

// clampInterval applies spec floor/ceiling bounds. Zero becomes
// DefaultPollInterval so caller-side "I forgot to set this" falls
// through to the spec value.
func clampInterval(d time.Duration) time.Duration {
	if d == 0 {
		return DefaultPollInterval
	}
	if d < MinPollInterval {
		return MinPollInterval
	}
	if d > MaxPollInterval {
		return MaxPollInterval
	}
	return d
}

// Run blocks until ctx is cancelled. Sweeps immediately on entry
// (so launch-time kill-switch entries apply without waiting a full
// interval) then ticks.
func (p *Poller) Run(ctx context.Context) {
	p.Sweep(ctx)
	t := time.NewTicker(p.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.Sweep(ctx)
		}
	}
}

// Sweep runs one iteration of the poll loop. Exposed so tests can
// drive sweeps without spinning up a goroutine + waiting a tick.
func (p *Poller) Sweep(ctx context.Context) {
	body, err := p.cfg.Catalog.FetchRevocations(ctx)
	if err != nil {
		p.cfg.Logger.Warn("revocation: fetch failed", "err", err)
		return
	}
	if len(body) == 0 {
		return // empty list is the common case
	}
	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		p.cfg.Logger.Warn("revocation: parse failed", "err", err)
		return
	}
	if resp.Version != 1 {
		p.cfg.Logger.Warn("revocation: unsupported version", "version", resp.Version)
		return
	}
	installed := p.cfg.Installed()

	for _, entry := range resp.Entries {
		for _, inst := range installed {
			hit, err := entry.Matches(inst.Publisher, inst.Name, inst.Version)
			if err != nil {
				p.cfg.Logger.Warn("revocation: entry skipped", "name", entry.Name, "err", err)
				break // move on to next entry; this one is malformed
			}
			if !hit {
				continue
			}
			if actErr := p.cfg.OnAction(ctx, entry, inst); actErr != nil {
				p.cfg.Logger.Error("revocation: action failed",
					"name", entry.Name,
					"action", entry.NormalisedAction(),
					"target", inst.Publisher+"/"+inst.Name+"@"+inst.Version,
					"err", actErr,
				)
			}
		}
	}
}
