# Developer Guide

This document is the authoritative reference for developing devkit. It covers the architecture, every package, the data flow, conventions, and how to add new features. Intended for humans and AI assistants alike.

---

## Orientation

devkit is a single Go binary. There is no server, no database, no external API calls. It reads markdown files from `~/.devkit/` and writes markdown files to a target project directory.

```
~/.devkit/  (user data, private)
    ↓
devkit generate <path>
    ↓
~/project/CLAUDE.md, AGENTS.md, GEMINI.md, ...  (AI config files)
```

**Module path:** `github.com/dklinux7/devkit`  
**Go version:** 1.26.4 (see `go.mod`)  
**Entry point:** `main.go` (root package)  
**Binary name:** `devkit`

---

## Repo Layout

```
.
├── main.go                    ← cobra root + run() entry point
├── init.go                    ← devkit init command
├── generate.go                ← devkit generate command
├── reset.go                   ← devkit reset command
├── search.go                  ← devkit search command
├── e2e_test.go                ← testscript TestMain + TestScript
│
├── internal/
│   ├── fs/                    ← filesystem abstraction
│   │   ├── fs.go              ← FS interface
│   │   ├── osfs.go            ← real OS implementation
│   │   └── memfs.go           ← in-memory implementation (tests only)
│   ├── config/                ← workspace.yaml loading + data dir resolution
│   ├── devctx/                ← loads identity, context, donts, lessons
│   ├── composer/              ← concatenates sources into one blob
│   ├── generator/             ← writes output files to target directory
│   └── search/                ← searches markdown files (ripgrep + native fallback)
│
├── templates/                 ← embedded scaffold files (go:embed all:templates)
│   ├── workspace.yaml
│   ├── identity/ai.md
│   ├── identity/engineering.md
│   ├── donts.md
│   ├── tools.md
│   ├── contexts/work.md
│   ├── prompts/               ← reusable prompt templates
│   ├── analysis.tmpl.md       ← source code analysis template
│   └── research.tmpl.md       ← research/ticket workflow template
│
├── testdata/script/           ← e2e test scripts (.txtar)
├── docs/                      ← developer and user documentation
│   ├── developer.md           ← this file
│   ├── testing.md             ← how to run/write/debug tests
│   └── setup/
│       └── github-multi-account.md
│
├── .github/
│   ├── workflows/
│   │   ├── ci.yaml            ← test, lint, vuln, licenses, secrets
│   │   ├── release.yaml       ← goreleaser on git tag v*
│   │   └── deps.yaml          ← weekly govulncheck + trivy
│   └── dependabot.yml
│
├── .goreleaser.yaml
├── Makefile
├── go.mod
└── go.sum
```

---

## Data Flow: `devkit generate`

This is the core command. Every other command is simpler.

```
1. config.DataDir()
       → $DEVKIT_HOME env var, or os.UserHomeDir() + "/.devkit"

2. config.Load(fsys, dataDir)
       → reads workspace.yaml
       → validates name + active_context are set
       → returns *Workspace{Name, ActiveContext}

3. devctx.Load(fsys, dataDir, activeContext, includeLessons)
       → reads identity/*.md  (sorted glob, frontmatter stripped)
       → reads contexts/<active>.md  OR  contexts/<active>/*.md (folder context)
       → reads donts.md
       → optionally reads lessons/*.md
       → returns *Sources{Identity [][]byte, Context []byte, Donts []byte, Lessons [][]byte}

4. composer.Compose(sources, force)
       → joins sections with \n\n in order: identity... → context → donts → lessons...
       → prepends header comment
       → enforces size: >16KB warns, >32KB fails (unless --force)
       → returns *Result{Content string, Size int, Warnings []string}

5. generator.Generate(fsys, targetDir, content, ws, templateDir)
       → writes content to every MarkdownTargets file
       → for each StructuredTargets file, if a .tmpl exists in templateDir, renders and writes
       → tracks which files were overwritten (existed and differed)
       → returns *Result{Written []string, Overwritten []string}
```

**Composition order is intentional:** `donts.md` is last because LLMs weight end-of-prompt content more heavily.

---

## Package Reference

### `internal/fs`

The filesystem abstraction that makes every package unit-testable without touching disk.

```go
type FS interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, perm os.FileMode) error
    ReadDir(path string) ([]os.DirEntry, error)
    Glob(pattern string) ([]string, error)
    Exists(path string) bool
    MkdirAll(path string, perm os.FileMode) error
    Stat(path string) (os.FileInfo, error)
}
```

- **`NewOsFS()`** — wraps the real OS. Used in production code.
- **`NewMemFS()`** — in-memory map. Used in all unit tests. Never touches disk.

