// Package internal holds the spec component implementation. Only the spec
// component's interface package may import it.
package internal

import (
	"fmt"
	"go/token"
	"strings"
)

// Parse validates the three raw inputs and returns the normalised name,
// interface and trimmed text. iface defaults to name when empty.
func Parse(name, iface, text string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	iface = strings.TrimSpace(iface)
	text = strings.TrimSpace(text)

	if name == "" {
		return "", "", "", fmt.Errorf("spec: name is required")
	}
	if !isIdent(name) {
		return "", "", "", fmt.Errorf("spec: name %q is not a valid Go identifier", name)
	}
	if iface == "" {
		iface = name
	}
	if !isIdent(iface) {
		return "", "", "", fmt.Errorf("spec: interface %q is not a valid Go identifier", iface)
	}
	if text == "" {
		return "", "", "", fmt.Errorf("spec: specification text is empty")
	}
	return name, iface, text, nil
}

// isIdent reports whether s is a non-keyword Go identifier usable as both a
// package name and a directory name.
func isIdent(s string) bool {
	if s == "" || token.IsKeyword(s) {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}
