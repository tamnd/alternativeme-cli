package alternativeme

import (
	"testing"
)

// These tests are offline: they exercise the URI driver's pure string functions.
// Network behaviour is covered in alternativeme_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "alternativeme" {
		t.Errorf("Scheme = %q, want alternativeme", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "alternativeme" {
		t.Errorf("Identity.Binary = %q, want alternativeme", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in, typ, id string
	}{
		{"feargreed", "index", "feargreed"},
		{"index", "index", "feargreed"},
		{"2024-01-01", "date", "2024-01-01"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("index", "feargreed")
	want := "https://alternative.me/crypto/fear-and-greed-index/"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("bogus", "x")
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

func TestClassifyUnknown(t *testing.T) {
	_, _, err := Domain{}.Classify("notaresource")
	if err == nil {
		t.Error("expected error for unknown input, got nil")
	}
}
