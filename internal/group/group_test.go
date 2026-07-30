package group_test

import (
	"testing"

	"github.com/cashie/depths/internal/group"
	"github.com/cashie/depths/internal/sample"
)

func TestBuildGroupsChrome(t *testing.T) {
	procs := []sample.Proc{
		{PID: 1, Name: "Google Chrome", RSS: 1 << 30},
		{PID: 2, Name: "Google Chrome Helper", RSS: 400 << 20},
		{PID: 3, Name: "Slack", RSS: 300 << 20},
		{PID: 4, Name: "WeirdApp", RSS: 50 << 20},
	}
	gs := group.Build(procs)
	if len(gs) < 2 {
		t.Fatalf("expected multiple groups, got %d", len(gs))
	}
	var chrome *group.Group
	for i := range gs {
		if gs[i].ID == "chrome" {
			chrome = &gs[i]
			break
		}
	}
	if chrome == nil {
		t.Fatal("chrome group missing")
	}
	if len(chrome.Procs) != 2 {
		t.Fatalf("chrome procs=%d want 2", len(chrome.Procs))
	}
	if chrome.RSS != (1<<30)+(400<<20) {
		t.Fatalf("chrome rss=%d", chrome.RSS)
	}
}

func TestFilterByKinds(t *testing.T) {
	gs := []group.Group{
		{ID: "chrome", Kind: group.KindBrowser},
		{ID: "slack", Kind: group.KindChat},
	}
	out := group.FilterByKinds(gs, []group.Kind{group.KindBrowser})
	if len(out) != 1 || out[0].ID != "chrome" {
		t.Fatalf("unexpected: %+v", out)
	}
}
