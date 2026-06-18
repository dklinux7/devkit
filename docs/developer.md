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
**Go version:** 1.26 (pinned in `mise.toml`; go.mod for toolchain constraints)  
**Entry point:** `main.go` (root package)  
**Binary name:** `devkit`

---

## Repo Layout

```
.
├── main.go                    ← cobra root, verbose flag, debugf(), run() entry point
├── init.go                    ← devkit init command
├── generate.go                ← devkit generate command
├── compose_helper.go          ← resolveComposed() shared helper + composedContext struct
├── status.go                  ← devkit status command
├── doctor.go                  ← devkit doctor command
├── diff.go                    ← devkit diff command
├── lint.go                    ← devkit lint command
├── context.go                 ← devkit context ls command
├── sync.go                    ← devkit sync command
├── version.go                 ← devkit version command
├── untrack.go                 ← devkit untrack command
├── reset.go                   ← devkit reset command
├── search.go                  ← devkit search command
├── e2e_test.go                ← testscript TestMain + TestScript
│
├── internal/
│   ├── fs/                    ← filesystem abstraction
│   │   ├── fs.go              ← FS interface
│   │   ├── osfs.go            ← real OS implementation (atomic write via rename)
│   │   └── memfs.go           ← in-memory implementation (tests only)
│   ├── config/                ← workspace.yaml loading + data dir resolution
│   ├── devctx/                ← loads identity, context, donts, lessons; parses MCP servers
│   │   ├── devctx.go
│   │   └── mcpservers.go
│   ├── composer/              ← concatenates sources into one blob
│   ├── generator/             ← writes output files to target directory
│   ├── registry/              ← reads/writes ~/.devkit/projects.txt
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
│       ├── github-multi-account.md
│       └── new-machine.md
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
├── mise.toml                  ← pins Go and golangci-lint versions for local dev
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
       → returns *Workspace{Name, ActiveContext, ExtraTargets, BackupRecipient}

3. devctx.Load(fsys, dataDir, activeContext, includeLessons)
       → validates activeContext: rejects ".." and absolute paths
       → reads identity/*.md  (sorted glob, frontmatter stripped)
       → reads contexts/<active>.md  OR  contexts/<active>/*.md (folder context)
       → reads donts.md
       → optionally reads lessons/*.md
       → returns *Sources{Identity [][]byte, Context []byte, RawContext []byte, Donts []byte, Lessons [][]byte}
       (RawContext is the raw bytes of the context file, used for MCP server extraction)

4. composer.Compose(sources, force)
       → joins sections with \n\n in order: identity... → context → donts → lessons...
       → prepends header comment
       → enforces size: >16KB warns, >32KB fails (unless --force)
       → returns *Result{Content string, Size int, Warnings []string}

5. generator.Generate(fsys, targetDir, content, ws, templateDir)
       → writes content to every MarkdownTargets file (including ws.ExtraTargets)
       → writes MDCFrontmatter + content to every MDCTargets file
       → for each StructuredTargets file, if a .tmpl exists in templateDir, renders and writes
       → path traversal check: filepath.Clean(path) must have cleanTarget as prefix
       → tracks which files were overwritten (existed and differed)
       → returns *Result{Written []string, Overwritten []string}

6. buildMCPJSON(sources)
       → calls devctx.ParseMCPServers(sources.RawContext) to extract mcp_servers from context frontmatter
       → if any servers found, marshals to JSON and writes .mcp.json in targetDir

7. registry.Append(fsys, dataDir, targetDir)
       → appends targetDir to ~/.devkit/projects.txt if not already present

8. writeSkillsFile(cmd, fsys, content)
       → if ~/.claude/skills/ exists, writes content to ~/.claude/skills/devkit-context.md

9. checkGitignore(cmd, targetDir)
       → if targetDir is a git repo, warns if CLAUDE.md/AGENTS.md are not in .gitignore
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

- **`NewOsFS()`** — wraps the real OS. `WriteFile` uses write-to-temp + rename for atomic writes. Used in production code.
- **`NewMemFS()`** — in-memory map. Supports `ModTimes` for mtime-based tests. Used in all unit tests. Never touches disk.

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
    Name            string   `yaml:"name"`
    ActiveContext   string   `yaml:"active_context"`
    ExtraTargets    []string `yaml:"extra_targets"`
    BackupRecipient string   `yaml:"backup_recipient"`
}
```

