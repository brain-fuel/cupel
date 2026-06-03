// Package architecture is the public interface of the architecture component.
// It enforces the goforge architecture on generated output: it strips any
// markdown fencing from the model's reply, verifies the result is valid Go
// declaring the expected external test package, gofmt-formats it, and renders
// the surrounding goforge brick (interface package, internal implementation,
// and component.yaml) so the generated tests live in a check-valid component.
//
// architecture is a deterministic, pure brick: it performs no I/O. The gen
// component is responsible for writing the rendered files to disk.
package architecture

import (
	"goforge.dev/cupel/components/architecture/internal"
	"goforge.dev/cupel/components/spec"
)

// File is a rendered file destined for a path relative to the component dir.
type File struct {
	// RelPath is relative to components/<name>/ (e.g. "user.go",
	// "internal/core.go", "user_test.go", "component.yaml").
	RelPath string
	// Content is the file body.
	Content string
	// Generated marks files produced from the model reply (the tests) versus
	// scaffolding written only when missing.
	Generated bool
}

// EnforceTests validates and gofmt-formats raw model output as the test file for
// s. It returns the formatted source or an error explaining how the output
// violated the goforge/Go contract.
func EnforceTests(s spec.Spec, raw string) (string, error) {
	return internal.EnforceTests(s.TestPackage(), raw)
}

// Brick renders every file of the goforge component for s, given the workspace
// module path. The _test.go file uses formattedTests (already validated by
// EnforceTests). Scaffolding files (interface, internal, component.yaml) are
// marked non-Generated so callers can decline to overwrite an existing brick.
func Brick(s spec.Spec, module, formattedTests string) []File {
	out := internal.Brick(internal.BrickView{
		Name:        s.Name,
		Interface:   s.Interface,
		TestPackage: s.TestPackage(),
		Module:      module,
		Tests:       formattedTests,
	})
	files := make([]File, len(out))
	for i, f := range out {
		files[i] = File{RelPath: f.RelPath, Content: f.Content, Generated: f.Generated}
	}
	return files
}