**Rule:** All `internal/` packages accept `fs.FS` as a parameter. No package calls `os.ReadFile` directly — they always go through the interface.

---

### `internal/config`

Loads `workspace.yaml` and resolves the data directory.

**`DataDir() (string, error)`**
- Returns `$DEVKIT_HOME` if set.
- Otherwise `os.UserHomeDir() + "/.devkit"`.
- Never uses `os.UserConfigDir()` — it returns wrong paths on macOS and conflates config/data on Linux.

**`Load(fsys fs.FS, dataDir string) (*Workspace, error)`**
- Reads and parses `workspace.yaml`.
- Returns an error if `name` or `active_context` are empty — these are the only two required fields.

**`Workspace` struct:**
```go
type Workspace struct {
    Name          string `yaml:"name"`
    ActiveContext string `yaml:"active_context"`
}
```

Adding a field to `workspace.yaml`: add it to the struct with a yaml tag. It becomes available in generator templates via `{{.Workspace.FieldName}}`. Add validation in `Load` if required.

---

### `internal/devctx`

Loads and prepares source content from `~/.devkit/`.

**`Load(fsys fs.FS, dataDir, activeContext string, includeLessons bool) (*Sources, error)`**

- **Identity:** Glob `identity/*.md`, read each, strip frontmatter, append to `Sources.Identity` slice. Order is filesystem-sorted (alphabetical).
- **Context (flat file):** Try `contexts/<activeContext>.md` first.
- **Context (folder):** If the `.md` file doesn't exist, try `contexts/<activeContext>/` as a directory. Globs `*.md` inside it, concatenates with `\n\n`.
- **Donts:** Reads `donts.md`. Missing file is not an error — `Sources.Donts` stays nil.
- **Lessons:** Only loaded when `includeLessons=true`. Glob `lessons/*.md`.

**`StripFrontmatter(content []byte) []byte`**  
Strips YAML frontmatter (`---\n...\n---\n`) from the start of a file. Frontmatter is used for metadata (date, tags, company) but must not appear in generated AI config.

**`Sources` struct:**
```go
type Sources struct {
    Identity [][]byte  // one entry per identity/*.md file
    Context  []byte    // composed from context file or folder
    Donts    []byte
    Lessons  [][]byte  // nil unless includeLessons=true
}
```

---

### `internal/composer`

Single function: concatenates sources into one content blob.

**`Compose(sources *devctx.Sources, force bool) (*Result, error)`**

Order: identity sections → context → donts → lessons (if present). Each non-empty section joined with `\n\n`. Prepends the `Header` constant.

**Constants:**
```go
Header   = "<!-- Generated by devkit. Do not edit. Contains private context. Source: ~/.devkit/ -->"
WarnSize = 16 * 1024   // 16KB
FailSize = 32 * 1024   // 32KB
```

Size enforcement: over `WarnSize` adds a warning to `Result.Warnings`. Over `FailSize` returns an error unless `force=true`.

---

### `internal/generator`

Writes the composed content to the target project directory.

**Target lists:**

```go
// Always written — same content (the composed markdown blob)
var MarkdownTargets = []string{
    "CLAUDE.md",
    "AGENTS.md",
    "GEMINI.md",
    ".cursorrules",
    ".windsurfrules",
    ".github/copilot-instructions.md",
}

// Written only when a matching .tmpl file exists in templateDir
var StructuredTargets = []string{
    "opencode.json",
    ".claude/settings.json",
}
```

**`Generate(fsys fs.FS, targetDir, content string, ws *config.Workspace, templateDir string) (*Result, error)`**

1. For each `MarkdownTargets` entry: check if the file exists and differs (→ add to `Overwritten`), then call `ensureParentDir` + `WriteFile`.
2. For each `StructuredTargets` entry: check if `<name>.tmpl` exists in `templateDir`. If yes, parse as `text/template`, execute with `TemplateData{Workspace, Content}`, write rendered output.

**`TemplateData` struct:**
```go
type TemplateData struct {
    Workspace *config.Workspace  // .Workspace.Name, .Workspace.ActiveContext
    Content   string             // the full composed markdown blob
}
```

**Adding a new AI tool (markdown output):** Add its config filename to `MarkdownTargets`. No other changes needed.

**Adding a new AI tool (structured/JSON/TOML output):** Add its filename to `StructuredTargets`, then add a `<filename>.tmpl` file in `templates/`. Use Go `text/template` syntax. Access workspace fields via `{{.Workspace.Name}}`.

**`ensureParentDir`** creates the parent directory with `MkdirAll` if it doesn't exist. This is how `.github/copilot-instructions.md` and `.claude/settings.json` work — subdirectories are created automatically.

---

### `internal/search`

Searches all markdown files under `dataDir` for a query string.

