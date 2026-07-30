package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cashie/depths/internal/app"
	"github.com/cashie/depths/internal/surface"
)

const (
	clockEvery     = 250 * time.Millisecond
	pollEvery      = 1 * time.Second
	menuStatusEvery = 3 * time.Second
)

type MenuAction string

const (
	ActionQuit   MenuAction = "quit"
	ActionScout  MenuAction = "scout"
	ActionDryRun MenuAction = "dry-run"
)

// screen modes inside one Bubble Tea session (menu → scout without re-init).
const (
	screenMenu    = "menu"
	screenOverlay = "overlay"
	screenScout   = "scout"
)

type Config struct {
	Options app.Options
	Live    bool
}

type model struct {
	screen string

	// menu
	cursor  int
	items   []surface.MenuItem
	overlay string

	// scout
	cfg         Config
	report      app.ScoutReport
	err         string
	updatedAt   time.Time
	clock       time.Time
	loading     bool
	frames      int
	pollGen     int
	liveOn      bool
	directScout bool // launched via `depths scout` — esc/q exits, no menu

	width  int
	height int
}

type clockMsg time.Time
type pollMsg struct{}
type scoutResultMsg struct {
	report app.ScoutReport
	err    error
	at     time.Time
	gen    int
}

func defaultMenuItems() []surface.MenuItem {
	return []surface.MenuItem{
		{ID: "scout", Title: "Scout", Desc: "Watch memory pressure · live"},
		{ID: "dry-run", Title: "Dry-run", Desc: "Preview reclaim plan · no signals"},
		{ID: "profiles", Title: "Profiles", Desc: "gentle · focus · operator"},
		{ID: "safety", Title: "Safety", Desc: "Invariants before any claim"},
		{ID: "help", Title: "Help", Desc: "Commands and options"},
	}
}

func newRoot(cfg Config, initial app.ScoutReport, startScout bool) model {
	cfg.Options.Live = true
	m := model{
		screen:    screenMenu,
		items:     defaultMenuItems(),
		cfg:       cfg,
		report:    initial,
		updatedAt: time.Now(),
		clock:     time.Now(),
		width:     80,
		height:    24,
	}
	if startScout {
		m.screen = screenScout
		m.liveOn = true
		m.directScout = true
	}
	return m
}

func (m model) Init() tea.Cmd {
	if m.screen == screenScout && m.liveOn {
		return tea.Batch(scheduleClock(), schedulePoll(), func() tea.Msg { return pollMsg{} })
	}
	// Home: refresh STATUS panel periodically.
	return tea.Batch(scheduleClock(), scheduleMenuStatus(), func() tea.Msg { return pollMsg{} })
}

func scheduleClock() tea.Cmd {
	return tea.Tick(clockEvery, func(t time.Time) tea.Msg { return clockMsg(t) })
}

func schedulePoll() tea.Cmd {
	return tea.Tick(pollEvery, func(t time.Time) tea.Msg { return pollMsg{} })
}

func scheduleMenuStatus() tea.Cmd {
	return tea.Tick(menuStatusEvery, func(t time.Time) tea.Msg { return pollMsg{} })
}

func pollCmd(gen int, opt app.Options) tea.Cmd {
	opt.Live = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		report, _, _, err := app.Scout(ctx, opt)
		return scoutResultMsg{report: report, err: err, at: time.Now(), gen: gen}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case clockMsg:
		m.clock = time.Time(msg)
		if m.screen == screenScout && m.liveOn {
			m.frames++
			return m, scheduleClock()
		}
		// Menu: tick so "Last scout Xs ago" stays current.
		if m.screen == screenMenu {
			return m, scheduleClock()
		}
		return m, nil

	case pollMsg:
		if m.screen == screenMenu {
			cmds := []tea.Cmd{scheduleMenuStatus()}
			if !m.loading {
				m.loading = true
				m.pollGen++
				cmds = append(cmds, pollCmd(m.pollGen, m.cfg.Options))
			}
			return m, tea.Batch(cmds...)
		}
		if m.screen != screenScout || !m.liveOn {
			return m, nil
		}
		cmds := []tea.Cmd{schedulePoll()}
		if !m.loading {
			m.loading = true
			m.pollGen++
			cmds = append(cmds, pollCmd(m.pollGen, m.cfg.Options))
		}
		return m, tea.Batch(cmds...)

	case scoutResultMsg:
		if msg.gen != m.pollGen {
			return m, nil
		}
		m.loading = false
		m.updatedAt = msg.at
		m.clock = msg.at
		if m.screen == screenMenu {
			if msg.err == nil {
				m.report = msg.report
			}
			return m, nil
		}
		if m.screen != screenScout {
			return m, nil
		}
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.report = msg.report
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenOverlay:
		switch msg.String() {
		case "q", "esc", "enter", "backspace":
			m.screen = screenMenu
			m.overlay = ""
			return m, tea.Batch(scheduleClock(), scheduleMenuStatus())
		}
		return m, nil

	case screenScout:
		switch msg.String() {
		case "q", "esc":
			if m.directScout {
				return m, tea.Quit
			}
			// From starter: return to menu, resume STATUS refresh.
			m.screen = screenMenu
			m.liveOn = false
			m.loading = false
			return m, tea.Batch(scheduleClock(), scheduleMenuStatus(), func() tea.Msg { return pollMsg{} })
		case "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.pollGen++
			return m, pollCmd(m.pollGen, m.cfg.Options)
		}
		return m, nil

	default: // menu
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "1", "2", "3", "4", "5":
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(m.items) {
				m.cursor = idx
				return m.activate()
			}
		case "?", "h":
			m.screen = screenOverlay
			m.overlay = surface.HelpText("0.1.0")
		case "enter", " ":
			return m.activate()
		}
	}
	return m, nil
}

