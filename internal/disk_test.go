package pgbackup

import (
	"testing"
)

func TestUsedPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		u    Usage
		want float64
	}{
		{"zero total", Usage{Total: 0, Used: 0}, 0},
		{"half", Usage{Total: 100, Used: 50}, 50},
		{"full", Usage{Total: 100, Used: 100}, 100},
		{"empty", Usage{Total: 100, Used: 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.u.UsedPercent(); got != tt.want {
				t.Errorf("UsedPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	u, err := getDiskUsage(t.TempDir())
	if err != nil {
		t.Fatalf("getDiskUsage() error = %v", err)
	}

	if u.Total == 0 {
		t.Error("Total = 0, want > 0")
	}
	if u.Used > u.Total {
		t.Errorf("Used (%d) > Total (%d)", u.Used, u.Total)
	}
	if u.Available > u.Total {
		t.Errorf("Available (%d) > Total (%d)", u.Available, u.Total)
	}
	if p := u.UsedPercent(); p < 0 || p > 100 {
		t.Errorf("UsedPercent() = %v, out of range [0,100]", p)
	}
}

func TestGet_NonexistentPath(t *testing.T) {
	t.Parallel()

	if _, err := getDiskUsage("/nonexistent/path/that/should/not/exist/12345"); err == nil {
		t.Error("getDiskUsage() error = nil, want error for nonexistent path")
	}
}