`ExtraTargets` is a list of additional filenames to write during generate (same content as `MarkdownTargets`). `BackupRecipient` is reserved for Milestone 3 archive/backup.

Adding a field to `workspace.yaml`: add it to the struct with a yaml tag. It becomes available in generator templates via `{{.Workspace.FieldName}}`. Add validation in `Load` if required.

---

### `internal/devctx`

Loads and prepares source content from `~/.devkit/`.

**`Load(fsys fs.FS, dataDir, activeContext string, includeLessons bool) (*Sources, error)`**

- **Validation:** Rejects `activeContext` containing `..` or that is absolute. Also verifies the resolved context path does not escape `contexts/` via `filepath.Clean` + `strings.HasPrefix`.
- **Identity:** Glob `identity/*.md`, read each, strip frontmatter, append to `Sources.Identity` slice. Order is filesystem-sorted (alphabetical).
- **Context (flat file):** Try `contexts/<activeContext>.md` first. Sets both `RawContext` (raw bytes) and `Context` (frontmatter stripped).
- **Context (folder):** If the `.md` file doesn't exist, try `contexts/<activeContext>/` as a directory. Globs `*.md` inside it, concatenates stripped content with `\n\n`. `RawContext` is not set for folder contexts.
- **Donts:** Reads `donts.md`. Missing file is not an error — `Sources.Donts` stays nil.
- **Lessons:** Only loaded when `includeLessons=true`. Glob `lessons/*.md`.

**`StripFrontmatter(content []byte) []byte`**  
Strips YAML frontmatter (`---\n...\n---\n`) from the start of a file. Frontmatter is used for metadata but must not appear in generated AI config.

**`Sources` struct:**
```go
type Sources struct {
    Identity   [][]byte  // one entry per identity/*.md file
    Context    []byte    // composed from context file or folder (frontmatter stripped)
    RawContext []byte    // raw bytes of the context file (for MCP extraction)
    Donts      []byte
    Lessons    [][]byte  // nil unless includeLessons=true
}
```

**`ParseMCPServers(content []byte) map[string]MCPServer`** (in `mcpservers.go`)

Extracts the `mcp_servers` key from YAML frontmatter in the raw context file. Returns a map of server name → `MCPServer{Command, Args, Env}`. Used by `buildMCPJSON` in `generate.go` to produce `.mcp.json`.

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
// Always written — same composed markdown content
var MarkdownTargets = []string{
    "CLAUDE.md",
    "AGENTS.md",
    "GEMINI.md",
    "CONVENTIONS.md",
    ".cursorrules",
    ".windsurfrules",
    ".github/copilot-instructions.md",
    ".claude/rules/devkit-context.md",
    ".kiro/steering/identity.md",
}

// Written with MDCFrontmatter prepended
var MDCTargets = []string{
    ".cursor/rules/devkit-context.mdc",
}

const MDCFrontmatter = "---\ndescription: devkit identity and context\nalwaysApply: true\n---\n\n"

