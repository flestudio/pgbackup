package pgbackup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	contentType     = "application/zstd"
	filenamePrefix  = "backup_"
	filenameSuffix  = ".sql.zst"
	timestampFormat = "20060102_150405"
	notifyTimeout   = 30 * time.Second

	dirPerm  = 0o700
	filePerm = 0o600
)

type Storage interface {
	Key(name string) string
	Upload(path, key, contentType string) error
}

type Notifier interface {
	Send(ctx context.Context, payload Payload) error
}

type App struct {
	cfg      Config
	dumper   Dumper
	store    Storage
	notifier Notifier
	log      *slog.Logger

	now       func() time.Time
	diskUsage func(path string) (Usage, error)
}

func New(cfg Config, log *slog.Logger) *App {
	a := &App{
		cfg:       cfg,
		dumper:    NewPgDumper(cfg.DumpOptions()),
		store:     NewStore(cfg.StorageConfig()),
		log:       log,
		now:       time.Now,
		diskUsage: getDiskUsage,
	}
	if cfg.NotifyEnabled() {
		a.notifier = NewDiscordClient(cfg.DiscordWebhookURL, nil)
	}
	return a
}

type backupFile struct {
	name      string
	path      string
	size      int64
	createdAt time.Time
}

func (a *App) Run(ctx context.Context) error {
	file, err := a.create(ctx)
	if err != nil {
		a.log.Error("create backup", "error", err)
		a.notify(FailurePayload(err.Error()))
		return err
	}
	a.log.Info("backup created", "path", file.path, "size", file.size)

	if err := a.upload(file); err != nil {
		a.log.Error("upload backup", "error", err)
		a.notify(FailurePayload(err.Error()))
		return err
	}

	a.notify(a.successPayload(file))
	a.prune()
	return nil
}

func (a *App) create(ctx context.Context) (backupFile, error) {
	createdAt := a.now()
	name := filenamePrefix + createdAt.Format(timestampFormat) + filenameSuffix
	path := filepath.Join(a.cfg.OutputDir, name)

	if err := os.MkdirAll(a.cfg.OutputDir, dirPerm); err != nil {
		return backupFile{}, fmt.Errorf("create output dir: %w", err)
	}

	size, err := a.writeDump(ctx, path)
	if err != nil {
		_ = os.Remove(path)
		return backupFile{}, err
	}
	return backupFile{name: name, path: path, size: size, createdAt: createdAt}, nil
}

// pg_dump compresses its own output (--compress=zstd), so we stream it straight to disk.
func (a *App) writeDump(ctx context.Context, path string) (int64, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return 0, fmt.Errorf("create backup file: %w", err)
	}
	if err := a.dumper.Dump(ctx, f); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, fmt.Errorf("close backup file: %w", err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("stat backup file: %w", err)
	}
	return fi.Size(), nil
}

func (a *App) upload(file backupFile) error {
	key := a.store.Key(file.name)
	a.log.Info("uploading backup", "key", key)
	if err := a.store.Upload(file.path, key, contentType); err != nil {
		return err
	}
	a.log.Info("backup uploaded", "key", key)
	return nil
}

func (a *App) prune() {
	if !a.cfg.RetentionEnabled() {
		return
	}
	deleted, err := pruneLocal(a.cfg.OutputDir, a.cfg.Cutoff(a.now()))
	switch {
	case err != nil:
		a.log.Warn("prune backups", "error", err)
	case len(deleted) > 0:
		a.log.Info("pruned backups", "count", len(deleted))
	}
}

func (a *App) successPayload(file backupFile) Payload {
	du, err := a.diskUsage(a.cfg.OutputDir)
	if err != nil {
		a.log.Warn("disk usage", "error", err)
	}
	return SuccessPayload(BackupInfo{
		Path:      file.path,
		RemoteKey: a.store.Key(file.name),
		Size:      uint64(file.size),
		CreatedAt: file.createdAt,
	}, du)
}

// Detached from the run context so a failure notification still goes out on cancel (SIGTERM).
func (a *App) notify(p Payload) {
	if a.notifier == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()
	if err := a.notifier.Send(ctx, p); err != nil {
		a.log.Warn("send notification", "error", err)
	}
}

func pruneLocal(dir string, cutoff time.Time) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, filenamePrefix+"*"+filenameSuffix))
	if err != nil {
		return nil, err
	}
	var deleted []string
	for _, path := range matches {
		fi, err := os.Stat(path)
		if err != nil {
			return deleted, fmt.Errorf("stat %q: %w", path, err)
		}
		if !fi.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(path); err != nil {
			return deleted, fmt.Errorf("remove %q: %w", path, err)
		}
		deleted = append(deleted, path)
	}
	return deleted, nil
}
