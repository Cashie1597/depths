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

// DEPTHS wordmark — compact, tool-first.
var asciiLogo = []string{
	` ____  _____ ____ _____ _   _ ____`,
	`|  _ \| ____|  _ \_   _| | | / ___|`,
	`| | | |  _| | |_) || | | |_| \___ \`,
	`| |_| | |___|  __/ | | |  _  |___) |`,
	`|____/|_____|_|    |_| |_| |_|____/`,
}

// HomeStatus is the operator summary for the starter screen.
type HomeStatus struct {
	Pressure   string // normal|warn|critical|unknown
	SwapActive bool
	UsedPct    float64
	LastScout  time.Time
	TopTarget  string
	GateHeld   bool
}

// StatusFromReport maps a scout report into home STATUS fields.
func StatusFromReport(r app.ScoutReport, lastScout time.Time) HomeStatus {
	st := HomeStatus{
		Pressure:   r.Memory.Pressure,
		SwapActive: r.Memory.SwapUsed > 0,
		UsedPct:    r.Memory.UsedPercent,
		LastScout:  lastScout,
		GateHeld:   !r.PressureOK,
	}
	if len(r.Groups) > 0 {
		st.TopTarget = r.Groups[0].Label
		if st.TopTarget == "" {
			st.TopTarget = r.Groups[0].ID
		}
	}
	if st.Pressure == "" {
		st.Pressure = sample.PressureUnknown
	}
	return st
}

// BrandBlock for help / plain identity.
func BrandBlock(color bool, width int) string {
	if width < 48 {
		width = 48
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", renderASCII(color, width))
	fmt.Fprintf(&b, "%s\n", center(tone(color, Mute, "quiet observation layer"), width))
	return b.String()
}

func renderASCII(color bool, width int) string {
	var lines []string
	for i, line := range asciiLogo {
		styled := line
		if color {
			c := NeonSwap
			if i >= 3 {
				c = NeonFree
			}
			styled = lipgloss.NewStyle().Bold(true).Foreground(c).Background(Abyss).Render(line)
		}
		lines = append(lines, center(styled, width))
	}
	return strings.Join(lines, "\n")
}

// HelpText is Mole-grammar help, DEPTHS voice.
func HelpText(version string) string {
	var b strings.Builder
	w := 56
	fmt.Fprintf(&b, "%s\n\n", BrandBlock(false, w))
	fmt.Fprintf(&b, "COMMANDS\n\n")
	cmds := [][2]string{
		{"depths", "Interactive observation console"},
		{"depths scout", "Watch memory pressure (live)"},
		{"depths scout --plain", "One-shot console text"},
		{"depths claim --dry-run", "Preview reclaim plan · no signals"},
		{"depths claim --groups=…", "Claim after confirm (SIGTERM→KILL)"},
		{"depths profiles", "List observation profiles"},
		{"depths version", "Print version"},
	}
	for _, c := range cmds {
		fmt.Fprintf(&b, "  %-28s %s\n", c[0], c[1])
	}
	fmt.Fprintf(&b, "\nPROFILES\n\n")
	fmt.Fprintf(&b, "  %-28s %s\n", "gentle", "High gate · browsers only")
	fmt.Fprintf(&b, "  %-28s %s\n", "focus", "Medium gate · browser/chat/media")
	fmt.Fprintf(&b, "  %-28s %s\n", "operator", "Lower gate · broader userland")
	fmt.Fprintf(&b, "\nOPTIONS\n\n")
	opts := [][2]string{
		{"--dry-run", "Plan only · never send signals"},
		{"--profile=NAME", "gentle | focus | operator"},
		{"--groups=ID,…", "Limit claim/scout to group ids"},
		{"--force-pressure", "Bypass pressure gate (explicit)"},
		{"--json", "Machine-readable output"},
		{"--plain", "Text console · no TUI"},
		{"--once", "Full TUI · no live poll"},
	}
	for _, o := range opts {
		fmt.Fprintf(&b, "  %-28s %s\n", o[0], o[1])
	}
	fmt.Fprintf(&b, "\nversion %s · read SAFETY.md before live claim\n", version)
	return b.String()
}

// MenuView — state first, then actions. Footer pinned.
func MenuView(color bool, width, height, cursor int, items []MenuItem, st HomeStatus) string {
	if width < 52 {
		width = 52
	}
	innerW := width - 8
	if innerW < 44 {
		innerW = 44
	}
	boxW := min(innerW, 48)

	var top strings.Builder
	fmt.Fprintf(&top, "%s\n\n", renderASCII(color, innerW))
	fmt.Fprintf(&top, "%s\n\n", center(tone(color, Mute, "quiet observation layer"), innerW))
	fmt.Fprintf(&top, "%s\n\n", statusPanel(color, boxW, st))

	for i, it := range items {
		if color {
			var row string
			if i == cursor {
				mark := lipgloss.NewStyle().Bold(true).Foreground(NeonFree).Background(Abyss).Render("›")
				n := lipgloss.NewStyle().Bold(true).Foreground(NeonSwap).Background(Abyss).Render(fmt.Sprintf("%d.", i+1))
				title := lipgloss.NewStyle().Bold(true).Foreground(Foam).Background(Abyss).Render(fmt.Sprintf(" %-10s", it.Title))
				desc := lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Render("  "+it.Desc)
				row = mark + " " + n + title + desc
			} else {
				mark := lipgloss.NewStyle().Foreground(Silt).Background(Abyss).Render(" ")
				n := lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Render(fmt.Sprintf("%d.", i+1))
				title := lipgloss.NewStyle().Foreground(Foam).Background(Abyss).Render(fmt.Sprintf(" %-10s", it.Title))
				desc := lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Render("  "+it.Desc)
				row = mark + " " + n + title + desc
			}
			fmt.Fprintf(&top, "  %s\n", row)
			if i < len(items)-1 {
				fmt.Fprintf(&top, "\n")
			}
		} else {
			marker := " "
			if i == cursor {
				marker = "›"
			}
			fmt.Fprintf(&top, "  %s %d. %-10s  %s\n", marker, i+1, it.Title, it.Desc)
			if i < len(items)-1 {
				fmt.Fprintf(&top, "\n")
			}
		}
	}

	var foot strings.Builder
	fmt.Fprintf(&foot, "%s\n", tone(color, Mute, "↑ ↓ Enter    ? Help    Q Quit"))
	prompt := "depths ›"
	if color {
		prompt = lipgloss.NewStyle().Bold(true).Foreground(NeonSwap).Background(Abyss).Render("depths ›")
	}
	fmt.Fprintf(&foot, "%s%s", prompt, tone(color, Mute, "  state before action"))

	topBlock := strings.TrimRight(top.String(), "\n")
	footBlock := strings.TrimRight(foot.String(), "\n")

	if !color {
		return "\n" + topBlock + "\n\n\n" + footBlock + "\n"
	}

	padX := 4
	topStyled := lipgloss.NewStyle().
		Foreground(Foam).
		Background(Abyss).
		Padding(1, padX).
		Width(width).
		Render(topBlock)

	footStyled := lipgloss.NewStyle().
		Foreground(Foam).
		Background(Abyss).
		Padding(0, padX, 1, padX).
		Width(width).
		Render(footBlock)

	if height <= 0 {
		return lipgloss.JoinVertical(lipgloss.Left, topStyled, "", footStyled)
	}

	used := lipgloss.Height(topStyled) + lipgloss.Height(footStyled)
	gap := height - used
	if gap < 0 {
		gap = 0
	}
	// Upper bias: status+menu read as one console, not floating in void.
	topPad := gap / 5
	botPad := gap - topPad

	spacer := func(n int) string {
		if n <= 0 {
			return ""
		}
		return lipgloss.NewStyle().Background(Abyss).Width(width).Height(n).Render("")
	}

	parts := []string{}
	if topPad > 0 {
		parts = append(parts, spacer(topPad))
	}
	parts = append(parts, topStyled)
	if botPad > 0 {
		parts = append(parts, spacer(botPad))
	}
	parts = append(parts, footStyled)

	out := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.NewStyle().
		Background(Abyss).
		Width(width).
		Height(height).
		MaxHeight(height).
		Render(out)
}

func statusPanel(color bool, width int, st HomeStatus) string {
	pressure := strings.ToUpper(st.Pressure)
	if pressure == "" {
		pressure = "UNKNOWN"
	}
	swap := "QUIET"
	if st.SwapActive {
		swap = "ACTIVE"
	}
	last := "never"
	if !st.LastScout.IsZero() {
		d := time.Since(st.LastScout).Round(time.Second)
		if d < time.Minute {
			last = fmt.Sprintf("%ds ago", int(d.Seconds()))
		} else if d < time.Hour {
			last = fmt.Sprintf("%dm ago", int(d.Minutes()))
		} else {
			last = st.LastScout.Format("15:04")
		}
	}
	gate := "CLEAR"
	if st.GateHeld {
		gate = "HELD"
	}

	rows := []struct {
		label string
		value string
		c     lipgloss.Color
	}{
		{"Memory pressure", pressure, pressureColor(st.Pressure)},
		{"Swap", swap, swapColor(st.SwapActive)},
		{"Resident", fmt.Sprintf("%.0f%%", st.UsedPct), NeonSwap},
		{"Gate", gate, gateColor(st.GateHeld)},
		{"Last scout", last, NeonSwap},
	}
	if st.TopTarget != "" {
		rows = append(rows, struct {
			label string
			value string
			c     lipgloss.Color
		}{"Top target", st.TopTarget, NeonSwap})
	}

	if !color {
		lines := []string{"STATUS", "---", ""}
		for _, r := range rows {
			lines = append(lines, fmt.Sprintf("%-18s %s", r.label, r.value))
		}
		lines = append(lines, "")
		return panel(false, width, lines)
	}

	inner := width - 2
	var b strings.Builder
	top := "┌" + strings.Repeat("─", inner) + "┐"
	bot := "└" + strings.Repeat("─", inner) + "┘"
	mid := "├" + strings.Repeat("─", inner) + "┤"
	fmt.Fprintf(&b, "%s\n", boxLine(true, top))
	fmt.Fprintf(&b, "%s\n", statusBoxTitle(inner))
	fmt.Fprintf(&b, "%s\n", boxLine(true, mid))
	fmt.Fprintf(&b, "%s\n", boxLine(true, "│"+strings.Repeat(" ", inner)+"│"))
	for _, r := range rows {
		fmt.Fprintf(&b, "%s\n", statusBoxRow(inner, r.label, r.value, r.c))
	}
	fmt.Fprintf(&b, "%s\n", boxLine(true, "│"+strings.Repeat(" ", inner)+"│"))
	fmt.Fprintf(&b, "%s", boxLine(true, bot))
	return b.String()
}

func statusBoxTitle(inner int) string {
	title := lipgloss.NewStyle().Bold(true).Foreground(Foam).Background(Abyss).Render(" STATUS")
	plain := " STATUS"
	pad := max(0, inner-utf8.RuneCountInString(plain))
	left := lipgloss.NewStyle().Foreground(Silt).Background(Abyss).Render("│")
	right := lipgloss.NewStyle().Foreground(Silt).Background(Abyss).Render("│")
	return left + title + strings.Repeat(" ", pad) + right
}

func statusBoxRow(inner int, label, value string, c lipgloss.Color) string {
	labPlain := fmt.Sprintf(" %-18s", label)
	lab := lipgloss.NewStyle().Foreground(Mute).Background(Abyss).Render(labPlain)
	val := lipgloss.NewStyle().Bold(true).Foreground(c).Background(Abyss).Render(value)
	used := utf8.RuneCountInString(labPlain) + utf8.RuneCountInString(value)
	pad := max(1, inner-used)
	left := lipgloss.NewStyle().Foreground(Silt).Background(Abyss).Render("│")
	right := lipgloss.NewStyle().Foreground(Silt).Background(Abyss).Render("│")
	return left + lab + val + strings.Repeat(" ", pad) + right
}

func pressureColor(p string) lipgloss.Color {
	switch strings.ToLower(p) {
	case sample.PressureCritical:
		return NeonUsed
	case sample.PressureWarn:
		return NeonWarn
	case sample.PressureNormal:
		return NeonFree
	default:
		return Mute
	}
}

func swapColor(active bool) lipgloss.Color {
	if active {
		return NeonWarn
	}
	return NeonFree
}

func gateColor(held bool) lipgloss.Color {
	if held {
		return NeonUsed
	}
	return NeonFree
}

type MenuItem struct {
	Title string
	Desc  string
	ID    string
}
