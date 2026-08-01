package pgbackup

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

func TestNewStore_InvalidEndpoint(t *testing.T) {
	t.Parallel()

	for _, endpoint := range []string{"", "example.com", "://bad"} {
		if _, err := NewStore(S3Config{Bucket: "b", Endpoint: endpoint}); err == nil {
			t.Errorf("NewStore(endpoint=%q) error = nil, want error", endpoint)
		}
	}
}

// fakeS3 is a minimal path-style S3 server for exercising the minio-go client.
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
		var body []byte
		var err error
		if strings.HasPrefix(r.Header.Get("X-Amz-Content-Sha256"), "STREAMING") {
			body, err = decodeAWSChunked(r.Body)
		} else {
			body, err = io.ReadAll(r.Body)
		}
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.objects[parts[len(parts)-1]] = body
		f.mu.Unlock()
		w.Header().Set("ETag", `"fake-etag"`)
		w.WriteHeader(http.StatusOK)
	})
}

func decodeAWSChunked(r io.Reader) ([]byte, error) {
	br := bufio.NewReader(r)
	var out []byte
	for {
		header, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		sizeHex, _, _ := strings.Cut(strings.TrimSpace(header), ";")
		size, err := strconv.ParseUint(sizeHex, 16, 32)
		if err != nil {
			return nil, err
		}
		if size == 0 {
			return out, nil
		}
		chunk := make([]byte, size)
		if _, err := io.ReadFull(br, chunk); err != nil {
			return nil, err
		}
		out = append(out, chunk...)
		if _, err := br.Discard(2); err != nil {
			return nil, err
		}
	}
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

	store, err := NewStore(S3Config{Bucket: "bucket", Region: "auto", AccessKey: "k", SecretKey: "s", Endpoint: srv.URL, Prefix: "backups"})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	key := store.Key("backup.sql.zst")
	etag, err := store.Upload(t.Context(), tmp, key, "application/zstd")
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if etag != "fake-etag" {
		t.Errorf("Upload() etag = %q, want %q", etag, "fake-etag")
	}

	fake.mu.Lock()
	got := string(fake.objects[key])
	fake.mu.Unlock()
	if got != "compressed backup data" {
		t.Errorf("stored body = %q", got)
	}
}
