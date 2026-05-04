// Package settings exposes the on-disk config.toml as a runtime
// API: GET reads it, PUT writes it back, POST /restart self-execs
// the binary so the new values take effect.
//
// Sensitive fields (database.url, admin.password) are stripped from
// GET responses so the running token doesn't leak credentials, and
// preserved-on-empty during PUT so the UI can render masked inputs
// without echoing the value back.
package settings

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"

	"github.com/opendray/opendray-v2/internal/config"
)

// Service owns read+write access to a single config.toml file.
// Concurrent PUTs are serialised through mu so two operators can't
// stomp each other.
type Service struct {
	mu         sync.Mutex
	configPath string
	log        *slog.Logger
}

func NewService(configPath string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		configPath: configPath,
		log:        log.With("component", "settings"),
	}
}

// ConfigPath returns the absolute path the service reads/writes.
// Useful for the UI to show "loaded from /etc/opendray/config.toml".
func (s *Service) ConfigPath() string { return s.configPath }

// Get re-reads config.toml from disk and returns the current values.
// Sensitive fields are zeroed before return — see Settings.Strip.
func (s *Service) Get() (*config.Config, error) {
	c, err := s.readUnsafe()
	if err != nil {
		return nil, err
	}
	stripSensitive(c)
	return c, nil
}

// Update overwrites config.toml with `patch`. Sensitive fields with
// empty/zero values in `patch` are filled in from the on-disk
// version — that lets the UI omit them entirely when the operator
// hasn't typed a new value.
func (s *Service) Update(patch *config.Config) error {
	if patch == nil {
		return errors.New("nil patch")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, err := s.readUnsafe()
	if err != nil {
		return err
	}
	mergeSensitive(patch, cur)

	// Atomic write: encode → tmp → rename. Atomic rename on macOS/Linux
	// guarantees readers never see a half-written file.
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(patch); err != nil {
		return fmt.Errorf("encode toml: %w", err)
	}
	tmp := s.configPath + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.configPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	s.log.Info("config updated", "path", s.configPath)
	return nil
}

func (s *Service) readUnsafe() (*config.Config, error) {
	if s.configPath == "" {
		return nil, errors.New("no config path configured")
	}
	abs, _ := filepath.Abs(s.configPath)
	var c config.Config
	if _, err := toml.DecodeFile(abs, &c); err != nil {
		return nil, fmt.Errorf("decode %s: %w", abs, err)
	}
	return &c, nil
}

// stripSensitive zeroes secrets so GET responses don't echo them
// back to the browser. The UI should render password fields as
// masked + "Change…" buttons.
func stripSensitive(c *config.Config) {
	if c == nil {
		return
	}
	c.Database.URL = ""
	c.Admin.Password = ""
}

// mergeSensitive copies sensitive fields from `cur` to `patch` when
// `patch` left them empty — the UI's way of saying "don't touch".
func mergeSensitive(patch, cur *config.Config) {
	if patch.Database.URL == "" {
		patch.Database.URL = cur.Database.URL
	}
	if patch.Admin.Password == "" {
		patch.Admin.Password = cur.Admin.Password
	}
}
