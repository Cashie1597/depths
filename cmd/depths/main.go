package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"

	"github.com/cashie/depths/internal/app"
	"github.com/cashie/depths/internal/claim"
	"github.com/cashie/depths/internal/profile"
	"github.com/cashie/depths/internal/receipt"
	"github.com/cashie/depths/internal/sample"
	"github.com/cashie/depths/internal/surface"
	"github.com/cashie/depths/internal/tui"
)

var version = "0.2.0"

type CLI struct {
	Menu     MenuCmd     `cmd:"" default:"1" help:"Interactive observation console (default)."`
	Scout    ScoutCmd    `cmd:"" help:"Watch memory pressure (live by default)."`
	Claim    ClaimCmd    `cmd:"" help:"Claim process groups — dry-run first."`
	Profiles ProfilesCmd `cmd:"" help:"List observation profiles."`
	Version  VersionCmd  `cmd:"" help:"Print version."`
}

type MenuCmd struct{}

func (c *MenuCmd) Run() error {
	if !isTTY() {
		fmt.Print(surface.HelpText(version))
		return nil
	}
	return tui.RunMenu()
}

type ScoutCmd struct {
	Profile       string   `name:"profile" default:"gentle" help:"Profile: gentle|focus|operator."`
	ProfileFile   string   `name:"profile-file" help:"Override profile YAML path."`
	Groups        []string `name:"groups" help:"Limit to group IDs."`
	Limit         int      `name:"limit" help:"Max groups to show (0 = profile default)."`
	ForcePressure bool     `name:"force-pressure" help:"Ignore pressure/swap gate for display."`
	JSON          bool     `name:"json" help:"Machine-readable output."`
	Plain         bool     `name:"plain" help:"Force plain text (no TUI)."`
	Once          bool     `name:"once" help:"Single snapshot TUI (no live poll)."`
}

func (c *ScoutCmd) Run() error {
	opt := app.Options{
		ProfileName:   c.Profile,
		ProfileFile:   c.ProfileFile,
		GroupIDs:      c.Groups,
		ForcePressure: c.ForcePressure,
		Limit:         c.Limit,
		Live:          !c.Plain && !c.JSON && isTTY() && !c.Once,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	report, _, _, err := app.Scout(ctx, opt)
	if err != nil {
		return err
	}
	if c.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	if c.Plain || !isTTY() {
		fmt.Print(surface.Scout(report, false))
		return nil
	}
	return tui.RunScoutLive(tui.Config{Options: opt, Live: !c.Once}, report)
}

type ClaimCmd struct {
	Profile       string   `name:"profile" default:"gentle" help:"Profile: gentle|focus|operator."`
	ProfileFile   string   `name:"profile-file" help:"Override profile YAML path."`
	Groups        []string `name:"groups" help:"Group IDs to claim (required for live unless --all-matching)."`
	AllMatching   bool     `name:"all-matching" help:"Claim all groups matching the profile (dangerous)."`
	DryRun        bool     `name:"dry-run" help:"Print plan only — no signals."`
	Yes           bool     `name:"yes" help:"Skip interactive confirm (after plan is printed)."`
	ForceKill     bool     `name:"force-kill" help:"Allow SIGKILL-first path when profile permits."`
	ForcePressure bool     `name:"force-pressure" help:"Bypass pressure/swap gate."`
	JSON          bool     `name:"json" help:"Machine-readable output."`
}

func (c *ClaimCmd) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if !c.DryRun && !c.AllMatching && len(c.Groups) == 0 {
		return fmt.Errorf("live claim requires --groups=<id>[,…] or --all-matching (prefer --dry-run first)")
	}

	report, _, guard, err := app.Scout(ctx, app.Options{
		ProfileName:   c.Profile,
		ProfileFile:   c.ProfileFile,
		GroupIDs:      c.Groups,
		ForcePressure: c.ForcePressure,
	})
	if err != nil {
		return err
	}

	if !report.PressureOK && !c.DryRun {
		return fmt.Errorf("pressure gate blocked: %s (use --force-pressure to override)", report.PressureMsg)
	}

	interactive := !c.DryRun && !c.Yes && isTTY()
	showOnly := c.DryRun || interactive

	plan, err := app.BuildClaimPlan(report, guard, showOnly, c.ForceKill)
	if err != nil {
		return err
	}

	if c.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(plan); err != nil {
			return err
		}
	} else {
		printPlan(plan, report)
	}
	if c.DryRun {
		if !c.JSON {
			fmt.Println("dry-run complete — no signals sent")
		}
		return nil
	}

	if interactive {
		fmt.Print("Proceed with SIGTERM → grace → SIGKILL? Type 'claim' to continue: ")
		var line string
		_, _ = fmt.Scanln(&line)
		if strings.TrimSpace(line) != "claim" {
			fmt.Println("aborted")
			return nil
		}
		plan.DryRun = false
	} else if !c.Yes {
		return fmt.Errorf("refusing non-interactive live claim without --yes")
	} else {
		plan.DryRun = false
	}

	before := report.Memory
	result := claim.Execute(ctx, plan)

	var after *sample.Memory
	if snap, err := sample.Collect(ctx); err == nil {
		after = &snap.Memory
	}

	path, err := receipt.Write(receipt.Receipt{
		Version: version,
		Command: "claim",
		Profile: report.Profile.Name,
		DryRun:  false,
		Before:  before,
		After:   after,
		Result:  result,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: receipt write failed: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "receipt: %s\n", path)
	}

	if c.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	printResult(result, before, after)
	return nil
}

