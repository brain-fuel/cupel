module goforge.dev/cupel

go 1.26.0

// Go+ toolchain. `go tool goplus gen ./...` regenerates every *_gp.go from its
// .gp source; plain `go build`/`go test` consumes the committed generated Go.
tool goforge.dev/goplus/cmd/goplus

require (
	goforge.dev/goplus v0.138.0 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

// DEV replace: the local goplus carries the unreleased convergence fix needed to
// gen cupel's larger packages. REMOVE before release (never ship a replace).
