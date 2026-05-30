package pgbackup

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSuccessPayload(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	info := BackupInfo{
		Path:      "/backups/backup.sql.zst",
		RemoteKey: "backups/backup.sql.zst",
		Size:      1024 * 1024,
		CreatedAt: created,
	}
	du := Usage{Total: 100, Used: 40, Available: 60}

	p := SuccessPayload(info, du)

	if !strings.Contains(p.Content, "作成しました") {
		t.Errorf("Content = %q, want success message", p.Content)
	}
	if len(p.Embeds) != 2 {
		t.Fatalf("len(Embeds) = %d, want 2", len(p.Embeds))
	}
	if p.Embeds[0].Color != colorSuccess {
		t.Errorf("Embeds[0].Color = %#x, want %#x", p.Embeds[0].Color, colorSuccess)
	}

	fields := p.Embeds[0].Fields
	if len(fields) != 4 {
		t.Fatalf("len(fields) = %d, want 4", len(fields))
	}
	if fields[0].Value != info.Path {
		t.Errorf("path field = %q, want %q", fields[0].Value, info.Path)
	}
	if fields[1].Value != "1.00 MiB" {
		t.Errorf("size field = %q, want 1.00 MiB", fields[1].Value)
	}
	if fields[2].Value != created.Format(time.RFC3339) {
		t.Errorf("created field = %q, want %q", fields[2].Value, created.Format(time.RFC3339))
	}
	if fields[3].Value != info.RemoteKey {
		t.Errorf("remote key field = %q, want %q", fields[3].Value, info.RemoteKey)
	}

	diskInfo := p.Embeds[1]
	if diskInfo.Title != "ディスク情報" {
		t.Errorf("disk embed title = %q", diskInfo.Title)
	}
	if got := diskInfo.Fields[3].Value; got != "40.0%" {
		t.Errorf("disk used percent = %q, want 40.0%%", got)
	}
}

func TestSuccessPayload_OmitsEmptyRemoteKey(t *testing.T) {
	t.Parallel()

	p := SuccessPayload(BackupInfo{Path: "/backups/b.sql.zst", Size: 1}, Usage{Total: 1})
	if got := len(p.Embeds[0].Fields); got != 3 {
		t.Errorf("fields = %d, want 3 when RemoteKey is empty", got)
	}
}

func TestFailurePayload(t *testing.T) {
	t.Parallel()

	p := FailurePayload("boom")
	if !strings.Contains(p.Content, "失敗しました") {
		t.Errorf("Content = %q, want failure message", p.Content)
	}
	if len(p.Embeds) != 1 {
		t.Fatalf("len(Embeds) = %d, want 1", len(p.Embeds))
	}
	if p.Embeds[0].Description != "boom" {
		t.Errorf("Description = %q, want boom", p.Embeds[0].Description)
	}
	if p.Embeds[0].Color != colorFailure {
		t.Errorf("Color = %#x, want %#x", p.Embeds[0].Color, colorFailure)
	}
}

func TestClient_Send_Success(t *testing.T) {
	t.Parallel()

	var gotBody []byte
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewDiscordClient(srv.URL, srv.Client())
	payload := FailurePayload("test error")
	if err := client.Send(t.Context(), payload); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	var decoded Payload
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("server received invalid JSON: %v", err)
	}
	if decoded.Embeds[0].Description != "test error" {
		t.Errorf("received description = %q", decoded.Embeds[0].Description)
	}
}

func TestClient_Send_ErrorStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"invalid webhook"}`)
	}))
	defer srv.Close()

	client := NewDiscordClient(srv.URL, srv.Client())
	err := client.Send(t.Context(), FailurePayload("x"))
	if err == nil {
		t.Fatal("Send() error = nil, want error for 4xx")
	}
	if !strings.Contains(err.Error(), "invalid webhook") {
		t.Errorf("error = %v, want response body included", err)
	}
}

func TestClient_Send_ConnectionError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // close immediately to force a connection error

	client := NewDiscordClient(url, &http.Client{Timeout: time.Second})
	if err := client.Send(t.Context(), FailurePayload("x")); err == nil {
		t.Fatal("Send() error = nil, want connection error")
	}
}

func TestHumanizeBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   uint64
		want string
	}{
		{0, "0 Bytes"},
		{1023, "1023 Bytes"},
		{1024, "1.00 KiB"},
		{1536, "1.50 KiB"},
		{1024 * 1024, "1.00 MiB"},
		{1 << 30, "1.00 GiB"},
		{1 << 40, "1.00 TiB"},
		{^uint64(0), "16.00 EiB"},
	}
	for _, tt := range tests {
		if got := humanizeBytes(tt.in); got != tt.want {
			t.Errorf("humanizeBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
