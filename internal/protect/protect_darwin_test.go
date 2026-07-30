//go:build darwin

package protect_test

import (
	"testing"

	"github.com/cashie/depths/internal/protect"
	"github.com/cashie/depths/internal/sample"
)

func TestDarwinDenylist(t *testing.T) {
	procs := []sample.Proc{
		{PID: 100, Name: "WindowServer", PPID: 1},
		{PID: 300, Name: "kernel_task", PPID: 0},
		{PID: 50, Name: "foo", PPID: 1, Cmdline: "/usr/libexec/foo"},
	}
	g := protect.NewGuard(procs)
	for _, p := range procs {
		if d := g.Check(p); !d.Protected {
			t.Fatalf("%s should be protected: %s", p.Name, d.Reason)
		}
	}
}
