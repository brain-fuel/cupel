// Package internal holds the architecture component implementation: the logic
// that enforces the goforge brick layout on generated test output.
package internal

import (
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// BrickView is the projection of a spec needed to render a brick.
type BrickView struct {
	Name        string
	Interface   string
	TestPackage string
	Module      string
	Tests       string
}

// File is a rendered component file.
type File struct {
	RelPath   string
	Content   string
	Generated bool
}

// EnforceTests strips fencing, parses, validates the package, and formats raw.
func EnforceTests(wantPkg, raw string) (string, error) {
	src := stripFence(raw)
	if strings.TrimSpace(src) == "" {
		return "", fmt.Errorf("architecture: model returned no Go source")
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, wantPkg+".go", src, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("architecture: generated source is not valid Go: %w", err)
	}
	if f.Name.Name != wantPkg {
		return "", fmt.Errorf("architecture: generated tests declare package %q, want %q (external black-box test package)", f.Name.Name, wantPkg)
	}

	formatted, err := format.Source([]byte(src))
	if err != nil {
		return "", fmt.Errorf("architecture: generated source does not gofmt/compile: %w", err)
	}
	return string(formatted), nil
}

// stripFence removes a single surrounding markdown code fence if present.
func stripFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Drop the opening fence line (``` or ```go).
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:]
	}
	// Drop the trailing fence.
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// Brick renders the full set of component files.
func Brick(v BrickView) []File {
	return []File{
		{RelPath: v.Name + "_test.go", Content: ensureTrailingNewline(v.Tests), Generated: true},
		{RelPath: v.Name + ".go", Content: renderInterface(v), Generated: false},
		{RelPath: "internal/core.go", Content: renderCore(v), Generated: false},
		{RelPath: "component.yaml", Content: renderComponentYAML(v), Generated: false},
	}
}

func renderInterface(v BrickView) string {
	return fmt.Sprintf(`// Package %[1]s is the public interface of the %[2]s component.
//
// This stub was scaffolded by cupel so the generated tests have a brick to
// compile against. Replace the body with the real implementation described by
// the specification.
package %[1]s

import "%[3]s/components/%[2]s/internal"

// TODO: implement the API the generated %[2]s_test.go exercises, delegating to
// the internal package. The following is a placeholder to keep the brick valid.
func Describe() string { return internal.Describe() }
`, v.Interface, v.Name, v.Module)
}

func renderCore(v BrickView) string {
	return fmt.Sprintf(`// Package internal holds the %[1]s component implementation.
package internal

// Describe is a placeholder; replace it with the real %[1]s implementation.
func Describe() string { return %[2]q }
`, v.Name, v.Name+" component")
}

func renderComponentYAML(v BrickView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Metadata for the %s component (scaffolded by cupel).\n", v.Name)
	b.WriteString("#\n# worker_type classifies the component along two axes (see principles.yaml):\n")
	b.WriteString("#   determinism:   deterministic | non_deterministic\n")
	b.WriteString("#   effectfulness: pure | [input_effect] | [output_effect] | [input_effect, output_effect]\n")
	if v.Interface != v.Name {
		fmt.Fprintf(&b, "interface: %s\n", v.Interface)
	}
	b.WriteString("worker_type:\n")
	b.WriteString("  determinism: deterministic\n")
	b.WriteString("  effectfulness: pure\n")
	return b.String()
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
