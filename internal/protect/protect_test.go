package protect_test

import (
	"testing"

	"github.com/cashie/depths/internal/protect"
	"github.com/cashie/depths/internal/sample"
)

func TestSharedDenylistAndSelf(t *testing.T) {
	procs := []sample.Proc{
		{PID: 1, Name: "init", PPID: 0},
		{PID: 200, Name: "Google Chrome", PPID: 1, Cmdline: "/opt/google/chrome/chrome"},
		{PID: 400, Name: "sudo", PPID: 1},
		{PID: 401, Name: "sshd", PPID: 1},
	}
	g := protect.NewGuard(procs)

	cases := []struct {
		pid     int32
		protect bool
	}{
		{1, true},
		{200, false},
		{400, true},
		{401, true},
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
