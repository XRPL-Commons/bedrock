package jade

import "testing"

func TestDropsToXRP(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"0", "0"},
		{"1", "0.000001"},
		{"1000000", "1"},
		{"1500000", "1.5"},
		{"100000000", "100"},
		{"123456789", "123.456789"},
		{"100000001", "100.000001"},
		// Past int64 — make sure we still use big.Int correctly.
		{"99999999999999999999000000", "99999999999999999999"},
	}
	for _, tc := range cases {
		if got := dropsToXRP(tc.in); got != tc.want {
			t.Errorf("dropsToXRP(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDropsToXRPInvalid(t *testing.T) {
	if got := dropsToXRP("not-a-number"); got != "0" {
		t.Errorf("invalid input should yield %q, got %q", "0", got)
	}
}

func TestXRPToDrops(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"1", "1000000"},
		{"1.5", "1500000"},
		{"100", "100000000"},
		{"0.000001", "1"},
		{"123.456789", "123456789"},
	}
	for _, tc := range cases {
		got, err := xrpToDrops(tc.in)
		if err != nil {
			t.Errorf("xrpToDrops(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("xrpToDrops(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestXRPToDropsRejectsNonPositive(t *testing.T) {
	for _, in := range []string{"0", "-1", "-0.5"} {
		if _, err := xrpToDrops(in); err == nil {
			t.Errorf("xrpToDrops(%q) should reject non-positive input", in)
		}
	}
}

func TestXRPToDropsRejectsGarbage(t *testing.T) {
	if _, err := xrpToDrops("abc"); err == nil {
		t.Error("expected error parsing garbage input")
	}
}
