package pgbackup

import "golang.org/x/sys/unix"

type Usage struct {
	Total     uint64
	Used      uint64
	Available uint64 // bytes a non-root user can write
}

func (u Usage) UsedPercent() float64 {
	if u.Total == 0 {
		return 0
	}
	return float64(u.Used) / float64(u.Total) * 100
}

// Bsize is int64 on Linux but uint32 on Darwin, so cast everything to uint64.
func getDiskUsage(path string) (Usage, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return Usage{}, err
	}

	bsize := uint64(st.Bsize)
	total := st.Blocks * bsize
	free := st.Bfree * bsize

	return Usage{
		Total:     total,
		Used:      total - free,
		Available: st.Bavail * bsize,
	}, nil
}