func printPlan(plan claim.Plan, report app.ScoutReport) {
	fmt.Print(surface.BrandBlock(false, 56))
	fmt.Println()
	fmt.Printf("CLAIM PLAN · profile %s · dry_run=%v · grace %ds\n",
		plan.Profile, plan.DryRun, plan.GraceSeconds)
	fmt.Printf("pressure %s · swap %s · estimate≈%s\n\n",
		report.Memory.Pressure,
		sample.FormatBytes(report.Memory.SwapUsed),
		sample.FormatBytes(report.Estimate.EstimateFree),
	)
	for _, t := range plan.Targets {
		fmt.Printf("  · %-24s pid=%-7d rss=%-10s group=%s\n",
			t.Name, t.PID, sample.FormatBytes(t.RSS), t.GroupID)
	}
	if len(plan.Skipped) > 0 {
		fmt.Println("\nheld (protected):")
		for _, s := range plan.Skipped {
			fmt.Printf("  · %s\n", s)
		}
	}
}

func printResult(result claim.Result, before sample.Memory, after *sample.Memory) {
	fmt.Printf("claim finished  signaled=%d  errors=%d\n", len(result.Signaled), len(result.Errors))
	for _, s := range result.Signaled {
		status := "ok"
		if !s.OK {
			status = s.Error
		}
		fmt.Printf("  · %s pid=%d %s → %s\n", s.Name, s.PID, s.Signal, status)
	}
	if after != nil {
		fmt.Printf("pressure %s → %s  swap %s → %s\n",
			before.Pressure, after.Pressure,
			sample.FormatBytes(before.SwapUsed), sample.FormatBytes(after.SwapUsed),
		)
	}
}

type ProfilesCmd struct{}

func (c *ProfilesCmd) Run() error {
	fmt.Print(surface.BrandBlock(false, 56))
	fmt.Println()
	for _, p := range profile.List() {
		fmt.Printf("%-10s  min_pressure=%-8s  grace=%2ds  kinds=%v\n  %s\n\n",
			p.Name, p.MinPressure, p.GraceSeconds, p.AllowKinds, p.Description)
	}
	return nil
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Println("depths", version)
	return nil
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func main() {
	// Custom --help before kong so DEPTHS brand shows like Mole's help.
	for _, a := range os.Args[1:] {
		if a == "--help" || a == "-h" {
			fmt.Print(surface.HelpText(version))
			return
		}
	}

	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("depths"),
		kong.Description("Reclaim RAM under swap pressure. Dry-run first. Explicit risk."),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
