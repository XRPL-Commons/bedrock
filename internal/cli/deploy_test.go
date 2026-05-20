package cli

import (
	"strings"
	"testing"
)

// TestMaskSeedNeverLeaksMiddle is the property that matters: the middle
// (entropic) portion of the seed must never appear in the masked output.
// If anyone tightens or loosens maskSeed in the future, this test catches
// regressions of the seed-leak fix.
func TestMaskSeedNeverLeaksMiddle(t *testing.T) {
	seed := "sEdSeedExampleVeryLongPlaintextSensitive"
	got := maskSeed(seed)
	middle := seed[4 : len(seed)-4]
	if strings.Contains(got, middle) {
		t.Errorf("maskSeed leaked middle: input=%q masked=%q", seed, got)
	}
	if !strings.HasPrefix(got, seed[:4]) {
		t.Errorf("expected first 4 chars preserved: got %q", got)
	}
	if !strings.HasSuffix(got, seed[len(seed)-4:]) {
		t.Errorf("expected last 4 chars preserved: got %q", got)
	}
	if len(got) != len(seed) {
		t.Errorf("masked length should equal original length: got %d want %d", len(got), len(seed))
	}
}

func TestMaskSeedShort(t *testing.T) {
	for _, in := range []string{"", "abc", "12345678"} {
		got := maskSeed(in)
		if got == in {
			t.Errorf("maskSeed(%q) should mask short input, got %q", in, got)
		}
	}
}
