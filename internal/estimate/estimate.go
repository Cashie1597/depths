package estimate

import (
	"github.com/cashie/depths/internal/group"
)

// Result is intentionally conservative. RSS is not freeable RAM.
type Result struct {
	GrossRSS      uint64  `json:"gross_rss"`
	EstimateFree  uint64  `json:"estimate_freeable"`
	Confidence    string  `json:"confidence"` // low | medium
	Note          string  `json:"note"`
	Factor        float64 `json:"factor"`
}

// ForGroups applies a haircut for shared/compressed pages.
// Browser/Electron helpers use a lower factor (more sharing).
func ForGroups(groups []group.Group) Result {
	var gross uint64
	var weighted float64
	for _, g := range groups {
		gross += g.RSS
		factor := 0.55
		switch g.Kind {
		case group.KindBrowser, group.KindElectron:
			factor = 0.40
		case group.KindChat, group.KindMedia:
			factor = 0.50
		case group.KindDev, group.KindAgent:
			factor = 0.60
		default:
			factor = 0.55
		}
		weighted += float64(g.RSS) * factor
	}
	est := uint64(weighted)
	conf := "low"
	if len(groups) > 0 && gross > 0 {
		conf = "medium"
	}
	return Result{
		GrossRSS:     gross,
		EstimateFree: est,
		Confidence:   conf,
		Factor:       weighted / max(float64(gross), 1),
		Note:         "estimate only — RSS ≠ freeable; shared/compressed pages reduce reclaim",
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
