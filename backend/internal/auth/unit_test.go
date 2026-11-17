package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	good := map[string]string{
		"  Demo@Eneba.TEST ":       "demo@eneba.test",
		"vykintas.k@go.example.lt": "vykintas.k@go.example.lt",
	}
	for in, want := range good {
		got, err := normalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("normalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "nope", "a@b", "Name <a@b.com>", "a@@b.com"} {
		if _, err := normalizeEmail(in); err == nil {
			t.Errorf("normalizeEmail(%q): expected an error", in)
		}
	}
}

func TestHashTokenIsSHA256(t *testing.T) {
	h := hashToken("abc")
	if len(h) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(h))
	}
	if string(h) == "abc" || string(hashToken("abd")) == string(h) {
		t.Error("hash must differ from the token and between tokens")
	}
}
