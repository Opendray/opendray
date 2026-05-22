// Package audit subscribes to selected event-bus topics and persists
// them to the audit_log table.
//
// Per design §12, audit captures *lifecycle* events — session start /
// end / idle, integration registration + key rotation, channel
// configure — but never message bodies or other PII. Topics whose
// payload could contain user data (e.g. session.output, channel.message)
// must not subscribe through this sink.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

// subscribedPatterns is the audit-safe topic allowlist.
var subscribedPatterns = []string{
	"session.started",
	"session.ended",
	"session.idle",
	"admin.login_success",
	"admin.login_failed",
	"admin.logout",
	"channel.message_sent",
	"channel.message_received",
	"integration.registered",
	"integration.deregistered",
	"integration.key_rotated",
	"integration.health_changed",
	"provider.updated",
}

const subscriberBuffer = 256

// Sink is the audit-log writer. One Sink per opendray process; Run
// blocks until ctx is cancelled.
type Sink struct {
	pool *pgxpool.Pool
	bus  *eventbus.Hub
	log  *slog.Logger
}

func NewSink(pool *pgxpool.Pool, bus *eventbus.Hub, log *slog.Logger) *Sink {
	if log == nil {
		log = slog.Default()
	}
	return &Sink{pool: pool, bus: bus, log: log.With("component", "audit")}
}

// Run launches one consumer goroutine per allowlisted pattern and
// blocks until ctx is cancelled or the bus closes.
func (s *Sink) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, pattern := range subscribedPatterns {
		ch, unsub := s.bus.Subscribe(pattern, subscriberBuffer)
		wg.Add(1)
		go s.consume(ctx, &wg, ch, unsub)
	}
	wg.Wait()
}

func (s *Sink) consume(ctx context.Context, wg *sync.WaitGroup, ch <-chan eventbus.Event, unsub func()) {
	defer wg.Done()
	defer unsub()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			s.write(ctx, ev)
		}
	}
}

func (s *Sink) write(ctx context.Context, ev eventbus.Event) {
	subjectKind, subjectID := extractSubject(ev.Data)
	metadata, err := json.Marshal(ev.Data)
	if err != nil {
		s.log.Warn("marshal metadata", "topic", ev.Topic, "err", err)
		return
	}

	ts := ev.Time
	if ts.IsZero() {
		ts = time.Now()
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = s.pool.Exec(writeCtx, `
        INSERT INTO audit_log (ts, actor_kind, action, subject_kind, subject_id, metadata)
        VALUES ($1, 'system', $2, $3, $4, $5::jsonb)`,
		ts, ev.Topic,
		nullable(subjectKind), nullable(subjectID), metadata)
	if err != nil {
		s.log.Error("insert audit_log", "topic", ev.Topic, "err", err)
	}
}

// extractSubject inspects the event payload for a known subject id key
// and returns ("kind","id") if found.
func extractSubject(data any) (kind, id string) {
	m, ok := data.(map[string]any)
	if !ok {
		return "", ""
	}
	if v, ok := m["session_id"].(string); ok {
		return "session", v
	}
	if v, ok := m["integration_id"].(string); ok {
		return "integration", v
	}
	if v, ok := m["channel_id"].(string); ok {
		return "channel", v
	}
	if v, ok := m["provider_id"].(string); ok {
		return "provider", v
	}
	if v, ok := m["user"].(string); ok {
		return "admin", v
	}
	if v, ok := m["integration_id"].(string); ok {
		return "integration", v
	}
	return "", ""
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
