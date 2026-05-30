package pgbackup

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rhnvrm/simples3"
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
	s3     *simples3.S3
	bucket string
	prefix string
}

func NewStore(cfg S3Config) *Store {
	s3 := simples3.New(cfg.Region, cfg.AccessKey, cfg.SecretKey)
	s3.SetEndpoint(cfg.Endpoint)
	s3.SetClient(&http.Client{Timeout: 5 * time.Minute})
	return &Store{s3: s3, bucket: cfg.Bucket, prefix: cfg.Prefix}
}

func (s *Store) Key(name string) string {
	if s.prefix == "" {
		return name
	}
	return s.prefix + "/" + name
}

func (s *Store) Upload(path, key, contentType string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	_, err = s.s3.FilePut(simples3.UploadInput{
		Bucket:      s.bucket,
		ObjectKey:   key,
		ContentType: contentType,
		Body:        f,
	})
	if err != nil {
		return fmt.Errorf("upload %q: %w", key, err)
	}
	return nil
}
