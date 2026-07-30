package sample

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// Pressure levels mirror macOS memory_pressure vocabulary where possible.
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

// CollectLive is fast enough for 1s polls — ps snapshot, no per-PID CPU%.
func CollectLive(ctx context.Context) (Snapshot, error) {
	return collect(ctx, true)
}

func collect(ctx context.Context, live bool) (Snapshot, error) {
	if runtime.GOOS != "darwin" {
		return Snapshot{}, fmt.Errorf("depths: only macOS is supported (got %s)", runtime.GOOS)
	}

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("virtual memory: %w", err)
	}
	sw, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		sw = &mem.SwapMemoryStat{}
	}
	pressure := readMemoryPressure(ctx)

	var procs []Proc
	if live {
		procs, err = collectProcsPS(ctx)
		if err != nil {
			// Fallback: light gopsutil pass without CPU%.
			procs, err = collectProcsLight(ctx)
			if err != nil {
				return Snapshot{}, err
			}
		}
	} else {
		procs, err = collectProcsFull(ctx)
		if err != nil {
			return Snapshot{}, err
		}
	}

	return Snapshot{
		TakenAt: time.Now(),
		Memory: Memory{
			Total:       vm.Total,
			Used:        vm.Used,
			Available:   vm.Available,
			UsedPercent: vm.UsedPercent,
			SwapTotal:   sw.Total,
			SwapUsed:    sw.Used,
			Pressure:    pressure,
		},
		Procs: procs,
	}, nil
}

func collectProcsPS(ctx context.Context) ([]Proc, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	// pid ppid %mem rss(kb) command — one shot, live-friendly.
	out, err := exec.CommandContext(ctx, "ps", "-Aceo", "pid=,ppid=,pmem=,rss=,comm=").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	procs := make([]Proc, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid64, err := strconv.ParseInt(fields[0], 10, 32)
		if err != nil || pid64 <= 0 {
			continue
		}
		ppid64, _ := strconv.ParseInt(fields[1], 10, 32)
		memPct, _ := strconv.ParseFloat(fields[2], 64)
		rssKB, _ := strconv.ParseUint(fields[3], 10, 64)
		name := strings.Join(fields[4:], " ")
		procs = append(procs, Proc{
			PID:        int32(pid64),
			PPID:       int32(ppid64),
			Name:       name,
			Cmdline:    name, // enough for group matchers; full argv is slow
			RSS:        rssKB * 1024,
			MemPercent: memPct,
		})
	}
	return procs, nil
}

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

func readMemoryPressure(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	out, err := exec.CommandContext(ctx, "memory_pressure", "-Q").CombinedOutput()
	if err != nil {
		out2, err2 := exec.CommandContext(ctx, "memory_pressure").CombinedOutput()
		if err2 != nil {
			return PressureUnknown
		}
		out = out2
	}
	s := strings.ToLower(string(out))
	switch {
	case strings.Contains(s, "critical"):
		return PressureCritical
	case strings.Contains(s, "warn"):
		return PressureWarn
	case strings.Contains(s, "normal"):
		return PressureNormal
	}

	if i := strings.Index(s, "free percentage:"); i >= 0 {
		rest := strings.TrimSpace(s[i+len("free percentage:"):])
		rest = strings.TrimSuffix(strings.Fields(rest)[0], "%")
		if pct, err := strconv.ParseFloat(rest, 64); err == nil {
			switch {
			case pct < 15:
				return PressureCritical
			case pct < 35:
				return PressureWarn
			default:
				return PressureNormal
			}
		}
	}
	return PressureUnknown
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
