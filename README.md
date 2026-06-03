# cupel

`cupel` turns an English-language specification into Go tests by calling Claude,
and writes them into a **goforge** component. It produces three styles of test
for the same spec:

1. **Example tests** — concrete, readable cases (plus testable `Example` funcs).
2. **Parameterized / data-driven tests** — table-driven `t.Run` cases.
3. **Property-based tests** — invariants checked with the standard-library
   `testing/quick`.

The output is not a loose file: cupel **enforces the goforge architecture on
it**. The model reply is de-fenced, parsed, required to declare the external
`<iface>_test` package, gofmt-formatted, written into `components/<name>/`
alongside a valid interface/`internal`/`component.yaml` scaffold, and then
validated by running `goforge check`.

> A cupel is the porous vessel an assayer uses to refine raw metal — here it
> refines raw prose into tests.

## Built with goforge

cupel is itself a goforge workspace, so its own structure obeys the same
worker-type discipline it enforces on its output:

| brick          | kind      | worker type |
|----------------|-----------|-------------|
| `spec`         | component | `deterministic/pure` — parse & validate the spec |
| `prompt`       | component | `deterministic/pure` — build the Claude messages |
| `architecture` | component | `deterministic/pure` — validate/format output, render the brick |
| `claude`       | component | `non_deterministic/[input_effect, output_effect]` — Anthropic Messages API backend |
| `claudecode`   | component | `non_deterministic/[input_effect, output_effect]` — local Claude Code CLI backend |
| `gen`          | component | `non_deterministic/[input_effect, output_effect]` — orchestrate & write files |
| `cli`          | base      | the effect boundary (effective: `non_deterministic/[input_effect, output_effect]`) |

`goforge classify` shows the `cli` base is *declared* pure but *effectively*
effectful — effects propagate up from `claude`/`gen` through the dependency
closure, exactly as goforge intends.

## Install

```sh
go install goforge.dev/cupel/projects/cupel@latest   # the cupel CLI
go install goforge.dev/goforge@latest                 # needed for the check step
```

## Use

Run from inside any goforge workspace. `name:` is the target component.

```sh
# api backend (default) — calls the Anthropic API
export ANTHROPIC_API_KEY=sk-...
cupel name:user spec:"A user has a non-empty name and an email containing '@'.
Two users are equal iff their emails match (case-insensitive)."

# cli backend — drives the local Claude Code `claude` CLI instead (no API key)
cupel name:user backend:cli spec-file:user.md

# from stdin
cat order.md | cupel name:order
```

### Backends

| `backend:` | how it generates | needs |
|------------|------------------|-------|
| `api` (default) | Anthropic Messages API over HTTP, system prompt cached | `ANTHROPIC_API_KEY` |
| `cli` | shells out to `claude -p ... --append-system-prompt ...` in headless print mode | Claude Code installed (`claude` on PATH, or `CUPEL_CLAUDE_BIN`) |

Both backends feed the identical prompt and the same output enforcement, so the
generated tests and the trailing `goforge check` are the same either way.

Arguments (poly-style `key:value` / `:flag`):

| arg              | meaning |
|------------------|---------|
| `name:NAME`      | target goforge component (required) |
| `iface:PKG`      | public package of that component (default `NAME`) |
| `spec:"TEXT"`    | inline specification |
| `spec-file:PATH` | read the spec from a file |
| `backend:WHICH`  | `api` (default) or `cli` (drive the Claude Code CLI) |
| `model:ID`       | Claude model (`$CUPEL_MODEL`; api default `claude-opus-4-8`, cli default its own) |
| `root:DIR`       | workspace root (default: discovered from cwd) |
| `:force`         | overwrite the interface/internal/yaml scaffold |
| `:dry-run`       | render everything, write nothing |
| `:no-check`      | skip the trailing `goforge check` |

cupel writes `components/<name>/<name>_test.go` (always) and, when the brick
does not yet exist, a placeholder interface, `internal/core.go`, and
`component.yaml` so the result is a complete, check-valid goforge component.

## Develop

```sh
go test ./...           # unit tests for every brick (offline)
goforge check           # structure, boundary, worker-type, project deps
goforge classify        # declared vs. effective worker types
```
