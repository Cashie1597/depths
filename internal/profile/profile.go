package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cashie/depths/internal/group"
	"gopkg.in/yaml.v3"
)

type Profile struct {
	Name              string   `yaml:"name" json:"name"`
	Description       string   `yaml:"description" json:"description"`
	MinPressure       string   `yaml:"min_pressure" json:"min_pressure"` // normal|warn|critical
	MinSwapUsedBytes  uint64   `yaml:"min_swap_used_bytes" json:"min_swap_used_bytes"`
	AllowKinds        []string `yaml:"allow_kinds" json:"allow_kinds"`
	GraceSeconds      int      `yaml:"grace_seconds" json:"grace_seconds"`
	AllowForceKill    bool     `yaml:"allow_force_kill" json:"allow_force_kill"`
	MaxGroups         int      `yaml:"max_groups" json:"max_groups"`
}

func (p Profile) Kinds() []group.Kind {
	out := make([]group.Kind, 0, len(p.AllowKinds))
	for _, k := range p.AllowKinds {
		out = append(out, group.Kind(strings.ToLower(k)))
	}
	return out
}

func Builtin() map[string]Profile {
	return map[string]Profile{
		"gentle": {
			Name:             "gentle",
			Description:      "High gate; browsers/helpers only; long grace",
			MinPressure:      "warn",
			MinSwapUsedBytes: 512 << 20,
			AllowKinds:       []string{"browser"},
			GraceSeconds:     15,
			AllowForceKill:   false,
			MaxGroups:        3,
		},
		"focus": {
			Name:             "focus",
			Description:      "Medium gate; browsers + chat + media",
			MinPressure:      "warn",
			MinSwapUsedBytes: 256 << 20,
			AllowKinds:       []string{"browser", "chat", "media"},
			GraceSeconds:     10,
			AllowForceKill:   false,
			MaxGroups:        5,
		},
		"operator": {
			Name:             "operator",
			Description:      "Lower gate; broader userland — hard denylist still applies",
			MinPressure:      "normal",
			MinSwapUsedBytes: 64 << 20,
			AllowKinds:       []string{"browser", "chat", "media", "electron", "dev", "agent", "other"},
			GraceSeconds:     5,
			AllowForceKill:   true,
			MaxGroups:        12,
		},
	}
}

func Load(name, overridePath string) (Profile, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "gentle"
	}

	if overridePath != "" {
		return loadFile(overridePath)
	}

	// Optional user override: ~/.config/depths/profiles/<name>.yaml
	home, _ := os.UserHomeDir()
	userPath := filepath.Join(home, ".config", "depths", "profiles", name+".yaml")
	if st, err := os.Stat(userPath); err == nil && !st.IsDir() {
		return loadFile(userPath)
	}

	// Bundled profiles next to module (dev) or embedded defaults.
	if p, ok := Builtin()[name]; ok {
		return p, nil
	}

	// Try ./profiles/<name>.yaml
	local := filepath.Join("profiles", name+".yaml")
	if st, err := os.Stat(local); err == nil && !st.IsDir() {
		return loadFile(local)
	}

	return Profile{}, fmt.Errorf("unknown profile %q (gentle|focus|operator)", name)
}

func loadFile(path string) (Profile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := yaml.Unmarshal(b, &p); err != nil {
		return Profile{}, err
	}
	if p.Name == "" {
		p.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if p.GraceSeconds <= 0 {
		p.GraceSeconds = 10
	}
	return p, nil
}

func List() []Profile {
	b := Builtin()
	order := []string{"gentle", "focus", "operator"}
	out := make([]Profile, 0, len(order))
	for _, n := range order {
		out = append(out, b[n])
	}
	return out
}