func (m model) activate() (tea.Model, tea.Cmd) {
	id := m.items[m.cursor].ID
	switch id {
	case "help":
		m.screen = screenOverlay
		m.overlay = surface.HelpText("0.1.0")
		return m, nil
	case "safety":
		m.screen = screenOverlay
		m.overlay = safetyOverlay()
		return m, nil
	case "profiles":
		m.screen = screenOverlay
		m.overlay = profilesOverlay()
		return m, nil
	case "dry-run":
		// Stay in-process: show dry-run as overlay text (fast).
		m.screen = screenOverlay
		m.overlay = dryRunOverlay(m.cfg.Options.ProfileName)
		return m, nil
	case "scout":
		m.screen = screenScout
		m.liveOn = true
		m.loading = false
		m.pollGen = 0
		m.clock = time.Now()
		m.frames = 0
		m.err = ""
		// Kick live loop immediately inside the same program.
		return m, tea.Batch(scheduleClock(), schedulePoll(), func() tea.Msg { return pollMsg{} })
	}
	return m, nil
}

func (m model) View() string {
	w, h := m.width, m.height
	if w < 52 {
		w = 52
	}
	if h < 18 {
		h = 18
	}

	switch m.screen {
	case screenOverlay:
		body := m.overlay + "\n\n  esc · return to depths"
		content := lipgloss.NewStyle().
			Foreground(surface.Foam).
			Background(surface.Abyss).
			Padding(1, 4).
			Width(w).
			Render(body)
		ch := lipgloss.Height(content)
		if ch < h {
			filler := lipgloss.NewStyle().Background(surface.Abyss).Width(w).Height(h - ch).Render("")
			content = lipgloss.JoinVertical(lipgloss.Left, content, filler)
		}
		return lipgloss.NewStyle().Background(surface.Abyss).Width(w).Height(h).MaxHeight(h).Render(content)

	case screenScout:
		pulse := "●"
		if m.frames%2 == 0 {
			pulse = "○"
		}
		return surface.ScoutSized(m.report, surface.Options{
			Color:     true,
			Width:     w,
			Height:    h,
			FullBleed: true,
			Live:      m.liveOn,
			UpdatedAt: m.clock,
			Err:       m.err,
			Loading:   m.loading,
			Pulse:     pulse,
		})

	default:
		st := surface.StatusFromReport(m.report, m.updatedAt)
		return surface.MenuView(true, w, h, m.cursor, m.items, st)
	}
}

func safetyOverlay() string {
	return `SAFETY · before any live claim

  Default is observe-only.
  Live claim needs explicit confirm (type claim) or --yes.
  Hard denylist cannot be weakened by profiles.
  Never kill DEPTHS ancestors (shell / terminal / self).
  PID identity checked at signal time.
  SIGTERM → grace → SIGKILL.
  Receipt written for every live claim.

  Read SAFETY.md in the project root.`
}

func profilesOverlay() string {
	return `PROFILES · observation policy

  gentle     high gate · browsers only · long grace
  focus      medium gate · browser / chat / media
  operator   lower gate · broader userland
             hard denylist still applies

  Gate language: HELD · READY · CLEAR · WATCHING`
}

func dryRunOverlay(profile string) string {
	if profile == "" {
		profile = "gentle"
	}
	opt := app.Options{ProfileName: profile, ForcePressure: true, Live: true}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	report, _, guard, err := app.Scout(ctx, opt)
	if err != nil {
		return "DRY-RUN\n\n  signal fault · " + err.Error()
	}
	plan, err := app.BuildClaimPlan(report, guard, true, false)
	if err != nil {
		return "DRY-RUN\n\n  signal fault · " + err.Error()
	}
	var b string
	b += "DRY-RUN · no signals\n\n"
	b += fmt.Sprintf("profile  %s\n", plan.Profile)
	b += fmt.Sprintf("targets  %d\n", len(plan.Targets))
	b += fmt.Sprintf("grace    %ds\n\n", plan.GraceSeconds)
	for _, t := range plan.Targets {
		b += fmt.Sprintf("  · %-22s pid=%-7d group=%s\n", t.Name, t.PID, t.GroupID)
	}
	if len(plan.Targets) == 0 {
		b += "  No claimable targets under this profile.\n"
	}
	b += "\ndepths › held · dry-run complete"
	return b
}

// RunMenu is the `depths` starter — one session, scout stays live on Enter.
func RunMenu() error {
	opt := app.Options{ProfileName: "gentle", Live: true}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	report, _, _, err := app.Scout(ctx, opt)
	if err != nil {
		report = app.ScoutReport{}
	}
	m := newRoot(Config{Options: opt, Live: true}, report, false)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// RunScoutLive is `depths scout` — jump straight into live watch.
func RunScoutLive(cfg Config, initial app.ScoutReport) error {
	cfg.Live = true
	cfg.Options.Live = true
	m := newRoot(cfg, initial, true)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Chosen kept for compatibility; unused by unified menu.
type Chosen struct {
	Action MenuAction
}
