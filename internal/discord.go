package pgbackup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const avatarURL = "https://github.com/flestudio.png"

const (
	colorSuccess = 0x57F287 // green
	colorFailure = 0xED4245 // red
)

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
}

type Payload struct {
	Content   string  `json:"content,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

type BackupInfo struct {
	Path      string
	RemoteKey string
	Size      uint64
	CreatedAt time.Time
}

func SuccessPayload(info BackupInfo, du Usage) Payload {
	fields := []EmbedField{
		{Name: "ファイル", Value: info.Path},
		{Name: "サイズ", Value: humanizeBytes(info.Size)},
		{Name: "作成日時", Value: info.CreatedAt.Format(time.RFC3339)},
	}
	if info.RemoteKey != "" {
		fields = append(fields, EmbedField{Name: "S3 オブジェクト", Value: info.RemoteKey})
	}
	return Payload{
		Content:   "✅ バックアップを作成しました",
		AvatarURL: avatarURL,
		Embeds:    []Embed{{Title: "バックアップ情報", Color: colorSuccess, Fields: fields}, diskEmbed(du)},
	}
}

func FailurePayload(cause string) Payload {
	return Payload{
		Content:   "🔥 バックアップの作成に失敗しました",
		AvatarURL: avatarURL,
		Embeds:    []Embed{{Title: "エラー", Description: cause, Color: colorFailure}},
	}
}

func diskEmbed(du Usage) Embed {
	return Embed{
		Title: "ディスク情報",
		Fields: []EmbedField{
			{Name: "総容量", Value: humanizeBytes(du.Total), Inline: true},
			{Name: "使用中", Value: humanizeBytes(du.Used), Inline: true},
			{Name: "空き", Value: humanizeBytes(du.Available), Inline: true},
			{Name: "使用率", Value: fmt.Sprintf("%.1f%%", du.UsedPercent()), Inline: true},
		},
	}
}

type DiscordClient struct {
	webhookURL string
	httpClient *http.Client
}

func NewDiscordClient(webhookURL string, httpClient *http.Client) *DiscordClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &DiscordClient{webhookURL: webhookURL, httpClient: httpClient}
}

func (c *DiscordClient) Send(ctx context.Context, payload Payload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook returned %s: %s", resp.Status, bytes.TrimSpace(snippet))
	}
	return nil
}

var byteUnits = [...]string{"Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

func humanizeBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d Bytes", b)
	}
	value := float64(b)
	i := 0
	for value >= unit && i < len(byteUnits)-1 {
		value /= unit
		i++
	}
	return fmt.Sprintf("%.2f %s", value, byteUnits[i])
}
