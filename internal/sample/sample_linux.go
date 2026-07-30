//go:build linux

package sample

import (
	"bufio"
	"context"
	"fmt"
	"os"
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
	pressure := readPSI(ctx)
	if pressure == PressureUnknown {
		pressure = pressureFromMetrics(vm.UsedPercent, sw.Used)
	}

	var procs []Proc
	if live {
		procs, err = collectProcsLight(ctx)
		if err != nil {
			return Snapshot{}, err
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

// readPSI maps Linux /proc/pressure/memory into DEPTHS pressure vocabulary.
// some avg10 = stall % over 10s; thresholds are conservative observation gates.
func readPSI(ctx context.Context) string {
	_ = ctx
	f, err := os.Open("/proc/pressure/memory")
	if err != nil {
		return PressureUnknown
	}
	defer f.Close()

	var someAvg10 float64
	var found bool
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "some ") {
			continue
		}
		// some avg10=0.00 avg60=0.00 avg300=0.00 total=…
		for _, field := range strings.Fields(line) {
			if strings.HasPrefix(field, "avg10=") {
				v := strings.TrimPrefix(field, "avg10=")
				if pct, err := strconv.ParseFloat(v, 64); err == nil {
					someAvg10 = pct
					found = true
				}
			}
		}
	}
	if !found {
		return PressureUnknown
	}
	switch {
	case someAvg10 >= 30:
		return PressureCritical
	case someAvg10 >= 5:
		return PressureWarn
	default:
		return PressureNormal
	}
}
