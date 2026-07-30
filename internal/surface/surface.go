package surface

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/cashie/depths/internal/app"
	"github.com/cashie/depths/internal/sample"
)

// DEPTHS-AES-001 terminal palette.
const (
	HexAbyss    = "#0A0C0B"
	HexTrench   = "#121614"
	HexSilt     = "#2A332E"
	HexFoam     = "#C8D2CA"
	HexDepth    = "#5FAF8A"
	HexPressure = "#C4A35A"
	HexBreach   = "#B85C4A"
	HexMute     = "#6B756F"
)

// DEPTHS-AES-001 terminal palette + neon meter accents.
var (
	Abyss    = lipgloss.Color(HexAbyss)
	Trench   = lipgloss.Color(HexTrench)
	Silt     = lipgloss.Color(HexSilt)
	Foam     = lipgloss.Color(HexFoam)
	Depth    = lipgloss.Color(HexDepth)
	Pressure = lipgloss.Color(HexPressure)
	Breach   = lipgloss.Color(HexBreach)
	Mute     = lipgloss.Color(HexMute)

	NeonUsed = lipgloss.Color("#FF2D6A") // hot pink — blocked / risk
	NeonFree = lipgloss.Color("#39FF14") // lime — safe / confirmed
	NeonSwap = lipgloss.Color("#00F0FF") // cyan — information
	NeonWarn = lipgloss.Color("#FF9F1C") // orange — attention
	NeonDim  = lipgloss.Color("#1A2A28") // track
)

type Options struct {
	Color     bool
	Width     int
	Height    int
	FullBleed bool
	Live      bool
	UpdatedAt time.Time
	Err       string
	Loading   bool
	Pulse     string
}

func Scout(r app.ScoutReport, color bool) string {
	return ScoutSized(r, Options{Color: color, Width: 72})
}

func ScoutSized(r app.ScoutReport, opt Options) string {
	w := opt.Width
	if w < 52 {
		w = 52
	}
	padX := 3
	if opt.FullBleed {
		padX = 4
	}
	inner := w - padX*2
	if inner < 44 {
		inner = 44
	}

	body := renderConsole(r, opt, inner)

	if !opt.Color {
		return body
	}

	content := lipgloss.NewStyle().
		Foreground(Foam).
		Background(Abyss).
		Padding(1, padX).
		Width(w).
		Render(body)

	if opt.Height > 0 {
		h := lipgloss.Height(content)
		if h < opt.Height {
			filler := lipgloss.NewStyle().
				Background(Abyss).
				Width(w).
				Height(opt.Height - h).
				Render("")
			content = lipgloss.JoinVertical(lipgloss.Left, content, filler)
		}
		content = lipgloss.NewStyle().
			Background(Abyss).
			Width(w).
			Height(opt.Height).
			MaxHeight(opt.Height).
			Render(content)
	}
	return content
}

