package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	log = c.Logger(os.Stderr)

	if err := c.Validate(); err != nil {
		log.Error("invalid configuration", "error", err)
		return err
	}

	app, err := pgbackup.New(c.Config, log)
	if err != nil {
		log.Error("initialize app", "error", err)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, 6*time.Hour)
	defer cancel()

	return app.Run(ctx)
}