// Written only when a matching .tmpl file exists in templateDir
var StructuredTargets = []string{
    "opencode.json",
    ".claude/settings.json",
}
```

`ws.ExtraTargets` from `workspace.yaml` are appended to the markdown targets list at generate time.

**`Generate(fsys fs.FS, targetDir, content string, ws *config.Workspace, templateDir string) (*Result, error)`**

1. For each markdown target (MarkdownTargets + ExtraTargets): validate path does not escape `targetDir` using `filepath.Clean` + `strings.HasPrefix`, then check if file exists and differs (→ add to `Overwritten`), call `ensureParentDir` + `WriteFile`.
2. For each MDC target: same path validation, write `MDCFrontmatter + content`.
3. For each StructuredTargets entry: check if `<name>.tmpl` exists in `templateDir`. If yes, parse as `text/template`, execute with `TemplateData{Workspace, Content}`, write rendered output.

**Path traversal protection:** Every target path is resolved with `filepath.Clean` and checked to have `cleanTarget + os.PathSeparator` as a prefix. This prevents `extra_targets` or any other name from escaping the project directory.

**`TemplateData` struct:**
```go
type TemplateData struct {
    Workspace *config.Workspace  // .Workspace.Name, .Workspace.ActiveContext, etc.
    Content   string             // the full composed markdown blob
}
```

**Adding a new AI tool (markdown output):** Add its config filename to `MarkdownTargets`. No other changes needed.

**Adding a new AI tool (MDC output):** Add its filename to `MDCTargets`. The `MDCFrontmatter` header is prepended automatically.

**Adding a new AI tool (structured/JSON/TOML output):** Add its filename to `StructuredTargets`, then add a `<filename>.tmpl` file in `templates/`. Use Go `text/template` syntax.

**`ensureParentDir`** creates the parent directory with `MkdirAll` if it doesn't exist. This is how `.github/`, `.claude/`, `.kiro/`, and `.cursor/rules/` are created automatically.

---

### `internal/registry`

Manages `~/.devkit/projects.txt` — one absolute path per line.

**`Append(fsys fs.FS, dataDir, targetPath string) error`**  
Adds `targetPath` to `projects.txt` if not already present. Creates the file if it doesn't exist.

**`ReadAll(fsys fs.FS, dataDir string) ([]string, error)`**  
Returns all paths. Missing file returns empty slice (not an error). If the file exists but cannot be read (e.g., permission denied), returns an error.

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
var verbose bool

func main()       // calls os.Exit(run())
func run() int    // calls rootCmd.Execute(), returns 0 or 1
func debugf(format string, args ...any)  // prints to os.Stderr when --verbose/-v is set
```

`run()` is separate from `main()` so testscript can register it as an in-process binary via `TestMain`. Never call `os.Exit` directly from a command — return an error instead.

`--verbose`/`-v` is a persistent flag on the root command. `debugf` writes to `os.Stderr` directly (not `cmd.ErrOrStderr`) because it's a global, not scoped to a cobra command.

### `init.go`

Scaffolds `~/.devkit/` from the embedded `templates/` directory.

- Fails if `workspace.yaml` already exists in `dataDir` (tells the user to `reset` instead).
- Uses `fs.WalkDir(TemplateFS, ...)` to copy every embedded file.
- `TemplateFS` is declared in `main.go` with `//go:embed all:templates`. The `all:` prefix is required to capture dotfiles (`.cursorrules`, etc.).
- Creates `~/.devkit/` with mode `0700`; files are written with mode `0600`.

### `compose_helper.go`

Not a command — shared infrastructure used by `status`, `doctor`, and `diff`.

**`composedContext` struct:**
```go
type composedContext struct {
    fsys    dkfs.FS
    dataDir string
    ws      *config.Workspace
    result  *composer.Result
}
```

**`resolveComposed(includeLessons bool, force bool) (*composedContext, error)`**  
Runs steps 1–4 of the data flow (DataDir → Load config → Load context → Compose) and returns the result. Calls `debugf` at each step so `--verbose` traces the pipeline.

### `generate.go`

Drives the full pipeline: `DataDir → Load config → Load context → Compose → Generate`.

**Flags:** `--dry-run`, `--include-lessons`, `--force`, `--all`, `--quiet`.

- `--dry-run` calls `printDryRun` which shows a 20-line preview of `CLAUDE.md` and the file list — it never writes anything. Cannot be combined with `--all`.
- `--all` regenerates every path in `projects.txt`. Skips directories that no longer exist (with a warning).
- `--quiet` suppresses the success output (useful in scripts).

