package pgbackup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeDumper struct {
	data []byte
	err  error
}

func (f *fakeDumper) Dump(_ context.Context, w io.Writer) error {
	if f.err != nil {
		return f.err
	}
	_, err := w.Write(f.data)
	return err
}

type uploadRecord struct {
	key         string
	data        []byte
	contentType string
}

type fakeStore struct {
	prefix    string
	uploads   []uploadRecord
	uploadErr error
}

func (s *fakeStore) Key(name string) string {
	if s.prefix == "" {
		return name
	}
	return s.prefix + "/" + name
}

func (s *fakeStore) Upload(_ context.Context, path, key, contentType string) (string, error) {
	if s.uploadErr != nil {
		return "", s.uploadErr
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s.uploads = append(s.uploads, uploadRecord{key: key, data: data, contentType: contentType})
	return "fake-etag", nil
}

type fakeNotifier struct {
	sent    []Payload
	ctxErrs []error // ctx.Err() captured at Send time
	err     error
}

func (n *fakeNotifier) Send(ctx context.Context, p Payload) error {
	n.sent = append(n.sent, p)
	n.ctxErrs = append(n.ctxErrs, ctx.Err())
	return n.err
}

func newTestApp(t *testing.T, dumper *fakeDumper, store *fakeStore, notifier Notifier) (*App, time.Time) {
	t.Helper()
	now := time.Now()
	cfg := Config{
		OutputDir:     t.TempDir(),
		S3Prefix:      "backups",
		RetentionDays: 7,
	}
	if store != nil {
		store.prefix = cfg.S3Prefix
	}
	a := &App{
		cfg:       cfg,
		dumper:    dumper,
		store:     store,
		notifier:  notifier,
		log:       slog.New(slog.DiscardHandler),
		now:       func() time.Time { return now },
		diskUsage: func(string) (Usage, error) { return Usage{Total: 100, Used: 50, Available: 50}, nil },
	}
	return a, now
}

func backupFiles(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "backup_*.sql.zst"))
	if err != nil {
		t.Fatalf("glob error = %v", err)
	}
	return matches
}

func TestRun_Success(t *testing.T) {
	t.Parallel()

	sql := []byte("-- PostgreSQL database dump\nSELECT 1;\n")
	store := &fakeStore{}
	notifier := &fakeNotifier{}
	a, _ := newTestApp(t, &fakeDumper{data: sql}, store, notifier)
	a.cfg.OutputDir = filepath.Join(a.cfg.OutputDir, "sub")

	if err := a.Run(t.Context()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	files := backupFiles(t, a.cfg.OutputDir)
	if len(files) != 1 {
		t.Fatalf("local backup files = %d, want 1", len(files))
	}
	// pg_dump compresses its own output, so the dump bytes are written verbatim.
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, sql) {
		t.Errorf("local backup = %q, want %q", raw, sql)
	}

	if len(store.uploads) != 1 {
		t.Fatalf("uploads = %d, want 1", len(store.uploads))
	}
	up := store.uploads[0]
	if up.contentType != contentType {
		t.Errorf("content-type = %q, want %q", up.contentType, contentType)
	}
	if filepath.Base(up.key) != filepath.Base(files[0]) {
		t.Errorf("upload key base = %q, want %q", filepath.Base(up.key), filepath.Base(files[0]))
	}
	if !bytes.Equal(up.data, sql) {
		t.Errorf("uploaded data = %q, want %q", up.data, sql)
	}

	if len(notifier.sent) != 1 || notifier.sent[0].Embeds[0].Color != colorSuccess {
		t.Error("want one success notification")
	}

	if fi, err := os.Stat(files[0]); err != nil {
		t.Fatal(err)
	} else if perm := fi.Mode().Perm(); perm != filePerm {
		t.Errorf("backup file perm = %o, want %o", perm, filePerm)
	}
	if fi, err := os.Stat(a.cfg.OutputDir); err != nil {
		t.Fatal(err)
	} else if perm := fi.Mode().Perm(); perm != dirPerm {
		t.Errorf("output dir perm = %o, want %o", perm, dirPerm)
	}
}

func TestRun_PrunesOldLocalBackups(t *testing.T) {
	t.Parallel()

	a, now := newTestApp(t, &fakeDumper{data: []byte("SELECT 1;")}, &fakeStore{}, &fakeNotifier{})
	oldTime := now.AddDate(0, 0, -30)

	oldBackup := filepath.Join(a.cfg.OutputDir, "backup_20200101_000000.sql.zst")
	keep := filepath.Join(a.cfg.OutputDir, "notes.txt")
	for _, p := range []string{oldBackup, keep} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, oldTime, oldTime); err != nil {
			t.Fatal(err)
		}
	}

	if err := a.Run(t.Context()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Stat(oldBackup); !errors.Is(err, os.ErrNotExist) {
		t.Error("old backup was not pruned")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Error("non-backup file was incorrectly removed")
	}
}

