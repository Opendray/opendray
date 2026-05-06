package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config drives an S3-compatible target. Plaintext fields —
// SecretKey is decrypted from backup_targets.config before reaching
// here.
//
// The same struct serves AWS S3, Cloudflare R2, Backblaze B2,
// MinIO self-hosted, DigitalOcean Spaces, Wasabi, 阿里云 OSS,
// 腾讯云 COS, etc. The differences are all encoded in Endpoint /
// Region / UseSSL.
type S3Config struct {
	Endpoint   string // "s3.amazonaws.com" / "<acct>.r2.cloudflarestorage.com" / "minio.local:9000"
	Region     string // "us-east-1" / "auto" / "" for libdefault
	Bucket     string
	AccessKey  string
	SecretKey  string
	UseSSL     bool   // default true; flip off for local MinIO over plain HTTP
	PathStyle  bool   // S3 path-style (legacy) vs. virtual-hosted; needed for some MinIO setups
	PathPrefix string // optional, e.g. "opendray/backups"
}

// S3Target writes blobs to an S3-compatible bucket via minio-go.
//
// Atomic writes: the SDK handles multipart upload internally so a
// crashed transfer leaves no half-written object visible at the
// final key (S3 semantics — objects appear only after the multipart
// completion, otherwise the multipart upload is abandoned and
// eventually GC'd by the bucket's lifecycle rules if configured).
type S3Target struct {
	id        string
	cfg       S3Config
	requestTO time.Duration
}

// NewS3Target validates cfg and constructs but does not connect.
// HealthCheck / Put / Get make per-op connections.
func NewS3Target(id string, cfg S3Config) (*S3Target, error) {
	if id == "" {
		return nil, errors.New("s3 target: id required")
	}
	if cfg.Endpoint == "" {
		return nil, errors.New("s3 target: endpoint required (e.g. s3.amazonaws.com)")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("s3 target: bucket required")
	}
	if cfg.AccessKey == "" {
		return nil, errors.New("s3 target: access_key required")
	}
	if cfg.SecretKey == "" {
		return nil, errors.New("s3 target: secret_key required")
	}
	cfg.PathPrefix = strings.Trim(cfg.PathPrefix, "/")
	return &S3Target{
		id:        id,
		cfg:       cfg,
		requestTO: 5 * time.Minute,
	}, nil
}

func (t *S3Target) Name() string     { return t.id }
func (t *S3Target) Kind() TargetKind { return TargetS3 }

func (t *S3Target) resolve(p string) (string, error) {
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

func (t *S3Target) client() (*minio.Client, error) {
	opts := &minio.Options{
		Creds:        credentials.NewStaticV4(t.cfg.AccessKey, t.cfg.SecretKey, ""),
		Secure:       t.cfg.UseSSL,
		BucketLookup: minio.BucketLookupAuto,
		Region:       t.cfg.Region,
	}
	if t.cfg.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}
	c, err := minio.New(t.cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("s3 client: %w", err)
	}
	return c, nil
}

func (t *S3Target) Put(ctx context.Context, p string, r io.Reader, size int64) (TargetRef, error) {
	dest, err := t.resolve(p)
	if err != nil {
		return TargetRef{}, err
	}
	client, err := t.client()
	if err != nil {
		return TargetRef{}, err
	}
	tctx, cancel := context.WithTimeout(ctx, t.requestTO)
	defer cancel()

	// Hash + count locally so we can return the same TargetRef
	// shape every other target uses. minio-go could give us the
	// server's ETag, but ETag != SHA-256 (it's MD5 for single PUT
	// or a multipart-aware hash for multipart), so we tee.
	hasher := sha256.New()
	tee := io.TeeReader(&ctxReader{ctx: ctx, r: r}, hasher)

	// size=-1 forces multipart (PartSize default 16 MiB). Pass the
	// real size when known so a single PUT is used.
	uploadSize := int64(-1)
	if size > 0 {
		uploadSize = size
	}

	info, err := client.PutObject(tctx, t.cfg.Bucket, dest, tee, uploadSize, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return TargetRef{}, fmt.Errorf("s3 put: %w", err)
	}

	return TargetRef{
		Target: t.id,
		Path:   p,
		Bytes:  info.Size,
		SHA256: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func (t *S3Target) Get(ctx context.Context, ref TargetRef) (io.ReadCloser, error) {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return nil, err
	}
	client, err := t.client()
	if err != nil {
		return nil, err
	}
	obj, err := client.GetObject(ctx, t.cfg.Bucket, dest, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3 get: %w", err)
	}
	// Probe StatObject so a missing key surfaces as ErrBackupNotFound
	// before the caller starts reading (otherwise the error appears
	// only on first Read, which is harder to handle in HTTP layers).
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		if isS3NotFound(err) {
			return nil, ErrBackupNotFound
		}
		return nil, fmt.Errorf("s3 stat: %w", err)
	}
	return obj, nil
}

func (t *S3Target) Delete(ctx context.Context, ref TargetRef) error {
	dest, err := t.resolve(ref.Path)
	if err != nil {
		return err
	}
	client, err := t.client()
	if err != nil {
		return err
	}
	tctx, cancel := context.WithTimeout(ctx, t.requestTO)
	defer cancel()
	err = client.RemoveObject(tctx, t.cfg.Bucket, dest, minio.RemoveObjectOptions{})
	if err != nil {
		if isS3NotFound(err) {
			return nil // idempotent
		}
		return fmt.Errorf("s3 remove: %w", err)
	}
	return nil
}

func (t *S3Target) HealthCheck(ctx context.Context) error {
	client, err := t.client()
	if err != nil {
		return err
	}
	tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// BucketExists is the cheapest auth+reachability probe — it
	// hits HEAD /<bucket> and validates credentials end-to-end.
	exists, err := client.BucketExists(tctx, t.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("s3 bucket-exists probe: %w", err)
	}
	if !exists {
		return fmt.Errorf("s3 target: bucket %q not found (or no permission)", t.cfg.Bucket)
	}
	// Round-trip a tiny probe object so we know the credentials
	// can also write.
	probeKey := ".healthcheck-" + NewDownloadToken()
	if t.cfg.PathPrefix != "" {
		probeKey = t.cfg.PathPrefix + "/" + probeKey
	}
	probeBody := strings.NewReader("ok")
	if _, err := client.PutObject(tctx, t.cfg.Bucket, probeKey, probeBody, 2,
		minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		return fmt.Errorf("s3 probe write: %w", err)
	}
	if err := client.RemoveObject(tctx, t.cfg.Bucket, probeKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("s3 probe cleanup: %w", err)
	}
	return nil
}

// isS3NotFound recognises minio-go's "NoSuchKey" / "NoSuchBucket"
// without leaking the SDK type into other files.
func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	e := minio.ToErrorResponse(err)
	switch e.Code {
	case "NoSuchKey", "NoSuchBucket", "NotFound":
		return true
	}
	return false
}
