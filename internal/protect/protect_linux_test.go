//go:build linux

package protect_test

import (
	"testing"

	"github.com/cashie/depths/internal/protect"
	"github.com/cashie/depths/internal/sample"
)

func TestLinuxDenylist(t *testing.T) {
	procs := []sample.Proc{
		{PID: 100, Name: "systemd", PPID: 1},
		{PID: 101, Name: "kthreadd", PPID: 0},
		{PID: 102, Name: "dockerd", PPID: 1},
		{PID: 50, Name: "foo", PPID: 1, Cmdline: "/usr/lib/systemd/systemd-foo"},
	}
	g := protect.NewGuard(procs)
	for _, p := range procs {
		if d := g.Check(p); !d.Protected {
			t.Fatalf("%s should be protected: %s", p.Name, d.Reason)
		}
	}
}
