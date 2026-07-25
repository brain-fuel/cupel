// Package spec is the public interface of the spec component. It parses and
// validates an English-language specification together with the name of the
// component the tests are being written for.
//
// spec is a deterministic, pure brick: given the same inputs it always yields
// the same Spec and reads nothing from the world.
package spec

import "goforge.dev/cupel/components/spec/internal"

// Spec is a validated test specification: a target component name, the Go
// package the generated tests live alongside, and the English prose describing
// the behaviour to be tested.
type Spec struct {
	// Name is the goforge component the tests target (e.g. "user"). It is a
	// valid Go identifier and a valid directory name.
	Name string
	// Interface is the public package name of that component. It defaults to
	// Name and is the package the example/table tests are written against.
	Interface string
	// Text is the English-language behaviour specification.
	Text string
}

// Parse validates name, interface and prose and returns a Spec. iface may be
// empty, in which case it defaults to name.
func Parse(name, iface, text string) (Spec, error) {
	n, i, t, err := internal.Parse(name, iface, text)
	if err != nil {
		return Spec{}, err
	}
	return Spec{Name: n, Interface: i, Text: t}, nil
}

// TestPackage is the package name the generated _test.go file declares: the
// black-box external test package, Interface + "_test".
func (s Spec) TestPackage() string { return s.Interface + "_test" }
