// Package cli is the cupel base: the entry-point layer that wires the spec,
// claude, and gen components into a runnable command. As a base it is the
// boundary where effects (argument parsing, stdin, environment, file system,
// the network, and shelling out to goforge) enter the system.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"goforge.dev/cupel/components/claude"
	"goforge.dev/cupel/components/claudecode"
	"goforge.dev/cupel/components/gen"
	"goforge.dev/cupel/components/spec"
)

// completer is a generation backend: it turns a prompt into a reply and can
// name the model it uses. Both the claude (API) and claudecode (CLI) clients
// satisfy it, and it satisfies gen.Completer.
type completer interface {
	Complete(ctx context.Context, system, user string) (string, error)
	Model() string
}

const usage = `cupel — generate example, table-driven, and property-based Go tests from an
English specification, written into a goforge component and validated against
the goforge architecture.

Usage:
  cupel name:NAME [spec:"..."|spec-file:PATH] [key:value ...] [:flags]
  cat spec.txt | cupel name:NAME

Arguments:
  name:NAME        target goforge component (required)
  iface:PKG        public package name of that component (default: NAME)
  spec:"TEXT"      inline English specification
  spec-file:PATH   read the specification from a file
  (stdin)          read the specification from stdin if neither is given
  backend:WHICH    generation backend: api (Anthropic API, default) | cli
                   (drive the local Claude Code "claude" CLI)
  model:ID         Claude model (default: env CUPEL_MODEL, else api uses
                   claude-opus-4-8 and cli uses its own default)
  root:DIR         workspace root (default: discovered from the cwd)

Flags:
  :force           overwrite scaffold files (interface/internal/component.yaml)
  :dry-run         render everything but write nothing
  :no-check        skip running "goforge check" on the result

Environment:
  ANTHROPIC_API_KEY   required for the api backend
  CUPEL_CLAUDE_BIN    override the claude binary used by the cli backend
`

// Run is the cupel entry point. It returns a process exit code.
func Run(args []string) int {
	return run(args, os.Stdin, os.Stdout, os.Stderr)
}

func run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	a := parse(argv)

	if a.flag("help") || a.named["name"] == "" && len(argv) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}

	name := a.named["name"]
	if name == "" {
		fmt.Fprintln(stderr, "cupel: missing required argument name:NAME (try: cupel :help)")
		return 2
	}

	text, err := specText(a, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "cupel: %v\n", err)
		return 1
	}

	s, err := spec.Parse(name, a.named["iface"], text)
	if err != nil {
		fmt.Fprintf(stderr, "cupel: %v\n", err)
		return 1
	}

	root, module, err := workspace(a.named["root"])
	if err != nil {
		fmt.Fprintf(stderr, "cupel: %v\n", err)
		return 1
	}

	cl, err := backendClient(a)
	if err != nil {
		fmt.Fprintf(stderr, "cupel: %v\n", err)
		return 1
	}
	fmt.Fprintf(stderr, "cupel: generating tests for %q via %s backend with %s...\n", s.Name, a.backend(), cl.Model())

	res, err := gen.Generate(context.Background(), cl, gen.Request{
		Spec:   s,
		Module: module,
		Root:   root,
		Force:  a.flag("force"),
		DryRun: a.flag("dry-run"),
	})
	if err != nil {
		fmt.Fprintf(stderr, "cupel: %v\n", err)
		return 1
	}

	report(stdout, res, a.flag("dry-run"))

	if a.flag("dry-run") || a.flag("no-check") {
		return 0
	}
	return runCheck(stdout, stderr, root)
}

func report(w io.Writer, res gen.Result, dry bool) {
	verb := "wrote"
	if dry {
		verb = "would write"
	}
	fmt.Fprintf(w, "component: %s\n", res.ComponentDir)
	for _, p := range res.Written {
		fmt.Fprintf(w, "  %s %s\n", verb, p)
	}
	for _, p := range res.Skipped {
		fmt.Fprintf(w, "  kept existing %s\n", p)
	}
}

// runCheck enforces the goforge architecture on the output by invoking the
// goforge CLI if it is on PATH. A missing goforge is a hint, not a failure.
func runCheck(stdout, stderr io.Writer, root string) int {
	bin, err := exec.LookPath("goforge")
	if err != nil {
		fmt.Fprintln(stderr, "cupel: goforge not on PATH; skipping architecture check (install: go install goforge.dev/goforge@latest)")
		return 0
	}
	fmt.Fprintln(stdout, "cupel: enforcing goforge architecture (goforge check)...")
	cmd := exec.Command(bin, "check")
	cmd.Dir = root
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "cupel: goforge check reported problems: %v\n", err)
		return 1
	}
	return 0
}

// specText resolves the specification from spec:, spec-file:, or stdin.
func specText(a args, stdin io.Reader) (string, error) {
	if v, ok := a.named["spec"]; ok {
		return v, nil
	}
	if path, ok := a.named["spec-file"]; ok {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", fmt.Errorf("no specification given (use spec:, spec-file:, or pipe it on stdin)")
	}
	return string(data), nil
}

// workspace discovers the goforge workspace root and module path. If start is
// empty it begins at the current directory and walks up to workspace.yaml.
func workspace(start string) (root, module string, err error) {
	if start == "" {
		start, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	root, err = findRoot(start)
	if err != nil {
		return "", "", err
	}
	module, err = modulePath(root)
	return root, module, err
}

func findRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, "workspace.yaml")); err == nil && !fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a goforge workspace (no workspace.yaml found above %s)", dir)
		}
		dir = parent
	}
}

// modulePath reads the module line from go.mod without pulling in x/mod.
func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("go.mod: missing module path")
}

// --- minimal poly-style argument parsing (key:value and :flag) ---

type args struct {
	named map[string]string
	flags map[string]bool
}

func (a args) flag(name string) bool { return a.flags[name] }

// backend returns the selected generation backend: "api" (default) or "cli".
func (a args) backend() string {
	if b := a.named["backend"]; b != "" {
		return b
	}
	return "api"
}

// backendClient builds the completer for the selected backend.
func backendClient(a args) (completer, error) {
	switch a.backend() {
	case "api":
		return claude.New(a.named["model"])
	case "cli":
		return claudecode.New(a.named["model"])
	default:
		return nil, fmt.Errorf("unknown backend %q (want api or cli)", a.backend())
	}
}

func parse(argv []string) args {
	a := args{named: map[string]string{}, flags: map[string]bool{}}
	for _, tok := range argv {
		switch {
		case strings.HasPrefix(tok, ":"):
			a.flags[tok[1:]] = true
		case strings.Contains(tok, ":"):
			k, v, _ := strings.Cut(tok, ":")
			a.named[k] = v
		}
	}
	return a
}
