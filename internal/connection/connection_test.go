package connection

import "testing"

func TestNormalizeAddress(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"127.0.0.1":        "127.0.0.1:5555",
		"127.0.0.1:16384":  "127.0.0.1:16384",
		" 127.0.0.1:5555 ": "127.0.0.1:5555",
	}
	for input, want := range cases {
		if got := normalizeAddress(input); got != want {
			t.Fatalf("normalizeAddress(%q) = %q, want %q", input, got, want)
		}
	}
}
