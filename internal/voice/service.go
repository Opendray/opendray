package voice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/opendray/opendray-v2/internal/mcp"
)

// Service is the channel-facing facade: give it a server id, get
// back a Client ready to call. Wraps the existing MCP loader +
// secrets file (per-call loading, mirroring catalog.SessionProvider's
// pattern so config / secret edits are picked up without a restart).
type Service struct {
	loader      *mcp.Loader
	secretsFile string
	log         *slog.Logger
}

// NewService wires the facade. log may be nil. secretsFile may be ""
// to disable ${KEY} substitution (placeholders will surface as
// missing-secret errors when the MCP server config references one).
func NewService(loader *mcp.Loader, secretsFile string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{loader: loader, secretsFile: secretsFile, log: log.With("component", "voice")}
}

// ErrUnknownProvider is returned when the channel references a voice
// MCP server that doesn't exist in the vault.
var ErrUnknownProvider = errors.New("voice: unknown provider")

// ErrProviderDisabled is returned when the server exists but is
// marked enabled=false in its mcp.json.
var ErrProviderDisabled = errors.New("voice: provider disabled")

// ResolveProvider looks up the id in the vault, applies secret
// substitution, and returns a Client ready to call. Missing secrets
// surface as a typed auth_failed error so the caller can render a
// useful message instead of a generic failure.
func (s *Service) ResolveProvider(ctx context.Context, id string) (*Client, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrUnknownProvider
	}
	if s == nil || s.loader == nil {
		return nil, ErrUnknownProvider
	}

	srv, err := s.loader.Get(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, id)
	}
	if !srv.Enabled {
		return nil, ErrProviderDisabled
	}

	resolved := srv
	if s.secretsFile != "" {
		secrets, err := mcp.LoadSecrets(s.secretsFile)
		if err != nil {
			s.log.Warn("voice: load secrets failed", "err", err, "file", s.secretsFile)
		} else {
			var missing []string
			resolved, missing = secrets.Resolve(srv)
			if len(missing) > 0 {
				return nil, &Error{
					Code:    "auth_failed",
					Message: fmt.Sprintf("missing secret(s): %s", strings.Join(missing, ", ")),
				}
			}
		}
	}

	return NewClient(resolved, s.log), nil
}
