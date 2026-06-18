# Testing Guide

devkit uses two layers of tests that both run under `go test ./...`:

| Layer | Location | What it tests |
|---|---|---|
| Unit tests | `internal/*/..._test.go` | Individual packages in isolation (MemFS, composer, generator, etc.) |
| E2E tests | `e2e_test.go` + `testdata/script/*.txtar` | The full binary end-to-end, every command |

The e2e layer uses **[testscript](https://github.com/rogpeppe/go-internal/tree/master/testscript)** — the same framework the Go toolchain itself uses internally. It runs the `devkit` binary in-process (no separate build step) against isolated temp directories so your real `~/.devkit/` is never touched.

---

## Running tests locally

### Run everything

```
go test ./...
```

### Run only e2e tests

```
go test -v -run TestScript .
```

### Run a single scenario

```
go test -v -run TestScript/generate .
go test -v -run TestScript/reset_yes .
```

The name after `/` is the `.txtar` filename without the extension.

### Run with verbose script tracing

```
go test -v -run TestScript . 2>&1 | less
```

Each step is printed with `>` prefix, stdout/stderr shown inline. Useful when debugging a failing script.

### Run unit tests only

```
go test ./internal/...
```

---

## Writing a new e2e test

1. Create a file in `testdata/script/` named after the scenario:

```
testdata/script/my_scenario.txtar
```

2. Write the script. Every script starts in a fresh temp directory (`$WORK`). Use `env DEVKIT_HOME=$WORK/devkit-home` to isolate from your real `~/.devkit/`.

### Minimal template

```
# one-line description of what this test proves
env DEVKIT_HOME=$WORK/devkit-home

exec devkit init
exec devkit <command> <args>
stdout 'expected output'
```

### Full reference — built-in commands

| Command | What it does |
|---|---|
| `env KEY=VALUE` | Set an environment variable |
| `exec cmd args` | Run a command, assert exit 0 |
| `! exec cmd args` | Run a command, assert non-zero exit |
| `stdout 'pattern'` | Assert stdout matches regex |
| `! stdout 'pattern'` | Assert stdout does NOT match |
| `stderr 'pattern'` | Assert stderr matches regex |
| `! stderr 'pattern'` | Assert stderr does NOT match |
| `exists path` | Assert file or dir exists |
| `! exists path` | Assert file or dir does NOT exist |
| `grep 'pattern' path` | Assert file content matches regex |
| `mkdir path` | Create a directory |
| `cp src dst` | Copy a file |
| `stdin path` | Feed a file as stdin to the next exec |
| `cd path` | Change working directory |
| `# comment` | Comment line (also used as section label in output) |

### Embedding files in the script

Append a `-- filename --` section at the bottom to create files in `$WORK` before the script runs:

```
# generate with a custom workspace config
env DEVKIT_HOME=$WORK/devkit-home
mkdir $WORK/project

exec devkit init
cp $WORK/workspace.yaml $WORK/devkit-home/workspace.yaml
exec devkit generate $WORK/project
stdout '✓ Generated'

-- workspace.yaml --
name: "Test User"
active_context: work
```

The file is placed at `$WORK/workspace.yaml`. Use `cp` to move it where the command expects it.

### Testing stdin (interactive prompts)

Use `stdin` to feed input before an `exec` that reads from stdin:

```
stdin $WORK/answer.txt
exec devkit reset
stdout 'Aborted'

-- answer.txt --
no
```

### Asserting failure

Prefix `exec` with `!` to assert the command exits non-zero:

```
! exec devkit generate $WORK/does-not-exist
stderr 'does not exist'
```

### Platform-specific steps

Wrap steps in guards if behaviour differs by OS:

```
[windows] exec devkit generate C:\project
[!windows] exec devkit generate /tmp/project
```

---

## Troubleshooting

### "stdout does not match"

The test output shows exactly what stdout contained. Check the indented block after `> exec ...` in the verbose output:

```
> exec devkit generate $WORK/project
[stdout]
✓ Generated 5 files in $WORK/project:
  CLAUDE.md, AGENTS.md ...
> stdout 'Generated 6 files'   ← your pattern
FAIL
```

Fix: update the pattern to match the actual output, or fix the command if the output is wrong.

### "exists: $WORK/project/CLAUDE.md: file not found"

The file wasn't created. Check whether the preceding `exec devkit generate` actually succeeded — add `stdout '✓ Generated'` before the `exists` check to confirm.

### "command not found: devkit"

testscript registers the binary via `TestMain`. If you see this, the `TestMain` function in `e2e_test.go` is not being invoked — make sure you're running from the root package (`.`), not from a sub-package.

### Test passes locally but fails on CI

Check whether the test depends on a tool (like `rg` / ripgrep) being installed. devkit's search falls back to a native Go scanner when `rg` is absent, so results are identical — but if you add a new command that shells out, make sure the CI runner has the dependency.

### Inspecting the temp directory after failure

Add `stop` at the point of failure to halt the script and print `$WORK`:

```
exec devkit init
stop   # script pauses here, $WORK path is printed
exec devkit generate $WORK/project
```

Then navigate to that path in a terminal to inspect files.

### Running with the race detector

```
go test -race -run TestScript .
```

---

## Is testscript actively maintained?

Yes. Key facts:

- **Used by the Go toolchain itself** — factored out of `cmd/go`'s own test infrastructure
- **Latest release**: v1.15.0 (May 2026)
- **Last commit**: June 2026
- **CI matrix**: ubuntu-latest, macos-latest, windows-latest on every PR
- **Used by**: Hugo, chezmoi, shfmt, rclone, asdf-vm, garble, and hundreds of other Go CLIs
- **Dependencies**: only `golang.org/x/sys`, `golang.org/x/tools`, `golang.org/x/mod` — all first-party Go packages
- **GitHub**: [rogpeppe/go-internal](https://github.com/rogpeppe/go-internal) — 983 stars, active issue tracker

If the Go toolchain keeps using it (it does), it will keep being maintained.

---

## Current test coverage

| Scenario | File |
|---|---|
| `init` creates scaffold | `init.txtar` |
| `init` fails when dir exists | `init_already_exists.txtar` |
| `generate` writes 5 AI config files | `generate.txtar` |
| `generate --dry-run` previews without writing | `generate_dry_run.txtar` |
| `generate` warns when overwriting changed files | `generate_overwrite.txtar` |
| `generate` fails on missing target dir | `generate_missing_dir.txtar` |
| `generate` fails without prior `init` | `generate_no_init.txtar` |
| `search` finds matches with file:line format | `search.txtar` |
| `search` with no matches | `search_no_match.txtar` |
| `search` is case-insensitive | `search_case_insensitive.txtar` |
| `reset` with `yes` wipes and re-inits | `reset_yes.txtar` |
| `reset` with `no` aborts cleanly | `reset_no.txtar` |
