package protect

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cashie/depths/internal/sample"
)

// Hard denylist — profiles cannot weaken these matches.
var hardNameDeny = []string{
	"kernel_task",
	"launchd",
	"WindowServer",
	"loginwindow",
	"UserEventAgent",
	"SystemUIServer",
	"Dock",
	"Finder",
	"cfprefsd",
	"opendirectoryd",
	"securityd",
	"trustd",
	"amfid",
	"syspolicyd",
	"coreservicesd",
	"configd",
	"powerd",
	"logd",
	"notifyd",
	"distnoted",
	"sharedfilelistd",
	"tccd",
	"sandboxd",
	"secd",
	"TrustEvaluationAgent",
	"backupd",
	"backupd-helper",
	"Time Machine",
	"mds",
	"mds_stores",
	"mdworker",
	"mdworker_shared",
	"Spotlight",
	"ssh-agent",
	"sudo",
	"depths",
}

type Decision struct {
	Protected bool   `json:"protected"`
	Reason    string `json:"reason,omitempty"`
}

type Guard struct {
	selfPID   int32
	ancestors map[int32]struct{}
}

func NewGuard(procs []sample.Proc) *Guard {
	self := int32(os.Getpid())
	anc := map[int32]struct{}{self: {}}
	byPID := make(map[int32]sample.Proc, len(procs))
	for _, p := range procs {
		byPID[p.PID] = p
	}
	cur := self
	for i := 0; i < 64; i++ {
		p, ok := byPID[cur]
		if !ok || p.PPID <= 0 {
			break
		}
		anc[p.PPID] = struct{}{}
		cur = p.PPID
	}
	return &Guard{selfPID: self, ancestors: anc}
}

func (g *Guard) Check(p sample.Proc) Decision {
	if p.PID <= 0 {
		return Decision{Protected: true, Reason: "invalid pid"}
	}
	if p.PID == g.selfPID {
		return Decision{Protected: true, Reason: "self"}
	}
	if _, ok := g.ancestors[p.PID]; ok {
		return Decision{Protected: true, Reason: "ancestor of depths"}
	}
	if p.PID == 0 || p.PID == 1 {
		return Decision{Protected: true, Reason: "system pid"}
	}
	base := filepath.Base(p.Name)
	lower := strings.ToLower(base)
	for _, deny := range hardNameDeny {
		if strings.EqualFold(base, deny) || strings.Contains(lower, strings.ToLower(deny)) {
			return Decision{Protected: true, Reason: "hard denylist: " + deny}
		}
	}
	// System paths under /System and /usr/libexec are protected by default.
	cmd := p.Cmdline
	if cmd == "" {
		cmd = p.Name
	}
	if strings.HasPrefix(cmd, "/System/") || strings.HasPrefix(cmd, "/usr/libexec/") || strings.HasPrefix(cmd, "/sbin/") {
		return Decision{Protected: true, Reason: "system path"}
	}
	return Decision{}
}

func (g *Guard) FilterClaimable(procs []sample.Proc) (claimable []sample.Proc, denied []sample.Proc) {
	for _, p := range procs {
		if d := g.Check(p); d.Protected {
			denied = append(denied, p)
			continue
		}
		claimable = append(claimable, p)
	}
	return claimable, denied
}
