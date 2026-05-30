package pgbackup

import (
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		Database:         "database",
		OutputDir:        "./backups",
		S3Bucket:         "bucket",
		S3Region:         "auto",
		S3AccessKey:      "key",
		S3SecretKey:      "secret",
		S3Endpoint:       "https://example.r2.cloudflarestorage.com",
		S3Prefix:         "backups",
		RetentionDays:    7,
		CompressionLevel: 8,
		PgUser:           "postgres",
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"valid", func(*Config) {}, false},
		{"min compression", func(c *Config) { c.CompressionLevel = 1 }, false},
		{"max compression", func(c *Config) { c.CompressionLevel = 22 }, false},
		{"compression too low", func(c *Config) { c.CompressionLevel = 0 }, true},
		{"compression too high", func(c *Config) { c.CompressionLevel = 23 }, true},
		{"negative compression", func(c *Config) { c.CompressionLevel = -1 }, true},
		{"zero retention ok", func(c *Config) { c.RetentionDays = 0 }, false},
		{"negative retention", func(c *Config) { c.RetentionDays = -1 }, true},
		{"empty database", func(c *Config) { c.Database = "" }, true},
		{"empty output dir", func(c *Config) { c.OutputDir = "" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := validConfig()
			tt.mutate(&c)
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetentionEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		days int
		want bool
	}{
		{7, true},
		{1, true},
		{0, false},
	}
	for _, tt := range tests {
		c := Config{RetentionDays: tt.days}
		if got := c.RetentionEnabled(); got != tt.want {
			t.Errorf("RetentionEnabled(days=%d) = %v, want %v", tt.days, got, tt.want)
		}
	}
}

func TestNotifyEnabled(t *testing.T) {
	t.Parallel()

	if (&Config{}).NotifyEnabled() {
		t.Error("NotifyEnabled() = true for empty URL, want false")
	}
	if !(&Config{DiscordWebhookURL: "https://discord.com/x"}).NotifyEnabled() {
		t.Error("NotifyEnabled() = false for set URL, want true")
	}
}

func TestCutoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	c := Config{RetentionDays: 7}
	want := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	if got := c.Cutoff(now); !got.Equal(want) {
		t.Errorf("Cutoff() = %v, want %v", got, want)
	}
}
