package profile_test

import (
	"testing"

	"github.com/cashie/depths/internal/profile"
)

func TestBuiltinProfiles(t *testing.T) {
	for _, name := range []string{"gentle", "focus", "operator"} {
		p, err := profile.Load(name, "")
		if err != nil {
			t.Fatal(err)
		}
		if p.Name != name {
			t.Fatalf("name=%s", p.Name)
		}
		if p.GraceSeconds <= 0 {
			t.Fatal("grace")
		}
		if len(p.AllowKinds) == 0 {
			t.Fatal("kinds")
		}
	}
}

func TestUnknownProfile(t *testing.T) {
	_, err := profile.Load("nope", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
