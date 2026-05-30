package pgbackup

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestStore_Key(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefix, name, want string
	}{
		{"backups", "b.sql.zst", "backups/b.sql.zst"},
		{"", "b.sql.zst", "b.sql.zst"},
		{"a/b", "c.zst", "a/b/c.zst"},
	}
	for _, tt := range tests {
		if got := (&Store{prefix: tt.prefix}).Key(tt.name); got != tt.want {
			t.Errorf("Key(%q) prefix %q = %q, want %q", tt.name, tt.prefix, got, tt.want)
		}
	}
}

// fakeS3 is a minimal path-style S3 server for exercising the simples3 client.
type fakeS3 struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (f *fakeS3) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.objects[parts[len(parts)-1]] = body
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
}

func TestStore_Upload(t *testing.T) {
	t.Parallel()

	fake := &fakeS3{objects: map[string][]byte{}}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	tmp := filepath.Join(t.TempDir(), "backup.sql.zst")
	if err := os.WriteFile(tmp, []byte("compressed backup data"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(S3Config{Bucket: "bucket", Region: "auto", AccessKey: "k", SecretKey: "s", Endpoint: srv.URL, Prefix: "backups"})
	key := store.Key("backup.sql.zst")
	if err := store.Upload(tmp, key, "application/zstd"); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	fake.mu.Lock()
	got := string(fake.objects[key])
	fake.mu.Unlock()
	if got != "compressed backup data" {
		t.Errorf("stored body = %q", got)
	}
}
