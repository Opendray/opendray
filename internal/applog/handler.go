package applog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Handler wraps an inner slog.Handler so every log record is also
// pushed into a ring Buffer (for the in-memory tail / WS endpoint).
// The inner handler is whatever the operator picked — usually a
// TextHandler or JSONHandler writing to a multi-writer that fans to
// stderr + an optional rotating file.
type Handler struct {
	inner slog.Handler
	ring  *Buffer

	// attrsMu protects attrs/groups so WithAttrs/WithGroup can
	// merge new metadata without racing concurrent Handle calls.
	attrsMu sync.Mutex
	attrs   []slog.Attr
	groups  []string
}

// NewHandler builds a Handler that forwards to `inner` and pushes
// every record into `ring`. Pass nil for `ring` to skip ring capture
// (rare; used in tests).
func NewHandler(inner slog.Handler, ring *Buffer) *Handler {
	return &Handler{inner: inner, ring: ring}
}

// Enabled is delegated; we don't filter further than the inner
// handler does.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle writes to the inner handler first (preserving its existing
// formatting / output destination), then captures a synthesised
// Record into the ring.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.inner.Handle(ctx, r); err != nil {
		return err
	}
	if h.ring != nil {
		h.ring.Push(toRecord(r, h.snapshotAttrs()))
	}
	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		inner:  h.inner.WithAttrs(attrs),
		ring:   h.ring,
		attrs:  append(append([]slog.Attr(nil), h.attrs...), attrs...),
		groups: append([]string(nil), h.groups...),
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		inner:  h.inner.WithGroup(name),
		ring:   h.ring,
		attrs:  append([]slog.Attr(nil), h.attrs...),
		groups: append(append([]string(nil), h.groups...), name),
	}
}

func (h *Handler) snapshotAttrs() []slog.Attr {
	h.attrsMu.Lock()
	defer h.attrsMu.Unlock()
	out := make([]slog.Attr, len(h.attrs))
	copy(out, h.attrs)
	return out
}

// toRecord materialises slog.Record into our display struct. The
// inherited base attrs (from WithAttrs / WithGroup chains) and the
// per-call attrs are both flattened into the Attrs map.
func toRecord(r slog.Record, base []slog.Attr) Record {
	attrs := make(map[string]any, r.NumAttrs()+len(base))
	for _, a := range base {
		attrs[a.Key] = a.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	level := LevelFromSlog(r.Level)
	rec := Record{
		Time:    r.Time,
		Level:   level,
		Message: r.Message,
		Attrs:   attrs,
	}
	rec.Text = renderText(rec)
	return rec
}

// renderText produces the single-line preview shown in the live-tail
// UI. Mirrors slog.NewTextHandler's default rendering closely so
// stderr and the UI agree visually.
//
//	"15:04:05.000 INFO message attr=value attr2=value2"
func renderText(r Record) string {
	var b strings.Builder
	b.WriteString(r.Time.Format("15:04:05.000"))
	b.WriteByte(' ')
	b.WriteString(r.Level)
	b.WriteByte(' ')
	b.WriteString(r.Message)
	for k, v := range r.Attrs {
		fmt.Fprintf(&b, " %s=%v", k, v)
	}
	return b.String()
}

// FileWriterOption configures the rotating-file writer.
type FileWriterOption struct {
	Path       string // ~/ already expanded by the caller
	MaxSizeMB  int    // default 10
	MaxBackups int    // default 5
	MaxAgeDays int    // default 30
	Compress   bool   // default false
}

// NewFileWriter returns an io.Writer wrapping a lumberjack rotator.
// Empty Path returns (nil, nil) — caller treats as "no file output".
func NewFileWriter(opt FileWriterOption) (io.Writer, error) {
	if strings.TrimSpace(opt.Path) == "" {
		return nil, nil
	}
	if opt.MaxSizeMB <= 0 {
		opt.MaxSizeMB = 10
	}
	if opt.MaxBackups <= 0 {
		opt.MaxBackups = 5
	}
	if opt.MaxAgeDays <= 0 {
		opt.MaxAgeDays = 30
	}
	// Ensure parent dir exists; lumberjack creates the file but not
	// intermediate directories.
	if dir := filepathDir(opt.Path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create log dir %s: %w", dir, err)
		}
	}
	return &lumberjack.Logger{
		Filename:   opt.Path,
		MaxSize:    opt.MaxSizeMB,
		MaxBackups: opt.MaxBackups,
		MaxAge:     opt.MaxAgeDays,
		Compress:   opt.Compress,
		LocalTime:  true,
	}, nil
}

// filepathDir returns the parent directory of p, using filepath.Dir
// so it handles both Unix (/) and Windows (\) path separators.
// Returns "" when there is no meaningful parent (equivalent to ".").
func filepathDir(p string) string {
	dir := filepath.Dir(p)
	if dir == "." {
		return ""
	}
	return dir
}
