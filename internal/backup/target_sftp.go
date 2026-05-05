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

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPConfig drives an SFTP target. Plaintext fields — Password and
// PrivateKey are decrypted from backup_targets.config before reaching
// here.
//
// Auth modes (in priority order):
//  1. PrivateKey set         → publickey auth, optional Password as the
//                              key passphrase
//  2. Password set, no key   → password auth
//
// HostKey can be an explicit OpenSSH-format public key string or
// empty; empty means "InsecureIgnoreHostKey, log a warning". Pinning
// is strongly encouraged for any non-LAN target.
type SFTPConfig struct {
	Host       string // "vps.example.com"
	Port       int    // default 22
	User       string
	Password   string // password or key passphrase
	PrivateKey string // PEM contents, not a file path
	HostKey    string // "ssh-ed25519 AAAA…" — pinned; empty disables pinning
	PathPrefix string // optional sub-folder; absolute or relative to user home
}

// SFTPTarget writes blobs to an SSH-accessible host via SFTP.
//
// Connections are short-lived per-operation: dial → handshake →
// open SFTP subsystem → read/write → close everything. SSH keep-
// alives + opendray's backup cadence (minutes apart) make this
// cheap, and avoiding pooled state simplifies error recovery.
type SFTPTarget struct {
	id        string
	cfg       SFTPConfig
	dialTO    time.Duration
	requestTO time.Duration
}

// NewSFTPTarget validates cfg and constructs but does not connect.
func NewSFTPTarget(id string, cfg SFTPConfig) (*SFTPTarget, error) {
	if id == "" {
		return nil, errors.New("sftp target: id required")
	}
	if cfg.Host == "" {
		return nil, errors.New("sftp target: host required")
	}
	if cfg.User == "" {
		return nil, errors.New("sftp target: user required")
	}
	if cfg.Password == "" && cfg.PrivateKey == "" {
		return nil, errors.New("sftp target: password or private_key required")
	}
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	cfg.PathPrefix = strings.TrimRight(cfg.PathPrefix, "/")
	return &SFTPTarget{
		id:        id,
		cfg:       cfg,
		dialTO:    30 * time.Second,
		requestTO: 5 * time.Minute,
	}, nil
}

func (t *SFTPTarget) Name() string     { return t.id }
func (t *SFTPTarget) Kind() TargetKind { return TargetSFTP }

// resolve returns the absolute path on the SFTP server.
//   - Absolute PathPrefix → result is "<prefix>/<p>"
//   - Relative or empty PathPrefix → result is interpreted by sftp
//     server (typically against $HOME of cfg.User)
func (t *SFTPTarget) resolve(p string) (string, error) {
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
		if strings.HasPrefix(t.cfg.PathPrefix, "/") {
			return t.cfg.PathPrefix + "/" + cleaned, nil
		}
		return t.cfg.PathPrefix + "/" + cleaned, nil
	}
	return cleaned, nil
}

func (t *SFTPTarget) sshConfig() (*ssh.ClientConfig, error) {
	auths := []ssh.AuthMethod{}
	if t.cfg.PrivateKey != "" {
		var signer ssh.Signer
		var err error
		if t.cfg.Password != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(t.cfg.PrivateKey), []byte(t.cfg.Password))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(t.cfg.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("sftp: parse private key: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	} else if t.cfg.Password != "" {
		auths = append(auths, ssh.Password(t.cfg.Password))
	}
	if len(auths) == 0 {
		return nil, errors.New("sftp: no auth method available")
	}

	hostKeyCallback := ssh.InsecureIgnoreHostKey() //nolint:gosec
	if t.cfg.HostKey != "" {
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(t.cfg.HostKey))
		if err != nil {
			return nil, fmt.Errorf("sftp: parse host_key: %w", err)
		}
		hostKeyCallback = ssh.FixedHostKey(key)
	}

	return &ssh.ClientConfig{
		User:            t.cfg.User,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback,
		Timeout:         t.dialTO,
	}, nil
}

