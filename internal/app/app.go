package app

import (
	"context"
	"fmt"

	"github.com/cashie/depths/internal/claim"
	"github.com/cashie/depths/internal/estimate"
	"github.com/cashie/depths/internal/group"
	"github.com/cashie/depths/internal/profile"
	"github.com/cashie/depths/internal/protect"
	"github.com/cashie/depths/internal/sample"
)

type ScoutReport struct {
	Memory     sample.Memory  `json:"memory"`
	Profile    profile.Profile `json:"profile"`
	Groups     []group.Group  `json:"groups"`
	Estimate   estimate.Result `json:"estimate"`
	DeniedTop  []Denied       `json:"denied_sample"`
	PressureOK bool           `json:"pressure_ok"`
	PressureMsg string        `json:"pressure_msg,omitempty"`
	TakenAt    string         `json:"taken_at"`
}

type Denied struct {
	PID    int32  `json:"pid"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type Options struct {
	ProfileName   string
	ProfileFile   string
	GroupIDs      []string
	ForcePressure bool
	Limit         int
	Live          bool // fast sampler for TUI polls
}

func Scout(ctx context.Context, opt Options) (ScoutReport, []sample.Proc, *protect.Guard, error) {
	var snap sample.Snapshot
	var err error
	if opt.Live {
		snap, err = sample.CollectLive(ctx)
	} else {
		snap, err = sample.Collect(ctx)
	}
	if err != nil {
		return ScoutReport{}, nil, nil, err
	}
	prof, err := profile.Load(opt.ProfileName, opt.ProfileFile)
	if err != nil {
		return ScoutReport{}, nil, nil, err
	}

	guard := protect.NewGuard(snap.Procs)
	claimable, denied := guard.FilterClaimable(snap.Procs)
	groups := group.Build(claimable)
	groups = group.FilterByKinds(groups, prof.Kinds())
	if len(opt.GroupIDs) > 0 {
		groups = group.FilterByIDs(groups, opt.GroupIDs)
	}
	max := prof.MaxGroups
	if opt.Limit > 0 {
		max = opt.Limit
	}
	if max > 0 && len(groups) > max {
		groups = groups[:max]
	}

	ok, msg := pressureAllows(snap.Memory, prof, opt.ForcePressure)
	est := estimate.ForGroups(groups)

	deniedSample := make([]Denied, 0, 8)
	for i, p := range denied {
		if i >= 8 {
			break
		}
		d := guard.Check(p)
		deniedSample = append(deniedSample, Denied{PID: p.PID, Name: p.Name, Reason: d.Reason})
	}

	return ScoutReport{
		Memory:      snap.Memory,
		Profile:     prof,
		Groups:      groups,
		Estimate:    est,
		DeniedTop:   deniedSample,
		PressureOK:  ok,
		PressureMsg: msg,
		TakenAt:     snap.TakenAt.Format(timeRFC3339),
	}, claimable, guard, nil
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func pressureAllows(m sample.Memory, p profile.Profile, force bool) (bool, string) {
	if force {
		return true, "forced (--force-pressure)"
	}
	need := sample.ParsePressureRank(p.MinPressure)
	have := sample.ParsePressureRank(m.Pressure)
	if have < need {
		return false, fmt.Sprintf("pressure %s below profile min %s", m.Pressure, p.MinPressure)
	}
	if m.SwapUsed < p.MinSwapUsedBytes {
		return false, fmt.Sprintf("swap used %s below profile min %s",
			sample.FormatBytes(m.SwapUsed), sample.FormatBytes(p.MinSwapUsedBytes))
	}
	return true, ""
}

func BuildClaimPlan(report ScoutReport, guard *protect.Guard, dryRun, forceKill bool) (claim.Plan, error) {
	if forceKill && !report.Profile.AllowForceKill {
		return claim.Plan{}, fmt.Errorf("profile %q does not allow --force-kill", report.Profile.Name)
	}
	return claim.BuildPlan(report.Groups, guard, report.Profile.Name, report.Profile.GraceSeconds, forceKill, dryRun), nil
}
