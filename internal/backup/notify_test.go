package backup

import (
	"log/slog"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
)

func TestServiceNotify_PublishesToBus(t *testing.T) {
	bus := eventbus.New(slog.Default())
	ch, unsub := bus.Subscribe("backup.failed", 4)
	defer unsub()

	s := &Service{bus: bus}
	s.notify("backup.failed", map[string]any{"backup_id": "bk_x", "error": "boom"})

	select {
	case ev := <-ch:
		if ev.Topic != "backup.failed" {
			t.Fatalf("topic = %q, want backup.failed", ev.Topic)
		}
		data, ok := ev.Data.(map[string]any)
		if !ok || data["backup_id"] != "bk_x" {
			t.Fatalf("unexpected event data: %#v", ev.Data)
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected a backup.failed event, got none")
	}
}

func TestServiceNotify_NilBusIsNoop(t *testing.T) {
	s := &Service{} // bus == nil
	// Must not panic.
	s.notify("backup.failed", map[string]any{"backup_id": "bk_x"})
}
