package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"
)

// SMBConfig holds an SMB target's connection params. Plaintext —
// the password is decrypted from backup_targets.config before
// reaching here. All fields are required except PathPrefix and
// Port (defaults to 445).
type SMBConfig struct {
	Host       string // "192.168.1.20" or "fileserver.local"
	Port       int    // default 445
	Share      string // "Claude_Workspace"
	User       string
	Password   string
	PathPrefix string // optional, e.g. "opendray/backups"
}

// SMBTarget writes blobs to an SMB/CIFS share using a pure-Go
// client (no host cifs-utils / mount.cifs dependency, so it works
// inside an unprivileged LXC).
//
// Connections are short-lived per-operation: dial, auth, mount,
// read/write, unmount, close. This avoids holding stale sockets
// across long idle periods at the cost of an extra round-trip per
// op — fine for backup cadence (minutes apart, not milliseconds).
type SMBTarget struct {
	id        string
	cfg       SMBConfig
	dialTO    time.Duration
	requestTO time.Duration
}

// NewSMBTarget validates cfg and returns a target. It does NOT
// connect; HealthCheck / Put / Get are responsible for that.
func NewSMBTarget(id string, cfg SMBConfig) (*SMBTarget, error) {
	if id == "" {
		return nil, errors.New("smb target: id required")
	}
	if cfg.Host == "" {
		return nil, errors.New("smb target: host required")
	}
	if cfg.Share == "" {
		return nil, errors.New("smb target: share required")
	}
	if cfg.User == "" {
		return nil, errors.New("smb target: user required")
	}
	if cfg.Port == 0 {
		cfg.Port = 445
	}
	cfg.PathPrefix = strings.Trim(cfg.PathPrefix, "/")
	return &SMBTarget{
		id:        id,
		cfg:       cfg,
		dialTO:    30 * time.Second,
		requestTO: 5 * time.Minute,
	}, nil
}

func (t *SMBTarget) Name() string     { return t.id }
func (t *SMBTarget) Kind() TargetKind { return TargetSMB }

// resolve returns the path *inside the SMB share*, with PathPrefix
// applied and traversal rejected. We reject any ".." anywhere in
// the input — even segments that would be neutralised by Clean —
// because their presence indicates intent to escape and we'd
// rather refuse than guess.
func (t *SMBTarget) resolve(p string) (string, error) {
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
	return cleaned, nil
}

func (t *SMBTarget) dial(ctx context.Context) (*smb2.Session, *smb2.Share, func(), error) {
	addr := fmt.Sprintf("%s:%d", t.cfg.Host, t.cfg.Port)

	dialer := net.Dialer{Timeout: t.dialTO}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("smb dial %s: %w", addr, err)
	}

	sd := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     t.cfg.User,
			Password: t.cfg.Password,
		},
	}
	sess, err := sd.DialContext(ctx, conn)
	if err != nil {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("smb auth: %w", err)
	}

	share, err := sess.Mount(t.cfg.Share)
	if err != nil {
		_ = sess.Logoff()
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("smb mount %q: %w", t.cfg.Share, err)
	}

	cleanup := func() {
		_ = share.Umount()
		_ = sess.Logoff()
		_ = conn.Close()
	}
	return sess, share, cleanup, nil
}

func (t *SMBTarget) Put(ctx context.Context, p string, r io.Reader, _ int64) (TargetRef, error) {
	dest, err := t.resolve(p)
	if err != nil {
		return TargetRef{}, err
	}
	_, share, done, err := t.dial(ctx)
	if err != nil {
		return TargetRef{}, err
	}
	defer done()

	if err := mkdirAllSMB(share, path.Dir(dest)); err != nil {
		return TargetRef{}, fmt.Errorf("smb mkdir parent: %w", err)
	}

	// Write to a sibling tmp file then rename for atomicity.
	tmp := path.Join(path.Dir(dest), "."+path.Base(dest)+".part")
	f, err := share.Create(tmp)
	if err != nil {
		return TargetRef{}, fmt.Errorf("smb create tmp: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = share.Remove(tmp)
		}
	}()

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(f, hasher), &ctxReader{ctx: ctx, r: r})
	if cerr := f.Close(); cerr != nil && copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		return TargetRef{}, fmt.Errorf("smb write: %w", copyErr)
	}

	if err := share.Rename(tmp, dest); err != nil {
		return TargetRef{}, fmt.Errorf("smb rename: %w", err)
	}
	committed = true

	return TargetRef{
		Target: t.id,
		Path:   p, // store user-facing path, not the prefixed form
		Bytes:  written,
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (t *SMBTarget) Get(ctx context.Context, ref TargetRef) (io.ReadCloser, error) {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return nil, err
	}
	_, share, done, err := t.dial(ctx)
	if err != nil {
		return nil, err
	}
	f, err := share.Open(dest)
	if err != nil {
		done()
		if os.IsNotExist(err) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("smb open: %w", err)
	}
	// Wrap so Close also tears down the SMB session.
	return &smbFileCloser{File: f, done: done}, nil
}

type smbFileCloser struct {
	*smb2.File
	done func()
}

func (s *smbFileCloser) Close() error {
	err := s.File.Close()
	s.done()
	return err
}

func (t *SMBTarget) Delete(ctx context.Context, ref TargetRef) error {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return err
	}
	_, share, done, err := t.dial(ctx)
	if err != nil {
		return err
	}
	defer done()
	if err := share.Remove(dest); err != nil {
		if os.IsNotExist(err) {
			return nil // idempotent
		}
		return fmt.Errorf("smb remove: %w", err)
	}
	return nil
}

func (t *SMBTarget) HealthCheck(ctx context.Context) error {
	_, share, done, err := t.dial(ctx)
	if err != nil {
		return err
	}
	defer done()
	probe := ".healthcheck-" + NewDownloadToken()
	if t.cfg.PathPrefix != "" {
		// also exercises the prefix dir creation
		if err := mkdirAllSMB(share, t.cfg.PathPrefix); err != nil {
			return fmt.Errorf("smb mkdir prefix: %w", err)
		}
		probe = t.cfg.PathPrefix + "/" + probe
	}
	f, err := share.Create(probe)
	if err != nil {
		return fmt.Errorf("smb probe create: %w", err)
	}
	if _, err := f.Write([]byte("ok")); err != nil {
		_ = f.Close()
		_ = share.Remove(probe)
		return fmt.Errorf("smb probe write: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = share.Remove(probe)
		return fmt.Errorf("smb probe close: %w", err)
	}
	if err := share.Remove(probe); err != nil {
		return fmt.Errorf("smb probe remove: %w", err)
	}
	return nil
}

// mkdirAllSMB creates the directory and any parents on the share.
// SMB has no native "mkdir -p"; recurse manually.
func mkdirAllSMB(share *smb2.Share, dir string) error {
	dir = strings.Trim(dir, "/")
	if dir == "" || dir == "." {
		return nil
	}
	parts := strings.Split(dir, "/")
	cur := ""
	for _, p := range parts {
		if cur == "" {
			cur = p
		} else {
			cur = cur + "/" + p
		}
		if _, err := share.Stat(cur); err == nil {
			continue
		}
		if err := share.Mkdir(cur, 0o700); err != nil {
			// Tolerate races: another goroutine may have just made it.
			if os.IsExist(err) {
				continue
			}
			return fmt.Errorf("mkdir %s: %w", cur, err)
		}
	}
	return nil
}
