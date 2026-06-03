// Command cupel generates example, parameterized/data-driven, and property-based
// Go tests from an English specification by calling Claude, writing them into a
// goforge component and validating the result against the goforge architecture.
package main

import (
	"os"

	"goforge.dev/cupel/bases/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