After generating, `generate.go` also:
- Calls `buildMCPJSON` to write `.mcp.json` if the context frontmatter declares `mcp_servers`.
- Calls `registry.Append` to record the path in `projects.txt`.
- Calls `writeSkillsFile` to mirror content to `~/.claude/skills/devkit-context.md` if that directory exists.
- Calls `checkGitignore` to warn if the target is a git repo without the generated files in `.gitignore`.

### `reset.go`

Re-scaffolds `~/.devkit/`. Default (non-destructive): only adds files that don't already exist. `--hard`: deletes all of `~/.devkit/` and re-initializes.

Both modes prompt for `yes` confirmation via `bufio.Scanner` on `cmd.InOrStdin()` (not `os.Stdin` — this makes the confirmation testable in testscript).

### `status.go`

`devkit status` — shows sync state for all tracked project paths.

Uses `resolveComposed` to get the current composed content, then compares it byte-for-byte to `CLAUDE.md` in each tracked path.

States: `✓ in-sync`, `✗ stale`, `⚠ not generated`, `? missing`.  
Summary line: `N in-sync, N stale, N missing`.

### `doctor.go`

`devkit doctor` — mtime-based stale check. Distinct from `status`: instead of comparing file content, it checks whether source files are newer than the generated `CLAUDE.md`.

Uses `resolveComposed` to load config, then:
1. Finds the latest mtime across `identity/*.md`, the active context file(s), and `donts.md`.
2. For each tracked project, compares that mtime to `CLAUDE.md`'s mtime.

States: `✓ up-to-date`, `✗ stale`, `⚠ not generated`, `⚠ unreadable`, `? missing`.

### `diff.go`

`devkit diff <path>` — shows what `devkit generate` would change for a given path.

Checks MarkdownTargets, ExtraTargets, MDCTargets (with MDCFrontmatter prepended), and StructuredTargets. For each file: compares current disk content to what would be written.

**Flags:** `--check` — exits with code 1 if any files would change. Useful in CI to enforce that generated files are committed.

### `lint.go`

`devkit lint` — validates `~/.devkit/` source files without generating anything.

Checks:
1. `workspace.yaml` is valid and has required fields.
2. Active context exists (as a file or directory).
3. `identity/` has at least one `.md` file.
4. Each `.md` file: warns if over 8KB, warns if it contains unexpanded `${VAR}` patterns.
5. Estimated composed size: warns over 16KB, errors over 32KB.

Exits non-zero if there are any errors (warnings alone do not fail).

### `context.go`

`devkit context ls` — lists all contexts in `~/.devkit/contexts/`.

For each entry: shows name, size, and last-modified date. Folder contexts aggregate size across all `.md` files inside. The active context is marked with `*active*`.

### `sync.go`

`devkit sync` — runs `git pull --rebase` then `git push` on `~/.devkit/`.

Requires `~/.devkit/` to be a git repository. Errors with setup instructions if `.git` is not present.

### `version.go`

`devkit version` — prints `devkit <version>`. The `version` variable is set at build time by goreleaser via `-ldflags "-X main.version=vX.Y.Z"`. Defaults to `"dev"` in local builds.

### `untrack.go`

`devkit untrack <path>` — removes a project path from `projects.txt`.

Resolves the argument to an absolute path, reads `projects.txt`, filters out the matching line, and rewrites the file. Errors if the path is not tracked.

### `search.go`

`devkit search <query>` — searches all markdown files under `~/.devkit/`.

Calls `config.DataDir()`, then `search.Search()`, then prints `file:line: text` results.

