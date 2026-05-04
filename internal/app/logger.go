package app

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/opendray/opendray-v2/internal/applog"
	"github.com/opendray/opendray-v2/internal/config"
)

// newLogger builds the global slog.Logger plus an in-memory ring
// buffer that the /admin/logs endpoints can serve. Output fans to:
//
//   - os.Stderr (always)
//   - rotating file (when cfg.File is set)
//   - the returned ring buffer (always; ~2,000 records, ~1-2 MB)
//
// The ring is returned so the caller can hand it to the HTTP layer.
func newLogger(cfg config.LogConfig) (*slog.Logger, *applog.Buffer, error) {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	// Underlying writer: stderr + (optional) rotating file.
	writers := []io.Writer{os.Stderr}
	if path := strings.TrimSpace(cfg.File); path != "" {
		fw, err := applog.NewFileWriter(applog.FileWriterOption{Path: expandPath(path)})
		if err != nil {
			return nil, nil, err
		}
		if fw != nil {
			writers = append(writers, fw)
		}
	}
	out := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{Level: level}
	var inner slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		inner = slog.NewJSONHandler(out, opts)
	} else {
		inner = slog.NewTextHandler(out, opts)
	}

	ring := applog.NewBuffer(2000)
	return slog.New(applog.NewHandler(inner, ring)), ring, nil
}
