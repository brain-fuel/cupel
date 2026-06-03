package spec_test

import (
	"strings"
	"testing"
	"testing/quick"

	"goforge.dev/cupel/components/spec"
)

// Example test: one concrete, readable case.
func TestParse_DefaultsInterfaceToName(t *testing.T) {
	s, err := spec.Parse("user", "", "A user has a non-empty name.")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Interface != "user" {
		t.Errorf("Interface = %q, want %q", s.Interface, "user")
	}
	if s.TestPackage() != "user_test" {
		t.Errorf("TestPackage = %q, want %q", s.TestPackage(), "user_test")
	}
}

// Table-driven test: data-driven validation rules.
func TestParse_Validation(t *testing.T) {
	cases := []struct {
		name      string
		comp      string
		iface     string
		text      string
		wantError bool
	}{
		{"ok", "user", "", "non-empty name", false},
		{"ok custom iface", "user", "account", "x", false},
		{"empty name", "", "", "x", true},
		{"bad name", "2cool", "", "x", true},
		{"keyword name", "func", "", "x", true},
		{"bad iface", "user", "2cool", "x", true},
		{"empty text", "user", "", "   ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := spec.Parse(tc.comp, tc.iface, tc.text)
			if (err != nil) != tc.wantError {
				t.Errorf("Parse(%q,%q,%q) error = %v, wantError = %v", tc.comp, tc.iface, tc.text, err, tc.wantError)
			}
		})
	}
}

// Property-based test: any valid spec round-trips its package naming.
func TestParse_TestPackageProperty(t *testing.T) {
	prop := func(text string) bool {
		text = strings.TrimSpace(text)
		if text == "" {
			return true // vacuous: empty text is rejected, nothing to assert
		}
		s, err := spec.Parse("user", "account", text)
		if err != nil {
			return false
		}
		return s.TestPackage() == s.Interface+"_test"
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Error(err)
	}
}