func TestRun_RetentionDisabled(t *testing.T) {
	t.Parallel()

	a, now := newTestApp(t, &fakeDumper{data: []byte("x")}, &fakeStore{}, nil)
	a.cfg.RetentionDays = 0

	old := filepath.Join(a.cfg.OutputDir, "backup_20200101_000000.sql.zst")
	if err := os.WriteFile(old, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := now.AddDate(0, 0, -30)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	if err := a.Run(t.Context()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(old); err != nil {
		t.Error("old backup removed despite retention disabled")
	}
}

func TestRun_DumpFailure(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	notifier := &fakeNotifier{}
	a, _ := newTestApp(t, &fakeDumper{err: errors.New("pg_dump exploded")}, store, notifier)

	if err := a.Run(t.Context()); err == nil {
		t.Fatal("Run() error = nil, want error")
	}

	if files := backupFiles(t, a.cfg.OutputDir); len(files) != 0 {
		t.Errorf("partial backup files remain: %v", files)
	}
	if len(store.uploads) != 0 {
		t.Errorf("uploads = %d, want 0", len(store.uploads))
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("notifications = %d, want 1", len(notifier.sent))
	}
	if notifier.sent[0].Embeds[0].Color != colorFailure {
		t.Error("notification is not a failure embed")
	}
	if notifier.sent[0].Embeds[0].Description != "pg_dump exploded" {
		t.Errorf("failure description = %q", notifier.sent[0].Embeds[0].Description)
	}
}

func TestRun_UploadFailure(t *testing.T) {
	t.Parallel()

	store := &fakeStore{uploadErr: errors.New("s3 unreachable")}
	notifier := &fakeNotifier{}
	a, _ := newTestApp(t, &fakeDumper{data: []byte("SELECT 1;")}, store, notifier)

	if err := a.Run(t.Context()); err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if len(notifier.sent) != 1 || notifier.sent[0].Embeds[0].Color != colorFailure {
		t.Error("want one failure notification")
	}
}

func TestRun_NoNotifier(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	a, _ := newTestApp(t, &fakeDumper{data: []byte("x")}, store, nil)

	if err := a.Run(t.Context()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(store.uploads) != 1 {
		t.Errorf("uploads = %d, want 1", len(store.uploads))
	}
}

func TestRun_NotifierFailureNonFatal(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{err: errors.New("discord down")}
	a, _ := newTestApp(t, &fakeDumper{data: []byte("x")}, &fakeStore{}, notifier)

	if err := a.Run(t.Context()); err != nil {
		t.Fatalf("Run() error = %v, want nil despite notifier failure", err)
	}
}

func TestRun_NotifiesEvenWhenContextCanceled(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	a, _ := newTestApp(t, &fakeDumper{err: errors.New("interrupted")}, &fakeStore{}, notifier)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := a.Run(ctx); err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("notifications = %d, want 1", len(notifier.sent))
	}
	if notifier.ctxErrs[0] != nil {
		t.Errorf("notification ctx err = %v, want nil (detached from canceled run)", notifier.ctxErrs[0])
	}
}

func TestPruneLocal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Now()
	cutoff := now.AddDate(0, 0, -7)

	files := []struct {
		name    string
		age     time.Duration
		deleted bool
	}{
		{"backup_1.sql.zst", 10 * 24 * time.Hour, true},
		{"backup_2.sql.zst", 8 * 24 * time.Hour, true},
		{"backup_3.sql.zst", 3 * 24 * time.Hour, false},
		{"backup_4.sql.zst", 0, false},
		{"other.txt", 30 * 24 * time.Hour, false},    // name doesn't match
		{"backup_x.sql", 30 * 24 * time.Hour, false}, // extension doesn't match
	}
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		mt := now.Add(-f.age)
		if err := os.Chtimes(path, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := pruneLocal(dir, cutoff)
	if err != nil {
		t.Fatalf("pruneLocal() error = %v", err)
	}
	if len(deleted) != 2 {
		t.Errorf("deleted %d files, want 2: %v", len(deleted), deleted)
	}
	for _, f := range files {
		_, statErr := os.Stat(filepath.Join(dir, f.name))
		gone := errors.Is(statErr, os.ErrNotExist)
		if gone != f.deleted {
			t.Errorf("file %q: deleted=%v, want %v", f.name, gone, f.deleted)
		}
	}
}

func TestPruneLocal_EmptyDir(t *testing.T) {
	t.Parallel()

	deleted, err := pruneLocal(t.TempDir(), time.Now())
	if err != nil {
		t.Fatalf("pruneLocal() error = %v", err)
	}
	if len(deleted) != 0 {
		t.Errorf("deleted = %v, want empty", deleted)
	}
}
