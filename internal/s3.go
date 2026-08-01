package pgbackup

import (
	"context"
	"fmt"
	"net/url"

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
}

func NewStore(cfg S3Config) (*Store, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint %q: %w", cfg.Endpoint, err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("endpoint must be a URL like https://<account>.r2.cloudflarestorage.com, got %q", cfg.Endpoint)
	}
	client, err := minio.New(u.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: u.Scheme != "http",
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	return &Store{client: client, bucket: cfg.Bucket, prefix: cfg.Prefix}, nil
}

func (s *Store) Key(name string) string {
	if s.prefix == "" {
		return name
	}
	return s.prefix + "/" + name
}

func (s *Store) Upload(ctx context.Context, path, key, contentType string) error {
	_, err := s.client.FPutObject(ctx, s.bucket, key, path, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("upload %q: %w", key, err)
	}
	return nil
}
