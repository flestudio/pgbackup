package pgbackup

import (
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"
)

const progressInterval = time.Minute

func (c *Config) Logger(w io.Writer) *slog.Logger {
	var level slog.Level
	_ = level.UnmarshalText([]byte(c.LogLevel))
	opts := &slog.HandlerOptions{Level: level}
	if c.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

var byteUnits = [...]string{"Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}

func humanizeBytes(b uint64) string {
	if b < 1024 {
		return fmt.Sprintf("%d Bytes", b)
	}
	exp := min(int(math.Log2(float64(b)))/10, len(byteUnits)-1)
	return fmt.Sprintf("%.2f %s", float64(b)/math.Exp2(float64(exp*10)), byteUnits[exp])
}

func transferSpeed(n int64, d time.Duration) string {
	if secs := d.Seconds(); secs > 0 {
		return humanizeBytes(uint64(float64(n)/secs)) + "/s"
	}
	return humanizeBytes(uint64(n)) + "/s"
}
