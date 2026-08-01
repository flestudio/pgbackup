package pgbackup

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Config struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	Endpoint  string
	Prefix    string
}

type Store struct {
	client *minio.Client
	bucket string
	prefix string
	log    *slog.Logger
}

func NewStore(cfg S3Config, log *slog.Logger) (*Store, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", cfg.Endpoint, err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("endpoint must be a URL like https://<account>.r2.cloudflarestorage.com, got %q", cfg.Endpoint)
	}
	client, err := minio.New(u.Host, &minio.Options{
		Creds:      credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:     u.Scheme != "http",
		Region:     cfg.Region,
		MaxRetries: 30,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	return &Store{client: client, bucket: cfg.Bucket, prefix: cfg.Prefix, log: log}, nil
}

func (s *Store) Key(name string) string {
	if s.prefix == "" {
		return name
	}
	return s.prefix + "/" + name
}

func (s *Store) Upload(ctx context.Context, path, key, contentType string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", path, err)
	}
	prog := &progressReader{}
	stop := s.logUploadProgress(key, fi.Size(), prog)
	defer stop()
	info, err := s.client.FPutObject(ctx, s.bucket, key, path, minio.PutObjectOptions{
		ContentType:    contentType,
		SendContentMd5: true,
		Progress:       prog,
	})
	if err != nil {
		return "", fmt.Errorf("upload %q: %w", key, err)
	}
	return info.ETag, nil
}

type progressReader struct {
	n atomic.Int64
}

func (p *progressReader) Read(b []byte) (int, error) {
	p.n.Add(int64(len(b)))
	return len(b), nil
}

func (s *Store) logUploadProgress(key string, total int64, prog *progressReader) func() {
	start := time.Now()
	ticker := time.NewTicker(progressInterval)
	done := make(chan struct{})
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				n := prog.n.Load()
				elapsed := time.Since(start)
				attrs := []any{
					"key", key,
					"bytes", n,
					"progress", humanizeBytes(uint64(n)) + "/" + humanizeBytes(uint64(total)),
					"speed", transferSpeed(n, elapsed),
				}
				if total > 0 {
					attrs = append(attrs, "percent", min(n*100/total, 100))
				}
				if n > 0 && n < total {
					eta := time.Duration(float64(total-n) / float64(n) * float64(elapsed))
					attrs = append(attrs, "eta", eta.Round(time.Second))
				}
				s.log.Info("upload progress", attrs...)
			}
		}
	}()
	return func() { close(done) }
}