func (t *SFTPTarget) dial(ctx context.Context) (*ssh.Client, *sftp.Client, func(), error) {
	cfg, err := t.sshConfig()
	if err != nil {
		return nil, nil, nil, err
	}

	d := net.Dialer{Timeout: t.dialTO}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", t.cfg.Host, t.cfg.Port))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("sftp dial: %w", err)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, fmt.Sprintf("%s:%d", t.cfg.Host, t.cfg.Port), cfg)
	if err != nil {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("sftp ssh handshake: %w", err)
	}
	sshClient := ssh.NewClient(c, chans, reqs)

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, nil, fmt.Errorf("sftp subsystem: %w", err)
	}
	cleanup := func() {
		_ = sftpClient.Close()
		_ = sshClient.Close()
	}
	return sshClient, sftpClient, cleanup, nil
}

func (t *SFTPTarget) ensureDir(c *sftp.Client, dir string) error {
	if dir == "" || dir == "." || dir == "/" {
		return nil
	}
	if _, err := c.Stat(dir); err == nil {
		return nil
	}
	if err := c.MkdirAll(dir); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return nil
}

func (t *SFTPTarget) Put(ctx context.Context, p string, r io.Reader, _ int64) (TargetRef, error) {
	dest, err := t.resolve(p)
	if err != nil {
		return TargetRef{}, err
	}
	_, c, done, err := t.dial(ctx)
	if err != nil {
		return TargetRef{}, err
	}
	defer done()

	if err := t.ensureDir(c, path.Dir(dest)); err != nil {
		return TargetRef{}, fmt.Errorf("sftp mkdir parent: %w", err)
	}

	tmp := path.Join(path.Dir(dest), "."+path.Base(dest)+".part")
	f, err := c.Create(tmp)
	if err != nil {
		return TargetRef{}, fmt.Errorf("sftp create tmp: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = c.Remove(tmp)
		}
	}()

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(f, hasher), &ctxReader{ctx: ctx, r: r})
	if cerr := f.Close(); cerr != nil && copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		return TargetRef{}, fmt.Errorf("sftp write: %w", copyErr)
	}

	// SFTP's Rename doesn't overwrite by default; remove first
	// (best-effort) so updating an existing key works.
	_ = c.Remove(dest)
	if err := c.Rename(tmp, dest); err != nil {
		return TargetRef{}, fmt.Errorf("sftp rename: %w", err)
	}
	committed = true

	return TargetRef{
		Target: t.id,
		Path:   p,
		Bytes:  written,
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (t *SFTPTarget) Get(ctx context.Context, ref TargetRef) (io.ReadCloser, error) {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return nil, err
	}
	_, c, done, err := t.dial(ctx)
	if err != nil {
		return nil, err
	}
	f, err := c.Open(dest)
	if err != nil {
		done()
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("sftp open: %w", err)
	}
	return &sftpFileCloser{File: f, done: done}, nil
}

type sftpFileCloser struct {
	*sftp.File
	done func()
}

func (s *sftpFileCloser) Close() error {
	err := s.File.Close()
	s.done()
	return err
}

func (t *SFTPTarget) Delete(ctx context.Context, ref TargetRef) error {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return err
	}
	_, c, done, err := t.dial(ctx)
	if err != nil {
		return err
	}
	defer done()
	if err := c.Remove(dest); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("sftp remove: %w", err)
	}
	return nil
}

func (t *SFTPTarget) HealthCheck(ctx context.Context) error {
	_, c, done, err := t.dial(ctx)
	if err != nil {
		return err
	}
	defer done()

	probe := ".healthcheck-" + NewDownloadToken()
	if t.cfg.PathPrefix != "" {
		if err := t.ensureDir(c, t.cfg.PathPrefix); err != nil {
			return fmt.Errorf("sftp probe mkdir prefix: %w", err)
		}
		probe = t.cfg.PathPrefix + "/" + probe
	}

	f, err := c.Create(probe)
	if err != nil {
		return fmt.Errorf("sftp probe create: %w", err)
	}
	if _, err := f.Write([]byte("ok")); err != nil {
		_ = f.Close()
		_ = c.Remove(probe)
		return fmt.Errorf("sftp probe write: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = c.Remove(probe)
		return fmt.Errorf("sftp probe close: %w", err)
	}
	if err := c.Remove(probe); err != nil {
		return fmt.Errorf("sftp probe remove: %w", err)
	}
	return nil
}