**`Search(fsys fs.FS, dataDir, query string) ([]Match, error)`**

1. If `rg` (ripgrep) is on `$PATH`, delegates to `searchRipgrep`.
2. Otherwise falls back to `searchNative`.

Both return `[]Match{File, Line, Text}`. Results are equivalent — the fallback exists so devkit works with no external dependencies.

**`searchRipgrep`:** Runs `rg --line-number --no-heading --glob '*.md' -- <query> <dataDir>`. The `--` separator is mandatory — it prevents the query from being interpreted as a flag (injection protection).

**`searchNative`:** Uses `regexp.Compile("(?i)" + regexp.QuoteMeta(query))` for case-insensitive literal search. Walks `*.md` files via `findMarkdownFiles`.

---

## Commands (root package)

Each command file registers itself in its `init()` via `rootCmd.AddCommand(...)`.

### `main.go`

```go
func main()     // calls os.Exit(run())
func run() int  // calls rootCmd.Execute(), returns 0 or 1
```

`run()` is separate from `main()` so testscript can register it as an in-process binary via `TestMain`. Never call `os.Exit` directly from a command — return an error instead.

### `init.go`

Scaffolds `~/.devkit/` from the embedded `templates/` directory.

- Fails if `dataDir` already exists (tells the user to `reset` instead).
- Uses `fs.WalkDir(TemplateFS, ...)` to copy every embedded file.
- `TemplateFS` is declared in `main.go` with `//go:embed all:templates`. The `all:` prefix is required to capture dotfiles (`.cursorrules`, etc.).

### `generate.go`

Drives the full pipeline: `DataDir → Load config → Load context → Compose → Generate`.

Flags: `--dry-run`, `--include-lessons`, `--force`.

`--dry-run` calls `printDryRun` which shows a 20-line preview of `CLAUDE.md` and the file list — it never writes anything.

### `reset.go`

Prompts for `yes` confirmation via `bufio.Scanner` on `os.Stdin`, then calls `os.RemoveAll(dataDir)` followed by `runInit`.

### `search.go`

Calls `config.DataDir()`, then `search.Search()`, then prints `file:line: text` results.

---

## Embedded Templates

Templates are baked into the binary at compile time:

```go
//go:embed all:templates
var TemplateFS embed.FS
```

The `all:` prefix is required — without it, `go:embed` silently skips files whose names start with `.` (like `.cursorrules`).

`devkit init` copies the embedded tree to `~/.devkit/` verbatim. These files are starter scaffolds — the user edits them after init.

**Structured config templates** (e.g., `templates/opencode.json.tmpl`) are also embedded but are used by the generator at generate-time, not init-time. They are Go `text/template` files rendered with `TemplateData`.

**Adding a new scaffold file:** Drop it into `templates/`. It will be copied on `devkit init` automatically.

---

## Adding a New Command

1. Create `<command>.go` in the root package (`package main`).
2. Declare a `var <command>Cmd = &cobra.Command{...}`.
3. In `init()`, call `rootCmd.AddCommand(<command>Cmd)`.
4. Add at least one `.txtar` e2e test in `testdata/script/`.

Pattern — every command follows the same shape:
```go
var myCmd = &cobra.Command{
    Use:   "mycommand <arg>",
    Short: "One-line description",
    RunE:  runMyCommand,
}

func init() {
    rootCmd.AddCommand(myCmd)
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    dataDir, err := config.DataDir()
    if err != nil {
        return err
    }
    fsys := dkfs.NewOsFS()
    // ... do work ...
    _, _ = fmt.Fprintf(cmd.OutOrStdout(), "result\n")
    return nil
}
```

**Stdout vs stderr:** User-facing output goes to `cmd.OutOrStdout()`. Warnings go to `cmd.ErrOrStderr()`. This keeps `devkit generate | jq` pipeable and makes testscript assertions reliable (`stdout` vs `stderr`).

**Error returns:** Return errors up the stack. Never call `os.Exit` in a command. Cobra handles printing and exit code via `RunE`.

**Unchecked returns:** All `fmt.Fprintf`/`fmt.Fprintln` return values must be assigned to `_, _`. golangci-lint's errcheck linter enforces this. Writing to stdout/stderr can fail (piped output, closed fd) and the linter requires acknowledgement.

---

## Adding a New Field to workspace.yaml

1. Add the field to `Workspace` struct in `internal/config/config.go` with a yaml tag.
2. Add validation in `config.Load()` if the field is required.
3. Update `templates/workspace.yaml` so `devkit init` scaffolds it.
4. If it affects generation, pass it through via `TemplateData` in `generator.go`.
5. Update `internal/config/config_test.go` with a test case.

---

## CI Pipeline

