//go:build darwin

package sample

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

func collect(ctx context.Context, live bool) (Snapshot, error) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("virtual memory: %w", err)
	}
	sw, err := mem.SwapMemoryWithContext(ctx)
	if err != nil {
		sw = &mem.SwapMemoryStat{}
	}
	pressure := readMemoryPressure(ctx)
	if pressure == PressureUnknown {
		pressure = pressureFromMetrics(vm.UsedPercent, sw.Used)
	}

	var procs []Proc
	if live {
		procs, err = collectProcsPS(ctx)
		if err != nil {
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
	// pid ppid %mem rss(kb) command — one shot, live-friendly (BSD ps).
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
			Cmdline:    name,
			RSS:        rssKB * 1024,
			MemPercent: memPct,
		})
	}
	return procs, nil
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
