package integration

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

const (
	defaultHealthInterval = 30 * time.Second
	defaultHealthTimeout  = 5 * time.Second
)

// HealthChecker periodically probes /health on each enabled integration
// and updates HealthStatus. Per design §11: two consecutive non-2xx OR
// status="unhealthy" => mark unhealthy.
type HealthChecker struct {
	svc      *Service
	bus      *eventbus.Hub
	log      *slog.Logger
	interval time.Duration
	timeout  time.Duration
	client   *http.Client

	mu       sync.Mutex
	failures map[string]int // integration id → consecutive failure count
}

func NewHealthChecker(svc *Service, bus *eventbus.Hub, log *slog.Logger) *HealthChecker {
	if log == nil {
		log = slog.Default()
	}
	timeout := defaultHealthTimeout
	return &HealthChecker{
		svc:      svc,
		bus:      bus,
		log:      log.With("component", "integration.health"),
		interval: defaultHealthInterval,
		timeout:  timeout,
		client:   &http.Client{Timeout: timeout},
		failures: make(map[string]int),
	}
}

// Run blocks until ctx is cancelled. Probes every existing integration
// immediately, then every interval, plus on every integration.registered
// event so newly-added rows transition out of "unknown" without waiting
// for the next tick.
func (h *HealthChecker) Run(ctx context.Context) {
	h.tick(ctx)

	var registered <-chan eventbus.Event
	if h.bus != nil {
		ch, unsub := h.bus.Subscribe("integration.registered", 16)
		defer unsub()
		registered = ch
	}

	t := time.NewTicker(h.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.tick(ctx)
		case ev, ok := <-registered:
			if !ok {
				registered = nil
				continue
			}
			h.probeFromEvent(ctx, ev)
		}
	}
}

func (h *HealthChecker) probeFromEvent(ctx context.Context, ev eventbus.Event) {
	data, _ := ev.Data.(map[string]any)
	id, _ := data["integration_id"].(string)
	if id == "" {
		return
	}
	i, err := h.svc.Get(ctx, id)
	if err != nil {
		return
	}
	h.probe(ctx, i)
}

func (h *HealthChecker) tick(ctx context.Context) {
	list, err := h.svc.List(ctx)
	if err != nil {
		h.log.Error("list integrations", "err", err)
		return
	}
	for _, i := range list {
		if !i.Enabled {
			continue
		}
		h.probe(ctx, i)
	}
}

func (h *HealthChecker) probe(ctx context.Context, i Integration) {
	url := i.BaseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	var (
		next    HealthStatus
		payload map[string]any
	)

	resp, err := h.client.Do(req)
	switch {
	case err != nil:
		next = h.bumpFail(i.ID)
		payload = map[string]any{"error": err.Error()}
	case resp.StatusCode/100 != 2:
		resp.Body.Close()
		next = h.bumpFail(i.ID)
		payload = map[string]any{"http_status": resp.StatusCode}
	default:
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var ping HealthPing
		_ = json.Unmarshal(body, &ping)
		switch ping.Status {
		case "":
			// Endpoint exists but didn't return our schema — treat as healthy.
			next = HealthHealthy
			h.resetFail(i.ID)
		case "healthy":
			next = HealthHealthy
			h.resetFail(i.ID)
		case "degraded":
			next = HealthDegraded
		case "unhealthy":
			next = HealthUnhealthy
		default:
			next = HealthDegraded
		}
		payload = map[string]any{
			"status":      ping.Status,
			"version":     ping.Version,
			"busy_ratio":  ping.BusyRatio,
			"queue_depth": ping.QueueDepth,
		}
	}

	if err := h.svc.SetHealth(ctx, i.ID, i.HealthStatus, next, payload); err != nil {
		h.log.Error("set health", "id", i.ID, "err", err)
	}
}

func (h *HealthChecker) bumpFail(id string) HealthStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failures[id]++
	if h.failures[id] >= 2 {
		return HealthUnhealthy
	}
	return HealthDegraded
}

func (h *HealthChecker) resetFail(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.failures, id)
}