All checks run on every push and PR to `main`.

| Job | Command | What it checks |
|---|---|---|
| `test` | `go test ./...` | Unit tests + e2e testscript suite |
| `test` | `CGO_ENABLED=0 go build .` | Binary compiles cleanly |
| `lint` | `golangci-lint run` v2.12.2 | Style, errcheck, unused vars, etc. |
| `vuln` | `govulncheck -show verbose ./...` | Known CVEs in deps |
| `licenses` | `go-licenses check` | Only MIT/Apache/BSD deps allowed |
| `secrets` | `trufflehog` | No credentials in git history |

**Weekly jobs** (`deps.yaml`): `govulncheck` + `trivy` filesystem scan.

**Release** (`release.yaml`): Triggered on `git push origin vX.Y.Z`. goreleaser builds 6 platform binaries (linux/darwin/windows × amd64/arm64), creates GitHub release with checksums.

### Running CI checks locally

```bash
make test        # go test ./...
make lint        # golangci-lint run
make vuln        # govulncheck ./...
make licenses    # go-licenses check
make check       # all of the above
make build       # compile binary
make install     # go install (puts devkit on $PATH)
```

---

## Testing

Full detail in `docs/testing.md`. Summary:

- **Unit tests:** `internal/*/..._test.go`. Use `fs.NewMemFS()` — never touch real disk.
- **E2e tests:** `testdata/script/*.txtar`. Each script is an isolated scenario. `env DEVKIT_HOME=$WORK/...` keeps tests isolated from real `~/.devkit/`.
- **Run all:** `go test ./...`
- **Run one e2e:** `go test -v -run TestScript/generate .`

The testscript framework runs `devkit` in-process — no separate build step. `TestMain` in `e2e_test.go` registers `run()` as the binary entrypoint.

---

## Key Invariants

These are non-obvious constraints that must be preserved:

1. **`run()` not `os.Exit()`** — `main.go` uses `os.Exit(run())`. Commands return errors. This is required for testscript in-process execution.

2. **`all:templates` embed prefix** — Without `all:`, dotfiles are silently skipped. Always use `//go:embed all:templates`.

3. **`--` before ripgrep query** — `exec.Command("rg", ..., "--", query, dataDir)`. Without `--`, a query starting with `-` is interpreted as a flag.

4. **Frontmatter stripped before composition** — `devctx.Load` strips YAML frontmatter from every source file. Frontmatter must never appear in generated AI config files.

5. **`donts.md` is last** — Composition order: identity → context → donts → lessons. Donts last because LLMs weight end-of-prompt higher.

6. **`cmd.OutOrStdout()` not `os.Stdout`** — All command output uses cobra's writer. testscript captures it; `os.Stdout` bypasses capture.

7. **No `os.UserConfigDir()`** — Returns `~/Library/Application Support` on macOS. Always use `os.UserHomeDir() + "/.devkit"` or `$DEVKIT_HOME`.

8. **`ensureParentDir` before WriteFile** — Generator calls this for every target. It's what creates `.github/` for `copilot-instructions.md` and `.claude/` for `settings.json`.

---

## Dependency Policy

- Only `MIT`, `Apache-2.0`, `BSD-2-Clause`, `BSD-3-Clause`, `ISC` licenses allowed. Enforced by `go-licenses` in CI.
- No CGO. Binary is always `CGO_ENABLED=0`.
- No third-party test frameworks — the standard library `testing` package + `testscript` (which depends only on `golang.org/x/*`).
- No `replace` directives in `go.mod`.
- To add a dependency: `go get <module>`, run `make check`, confirm license with `go-licenses csv ./...`.

---

## Milestone 2.5 — What's Next

The design is fully specified in `devkit-workspace-design.md`. Implementation order recommendation:

| Priority | Feature | Why |
|---|---|---|
| 1 | `extra_targets` in workspace.yaml | Enables new tool support without a release |
| 2 | `projects.txt` registry + `--all` | Makes multi-project updates a one-liner |
| 3 | `devkit context ls` | Quality of life, small scope |
| 4 | `devkit doctor` | Useful once `--all` exists |
| 5 | `devkit search --interactive` | Requires go-fuzzyfinder dep |
| 6 | `devkit sync` | Requires user has a private git repo for `~/.devkit/` |
| 7 | `.mcp.json` generation | Requires YAML frontmatter parsing in devctx |

New internal packages needed for 2.5:
- `internal/registry` — read/write `~/.devkit/projects.txt`
- `internal/mcp` — parse `mcp_servers` frontmatter, render `.mcp.json`

New command files needed:
- `context.go` — `devkit context ls`
- `doctor.go` — `devkit doctor`
- `sync.go` — `devkit sync`
