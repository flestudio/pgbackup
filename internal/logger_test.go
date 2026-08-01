package pgbackup

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestConfig_Logger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format, wantSub string
	}{
		{"text", "msg=hello"},
		{"json", `"msg":"hello"`},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			(&Config{LogLevel: "info", LogFormat: tt.format}).Logger(&buf).Info("hello")
			if !strings.Contains(buf.String(), tt.wantSub) {
				t.Errorf("output %q does not contain %q", buf.String(), tt.wantSub)
			}
		})
	}
}

func TestConfig_Logger_Levels(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	(&Config{LogLevel: "info"}).Logger(&buf).Debug("hidden")
	if buf.Len() != 0 {
		t.Errorf("debug at info level logged %q", buf.String())
	}
	(&Config{LogLevel: "debug"}).Logger(&buf).Debug("shown")
	if !strings.Contains(buf.String(), "msg=shown") {
		t.Errorf("debug at debug level logged %q, want msg=shown", buf.String())
	}
}

func TestLogDumpProgress(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f, err := os.Create(filepath.Join(t.TempDir(), "dump"))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = f.Close() }()
		if _, err := f.Write(make([]byte, 2048)); err != nil {
			t.Fatal(err)
		}

		var buf bytes.Buffer
		a := &App{log: slog.New(slog.NewTextHandler(&buf, nil))}
		stop := a.logDumpProgress(f)

		time.Sleep(progressInterval + time.Second)
		synctest.Wait()
		stop()

		got := buf.String()
		for _, want := range []string{`msg="dumping database"`, "bytes=2048", "size=", "speed=", "elapsed="} {
			if !strings.Contains(got, want) {
				t.Errorf("output %q does not contain %q", got, want)
			}
		}
	})
}

func TestLogUploadProgress(t *testing.T) {
	tests := []struct {
		name    string
		read    int
		wants   []string
		forbids []string
	}{
		{
			name:  "halfway",
			read:  500,
			wants: []string{`msg="upload progress"`, "key=k", "bytes=500", "percent=50", "eta="},
		},
		{
			name:    "overcount capped",
			read:    1500,
			wants:   []string{"percent=100"},
			forbids: []string{"eta="},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var buf bytes.Buffer
				s := &Store{log: slog.New(slog.NewTextHandler(&buf, nil))}
				prog := &progressReader{}
				stop := s.logUploadProgress("k", 1000, prog)
				if _, err := prog.Read(make([]byte, tt.read)); err != nil {
					t.Fatal(err)
				}

				time.Sleep(progressInterval + time.Second)
				synctest.Wait()
				stop()

				got := buf.String()
				for _, want := range tt.wants {
					if !strings.Contains(got, want) {
						t.Errorf("output %q does not contain %q", got, want)
					}
				}
				for _, forbid := range tt.forbids {
					if strings.Contains(got, forbid) {
						t.Errorf("output %q contains %q", got, forbid)
					}
				}
			})
		})
	}
}

func TestTransferSpeed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int64
		d    time.Duration
		want string
	}{
		{10 * 1024 * 1024, time.Second, "10.00 MiB/s"},
		{1024, 2 * time.Second, "512 Bytes/s"},
		{5 * 1024, 0, "5.00 KiB/s"},
	}
	for _, tt := range tests {
		if got := transferSpeed(tt.n, tt.d); got != tt.want {
			t.Errorf("transferSpeed(%d, %v) = %q, want %q", tt.n, tt.d, got, tt.want)
		}
	}
}
