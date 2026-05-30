package pgbackup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPgDumper_Args(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts Options
		want []string
	}{
		{
			name: "database only",
			opts: Options{Database: "database"},
			want: []string{"--no-password", "--dbname", "database"},
		},
		{
			name: "with user",
			opts: Options{Database: "database", User: "postgres"},
			want: []string{"--username", "postgres", "--no-password", "--dbname", "database"},
		},
		{
			name: "with host and port",
			opts: Options{Database: "database", User: "postgres", Host: "localhost", Port: "5432"},
			want: []string{"--username", "postgres", "--host", "localhost", "--port", "5432", "--no-password", "--dbname", "database"},
		},
		{
			name: "host without user",
			opts: Options{Database: "database", Host: "db.example.com"},
			want: []string{"--host", "db.example.com", "--no-password", "--dbname", "database"},
		},
		{
			name: "with compression level",
			opts: Options{Database: "database", User: "postgres", CompressionLevel: 8},
			want: []string{"--username", "postgres", "--compress=zstd:8", "--no-password", "--dbname", "database"},
		},
		{
			name: "zero compression level omitted",
			opts: Options{Database: "database", CompressionLevel: 0},
			want: []string{"--no-password", "--dbname", "database"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewPgDumper(tt.opts).Args()
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Errorf("Args() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPgDumper_command(t *testing.T) {
	t.Parallel()

	if got := (&PgDumper{}).command(); got != "pg_dump" {
		t.Errorf("default command = %q, want pg_dump", got)
	}
	if got := (&PgDumper{Command: "/usr/bin/pg_dump"}).command(); got != "/usr/bin/pg_dump" {
		t.Errorf("command = %q, want /usr/bin/pg_dump", got)
	}
}

func writeFakeCommand(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-pg_dump")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake command: %v", err)
	}
	return path
}

// These exec tests avoid t.Parallel(): a concurrent fork can inherit the
// just-written executable's write fd and cause ETXTBSY.

func TestDump_Success(t *testing.T) {
	cmd := writeFakeCommand(t, "#!/bin/sh\nprintf '%s\\n' '-- PostgreSQL database dump' 'SELECT 1;'\n")
	d := NewPgDumper(Options{Database: "test", User: "postgres"})
	d.Command = cmd

	var out strings.Builder
	if err := d.Dump(t.Context(), &out); err != nil {
		t.Fatalf("Dump() error = %v", err)
	}
	if !strings.Contains(out.String(), "PostgreSQL database dump") {
		t.Errorf("stdout = %q, missing dump header", out.String())
	}
	if !strings.Contains(out.String(), "SELECT 1;") {
		t.Errorf("stdout = %q, missing SQL", out.String())
	}
}

func TestDump_FailureWithStderr(t *testing.T) {
	cmd := writeFakeCommand(t, "#!/bin/sh\nprintf 'connection to server failed' >&2\nexit 1\n")
	d := NewPgDumper(Options{Database: "test"})
	d.Command = cmd

	var out strings.Builder
	err := d.Dump(t.Context(), &out)
	if err == nil {
		t.Fatal("Dump() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "pg_dump failed") {
		t.Errorf("error = %v, want prefix 'pg_dump failed'", err)
	}
	if !strings.Contains(err.Error(), "connection to server failed") {
		t.Errorf("error = %v, want stderr message included", err)
	}
}

func TestDump_FailureSilent(t *testing.T) {
	cmd := writeFakeCommand(t, "#!/bin/sh\nexit 3\n")
	d := NewPgDumper(Options{Database: "test"})
	d.Command = cmd

	var out strings.Builder
	err := d.Dump(t.Context(), &out)
	if err == nil {
		t.Fatal("Dump() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "pg_dump failed") {
		t.Errorf("error = %v, want prefix 'pg_dump failed'", err)
	}
}

func TestDump_CommandNotFound(t *testing.T) {
	d := NewPgDumper(Options{Database: "test"})
	d.Command = filepath.Join(t.TempDir(), "does-not-exist")

	var out strings.Builder
	if err := d.Dump(t.Context(), &out); err == nil {
		t.Fatal("Dump() error = nil, want error for missing command")
	}
}

func TestDump_CanceledContext(t *testing.T) {
	cmd := writeFakeCommand(t, "#!/bin/sh\nprintf 'ok'\n")
	d := NewPgDumper(Options{Database: "test"})
	d.Command = cmd

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before running

	var out strings.Builder
	if err := d.Dump(ctx, &out); err == nil {
		t.Fatal("Dump() error = nil, want error for canceled context")
	}
}
