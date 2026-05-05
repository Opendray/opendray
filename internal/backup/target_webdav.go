package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
)

// WebDAVConfig drives a WebDAV target. Plaintext fields —
// Password is decrypted from backup_targets.config.
//
// Targets typically encountered:
//   - Nextcloud:    https://cloud.example.com/remote.php/dav/files/<user>/
//   - Synology DSM: https://nas.local:5006/
//   - Box.com:      https://dav.box.com/dav
//   - 坚果云:        https://dav.jianguoyun.com/dav/
//   - generic Apache mod_dav over HTTPS
//
// We don't try to detect the flavour — the URL is given verbatim
// to gowebdav, which speaks the standard WebDAV class-2 surface.
type WebDAVConfig struct {
	BaseURL    string // full URL incl. path (e.g. https://cloud.example/remote.php/dav/files/me/)
	User       string
	Password   string
	PathPrefix string // optional sub-folder under BaseURL
}

// WebDAVTarget writes blobs to a WebDAV server with basic auth.
//
// gowebdav doesn't have native streaming-upload; we spool to a
// temp file then PUT via Write. For typical backup sizes (<= 1
// GiB) this is acceptable; larger deployments should prefer S3 or
// SFTP.
type WebDAVTarget struct {
	id       string
	cfg      WebDAVConfig
	dialTO   time.Duration
	requestTO time.Duration
}

// NewWebDAVTarget validates cfg and constructs but does not connect.
func NewWebDAVTarget(id string, cfg WebDAVConfig) (*WebDAVTarget, error) {
	if id == "" {
		return nil, errors.New("webdav target: id required")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("webdav target: base_url required (e.g. https://cloud.example.com/remote.php/dav/files/<user>/)")
	}
	if !strings.HasPrefix(cfg.BaseURL, "http://") && !strings.HasPrefix(cfg.BaseURL, "https://") {
		return nil, errors.New("webdav target: base_url must start with http(s)://")
	}
	if cfg.User == "" {
		return nil, errors.New("webdav target: user required")
	}
	cfg.PathPrefix = strings.Trim(cfg.PathPrefix, "/")
	return &WebDAVTarget{
		id:        id,
		cfg:       cfg,
		dialTO:    30 * time.Second,
		requestTO: 5 * time.Minute,
	}, nil
}

func (t *WebDAVTarget) Name() string     { return t.id }
func (t *WebDAVTarget) Kind() TargetKind { return TargetWebDAV }

func (t *WebDAVTarget) resolve(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("%w: empty path", ErrTargetRejectedPath)
	}
	if strings.ContainsRune(p, 0) {
		return "", fmt.Errorf("%w: null byte", ErrTargetRejectedPath)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", fmt.Errorf("%w: traversal segment in %q", ErrTargetRejectedPath, p)
		}
	}
	cleaned := path.Clean("/" + p)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if t.cfg.PathPrefix != "" {
		cleaned = t.cfg.PathPrefix + "/" + cleaned
	}
	return "/" + cleaned, nil // gowebdav expects leading slash
}

func (t *WebDAVTarget) client() *gowebdav.Client {
	c := gowebdav.NewClient(t.cfg.BaseURL, t.cfg.User, t.cfg.Password)
	c.SetTimeout(t.requestTO)
	c.SetTransport(&http.Transport{
		ResponseHeaderTimeout: t.dialTO,
	})
	return c
}

func (t *WebDAVTarget) ensureDir(c *gowebdav.Client, dir string) error {
	if dir == "" || dir == "/" {
		return nil
	}
	dir = strings.TrimRight(dir, "/")
	// MkdirAll is idempotent in gowebdav (existing path → no error).
	return c.MkdirAll(dir, 0o700)
}

func (t *WebDAVTarget) Put(ctx context.Context, p string, r io.Reader, _ int64) (TargetRef, error) {
	dest, err := t.resolve(p)
	if err != nil {
		return TargetRef{}, err
	}
	c := t.client()

	if err := t.ensureDir(c, path.Dir(dest)); err != nil {
		return TargetRef{}, fmt.Errorf("webdav mkdir: %w", err)
	}

	hasher := sha256.New()
	tee := io.TeeReader(&ctxReader{ctx: ctx, r: r}, hasher)

	// Spool through a temp file to know final size + retry on
	// network blips. gowebdav.WriteStream takes io.Reader and PUTs
	// directly, but doesn't expose retry; for backup we tolerate
	// the extra disk hop.
	tmp, err := os.CreateTemp("", "opendray-webdav-*")
	if err != nil {
		return TargetRef{}, fmt.Errorf("webdav tmp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	written, copyErr := io.Copy(tmp, tee)
	if cerr := tmp.Close(); cerr != nil && copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		return TargetRef{}, fmt.Errorf("webdav spool: %w", copyErr)
	}

	body, err := os.Open(tmpName)
	if err != nil {
		return TargetRef{}, fmt.Errorf("webdav reopen tmp: %w", err)
	}
	defer body.Close()

	if err := c.WriteStream(dest, body, 0o600); err != nil {
		return TargetRef{}, fmt.Errorf("webdav put %s: %w", dest, err)
	}

	return TargetRef{
		Target: t.id,
		Path:   p,
		Bytes:  written,
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (t *WebDAVTarget) Get(_ context.Context, ref TargetRef) (io.ReadCloser, error) {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return nil, err
	}
	c := t.client()
	rc, err := c.ReadStream(dest)
	if err != nil {
		if isWebDAVNotFound(err) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("webdav get %s: %w", dest, err)
	}
	return rc, nil
}

func (t *WebDAVTarget) Delete(_ context.Context, ref TargetRef) error {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return err
	}
	c := t.client()
	if err := c.Remove(dest); err != nil {
		if isWebDAVNotFound(err) {
			return nil
		}
		return fmt.Errorf("webdav remove: %w", err)
	}
	return nil
}

func (t *WebDAVTarget) HealthCheck(ctx context.Context) error {
	c := t.client()

	// Touch the base URL via PROPFIND on the root — fastest way
	// to confirm "auth + reachability" without writing.
	if _, err := c.ReadDir("/"); err != nil {
		return fmt.Errorf("webdav probe propfind: %w", err)
	}

	// Round-trip a tiny probe to confirm write access too.
	probe := ".healthcheck-" + NewDownloadToken()
	if t.cfg.PathPrefix != "" {
		probe = t.cfg.PathPrefix + "/" + probe
	}
	probe = "/" + strings.TrimLeft(probe, "/")

	if t.cfg.PathPrefix != "" {
		if err := t.ensureDir(c, t.cfg.PathPrefix); err != nil {
			return fmt.Errorf("webdav probe mkdir prefix: %w", err)
		}
	}
	if err := c.Write(probe, []byte("ok"), 0o600); err != nil {
		return fmt.Errorf("webdav probe write: %w", err)
	}
	if err := c.Remove(probe); err != nil {
		return fmt.Errorf("webdav probe cleanup: %w", err)
	}
	_ = ctx // gowebdav doesn't honour ctx; client timeouts cover us.
	return nil
}

func isWebDAVNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	// gowebdav surfaces 404 as a plain string error.
	s := err.Error()
	return strings.Contains(s, "404") || strings.Contains(s, "Not Found")
}
