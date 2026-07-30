package claim

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/cashie/depths/internal/group"
	"github.com/cashie/depths/internal/protect"
	"github.com/shirou/gopsutil/v4/process"
)

type Target struct {
	PID        int32  `json:"pid"`
	Name       string `json:"name"`
	CreateTime int64  `json:"create_time"`
	RSS        uint64 `json:"rss"`
	GroupID    string `json:"group_id"`
}

type Plan struct {
	DryRun       bool     `json:"dry_run"`
	Profile      string   `json:"profile"`
	GraceSeconds int      `json:"grace_seconds"`
	ForceKill    bool     `json:"force_kill"`
	Targets      []Target `json:"targets"`
	Skipped      []string `json:"skipped,omitempty"`
}

type Result struct {
	Plan      Plan           `json:"plan"`
	Signaled  []SignalRecord `json:"signaled"`
	Errors    []string       `json:"errors,omitempty"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   time.Time      `json:"ended_at"`
}

type SignalRecord struct {
	PID    int32  `json:"pid"`
	Name   string `json:"name"`
	Signal string `json:"signal"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

func BuildPlan(groups []group.Group, guard *protect.Guard, profileName string, grace int, forceKill, dryRun bool) Plan {
	plan := Plan{
		DryRun:       dryRun,
		Profile:      profileName,
		GraceSeconds: grace,
		ForceKill:    forceKill,
	}
	for _, g := range groups {
		for _, p := range g.Procs {
			if d := guard.Check(p); d.Protected {
				plan.Skipped = append(plan.Skipped, fmt.Sprintf("%s(%d): %s", p.Name, p.PID, d.Reason))
				continue
			}
			plan.Targets = append(plan.Targets, Target{
				PID:        p.PID,
				Name:       p.Name,
				CreateTime: p.CreateTime,
				RSS:        p.RSS,
				GroupID:    g.ID,
			})
		}
	}
	return plan
}

// Execute runs SIGTERM → grace → SIGKILL. Never runs when plan.DryRun.
func Execute(ctx context.Context, plan Plan) Result {
	res := Result{Plan: plan, StartedAt: time.Now()}
	if plan.DryRun {
		res.EndedAt = time.Now()
		return res
	}
	if len(plan.Targets) == 0 {
		res.EndedAt = time.Now()
		return res
	}

	for _, t := range plan.Targets {
		rec := signalOne(t, syscall.SIGTERM, "SIGTERM")
		res.Signaled = append(res.Signaled, rec)
		if !rec.OK {
			res.Errors = append(res.Errors, rec.Error)
		}
	}

	grace := time.Duration(plan.GraceSeconds) * time.Second
	if grace <= 0 {
		grace = 5 * time.Second
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		res.Errors = append(res.Errors, ctx.Err().Error())
		res.EndedAt = time.Now()
		return res
	case <-timer.C:
	}

	for _, t := range plan.Targets {
		if stillMatches(t) {
			rec := signalOne(t, syscall.SIGKILL, "SIGKILL")
			res.Signaled = append(res.Signaled, rec)
			if !rec.OK {
				res.Errors = append(res.Errors, rec.Error)
			}
		}
	}
	res.EndedAt = time.Now()
	return res
}

func signalOne(t Target, sig syscall.Signal, label string) SignalRecord {
	rec := SignalRecord{PID: t.PID, Name: t.Name, Signal: label}
	if !stillMatches(t) {
		rec.OK = false
		rec.Error = "pid identity mismatch or already exited — refused"
		return rec
	}
	proc, err := os.FindProcess(int(t.PID))
	if err != nil {
		rec.Error = err.Error()
		return rec
	}
	if err := proc.Signal(sig); err != nil {
		rec.Error = err.Error()
		return rec
	}
	rec.OK = true
	return rec
}

func stillMatches(t Target) bool {
	proc, err := os.FindProcess(int(t.PID))
	if err != nil {
		return false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	if t.CreateTime <= 0 {
		return true
	}
	p, err := process.NewProcess(t.PID)
	if err != nil {
		return false
	}
	ct, err := p.CreateTime()
	if err != nil {
		return false
	}
	return ct == t.CreateTime
}
