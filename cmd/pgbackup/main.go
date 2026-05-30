package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/joho/godotenv"

	pgbackup "github.com/flestudio/pgbackup/internal"
)

var version = "dev"

type cli struct {
	pgbackup.Config
	Version kong.VersionFlag `short:"v"`
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error("load .env", "error", err)
		return err
	}

	var c cli
	kong.Parse(&c,
		kong.Name("pgbackup"),
		kong.Description("PostgreSQL backup tool for flestudio with S3 upload and Discord notifications"),
		kong.Vars{"version": version},
		kong.UsageOnError(),
	)

	if err := c.Validate(); err != nil {
		log.Error("invalid configuration", "error", err)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return pgbackup.New(c.Config, log).Run(ctx)
}
