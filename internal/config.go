package pgbackup

import (
	"fmt"
	"time"
)

// Struct tags drive kong (flags/env/defaults) without importing it.
type Config struct {
	Database  string `help:"Database name to back up." env:"DATABASE" default:"database"`
	OutputDir string `help:"Directory to store local backups." env:"OUTPUT_DIR" default:"./backups" type:"path"`

	S3Bucket    string `name:"s3-bucket" help:"S3 bucket name." env:"S3_BUCKET" required:""`
	S3Region    string `name:"s3-region" help:"S3 region." env:"S3_REGION" default:"auto"`
	S3AccessKey string `name:"s3-access-key" help:"S3 access key ID." env:"S3_ACCESS_KEY" required:""`
	S3SecretKey string `name:"s3-secret-key" help:"S3 secret access key." env:"S3_SECRET_KEY" required:""`
	S3Endpoint  string `name:"s3-endpoint" help:"S3 endpoint URL (e.g. https://<account>.r2.cloudflarestorage.com)." env:"S3_ENDPOINT" required:""`
	S3Prefix    string `name:"s3-prefix" help:"Key prefix for uploaded objects." env:"S3_PREFIX" default:"backups"`

	DiscordWebhookURL string `help:"Discord webhook URL for notifications (optional)." env:"DISCORD_WEBHOOK_URL"`

	RetentionDays    int `help:"Delete local backups older than this many days (0 disables pruning)." env:"RETENTION_DAYS" default:"7"`
	CompressionLevel int `help:"zstd compression level (1-22)." env:"COMPRESSION_LEVEL" default:"8"`

	PgHost string `help:"PostgreSQL host (defaults to local socket)." env:"PGHOST"`
	PgPort string `help:"PostgreSQL port." env:"PGPORT"`
	PgUser string `help:"PostgreSQL user." env:"PGUSER" default:"postgres"`
}

func (c *Config) Validate() error {
	if c.CompressionLevel < 1 || c.CompressionLevel > 22 {
		return fmt.Errorf("compression level must be between 1 and 22, got %d", c.CompressionLevel)
	}
	if c.RetentionDays < 0 {
		return fmt.Errorf("retention days must not be negative, got %d", c.RetentionDays)
	}
	if c.Database == "" {
		return fmt.Errorf("database name must not be empty")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output directory must not be empty")
	}
	return nil
}

func (c *Config) RetentionEnabled() bool {
	return c.RetentionDays > 0
}

func (c *Config) NotifyEnabled() bool {
	return c.DiscordWebhookURL != ""
}

func (c *Config) Cutoff(now time.Time) time.Time {
	return now.AddDate(0, 0, -c.RetentionDays)
}

func (c *Config) StorageConfig() S3Config {
	return S3Config{
		Bucket:    c.S3Bucket,
		Region:    c.S3Region,
		AccessKey: c.S3AccessKey,
		SecretKey: c.S3SecretKey,
		Endpoint:  c.S3Endpoint,
		Prefix:    c.S3Prefix,
	}
}

func (c *Config) DumpOptions() Options {
	return Options{
		Database:         c.Database,
		User:             c.PgUser,
		Host:             c.PgHost,
		Port:             c.PgPort,
		CompressionLevel: c.CompressionLevel,
	}
}