**Flags:** `--interactive`. When set, pipes results to `fzf` if on PATH, otherwise falls back to `go-fuzzyfinder`.

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
| `test` | `go test -coverprofile=coverage.out ./...` | Unit tests + e2e testscript suite |
| `test` | `CGO_ENABLED=0 go build .` | Binary compiles cleanly |
| `lint` | `golangci-lint run` v2.12.2 | Style, errcheck, unused vars, etc. |
| `vuln` | `govulncheck -show verbose ./...` | Known CVEs in deps |
| `licenses` | `go-licenses check` | Only MIT/Apache/BSD deps allowed |
| `secrets` | `trufflehog` | No credentials in git history |

The `test` job runs on a matrix of `ubuntu-latest`, `macos-latest`, and `windows-latest`. Coverage is uploaded as a GitHub Actions artifact (from ubuntu only, 7-day retention).

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

`mise.toml` pins Go (`1.26`) and golangci-lint (`2.12.2`) for local dev consistency. Run `mise install` once to get the same versions CI uses.

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

1. **`run()` not `os.Exit()`** — `main.go` uses `os.Exit(run())`. Commands return errors. Required for testscript in-process execution.

2. **`all:templates` embed prefix** — Without `all:`, dotfiles are silently skipped. Always use `//go:embed all:templates`.

3. **`--` before ripgrep query** — `exec.Command("rg", ..., "--", query, dataDir)`. Without `--`, a query starting with `-` is interpreted as a flag.

4. **Frontmatter stripped before composition** — `devctx.Load` strips YAML frontmatter from every source file. Frontmatter must never appear in generated AI config files.

5. **`donts.md` is last** — Composition order: identity → context → donts → lessons. Donts last because LLMs weight end-of-prompt higher.

6. **`cmd.OutOrStdout()` not `os.Stdout`** — All command output uses cobra's writer. testscript captures it; `os.Stdout` bypasses capture.

7. **No `os.UserConfigDir()`** — Returns `~/Library/Application Support` on macOS. Always use `os.UserHomeDir() + "/.devkit"` or `$DEVKIT_HOME`.

8. **`ensureParentDir` before WriteFile** — Generator calls this for every target. It creates `.github/`, `.claude/`, `.kiro/`, and `.cursor/rules/` automatically.

9. **Path traversal protection** — `generator.Generate` validates every target path with `filepath.Clean` + `strings.HasPrefix` before writing. This applies to `MarkdownTargets`, `ExtraTargets`, `MDCTargets`, and `StructuredTargets`. `devctx.Load` applies the same check to `active_context`.

10. **File permissions** — `~/.devkit/` is created with mode `0700`. Files inside it are written with mode `0600`. Generated project files (CLAUDE.md, etc.) use `0644`.

11. **`cmd.InOrStdin()` not `os.Stdin`** — `reset.go` reads confirmation input via `cmd.InOrStdin()`. This allows testscript to inject stdin without touching the real terminal.

12. **`debugf` uses `os.Stderr` directly** — The `debugf` helper is global (not scoped to a cobra command), so it writes to `os.Stderr` rather than `cmd.ErrOrStderr()`. This is intentional — it cannot be injected by tests.

---

## Dependency Policy

- Only `MIT`, `Apache-2.0`, `BSD-2-Clause`, `BSD-3-Clause`, `ISC` licenses allowed. Enforced by `go-licenses` in CI.
- No CGO. Binary is always `CGO_ENABLED=0`.
- No third-party test frameworks — the standard library `testing` package + `testscript` (which depends only on `golang.org/x/*`).
- No `replace` directives in `go.mod`.
- To add a dependency: `go get <module>`, run `make check`, confirm license with `go-licenses csv ./...`.

---

## What's Next

Milestones 2.5 and 2.75 are complete. The next planned milestones are:

- **Milestone 3 (archive/backup):** Age-encrypted backup to a configurable recipient. The `backup_recipient` field in `workspace.yaml` is already reserved for this. See `devkit-workspace-design.md` for the full spec.
- **v2 (MCP server mode):** devkit running as a local MCP server so AI tools can query context directly without file generation. Also specified in `devkit-workspace-design.md`.