func renderConsole(r app.ScoutReport, opt Options, width int) string {
	color := opt.Color
	var b strings.Builder

	// Signature — quiet identity, not ASCII mascot.
	fmt.Fprintf(&b, "%s\n", center(tone(color, Mute, "◦"), width))
	fmt.Fprintf(&b, "%s\n", center(tone(color, Foam, "DEPTHS SCOUT"), width))
	fmt.Fprintf(&b, "%s\n\n", center(tone(color, Mute, "quiet observation layer"), width))

	mode := "OBSERVING"
	if opt.Live {
		mode = "WATCHING"
	}
	ts := ""
	if opt.Live {
		t := opt.UpdatedAt
		if t.IsZero() {
			t = time.Now()
		}
		pulse := opt.Pulse
		if pulse == "" {
			pulse = "●"
		}
		pulseOut := pulse
		if color {
			pc := NeonFree
			if pulse == "○" {
				pc = NeonSwap
			}
			pulseOut = lipgloss.NewStyle().Bold(true).Foreground(pc).Background(Abyss).Render(pulse)
		}
		samp := ""
		if opt.Loading {
			if color {
				samp = lipgloss.NewStyle().Foreground(NeonSwap).Background(Abyss).Render(" · sampling")
			} else {
				samp = " · sampling"
			}
		}
		ts = fmt.Sprintf(" · %s %s%s", pulseOut, t.Format("15:04:05"), samp)
	}

	boxW := min(width, 56)
	if width >= 72 {
		boxW = min(width, 64)
	}
	barW := boxW - 28
	if barW < 12 {
		barW = 12
	}

	usedPct := r.Memory.UsedPercent
	freePct := 0.0
	if r.Memory.Total > 0 {
		freePct = float64(r.Memory.Available) / float64(r.Memory.Total) * 100
	}
	swapPct := 0.0
	if r.Memory.SwapTotal > 0 {
		swapPct = float64(r.Memory.SwapUsed) / float64(r.Memory.SwapTotal) * 100
	} else if r.Memory.SwapUsed > 0 {
		swapPct = 100
	}

	if color {
		// Keep ANSI (neon pulse) out of the box panel — panel padding mangles escapes.
		watchLine := fmt.Sprintf("system observation · %s", strings.ToLower(mode))
		if opt.Live {
			t := opt.UpdatedAt
			if t.IsZero() {
				t = time.Now()
			}
			pulse := opt.Pulse
			if pulse == "" {
				pulse = "●"
			}
			pc := NeonFree
			if pulse == "○" {
				pc = NeonSwap
			}
			pulseOut := lipgloss.NewStyle().Bold(true).Foreground(pc).Background(Abyss).Render(pulse)
			samp := ""
			if opt.Loading {
				samp = lipgloss.NewStyle().Foreground(NeonSwap).Background(Abyss).Render(" sampling")
			}
			watchLine = lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Render("system observation · "+strings.ToLower(mode)+" · ") +
				pulseOut +
				lipgloss.NewStyle().Foreground(Foam).Background(Abyss).Render(" "+t.Format("15:04:05")) +
				samp
		}
		fmt.Fprintf(&b, "%s\n", panel(true, boxW, []string{
			"DEPTHS / SCOUT",
			"---",
			"MEMORY",
		}))
		fmt.Fprintf(&b, "%s\n\n", watchLine)
		fmt.Fprintf(&b, "%s\n", neonMeterBlock(
			usedPct, freePct, swapPct, barW,
			sample.FormatBytes(r.Memory.Used),
			sample.FormatBytes(r.Memory.Available),
			sample.FormatBytes(r.Memory.SwapUsed),
			sample.FormatBytes(r.Memory.Total),
		))
		fmt.Fprintf(&b, "\n%s\n", tone(true, Mute, fmt.Sprintf("pressure  %s", r.Memory.Pressure)))
		fmt.Fprintf(&b, "%s\n", neonGate(r))
	} else {
		memLines := []string{
			"DEPTHS / SCOUT",
			fmt.Sprintf("system observation · %s%s", strings.ToLower(mode), ts),
			"---",
			"MEMORY",
			"",
			meterRow("Used", usedPct, barW, sample.FormatBytes(r.Memory.Used)),
			meterRow("Free", freePct, barW, sample.FormatBytes(r.Memory.Available)),
			meterRow("Swap", swapPct, barW, sample.FormatBytes(r.Memory.SwapUsed)),
			"",
			fmt.Sprintf("%-5s %s", "total", sample.FormatBytes(r.Memory.Total)),
			"",
			fmt.Sprintf("pressure  %s", r.Memory.Pressure),
			"",
			gateVoice(r),
		}
		fmt.Fprintf(&b, "%s\n", panel(false, boxW, memLines))
	}

	if opt.Err != "" {
		fmt.Fprintf(&b, "\n%s\n", tone(color, Breach, "signal fault · "+opt.Err))
	}

	fmt.Fprintf(&b, "\n\n%s\n\n", section(color, "INTERPRETATION"))
	fmt.Fprintf(&b, "%s\n", tone(color, Foam, interpretMemory(r)))
	fmt.Fprintf(&b, "%s\n", tone(color, Mute, interpretGate(r)))

	fmt.Fprintf(&b, "\n\n%s\n\n", section(color, "SCOUT TARGET"))
	if len(r.Groups) == 0 {
		fmt.Fprintf(&b, "%s\n", tone(color, Mute, "No reclaim candidate under this profile."))
		fmt.Fprintf(&b, "%s\n", tone(color, Mute, "Continue watching."))
	} else {
		for i, g := range r.Groups {
			if i > 0 {
				fmt.Fprintf(&b, "\n\n")
			}
			est := estimateForGroup(r, i)
			fmt.Fprintf(&b, "%s\n", tone(color, Foam, g.Label))
			fmt.Fprintf(&b, "%s\n\n", tone(color, Mute, "id  "+g.ID+" · "+string(g.Kind)))
			fmt.Fprintf(&b, "%s\n\n", kv(color, "processes", fmt.Sprintf("%d", len(g.Procs))))
			fmt.Fprintf(&b, "%s\n", kv(color, "resident", sample.FormatBytes(g.RSS)))
			rel := 0.0
			if r.Estimate.GrossRSS > 0 {
				rel = float64(g.RSS) / float64(r.Estimate.GrossRSS) * 100
			}
			resBar := strikeBar(rel, 5)
			if color {
				resBar = lipgloss.NewStyle().Bold(true).Foreground(NeonSwap).Background(Abyss).Render(resBar)
			}
			fmt.Fprintf(&b, "\n%s %s\n\n", tone(color, Mute, "footprint"), resBar)
			fmt.Fprintf(&b, "%s\n\n", kv(color, "reclaim estimate", sample.FormatBytes(est)))
			fmt.Fprintf(&b, "%s\n", kv(color, "confidence", r.Estimate.Confidence))
		}
		fmt.Fprintf(&b, "\n%s\n", tone(color, Mute, r.Estimate.Note))
	}

	nextGroup := "<id>"
	if len(r.Groups) > 0 {
		nextGroup = r.Groups[0].ID
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", section(color, "NEXT"))
	fmt.Fprintf(&b, "%s\n", tone(color, Mute, "dry-run path · no signals until confirm"))
	fmt.Fprintf(&b, "%s\n", tone(color, Pressure, fmt.Sprintf(
		"depths claim --dry-run --profile=%s --groups=%s",
		r.Profile.Name, nextGroup,
	)))

	fmt.Fprintf(&b, "\n%s\n", rule(width, color))
	state := mode
	if !r.PressureOK {
		state = "HELD"
	} else if len(r.Groups) > 0 {
		state = "READY"
	} else {
		state = "CLEAR"
	}
	hint := "q quit · r refresh"
	if opt.Live {
		if color {
			live := lipgloss.NewStyle().Bold(true).Foreground(NeonFree).Background(Abyss).Render("live")
			hint = live + tone(color, Mute, " · q quit · r refresh")
		} else {
			hint = "live · q quit · r refresh"
		}
	}
	if color {
		fmt.Fprintf(&b, "%s%s\n",
			lipgloss.NewStyle().Bold(true).Foreground(NeonSwap).Background(Abyss).Render("depths ›"),
			lipgloss.NewStyle().Foreground(Foam).Background(Abyss).Render(
				fmt.Sprintf(" %s · scout · %s ", strings.ToLower(state), r.Profile.Name),
			)+hint,
		)
	} else {
		fmt.Fprintf(&b, "depths › %s · scout · %s  ·  %s\n",
			strings.ToLower(state), r.Profile.Name, hint)
	}
	return b.String()
}

func gateVoice(r app.ScoutReport) string {
	if r.PressureOK {
		return "gate      READY"
	}
	return "gate      HELD\nreason    " + softenReason(r.PressureMsg)
}

func softenReason(msg string) string {
	msg = strings.TrimSpace(msg)
	msg = strings.ReplaceAll(msg, "pressure normal below profile min warn", "pressure below profile warn")
	msg = strings.ReplaceAll(msg, "pressure unknown below profile min warn", "pressure reading unclear · held")
	return msg
}

func interpretMemory(r app.ScoutReport) string {
	switch r.Memory.Pressure {
	case sample.PressureCritical:
		return "Memory pressure critical. Intervention likely warranted."
	case sample.PressureWarn:
		return "Memory pressure elevated. Scout targets available for review."
	case sample.PressureNormal:
		if r.Memory.SwapUsed > 1<<30 {
			return "Pressure nominal, but swap is active. Watching."
		}
		return "Memory pressure stable. No intervention required."
	default:
		return "Pressure signal unclear. Holding aggressive paths."
	}
}

func interpretGate(r app.ScoutReport) string {
	if r.PressureOK {
		return "Gate CLEAR for profile " + r.Profile.Name + "."
	}
	return "Gate HELD · " + softenReason(r.PressureMsg) + "."
}

func estimateForGroup(r app.ScoutReport, idx int) uint64 {
	if len(r.Groups) == 0 || r.Estimate.GrossRSS == 0 {
		return 0
	}
	g := r.Groups[idx]
	// Proportional share of overall estimate.
	return uint64(float64(r.Estimate.EstimateFree) * (float64(g.RSS) / float64(r.Estimate.GrossRSS)))
}

func panel(color bool, width int, lines []string) string {
	if width < 30 {
		width = 30
	}
	inner := width - 2
	top := "┌" + strings.Repeat("─", inner) + "┐"
	bot := "└" + strings.Repeat("─", inner) + "┘"
	mid := "├" + strings.Repeat("─", inner) + "┤"

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", boxLine(color, top))
	for i, line := range lines {
		switch {
		case line == "---":
			fmt.Fprintf(&b, "%s\n", boxLine(color, mid))
		case line == "":
			fmt.Fprintf(&b, "%s\n", boxLine(color, "│"+strings.Repeat(" ", inner)+"│"))
		default:
			for _, part := range strings.Split(line, "\n") {
				label := part
				if i == 0 || part == "MEMORY" {
					// section titles stay as-is
				}
				vis := " " + label
				if utf8.RuneCountInString(vis) > inner {
					vis = truncate(vis, inner)
				}
				padded := vis + strings.Repeat(" ", max(0, inner-utf8.RuneCountInString(vis)))
				content := "│" + padded + "│"
				if color && (i == 0 || part == "MEMORY") {
					content = lipgloss.NewStyle().Foreground(Foam).Background(Abyss).Render(content)
					fmt.Fprintf(&b, "%s\n", content)
				} else {
					fmt.Fprintf(&b, "%s\n", boxLine(color, content))
				}
			}
		}
	}
	fmt.Fprintf(&b, "%s", boxLine(color, bot))
	return b.String()
}

func boxLine(color bool, s string) string {
	if color {
		return lipgloss.NewStyle().Foreground(Silt).Background(Abyss).Render(s)
	}
	return s
}

func meterRow(label string, pct float64, barW int, value string) string {
	return fmt.Sprintf("%-5s %s %5.1f%%  %s", label, blockBar(pct, barW), pct, value)
}

func neonMeterBlock(used, free, swap float64, barW int, usedV, freeV, swapV, totalV string) string {
	bars := strings.Join([]string{
		neonMeterRow("Used", used, NeonUsed, usedV, barW),
		neonMeterRow("Free", free, NeonFree, freeV, barW),
		neonMeterRow("Swap", swap, NeonSwap, swapV, barW),
	}, "\n")
	// Left-aligned with labels — never indented under the bar track.
	total := lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Render(
		fmt.Sprintf("%-5s %s", "total", totalV),
	)
	return bars + "\n\n" + total
}

func neonMeterRow(label string, pct float64, fill lipgloss.Color, value string, barW int) string {
	filled, empty := neonBarParts(pct, barW)
	lab := lipgloss.NewStyle().Foreground(Foam).Background(Abyss).Bold(true).Width(5).Render(label)
	bar := lipgloss.NewStyle().Foreground(fill).Background(Abyss).Bold(true).Render(filled) +
		lipgloss.NewStyle().Foreground(NeonDim).Background(Abyss).Render(empty)
	pctS := lipgloss.NewStyle().Foreground(fill).Background(Abyss).Bold(true).Render(fmt.Sprintf(" %5.1f%%", pct))
	val := lipgloss.NewStyle().Foreground(Foam).Background(Abyss).Render("  " + value)
	return lab + " " + bar + pctS + val
}

func neonBarParts(pct float64, width int) (filled, empty string) {
	if width < 8 {
		width = 8
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	n := int((pct/100)*float64(width) + 0.5)
	if n > width {
		n = width
	}
	return strings.Repeat("█", n), strings.Repeat("░", width-n)
}

func neonGate(r app.ScoutReport) string {
	if r.PressureOK {
		return lipgloss.NewStyle().Foreground(NeonFree).Background(Abyss).Bold(true).Render("gate  READY")
	}
	return lipgloss.NewStyle().Foreground(NeonUsed).Background(Abyss).Bold(true).Render("gate  HELD") +
		lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Render(" · "+softenReason(r.PressureMsg))
}

func coloredMeters(used, free, swap float64, barW int) string {
	return neonMeterBlock(used, free, swap, barW, "", "", "", "")
}

func blockBar(pct float64, width int) string {
	if width < 8 {
		width = 8
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	n := int((pct/100)*float64(width) + 0.5)
	if n > width {
		n = width
	}
	// Mole status progressBar glyphs.
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

// strikeBar is the short Mole process meter (▮▯).
func strikeBar(pct float64, width int) string {
	if width < 3 {
		width = 5
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	n := int((pct/100)*float64(width) + 0.5)
	if n > width {
		n = width
	}
	return strings.Repeat("▮", n) + strings.Repeat("▯", width-n)
}

func section(color bool, title string) string {
	// Slight tracking via spaced caps for section labels.
	spaced := strings.Join(strings.Split(title, ""), " ")
	if color {
		return lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Bold(true).Render(spaced)
	}
	return spaced
}

func kv(color bool, key, val string) string {
	if color {
		return tone(color, Mute, key) + "\n" + tone(color, Foam, val)
	}
	return key + "\n" + val
}

func rule(width int, color bool) string {
	if width < 20 {
		width = 40
	}
	line := strings.Repeat("─", width)
	if color {
		return lipgloss.NewStyle().Foreground(Silt).Background(Abyss).Render(line)
	}
	return line
}

func tone(color bool, c lipgloss.Color, s string) string {
	if !color {
		return s
	}
	return lipgloss.NewStyle().Foreground(c).Background(Abyss).Render(s)
}

func center(s string, width int) string {
	plain := stripANSI(s)
	n := utf8.RuneCountInString(plain)
	if n >= width {
		return s
	}
	pad := (width - n) / 2
	return strings.Repeat(" ", pad) + s
}

func stripANSI(s string) string {
	// Lipgloss styles embed ANSI; for centering signature we pass unstyled or approximate.
	var b strings.Builder
	in := false
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			in = true
			continue
		}
		if in {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				in = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	if n <= 1 {
		return string(runes[:n])
	}
	return string(runes[:n-1]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
