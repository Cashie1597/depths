package protect

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cashie/depths/internal/sample"
)

// Shared hard denies — self-preservation and remote/admin tools.
// OS-specific lists live in protect_darwin.go / protect_linux.go.
var sharedNameDeny = []string{
	"depths",
	"sudo",
	"ssh-agent",
	"sshd",
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
	for _, deny := range sharedNameDeny {
		if strings.EqualFold(base, deny) {
			return Decision{Protected: true, Reason: "hard denylist: " + deny}
		}
	}
	for _, deny := range osNameDeny {
		if strings.EqualFold(base, deny) {
			return Decision{Protected: true, Reason: "hard denylist: " + deny}
		}
	}
	cmd := p.Cmdline
	if cmd == "" {
		cmd = p.Name
	}
	for _, prefix := range osProtectedPrefixes {
		if strings.HasPrefix(cmd, prefix) {
			return Decision{Protected: true, Reason: "system path"}
		}
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
