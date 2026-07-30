package sample

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// Pressure vocabulary is stable across adapters (Darwin memory_pressure, Linux PSI).
const (
	PressureNormal   = "normal"
	PressureWarn     = "warn"
	PressureCritical = "critical"
	PressureUnknown  = "unknown"
)

type Memory struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	Pressure    string  `json:"pressure"`
}

type Proc struct {
	PID        int32   `json:"pid"`
	PPID       int32   `json:"ppid"`
	Name       string  `json:"name"`
	Cmdline    string  `json:"cmdline"`
	RSS        uint64  `json:"rss"`
	MemPercent float64 `json:"mem_percent"`
	CPUPercent float64 `json:"cpu_percent"`
	CreateTime int64   `json:"create_time"` // unix ms
	Username   string  `json:"username,omitempty"`
}

type Snapshot struct {
	TakenAt time.Time `json:"taken_at"`
	Memory  Memory    `json:"memory"`
	Procs   []Proc    `json:"processes"`
}

// Collect is the thorough path (claim identity). Prefer CollectLive for TUI polls.
func Collect(ctx context.Context) (Snapshot, error) {
	return collect(ctx, false)
}

// CollectLive is fast enough for 1s polls.
func CollectLive(ctx context.Context) (Snapshot, error) {
	return collect(ctx, true)
}

// collect is implemented per-OS in sample_*.go build-tagged files.

func collectProcsLight(ctx context.Context) ([]Proc, error) {
	list, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}
	procs := make([]Proc, 0, len(list))
	for _, p := range list {
		select {
		case <-ctx.Done():
			return procs, ctx.Err()
		default:
		}
		name, err := p.NameWithContext(ctx)
		if err != nil || name == "" {
			continue
		}
		var rss uint64
		if mi, err := p.MemoryInfoWithContext(ctx); err == nil && mi != nil {
			rss = mi.RSS
		}
		ppid, _ := p.PpidWithContext(ctx)
		create, _ := p.CreateTimeWithContext(ctx)
		procs = append(procs, Proc{
			PID:        p.Pid,
			PPID:       ppid,
			Name:       name,
			Cmdline:    name,
			RSS:        rss,
			CreateTime: create,
		})
	}
	return procs, nil
}

func collectProcsFull(ctx context.Context) ([]Proc, error) {
	list, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("processes: %w", err)
	}
	procs := make([]Proc, 0, len(list))
	for _, p := range list {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		pr, ok := readProcFull(ctx, p)
		if !ok {
			continue
		}
		procs = append(procs, pr)
	}
	return procs, nil
}

func readProcFull(ctx context.Context, p *process.Process) (Proc, bool) {
	name, err := p.NameWithContext(ctx)
	if err != nil || name == "" {
		return Proc{}, false
	}
	rss := uint64(0)
	if mi, err := p.MemoryInfoWithContext(ctx); err == nil && mi != nil {
		rss = mi.RSS
	}
	memPct, _ := p.MemoryPercentWithContext(ctx)
	ppid, _ := p.PpidWithContext(ctx)
	create, _ := p.CreateTimeWithContext(ctx)
	cmd, _ := p.CmdlineWithContext(ctx)
	user, _ := p.UsernameWithContext(ctx)

	return Proc{
		PID:        p.Pid,
		PPID:       ppid,
		Name:       name,
		Cmdline:    cmd,
		RSS:        rss,
		MemPercent: float64(memPct),
		CreateTime: create,
		Username:   user,
	}, true
}

// pressureFromMetrics is a portable fallback when OS pressure APIs are missing.
func pressureFromMetrics(usedPercent float64, swapUsed uint64) string {
	switch {
	case usedPercent >= 92 || swapUsed > 512*1024*1024:
		return PressureCritical
	case usedPercent >= 80 || swapUsed > 0:
		return PressureWarn
	default:
		return PressureNormal
	}
}

func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func ParsePressureRank(p string) int {
	switch p {
	case PressureCritical:
		return 3
	case PressureWarn:
		return 2
	case PressureNormal:
		return 1
	default:
		return 0
	}
}
