package pgbackup

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Empty Host/Port fall back to pg_dump's defaults (PGHOST/PGPORT env or local socket).
type Options struct {
	Database         string
	User             string
	Host             string
	Port             string
	CompressionLevel int // zstd level; pg_dump compresses its own output
}

type Dumper interface {
	Dump(ctx context.Context, w io.Writer) error
}

type PgDumper struct {
	Options Options
	Command string // empty means "pg_dump"; overridden in tests
}

func NewPgDumper(opts Options) *PgDumper {
	return &PgDumper{Options: opts}
}

func (d *PgDumper) command() string {
	return cmp.Or(d.Command, "pg_dump")
}

// No -f: the compressed dump streams to stdout.
func (d *PgDumper) Args() []string {
	var args []string
	if d.Options.User != "" {
		args = append(args, "--username", d.Options.User)
	}
	if d.Options.Host != "" {
		args = append(args, "--host", d.Options.Host)
	}
	if d.Options.Port != "" {
		args = append(args, "--port", d.Options.Port)
	}
	if d.Options.CompressionLevel > 0 {
		args = append(args, fmt.Sprintf("--compress=zstd:%d", d.Options.CompressionLevel))
	}
	args = append(args, "--no-password", "--dbname", d.Options.Database)
	return args
}

func (d *PgDumper) Dump(ctx context.Context, w io.Writer) error {
	cmd := exec.CommandContext(ctx, d.command(), d.Args()...)
	cmd.Stdout = w

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return fmt.Errorf("pg_dump failed: %w", err)
		}
		return fmt.Errorf("pg_dump failed: %w: %s", err, msg)
	}
	return nil
}
