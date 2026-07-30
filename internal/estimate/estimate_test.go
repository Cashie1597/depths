package estimate_test

import (
	"testing"

	"github.com/cashie/depths/internal/estimate"
	"github.com/cashie/depths/internal/group"
)

func TestEstimateBelowGross(t *testing.T) {
	gs := []group.Group{
		{ID: "chrome", Kind: group.KindBrowser, RSS: 2 << 30},
	}
	r := estimate.ForGroups(gs)
	if r.EstimateFree >= r.GrossRSS {
		t.Fatalf("estimate should haircut RSS: free=%d gross=%d", r.EstimateFree, r.GrossRSS)
	}
	if r.Confidence != "medium" {
		t.Fatalf("confidence=%s", r.Confidence)
	}
}
