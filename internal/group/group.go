package group

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/cashie/depths/internal/sample"
)

type Kind string

const (
	KindBrowser Kind = "browser"
	KindChat    Kind = "chat"
	KindMedia   Kind = "media"
	KindDev     Kind = "dev"
	KindElectron Kind = "electron"
	KindAgent   Kind = "agent"
	KindOther   Kind = "other"
)

type Group struct {
	ID          string        `json:"id"`
	Kind        Kind          `json:"kind"`
	Label       string        `json:"label"`
	Procs       []sample.Proc `json:"processes"`
	RSS         uint64        `json:"rss"`
	Claimable   bool          `json:"claimable"`
	SkipReason  string        `json:"skip_reason,omitempty"`
}

type rule struct {
	id    string
	kind  Kind
	label string
	match func(name, cmd string) bool
}

var rules = []rule{
	{id: "chrome", kind: KindBrowser, label: "Google Chrome", match: containsAny("Google Chrome", "Chrome Helper", "Chromium")},
	{id: "arc", kind: KindBrowser, label: "Arc", match: containsAny("Arc Helper", "Arc.app")},
	{id: "safari", kind: KindBrowser, label: "Safari", match: containsAny("Safari.app", "com.apple.WebKit.WebContent", "com.apple.WebKit.Networking", "com.apple.WebKit.GPU")},
	{id: "firefox", kind: KindBrowser, label: "Firefox", match: containsAny("firefox", "Firefox")},
	{id: "edge", kind: KindBrowser, label: "Microsoft Edge", match: containsAny("Microsoft Edge", "Edge Helper")},
	{id: "slack", kind: KindChat, label: "Slack", match: containsAny("Slack")},
	{id: "discord", kind: KindChat, label: "Discord", match: containsAny("Discord")},
	{id: "teams", kind: KindChat, label: "Microsoft Teams", match: containsAny("Microsoft Teams", "Teams")},
	{id: "spotify", kind: KindMedia, label: "Spotify", match: containsAny("Spotify")},
	{id: "music", kind: KindMedia, label: "Music", match: exactOrHelper("Music")},
	{id: "code", kind: KindDev, label: "VS Code / Cursor helpers", match: containsAny("Code Helper", "Cursor Helper", "Electron Helper")},
	{id: "node", kind: KindDev, label: "Node.js", match: exactName("node")},
	{id: "python", kind: KindDev, label: "Python", match: exactNamePrefix("python")},
	{id: "docker", kind: KindDev, label: "Docker", match: containsAny("com.docker", "Docker")},
	{id: "electron", kind: KindElectron, label: "Other Electron", match: containsAny("Electron")},
	{id: "agents", kind: KindAgent, label: "Local AI / agents", match: containsAny("ollama", "llama", "litellm", "opencode")},
}

func containsAny(needles ...string) func(string, string) bool {
	return func(name, cmd string) bool {
		hay := name + " " + cmd
		for _, n := range needles {
			if strings.Contains(hay, n) {
				return true
			}
		}
		return false
	}
}

func exactName(want string) func(string, string) bool {
	return func(name, _ string) bool {
		return strings.EqualFold(filepath.Base(name), want)
	}
}

func exactNamePrefix(prefix string) func(string, string) bool {
	return func(name, _ string) bool {
		return strings.HasPrefix(strings.ToLower(filepath.Base(name)), strings.ToLower(prefix))
	}
}

func exactOrHelper(app string) func(string, string) bool {
	return containsAny(app, app+" Helper")
}

// Build groups claimable processes by known app families.
func Build(procs []sample.Proc) []Group {
	buckets := map[string]*Group{}
	assigned := map[int32]struct{}{}

	for _, r := range rules {
		g := &Group{ID: r.id, Kind: r.kind, Label: r.label, Claimable: true}
		for _, p := range procs {
			if _, ok := assigned[p.PID]; ok {
				continue
			}
			if r.match(p.Name, p.Cmdline) {
				g.Procs = append(g.Procs, p)
				g.RSS += p.RSS
				assigned[p.PID] = struct{}{}
			}
		}
		if len(g.Procs) > 0 {
			buckets[r.id] = g
		}
	}

	// Singletons / leftovers as other:<name>
	for _, p := range procs {
		if _, ok := assigned[p.PID]; ok {
			continue
		}
		id := "other:" + sanitizeID(p.Name)
		g, ok := buckets[id]
		if !ok {
			g = &Group{ID: id, Kind: KindOther, Label: p.Name, Claimable: true}
			buckets[id] = g
		}
		g.Procs = append(g.Procs, p)
		g.RSS += p.RSS
	}

	out := make([]Group, 0, len(buckets))
	for _, g := range buckets {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RSS == out[j].RSS {
			return out[i].Label < out[j].Label
		}
		return out[i].RSS > out[j].RSS
	})
	return out
}

func sanitizeID(name string) string {
	name = filepath.Base(name)
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)
	if name == "" {
		return "unknown"
	}
	return name
}

func FilterByIDs(groups []Group, ids []string) []Group {
	if len(ids) == 0 {
		return groups
	}
	want := map[string]struct{}{}
	for _, id := range ids {
		want[id] = struct{}{}
	}
	var out []Group
	for _, g := range groups {
		if _, ok := want[g.ID]; ok {
			out = append(out, g)
		}
	}
	return out
}

func FilterByKinds(groups []Group, kinds []Kind) []Group {
	if len(kinds) == 0 {
		return groups
	}
	want := map[Kind]struct{}{}
	for _, k := range kinds {
		want[k] = struct{}{}
	}
	var out []Group
	for _, g := range groups {
		if _, ok := want[g.Kind]; ok {
			out = append(out, g)
		}
	}
	return out
}
