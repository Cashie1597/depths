package protect_test

import (
	"testing"

	"github.com/cashie/depths/internal/protect"
	"github.com/cashie/depths/internal/sample"
)

func TestHardDenylist(t *testing.T) {
	procs := []sample.Proc{
		{PID: 1, Name: "launchd", PPID: 0},
		{PID: 100, Name: "WindowServer", PPID: 1},
		{PID: 200, Name: "Google Chrome", PPID: 1, Cmdline: "/Applications/Google Chrome.app/..."},
		{PID: 300, Name: "kernel_task", PPID: 0},
	}
	g := protect.NewGuard(procs)

	cases := []struct {
		pid     int32
		protect bool
	}{
		{1, true},
		{100, true},
		{200, false},
		{300, true},
	}
	for _, tc := range cases {
		var p sample.Proc
		for _, x := range procs {
			if x.PID == tc.pid {
				p = x
				break
			}
		}
		d := g.Check(p)
		if d.Protected != tc.protect {
			t.Fatalf("pid %d name %s: protected=%v want %v (%s)", tc.pid, p.Name, d.Protected, tc.protect, d.Reason)
		}
	}
}

func TestSystemPathProtected(t *testing.T) {
	procs := []sample.Proc{
		{PID: 50, Name: "foo", PPID: 1, Cmdline: "/usr/libexec/foo"},
	}
	g := protect.NewGuard(procs)
	d := g.Check(procs[0])
	if !d.Protected {
		t.Fatal("expected system path protection")
	}
}
