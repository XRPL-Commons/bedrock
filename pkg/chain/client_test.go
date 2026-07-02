package chain

import "testing"

// TestToHTTPURL covers the WS→HTTP conversion, including the local-dev remap
// of WS port 6006 to JSON-RPC port 5005 (the fix that made the `jade` network
// family work against a local node).
func TestToHTTPURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"local ws remaps 6006 to 5005", "ws://localhost:6006", "http://localhost:5005"},
		{"ws keeps non-6006 port", "ws://localhost:1234", "http://localhost:1234"},
		{"remap only matches localhost host", "ws://example.com:6006", "http://example.com:6006"},
		{"wss becomes https", "wss://alphanet.nerdnest.xyz", "https://alphanet.nerdnest.xyz"},
		{"wss does not remap ports", "wss://node.example:6006", "https://node.example:6006"},
		{"http passes through", "http://localhost:5005", "http://localhost:5005"},
		{"https passes through", "https://alphanet.nerdnest.xyz", "https://alphanet.nerdnest.xyz"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ToHTTPURL(c.in); got != c.want {
				t.Errorf("ToHTTPURL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestNewClientAppliesToHTTPURL ensures the client talks JSON-RPC on the
// remapped local port rather than the WebSocket port.
func TestNewClientAppliesToHTTPURL(t *testing.T) {
	c := NewClient("ws://localhost:6006")
	if c.httpURL != "http://localhost:5005" {
		t.Errorf("NewClient httpURL = %q, want %q", c.httpURL, "http://localhost:5005")
	}
	if c.httpClient == nil {
		t.Error("NewClient httpClient is nil")
	}
}

func TestDropsToXRP(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1000000", "1.000000"},
		{"1500000", "1.500000"},
		{"0", "0.000000"},
		{"1", "0.000001"},
		{"100000000", "100.000000"},
		{"999999", "0.999999"},
		{"", "0.000000"}, // unparseable → zero
	}
	for _, c := range cases {
		if got := DropsToXRP(c.in); got != c.want {
			t.Errorf("DropsToXRP(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
