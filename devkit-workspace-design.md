# devkit — Personal Dev Workspace Design Document

**Status:** ACTIVE — Milestone 2.5 substantially complete, post-audit hardening in progress.
**Date started:** 2026-06-18
**Design locked:** 2026-06-18
**Last updated:** 2026-06-18 (engineering audit — security, architecture, code quality, AI workflow fixes)
**Goal:** Make you + AI maximally productive. Not portable, not generic, not elegant — productive.

---

## Problem Statement

Current workspace is tightly coupled to one employer. Need a personal workspace that:
- Works with any AI coding tool (Claude Code, OpenCode, Cursor, Copilot, Windsurf, Gemini CLI, Aider, Continue.dev, Zed, and whatever comes next)
- Works at any company (swap context, not rebuild)
- Works on macOS, Linux, Windows
- Philosophy: install binary → `devkit init` → `devkit generate ~/project` → done

### AI Tool Landscape (as of 2026)

15+ active AI coding tools exist. Two standards are converging:
- **AGENTS.md** — Linux Foundation / Agentic AI Foundation. 60k+ projects. Supported by every major tool. Plain markdown, no schema required. This is the `.editorconfig` of AI tools.
- **MCP (Model Context Protocol)** — Anthropic-originated, now broadly adopted. Standard JSON schema for connecting AI tools to live data sources and APIs. Every major tool supports it.

devkit's role: generate static identity + context files for all tools from a single source of truth. MCP servers (Jira, Slack, GitHub, Postgres) are the AI tool's job, not devkit's.

---

## Core Principles

> 1. Canonical format is markdown. All AI tools read it.
> 2. Zero known vulnerabilities. Clean deps at all times.
> 3. Public repo = the tool. User data = private, separate.
> 4. workspace.yaml = minimal fields. Everything else in markdown.
> 5. Hard output limit: >16KB warns, >32KB fails (unless --force).
> 6. If it doesn't help solve a production problem, understand a codebase, or use AI better within 30 days — don't build it in v1.
> 7. devkit provides static context. AI tools provide dynamic integrations. Never cross this boundary.

---

## Two-Location Architecture

```
github.com/<user>/devkit              ← PUBLIC REPO (the tool only)
├── cmd/devkit/                       ← Go CLI source
├── internal/                         ← CLI internals
├── templates/                        ← scaffold files (identity, context, prompts, tools)
│   ├── identity/
│   │   ├── engineering.tmpl.md
│   │   └── ai.tmpl.md
│   ├── context.tmpl.md
│   ├── analysis.tmpl.md             ← language-agnostic source code analysis
│   ├── research.tmpl.md             ← problem → research → solution → close
│   └── tools.tmpl.md               ← tools reference starter
├── .github/
│   └── workflows/
│       ├── ci.yaml                  ← lint, test, govulncheck, trivy, go-licenses
│       ├── release.yaml             ← goreleaser on tag
│       └── deps.yaml                ← weekly dep updates + scan
├── .github/dependabot.yml
├── go.mod / go.sum
├── Makefile
└── README.md                         ← includes "no license" notice

~/.devkit/                            ← PRIVATE (created by `devkit init`)
├── workspace.yaml                   ← 2 fields: name, active_context
├── donts.md                         ← "never do this" constraints
├── tools.md                         ← personal tools reference
├── identity/
│   ├── engineering.md               ← coding style, architecture, workflow, git
│   └── ai.md                        ← AI behavior, proficiency, learning vs doing
├── contexts/
│   ├── work.md                      ← flat file (promote to folder/ if it grows too large)
│   └── personal.md
├── prompts/                         ← reusable prompt templates (portable across AI tools)
│   ├── code-review.md
│   ├── root-cause.md
│   ├── architecture-analysis.md
│   └── source-analysis.md
├── findings/                        ← active research (ticket-linked, uses research template)
├── analyzed/                        ← source code analyses (uses analysis template)
├── lessons/                         ← archived knowledge (sanitized, tagged)
│   └── <company>-<years>.md
└── archive/                         ← compressed originals (never deleted)
    └── <company>-<date>.tar.gz
```

### Data directory locations

| OS | Default path | Env override |
|----|-------------|--------------|
| macOS | `~/.devkit/` | `$DEVKIT_HOME` |
| Linux | `~/.devkit/` | `$DEVKIT_HOME` |
| Windows | `%USERPROFILE%\.devkit\` | `$DEVKIT_HOME` |

**Implementation:** Use `os.UserHomeDir()` + `".devkit"`. `$DEVKIT_HOME` wins unconditionally on all platforms. Never use `os.UserConfigDir()` (returns wrong path on macOS and conflates config/data on Linux). Never hardcode path separators.

---

## What's Safe in Public Repo

- Go source code (the tool)
- Scaffold templates (generic placeholder content)
- CI workflows, Makefile
- README (includes "no license" notice)

### License Strategy

No LICENSE file. Default copyright = all rights reserved. README states:
```
This project is not licensed for use, modification, or distribution.
Source code is publicly visible for transparency and reference only.
All rights reserved.
```

### What Stays Private (~/.devkit/)

- Everything. workspace.yaml, identity/, contexts/, findings/, prompts/, lessons/ — all private.

---

## workspace.yaml — Schema

```yaml
name: "Your Name"
active_context: work
extra_targets:        # optional — additional output files beyond the defaults
  - .roo/system-prompt.md
  - .amp/context.md
backup_recipient: ""  # optional — age public key for devkit backup (Milestone 3)
```

Core fields:
- `name` — used in templates
- `active_context` — which context file/folder to load from `contexts/`

Optional fields:
- `extra_targets` — list of additional output file paths appended to the default markdown target list. Enables new AI tool support without a devkit release. Paths are relative to the target project directory.
- `backup_recipient` — age public key (`age1...`) used by `devkit backup`. Leave empty until Milestone 3 is built.

**What's NOT here:** ai_tools (generate everything — disk is free), integrations (belong in context prose), role/languages (in identity/ai.md).

**Switching context:** Edit `active_context`, run `devkit generate --all`. One line edit, all projects updated.

### Default markdown targets (always generated)

```
CLAUDE.md
AGENTS.md
GEMINI.md
CONVENTIONS.md                         ← Aider compatibility
.cursorrules                           ← deprecated but kept for backward compat
.windsurfrules
.github/copilot-instructions.md
.claude/rules/devkit-context.md        ← Claude Code scoped rules (preferred over CLAUDE.md)
.kiro/steering/identity.md             ← AWS Kiro
```

### Default structured targets (template-rendered)

```
opencode.json
.mcp.json                  ← project-level MCP server config (stub for devkit serve)
.claude/settings.json
```

### AI tool config file reference

| Tool | Files read | Format | devkit coverage |
|------|-----------|--------|----------------|
| Claude Code | `CLAUDE.md`, `.claude/rules/*.md`, `AGENTS.md` | Markdown | ✓ Full (CLAUDE.md + .claude/rules/devkit-context.md) |
| GitHub Copilot | `.github/copilot-instructions.md`, `AGENTS.md` | Markdown | ✓ Full |
| Cursor | `.cursor/rules/*.mdc` (primary), `.cursorrules` (deprecated) | Markdown + YAML frontmatter | ✓ Full (both generated) |
| Windsurf | `.windsurfrules` | Markdown | ✓ Full |
| Gemini CLI | `GEMINI.md`, `~/.gemini/settings.json` | Markdown + JSON | ✓ Full |
| OpenCode | `opencode.json`, `AGENTS.md` | JSONC | ✓ Full |
| Aider | `CONVENTIONS.md`, `.aider.conf.yml` | Markdown + YAML | ✓ Markdown (CONVENTIONS.md) |
| AWS Kiro | `.kiro/steering/identity.md` | Markdown | ✓ Full |
| Devin | `AGENTS.md`, `.devin/rules/*.md` | Markdown | Partial (AGENTS.md only; add .devin/rules/ via extra_targets) |
| Roo Code | `.roorules` | Markdown | Via extra_targets |
| Amp | `.amp/context.md` | Markdown | Via extra_targets |
| Continue.dev | `~/.continue/config.yaml` | YAML | ✗ Requires MCP (v2) |
| Zed | `settings.json` (`context_servers` key) | JSON | ✗ Requires MCP (v2) |
| Warp | `AGENTS.md` | Markdown | ✓ Full |
| JetBrains Junie | `AGENTS.md` | Markdown | ✓ Full |
| Cline | `.clinerules`, `.cursorrules` | Markdown | ✓ Via .cursorrules |

---

## Go CLI — `devkit`

### v1 Commands ✓ DONE

| Command | Purpose |
|---------|---------|
| `devkit init` | Creates ~/.devkit/, scaffolds from templates |
| `devkit generate <path>` | Reads identity + context + donts → writes all AI config files to target |
| `devkit generate --dry-run` | Show what would be generated |
| `devkit generate --include-lessons` | Append lessons at end of output |
| `devkit generate --force` | Bypass 32KB hard limit |
| `devkit search <query>` | Search across all ~/.devkit/ markdown (ripgrep if available, Go-native fallback) |
| `devkit reset` | Delete ~/.devkit/ and re-initialize with confirmation |

### Milestone 2.5 Commands

| Command | Purpose |
|---------|---------|
| `devkit generate --all` | Regenerate all previously-generated project paths (tracked in `~/.devkit/projects.txt`) |
| `devkit search --interactive` | Fuzzy search — fzf via `exec.LookPath`, go-fuzzyfinder fallback |
| `devkit context ls` | List all contexts with size + last-modified date |
| `devkit doctor` | Show which generated project files are stale (source newer than output) |
| `devkit sync` | git pull/push on ~/.devkit/ — makes multi-machine sync discoverable |

### v2 Commands (MCP era)

| Command | Purpose |
|---------|---------|
| `devkit serve` | Run devkit as a local MCP server — AI tools query identity/context/constraints live instead of reading stale files |
| `devkit archive` | AI-summarize findings → lessons, compress originals |
| `devkit backup` | Encrypted tarball of ~/.devkit/ via `filippo.io/age` |

### Removed commands (not needed for solo user)

| Command | Why |
|---------|-----|
| `devkit switch` | Edit one YAML line + `devkit generate --all`. Not worth a command. |
| `devkit validate` | You know if your tools work. |
| `devkit new-context` | `cp templates/context.tmpl.md ~/.devkit/contexts/new.md`. |
| `devkit diff` | `git diff CLAUDE.md` in target project. |
| `devkit capture` | `$EDITOR ~/.devkit/findings/new-note.md`. Not worth code. |
| `devkit prompt` | v2 candidate — prompts are manual-use for now. |

---

## Composition Order

When `devkit generate` builds the output:

```
1. identity/ai.md           (how to behave)
2. identity/engineering.md  (how to write code and work)
3. contexts/<active>.md     (company/project context)
4. donts.md                 (hard constraints LAST = highest LLM weight)
```

Each section separated by `\n\n`. Frontmatter (`---` blocks) stripped from source files during composition.

Lessons NOT injected by default. `--include-lessons` appends them at the end.

Size enforcement:
```
> 16KB: ⚠ Warning
> 32KB: ✗ Fail (use --force to override)
```

---

## Generation Logic (no adapters, no interfaces)

Composer outputs a canonical markdown blob. Generator writes it to markdown targets directly and renders non-markdown targets through templates.

```go
header := "<!-- Generated by devkit. Do not edit. Contains private context. Source: ~/.devkit/ -->\n\n"
content := header + compose(identity, context, donts)  // concatenate in order, \n\n between sections

// Default markdown targets + extra_targets from workspace.yaml
// IMPORTANT: Never append to a package-level slice — allocate a new slice to avoid mutation.
defaultTargets := []string{"CLAUDE.md", "AGENTS.md", "GEMINI.md", "CONVENTIONS.md",
    ".cursorrules", ".windsurfrules", ".github/copilot-instructions.md",
    ".claude/rules/devkit-context.md", ".kiro/steering/identity.md"}
allTargets := make([]string, 0, len(defaultTargets)+len(workspace.ExtraTargets))
allTargets = append(allTargets, defaultTargets...)
allTargets = append(allTargets, workspace.ExtraTargets...)

for _, name := range allTargets {
    // PATH TRAVERSAL PROTECTION: validate resolved path stays within target dir
    resolved := filepath.Clean(filepath.Join(target, name))
    if !strings.HasPrefix(resolved, filepath.Clean(target)+string(os.PathSeparator)) {
        return fmt.Errorf("target %q escapes target directory", name)
    }
    writeFile(resolved, content)
}

// Structured targets — render through templates (text/template)
for _, tmpl := range []string{"opencode.json", ".mcp.json", ".claude/settings.json"} {
    if templateExists(tmpl) {
        rendered := renderTemplate(tmpl, workspace, content)
        writeFile(filepath.Join(target, tmpl), rendered)
    }
}

// Track this path for devkit generate --all
appendToProjectsRegistry(target)
```

Every generated file starts with a header line so you never wonder "should I edit this directly?" The answer is always: edit the source in `~/.devkit/`, then regenerate.

**Overwrite behavior:** If a target file exists and differs from what would be generated, print which files are being overwritten. The header ("do not edit") is the contract — if you edited it manually, regeneration replaces it.

**`extra_targets`:** Any paths in `workspace.yaml.extra_targets` are appended to the default list. Relative to the target project directory. This is how new AI tools are supported without a devkit release — the user adds one line to workspace.yaml.

**Projects registry:** Every successful `devkit generate <path>` appends the absolute path to `~/.devkit/projects.txt`. `devkit generate --all` reads this file and regenerates each path, skipping any that no longer exist.

**`.mcp.json` generation (Milestone 2.5):** Generates a project-level MCP config from an `mcp_servers` section in the active context file. Format mirrors the cross-tool standard:
```yaml
# in contexts/work.md frontmatter (optional)
mcp_servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
  jira:
    command: npx
    args: ["-y", "@some/jira-mcp-server"]
```
devkit renders this into `.mcp.json` at the project root. Env var values with `${VAR}` are kept as-is — the user's shell resolves them at runtime.

Tool-specific structured configs are Go `text/template` files in the public repo's `templates/` dir. No interface, no adapter pattern — just template files + a render loop. New structured target: add one `.tmpl` file + one filename to the list.

---

## Identity (2 files)

### `~/.devkit/identity/engineering.md`
- Coding style, architecture taste, naming conventions
- Git workflow (conventional commits, branching, PR conventions)
- Debugging approach
- Risk tolerance
- Security posture
- Workflow preferences

### `~/.devkit/identity/ai.md`
- AI behavior rules (concise, ask when unclear, never commit without permission)
- Languages & proficiency (calibrates explanation depth)
- Learning vs doing mode
- Communication tone
- Review philosophy
- Collaboration style

---

## Prompts (reference directory)

Reusable prompt templates that work with ANY AI tool. Copy-paste into ChatGPT, use as Claude skills, reference from Cursor — fully portable.

```
~/.devkit/prompts/
├── code-review.md          ← "Review this code for: security, correctness, simplicity..."
├── root-cause.md           ← "Given this error/behavior, trace the root cause..."
├── architecture-analysis.md ← "Analyze this system's architecture..."
├── source-analysis.md      ← "Analyze this codebase using the standard template..."
├── design-review.md        ← "Review this design for: risks, missing pieces..."
└── incident-response.md    ← "This is broken in production. Help me..."
```

These are YOUR refined prompts that improve over time. Not wired into any v1 command — they're a curated reference you use manually. v2 candidate: `devkit prompt list/show <name>` (copy to clipboard).

---

## Templates (in public repo)

### Source Code Analysis Template (`templates/analysis.tmpl.md`)

Language-agnostic. Works for Go, Python, TypeScript, Java, Rust, anything.

```markdown
---
name: <service-name>
type: service | library | cli-tool | action | frontend
language: go | python | typescript | java | rust | other
framework: <framework or none>
analyzed: YYYY-MM-DD
repo: <repo URL or path>
---

# <Name> — Source Code Analysis

## 1. Overview
| Property | Value |
|----------|-------|
| Name | |
| Language/Runtime | |
| Framework | |
| Entry Point | `<file:line>` |
| API Type | REST / gRPC / GraphQL / CLI / library |
| Storage | |
| Messaging | |
| Auth | |

## 2. Architecture Diagram
## 3. Startup Sequence
## 4. API / Interface
## 5. Event / Message Handling
## 6. Data Model
## 7. Project Structure
## 8. Domain Models / Types
## 9. Dependencies (internal + external)
## 10. Inter-Service Communication
## 11. Configuration
## 12. Testing
## 13. Design Decisions
## 14. Operational Notes
```

### Research/Workflow Template (`templates/research.tmpl.md`)

Problem → research → solution → test → close lifecycle.

```markdown
---
ticket: <ID>
title: <short description>
status: research | implementing | testing | done
created: YYYY-MM-DD
---

# <Ticket> — <Title>

## Problem Statement
## Research (source analysis, data flow, root cause)
## Solution (proposed changes, risks)
## Implementation (what was actually done)
## Testing (checklist)
## Resolution (PR link, merged date)
```

### Tools Reference Template (`templates/tools.tmpl.md`)

Pre-filled with common dev tools, cross-platform install commands, docs links. User customizes in `~/.devkit/tools.md`.

---

## Security & Dependency Hygiene

| Layer | How |
|-------|-----|
| Go deps | Dependabot weekly PRs + CI blocks CVEs |
| CI scan | govulncheck + trivy + golangci-lint on every push |
| Binary | CGO_ENABLED=0, static binary |
| Supply chain | go.sum checked in, verified modules |
| Secrets | Never in any file. trufflehog in CI (AGPL-3.0 — fine for CI, not embedded). |
| Repo deps | MIT/Apache/BSD only — enforced via `go-licenses` in CI |
| Private data | Generated files contain private context. Header warns "do not commit to public repos." |

Maintainer workflow: `make update && make check` before tagging release.

### Security notes
- **Template injection**: `text/template` used for structured configs only. No custom `FuncMap` with side effects. Source markdown is concatenated raw (never template-rendered).
- **Command injection**: `exec.Command("rg", "--", query, ...)` — `--` separator prevents query being interpreted as flags. No shell interpolation.
- **Path traversal (FIXED post-audit)**: Both `active_context` and `extra_targets` MUST be validated to stay within their intended directories. `extra_targets` resolved paths must be within targetDir. `active_context` must resolve within `contexts/`. Implementation: `filepath.Clean` + `strings.HasPrefix` check after joining.
- **File permissions**: `~/.devkit/` MUST be created with 0700 (not 0755). Files within MUST be 0600. Contains private company context.
- **Supply chain**: All GitHub Actions in CI MUST be pinned to commit SHA, not mutable tags or branches. `trufflehog@main` is a supply chain attack vector.
- **Generated file leakage**: devkit SHOULD check if generated files are in .gitignore for git repos. Warn (not error) if not — prevents accidental push of private context to public repos.
- **Archive encryption (v2)**: Use `age` (modern, no GPG keyring complexity).

---

## MCP Server Mode (v2 — `devkit serve`)

Eliminates the "regenerate after identity change" problem. Instead of writing static files, devkit runs as a local MCP server that AI tools query on every session.

```
devkit serve
→ Speaks MCP over stdio (JSON-RPC 2.0)
→ Exposes three Resources:
    devkit://identity         ← composed identity/ai.md + identity/engineering.md
    devkit://context/{name}   ← contexts/<name>.md (default: active_context)
    devkit://constraints      ← donts.md
→ Claude Code (or any MCP client) calls these instead of reading CLAUDE.md
```

**Config in target project** (generated by `devkit generate`, Milestone 2.5+):
```json
{
  "mcpServers": {
    "devkit": {
      "command": "devkit",
      "args": ["serve"]
    }
  }
}
```

**Why this is v2, not now:**
- Creates a daemon dependency — `devkit` binary must be on PATH and runnable at AI tool startup
- Static files work fine and are already implemented
- MCP client support across tools is broad but not universal (Cursor, Copilot file-based rules still dominate)
- Build when the "forget to regenerate" problem becomes a daily friction point

**Implementation:** ~200 lines of Go. MCP stdio transport is JSON-RPC 2.0 line-delimited. No external SDK needed for a read-only resource server.

---

## Archive Pipeline (v2 — build only when findings > 50)

```
devkit archive --reason "leaving <company>"

1. Collect findings/*.md + analyzed/*.md
2. Single-pass or two-pass AI summarization (Summarizer interface)
3. Draft → lessons/<company>-<years>.md
4. Human review + confirm
5. Compress originals to archive/<company>-<date>.tar.gz
6. Clear findings/ and analyzed/

Summarizer implementations:
- ManualSummarizer: concat into paste-ready file for web LLM
- ClaudeSummarizer: API-based (requires key)
```

### Lessons file format

```markdown
---
company: acme-corp
period: 2024-2026
type: architecture | debugging | operations | security | testing | scalability
tags: [event-driven, messaging, kubernetes]
created: 2026-06-18
---

# Engineering Lessons — Acme Corp (2024–2026)
...
```

---

## Implementation Architecture

```
.                             ← root package (package main) — all cobra commands here
├── main.go                  ← root command, embed.FS, run() entry point
├── init.go                  ← scaffolds ~/.devkit/
├── generate.go              ← the core command (--dry-run, --include-lessons, --force, --all, --quiet)
├── reset.go                 ← delete + re-initialize with confirmation (--hard)
├── search.go                ← search entry point (--interactive)
├── status.go                ← sync state for all tracked projects
├── diff.go                  ← show what generate would change (--check for CI)
├── context.go               ← devkit context ls
├── doctor.go                ← stale detection (mtime-based)
├── lint.go                  ← source file validator
├── version.go               ← devkit version (ldflags-injected)
├── sync.go                  ← git pull/push on ~/.devkit/ (TODO)
└── serve.go                 ← MCP server mode (v2)

internal/
├── fs/           ← filesystem interface (testable with in-memory)
├── config/       ← loads workspace.yaml, resolves data dir via os.UserHomeDir()
├── devctx/       ← loads identity/*, contexts/<active>, donts.md (NOT "context" — avoids stdlib shadow)
├── composer/     ← composition order, frontmatter stripping, \n\n separators, size enforcement
├── generator/    ← write markdown targets + render template targets to target path
├── search/       ← ripgrep with Go-native fallback
└── registry/     ← read/write ~/.devkit/projects.txt
```

No model/ package needed — types are simple enough to live in each package. No adapter interface — it's a for-loop over filenames.

### FS interface

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

Notes:
- `Glob` needed for `identity/*` file discovery. `embed` directives use `//go:embed all:templates` (the `all:` prefix is required to capture dotfiles like `.cursorrules` in scaffold templates).
- `OsFS.WriteFile` MUST use a unique temp filename (e.g., `os.CreateTemp` in same directory) to avoid races on concurrent writes. Current fixed `.devkit-tmp` suffix is a race condition.
- All command-layer code should use the FS interface, not `os.*` directly. Known violations: `writeSkillsFile`, `init.go`, `reset.go`.

---

## Implementation Checklist

### Pre-work: Repo setup ✓ DONE
- [x] Create GitHub repo `devkit` (PUBLIC, no license) — github.com/dklinux7/devkit
- [x] Initialize Go module (go 1.25)
- [x] Set up CI (lint, test, govulncheck, trivy, go-licenses)
- [x] Configure Dependabot
- [x] Add trufflehog check
- [x] Write scaffold templates (identity, context, prompts, tools, research, analysis) — use `//go:embed all:templates` for dotfiles
- [x] Public repo .gitignore

### Milestone 1: `devkit generate` working ✓ DONE
- [x] `internal/fs` — interface (with Glob) + real implementation
- [x] `internal/config` — load workspace.yaml, resolve data dir via `os.UserHomeDir()` + ".devkit", `$DEVKIT_HOME` override
- [x] `internal/devctx` — load identity/*.md + contexts/<active> + donts.md (strip frontmatter from all sources)
- [x] `internal/composer` — composition order (identity → context → donts), `\n\n` separators, size check (16KB warn, 32KB fail)
- [x] `internal/generator` — markdown targets (direct write) + structured targets (template render), overwrite reporting
- [x] `cmd/devkit/init.go` — scaffold ~/.devkit/ from embedded templates
- [x] `cmd/devkit/generate.go` — the core command (--dry-run, --include-lessons, --force)
- [x] `cmd/devkit/main.go` — cobra root

### Milestone 2: Search + polish ✓ DONE
- [x] `internal/search` — Go-native fallback (filepath.WalkDir + regexp), ripgrep when available (exec.Command with `--` separator)
- [x] `cmd/devkit/search.go` — search entry point
- [x] Binary releases (goreleaser OSS — macOS/Linux/Windows/arm64)
- [x] README
- [x] Makefile
- [x] CI workflows (lint, test, govulncheck, go-licenses, trufflehog, trivy)
- [x] Dependabot (go modules + GitHub Actions)
- [x] `cmd/devkit/reset.go` — delete + re-initialize with confirmation
- [x] `templates/analysis.tmpl.md` + `templates/research.tmpl.md`
- [x] `docs/setup/github-multi-account.md`

### Milestone 2.5: Plug and play + quality of life
- [x] `extra_targets` in workspace.yaml — user-defined additional output files, merged with defaults in generator
- [x] `~/.devkit/projects.txt` registry — append on every `devkit generate`, read by `--all`
- [x] `devkit generate --all` — regenerate all tracked project paths
- [x] `devkit doctor` — compare mtime of identity/context sources vs generated files, print stale projects
- [x] `devkit context ls` — list contexts/ with size + last-modified date
- [x] `devkit search --interactive` — fzf via exec.LookPath, go-fuzzyfinder as fallback
- [x] `devkit sync` — git pull/push wrapper on ~/.devkit/
- [x] `.mcp.json` generation — parse `mcp_servers` from context frontmatter, render to project
- [x] `GEMINI.md` added to default markdown targets
- [x] `docs/setup/new-machine.md` — private git repo for ~/.devkit/ + mise setup guide
- [x] Atomic writes in generator (write to .devkit-tmp then rename)
- [x] `.cursor/rules/devkit-context.mdc` generation with Cursor frontmatter
- [x] `devkit status` — sync state for all tracked projects
- [x] `devkit diff <path>` + `--check` — CI integration for "forgot to regenerate"
- [x] Non-destructive `devkit reset` (default) + `--hard` for old behavior
- [x] `devkit lint` — source file validator
- [x] `devkit generate --quiet` and `--all` flags
- [x] `~/.claude/skills/devkit-context.md` output (hedge against SKILL.md ecosystem)

### Milestone 2.75: Post-Audit Hardening ✓ DONE
- [x] Fix slice mutation bug (`append(MarkdownTargets, ...)` in generator.go and generate.go)
- [x] Add path traversal validation for `extra_targets` and `active_context`
- [x] Pin all GitHub Actions to commit SHA (trufflehog, trivy, goreleaser, golangci-lint)
- [x] Fix `--dry-run --all` panic
- [x] Add `devkit version` command with ldflags injection
- [x] Change ~/.devkit/ permissions to 0700, files to 0600
- [x] Add `CONVENTIONS.md`, `.claude/rules/devkit-context.md`, `.kiro/steering/identity.md` to default targets
- [x] Fix OsFS temp file naming (use os.CreateTemp for uniqueness)
- [x] Extract shared `resolveComposed()` helper to eliminate pipeline duplication
- [x] Add .gitignore check warning on generate
- [x] Add macOS/Windows to CI test matrix
- [x] Write missing critical test cases (dry-run-all, doctor-stale, lint-errors, pipeline-lifecycle)
- [x] `devkit untrack <path>` — remove project from registry (medium backlog item, done early)
- [x] `status` and `doctor` are distinct: status uses content comparison, doctor uses mtime comparison
- [x] Add `--verbose` / `-v` flag — prints data dir, active context, composed size to stderr
- [x] Test coverage reporting in CI — `-coverprofile` + artifact upload
- [x] `mise.toml` for local dev tool versions (go 1.26, golangci-lint 2.12.2)
- [x] `registry.ReadAll` distinguishes permission error from not-found
- [x] `reset.go` uses `cmd.InOrStdin()` instead of `os.Stdin` (testable)

### Milestone 3: Archive + backup (when findings > 50)
- [ ] Summarizer interface (Manual + Claude implementations)
- [ ] Compression to archive/
- [ ] Encryption via `age` (`filippo.io/age` Go library — BSD-3-Clause, no external binary needed)
- [ ] `devkit backup` — encrypted tarball using age public key (`backup_recipient` in workspace.yaml, optional)
- [ ] `devkit archive` command

### v2: MCP server mode
- [ ] `devkit serve` — stdio MCP server exposing three resources: `identity`, `context/{name}`, `constraints`
- [ ] Add to scaffold: `.mcp.json` stub pointing at `devkit serve` for Claude Code / VS Code / Gemini CLI
- [ ] Document: once `devkit serve` is in `.mcp.json`, static file regeneration is optional for MCP-capable tools

---

## Rejected Ideas (final)

| Idea | Why |
|------|-----|
| Adapter interface | All tools read same markdown. A for-loop over filenames is sufficient. |
| ai_tools in workspace.yaml | Generate everything. Disk is free. |
| manifest.json | You'll never read it. Git already exists. |
| examples/ directory | OSS documentation for zero users. README suffices. |
| Public repo safety check | You know where you're generating. |
| 4 identity files | Won't maintain 4. Two (engineering + ai) is the maintainable number. |
| Context directories by default | Start flat. Promote to folder only when one file grows too large. |
| devkit switch / validate / doctor / new-context / diff / capture | Shell one-liners or not needed for solo user. |
| Work journal (daily .md files) | git log captures "what I did". Findings captures learnings. Journal is a third format for same info. |
| inbox/active/lessons/archive workflow | 20-50 findings is manageable. Adding workflow states = friction that discourages capturing. |
| Plugin architecture | Over-engineering for one user. |
| Knowledge folder taxonomy | Tags in lessons frontmatter. Folders are commitments. |
| Integrations block in workspace.yaml | AI reads prose in context files, not `kafka: enabled: true`. |
| Workflow runner (standup, daily, sprint) | Company-specific. Handled by AI tool config (MCP servers, skills) not devkit. devkit provides context, AI tools provide automation. |
| `.devkit.local.yaml` per-project | Context markdown IS the override. |
| Symlinks for output | Breaks cross-platform. Direct write to target. |
| Per-tool customization (Cursor globs, Copilot per-file) | Conscious trade-off: basic coverage everywhere > perfect for one tool. Template system allows tool-specific structured configs when needed. |
| `os.UserConfigDir()` | Returns wrong path on macOS (`~/Library/Application Support`), conflates config/data on Linux. Use `os.UserHomeDir()` + `.devkit`. |
| devkit update (CLI command) | Maintainer concern. Dependabot + CI. |
| Versioned contexts | Git history. |
| Team sharing | Company's job. |
| Separate projects/ from contexts/ | Company-scoped in practice. |
| model/ package | Types are simple enough to live in each package for this codebase size. |
| chezmoi for ~/.devkit/ sync | Overkill — copy model creates friction (must run `chezmoi add` after every edit). ~/.devkit/ is plain markdown + YAML, no secrets, no per-machine conditionals. A private git repo is sufficient and simpler. |
| age for normal use (v1/v2) | ~/.devkit/ contains no secrets — coding style, work context, constraints. OS disk encryption (FileVault/LUKS/BitLocker) + chmod 700 + private git repo is appropriate protection. age belongs only in `devkit backup` (Milestone 3). |
| just / Task to replace Makefile | Existing Makefile is clean, POSIX-compatible, 11 simple targets. Make is universally available on macOS/Linux with no install. Migrate to just only if Windows CI is ever added. Do not adopt mage (7-year-old unresolved bugs, stalled maintenance). |
| Task file generation in devkit generate | devkit doesn't know the project's language or build system. A generic Taskfile would require immediate customization and risks colliding with existing task files. Scope creep. |
| devbox for dev environment | No native Windows support (WSL only). Adds Nix as transitive dependency. mise covers the tool version management need with full cross-platform support. |
| sops for secrets | Team/GitOps tool designed for cloud KMS (AWS/GCP/Azure). Overkill for a solo dev. age handles the personal backup encryption use case without infrastructure overhead. |
| proto as mise alternative | 1,302 stars vs mise's 29,653. Single-maintainer company (moonrepo). Tiny ecosystem. Not a viable recommendation despite good Windows support. |
| Dynamic context injection (Jira/Slack/GitHub API) | devkit never makes API calls. That's the AI tool's job via MCP servers. Principle 7: devkit provides static context, AI tools provide dynamic integrations. |
| AI-to-AI context broker | Already solved — sub-agents inherit CLAUDE.md from the project. The generated file IS the broker. The MCP serve mode handles the edge case (agents outside any project dir). |
| Company onboarding packs | Behavior doesn't exist in the ecosystem yet. Build devkit sync first. Revisit when a company actually publishes a context pack. |
| Plugin hooks (pre/post generate shell scripts) | No concrete use case yet. Build when you have 3+ cases that can't be solved by a flag. Shell hooks are the right implementation when the time comes — not Go interfaces. |
| Plugin architecture / plugin registry | Over-engineering. `extra_targets` in workspace.yaml solves new-tool support. Template files solve structured config. No plugin system needed. |
| Full TUI (bubbletea) for search | `bubbles list.Model` requires adopting the bubbletea event loop across the entire app — significant architectural commitment for a single feature. fzf + go-fuzzyfinder fallback is sufficient. |
| CrewAI / LangGraph config generation | Agent framework configs are project-specific code, not identity/context. Wrong layer for devkit. |

---

## Companion Tools

Tools researched and evaluated for use alongside devkit. Decisions are final unless circumstances change.

| Tool | Verdict | Use | Notes |
|------|---------|-----|-------|
| **fzf** (junegunn/fzf, 81k★, MIT) | Build into devkit | `devkit search --interactive` primary path | Detect via `exec.LookPath("fzf")`. go-fuzzyfinder as fallback (no external binary). Do NOT use fzf as an embedded Go library — the `src` package has no stable API. |
| **go-fuzzyfinder** (ktr0731/go-fuzzyfinder, 519★, MIT) | Build into devkit | `devkit search --interactive` fallback | Pure Go, tcell-based, works on macOS/Linux/Windows. Clean API: `fuzzyfinder.Find(items, labelFn)`. Avoid go-fzf (koki-develop) — last release 2023, stale deps. |
| **mise** (jdx/mise, 29k★, MIT) | Document as companion | Runtime version management | Best cross-platform tool version manager. Full Windows native support. Bus factor caveat: Jeff Dickey is ~85% of commits, funded solo project. Windows env var injection requires `mise x` or shim mode (not transparent activation). |
| **age** (FiloSottile/age, 22k★, BSD-3-Clause) | Milestone 3 only | `devkit backup` encryption | `filippo.io/age` Go library embeds cleanly (~30 lines). Use public key mode (not passphrase) for backup. Optional `backup_recipient` field in workspace.yaml. OS disk encryption is sufficient for normal ~/.devkit/ use. |
| **private git repo** | Document as companion | ~/.devkit/ multi-machine sync | Simplest solution — no extra tool. `devkit sync` (v2) wraps git pull/push to make it discoverable. chezmoi is overkill for plain markdown files. |

---

## User Flow

### First time

```bash
go install github.com/<user>/devkit/cmd/devkit@latest
devkit init
# → Creates ~/.devkit/ with scaffolds
# → Prints next steps (see post-init guidance below)
```

### Day-to-day

```bash
devkit generate ~/projects/my-app
# → Writes CLAUDE.md, AGENTS.md, .cursorrules, etc.

devkit search "retry logic"
# → Searches across all ~/.devkit/ markdown
```

### Switching company

```bash
# Edit workspace.yaml: active_context: new-company
# Create contexts/new-company.md
devkit generate ~/projects/new-project
```

### Leaving company (v2)

```bash
devkit archive --reason "leaving acme-corp"
```

---

## CLI Help & Guidance

Every command should explain what to do next. The CLI is the documentation.

### `devkit init` post-run output

```
✓ Created ~/.devkit/

Next steps:
  1. Edit your identity:
     ~/.devkit/identity/ai.md          ← how AI should behave with you
     ~/.devkit/identity/engineering.md  ← your coding style, git workflow, preferences

  2. Set your constraints:
     ~/.devkit/donts.md                ← things AI must never do

  3. Create your first context:
     ~/.devkit/contexts/work.md        ← describe your company, repos, tools, team

  4. Add your repo clone commands:
     ~/.devkit/repos.md                ← git clone commands for working + analysis copies

  5. Generate AI config for a project:
     devkit generate ~/path/to/project

Run `devkit help` for all commands.
```

### `devkit generate` post-run output

```
✓ Generated 5 files in ~/projects/my-app:
  CLAUDE.md, AGENTS.md, .cursorrules, .windsurfrules, copilot-instructions.md

  Context: work (from ~/.devkit/contexts/work.md)
  Size: 11.2KB (under 16KB limit)

  ⚠ These files contain your private context. Add to .gitignore if repo is public.
```

### `devkit help` (root command)

```
devkit — personal dev workspace generator

Commands:
  init              Set up ~/.devkit/ with starter templates
  generate <path>   Compose identity + context → write AI config files to target
  search <query>    Search across all ~/.devkit/ markdown

Flags (generate):
  --dry-run           Show what would be generated without writing
  --include-lessons   Append lessons at end of output
  --force             Bypass 32KB size limit

How it works:
  1. You maintain identity, constraints, and context files in ~/.devkit/
  2. `devkit generate` composes them and writes AI config files to any project
  3. Every AI tool (Claude, Cursor, Copilot, Windsurf, OpenCode) reads the same context

Files:
  ~/.devkit/workspace.yaml         Your name + active context
  ~/.devkit/identity/              How you work and how AI should behave
  ~/.devkit/donts.md               Hard constraints (never do X)
  ~/.devkit/contexts/              One file per company/project
  ~/.devkit/prompts/               Reusable prompt templates (manual reference)
  ~/.devkit/repos.md               Clone commands for your repos
  ~/.devkit/findings/              Research notes (manual, not generated)

Workflow:
  New company  → write contexts/<name>.md, update workspace.yaml, run generate
  New project  → run `devkit generate ~/path/to/project`
  Update style → edit identity/ files, re-run generate on active projects
```

### `devkit generate --dry-run` output

```
Would generate 5 files in ~/projects/my-app:

--- CLAUDE.md (preview) ---
<!-- Generated by devkit. Do not edit. Contains private context. Source: ~/.devkit/ -->

[AI Behavior]
...first 10 lines of identity/ai.md...

[Engineering Style]
...first 10 lines of identity/engineering.md...

[Context: work]
...first 10 lines of contexts/work.md...

[Constraints]
...first 10 lines of donts.md...

Total: 11.2KB | Files: CLAUDE.md, AGENTS.md, .cursorrules, .windsurfrules, copilot-instructions.md
```

### README.md (public repo)

Short. Points to `devkit help`. No tutorial — the CLI IS the tutorial.

```markdown
# devkit

Personal dev workspace generator. Composes your identity, constraints, and 
company context into AI config files for any coding tool.

## Install

go install github.com/<user>/devkit/cmd/devkit@latest

## Usage

devkit init                    # set up ~/.devkit/
devkit generate ~/my-project   # write AI config files
devkit search "query"          # search your notes

Run `devkit help` for full details.

## What it does

You maintain markdown files describing how you work. devkit composes them 
and writes config files that Claude Code, Cursor, Copilot, Windsurf, and 
OpenCode all understand.

One source of truth → every AI tool gets the same context.

## Not licensed

This project is not licensed for use, modification, or distribution.
Source code is publicly visible for transparency and reference only.
All rights reserved.
```

---

## Open Questions (resolve while building)

1. **identity/ content** — Q&A to fill engineering.md and ai.md
2. **donts.md content** — Q&A to fill constraints
3. **prompts/ initial set** — which prompts to include in scaffold
4. **Multi-machine sync** — private git repo for ~/.devkit/ (most likely)
5. **Testing** — unit tests with in-memory FS (yes), integration tests (only if time)

## Workspace Layout (outside devkit)

devkit manages `~/.devkit/` (your identity and context). But your actual code lives in a workspace with two distinct areas:

```
~/dev/                            ← WORKING COPIES (you commit here)
├── <org>/
│   ├── service-a/
│   ├── service-b/
│   └── shared-lib/
└── personal/
    └── side-project/

~/dev/readonly/                   ← ANALYSIS COPIES (read-only, always on main)
├── <org>/
│   ├── service-a/
│   ├── service-b/
│   └── shared-lib/
└── sync.sh                      ← resets all to origin/main (safe to nuke)
```

### repos.md (in ~/.devkit/contexts/ or standalone)

A file listing clone commands for the repos you work with. SCM-agnostic — works with GitHub, GitLab, Bitbucket, etc.

```markdown
# Repos

## Working copies (~/dev/<org>/)
git clone git@github.com:<org>/service-a.git
git clone git@github.com:<org>/service-b.git
git clone git@github.com:<org>/shared-lib.git

## Analysis copies (~/dev/readonly/<org>/)
git clone git@github.com:<org>/service-a.git
git clone git@github.com:<org>/service-b.git
git clone git@github.com:<org>/shared-lib.git
```

**Rules:**
- Working copies: your branches, your commits, your PRs
- Analysis copies: always on main, force-reset via `sync.sh`, used by AI tools for cross-repo analysis
- `repos.md` lives in `~/.devkit/` (private) — update it when you join a company or onboard new repos
- `sync.sh` is a simple loop: `cd <repo> && git fetch origin && git reset --hard origin/main` for each analysis copy

**Why two folders:** AI tools analyzing code shouldn't accidentally see your in-progress branches. Clean main-branch copies give consistent analysis results.

### devkit generate integration

`repos.md` is NOT composed into AI config files — it's a reference for you. Your `contexts/<company>.md` file describes what the repos *are* (purpose, ownership, key services). The clone commands are operational, not context.

---

## Scope Boundary: devkit vs AI tool config

```
devkit (portable, static)          AI tool config (company-specific, dynamic)
─────────────────────────          ─────────────────────────────────────────
Identity (who you are)             MCP servers (Jira, Slack, GitHub APIs)
Context (what company uses)        Skills / slash commands (standup, research)
Constraints (donts.md)             Auth tokens / API keys
Prompts (reusable templates)       Hooks, automation, scheduled tasks
```

**Rule:** devkit generates context that tells the AI tool *what exists* (Jira projects, Slack channels, GitHub orgs). The AI tool config decides *how to interact* with those systems. When you join a new company, write the context file describing the tools, then configure your AI tool's integrations. devkit never makes API calls.

## Resolved Design Questions (from final review + engineering audit)

| Question | Resolution |
|----------|-----------|
| macOS path vs os.UserConfigDir() | Use `os.UserHomeDir()` + ".devkit" everywhere. Drop UserConfigDir. |
| opencode.toml gets markdown | Separate markdown targets (direct write) from structured targets (template render). |
| No separators between files | `\n\n` between every composed section. Strip frontmatter from sources. |
| 16KB too tight | Warn at 16KB, fail at 32KB. (Audit note: 16KB warning may be noise; consider raising to 24KB.) |
| donts.md ordering | Moved to LAST position (LLMs weight end-of-prompt more). Validated by audit. |
| Overwrite without warning | Print which files are overwritten. Header is the contract. |
| ripgrep runtime dep | Go-native fallback (WalkDir + regexp); rg used when available for speed. |
| `context` package shadows stdlib | Renamed to `devctx`. |
| embed misses dotfiles | Use `//go:embed all:templates`. |
| $DEVKIT_HOME vs XDG priority | $DEVKIT_HOME wins unconditionally on all platforms. |
| License enforcement | `go-licenses` in CI. |
| ripgrep flag injection | `--` separator before query arg. |
| Private data leakage | Generated file header warns "contains private context". Audit: also check .gitignore. |
| Prompts "first-class" but unused | Demoted to "reference directory". v2 candidate for CLI integration + MCP prompts. |
| findings/analyzed orphaned in v1 | Manual-use-only. No command manages them in v1. |
| Path traversal in extra_targets | **AUDIT FIX**: Must validate resolved path stays within target directory. |
| Path traversal in active_context | **AUDIT FIX**: Must validate resolved path stays within contexts/ directory. |
| `append(slice, ...)` on global var | **AUDIT FIX**: Never append to package-level slice. Always allocate new. |
| doctor vs status duplication | **AUDIT**: Consolidate. Content comparison (status) is canonical. |
| Same content to all targets | **AUDIT RISK**: Acceptable now but plan `TargetSpec` for per-target limits/filters. |
| CI actions pinned to tags | **AUDIT FIX**: Pin to SHA. Tags are mutable attack vectors. |
| ~/.devkit/ permissions | **AUDIT FIX**: 0700 for directory, 0600 for files. Contains private context. |

---

---

## Competitive Intelligence (2026-06-18)

Deep analysis of every known tool in this space. Purpose: steal every good idea, avoid every known failure mode, anticipate every ecosystem threat.

### Tools analyzed

| Tool | Stars | Language | Status |
|------|-------|----------|--------|
| Caliber | 1,141 | Python | Active |
| agentsync | 123 | Python | Active |
| contextai | 5 | Python | Stale |
| agentfiles | 0 | Go | Stale |
| ai-brain | 1 | Shell | Active |
| LynxPrompt | 41 | Python | Active |
| danielmiessler/Personal_AI_Infrastructure | 15,987 | Markdown | Active |
| wshobson/agents | 37,000 | Markdown/Template | Active |

---

### Steal List — Tier 1 (High value, low cost, build within 3 months)

**1. Hash-based sync status (`devkit status`)**
Stolen from: Caliber's sync detection approach.
Show 4-state output for each tracked project:
- `in-sync` — generated files match what would be produced now
- `source-newer` — identity/context changed since last generate
- `output-modified` — someone edited generated files manually
- `conflict` — both changed
Implementation: store SHA256 of composed content in `~/.devkit/projects.txt` alongside the path. Compare on `devkit status`. No mtime heuristics — hash the content.

**2. `devkit diff <path>`**
Stolen from: Caliber's diff view.
Show what would change if you ran generate now. Uses standard `diff` algorithm on current file content vs what would be written. Add `--check` flag that exits 1 if anything would change — plugs into CI to catch "forgot to regenerate" before PR merge.

**3. Content-block preservation (`<!-- devkit:begin -->` / `<!-- devkit:end -->`)**
Stolen from: wshobson/agents partial-update model.
Instead of overwriting the entire file, replace only the devkit-managed block. This lets team members add project-specific notes below the devkit block without losing them on regenerate. Implementation: write header block delimited by markers; on next generate, find markers and replace only that span. Fall back to full overwrite if markers missing.
**Caveat:** adds complexity. Only implement if "someone else edits CLAUDE.md" is a real problem you hit.

**4. `devkit lint` — source file validator**
Stolen from: LynxPrompt's prompt quality scoring approach.
Validate `~/.devkit/` files for common problems before they propagate to generated output:
- Frontmatter syntax errors
- Files exceeding individual size budgets (e.g., single context > 8KB)
- Missing required sections in identity files
- Template variables in identity files that won't be resolved
- Detects `${VAR}` patterns that might be accidentally left unexpanded
No LLM required — pure static analysis.

**5. Atomic writes (temp + rename)**
Stolen from: observed failure mode in agentfiles and contextai.
Current `WriteFile` is not atomic — a crash mid-write leaves a truncated CLAUDE.md. Fix: write to `<path>.devkit-tmp`, then `os.Rename()`. `Rename` is atomic on POSIX (same filesystem). Protects against partial writes during power loss or signal interrupt.

**6. `--quiet` flag on generate**
Stolen from: Caliber's `--silent` mode.
`devkit generate --quiet <path>` prints nothing on success, only errors to stderr. Needed for `devkit generate --all` in cron/CI contexts where output noise is undesirable.

**7. `devkit score` — identity quality rubric**
Stolen from: LynxPrompt's scoring concept, but deterministic (no LLM).
Score each identity file 0-100 based on measurable properties: word count, section coverage, specificity signals (has code examples, has named tools, has negative examples in donts.md). Print breakdown. Goal: make "my context is too vague" actionable without needing an AI to tell you.
Implementation: ~100 lines of Go. Heuristic, not ML.

**8. Non-destructive `devkit reset`**
Stolen from: every OSS tool that got GitHub issues saying "I accidentally ran reset and lost everything".
Current reset: `rm -rf ~/.devkit/` + re-init. New behavior:
- Preserve: `identity/`, `contexts/`, `findings/`, `prompts/`, `lessons/`
- Wipe and re-scaffold: `workspace.yaml`, `donts.md`, `tools.md`
- Add `--hard` flag for the current nuke-everything behavior
This makes reset useful for "reset config files to defaults" instead of "destroy all my work".

---

### Steal List — Tier 2 (High value, higher cost, build within 6 months)

**9. TELOS-structured identity (`identity/telos.md`)**
Stolen from: danielmiessler/Personal_AI_Infrastructure TELOS model.
Add a third identity file for professional mission/goals/values. TELOS = Telos (purpose), Epistemics (how you learn/think), Lens (worldview), Operations (how you work), Style (communication preferences). This is distinct from `engineering.md` (how you code) and `ai.md` (how AI should behave) — it's the "who you are professionally" layer that makes AI interactions feel more like working with someone who knows you.
Scaffold it in `devkit init`. Keep it optional — empty file means it's skipped in composition.

**10. `devkit wizard` — interactive terminal interview**
Stolen from: Caliber's onboarding flow + LynxPrompt's Q&A scaffolding.
`devkit wizard` runs an interactive terminal interview that fills in identity files with real answers instead of placeholder text. Questions like "What's your preferred git branching strategy?", "What are your top 3 coding constraints?", "What should AI never do?". Writes answers directly to the right files.
Why this matters: the biggest OSS failure mode is "works for author, confusing for anyone else." The wizard solves cold-start. If it takes 4 file edits to see value, most users bounce.

**11. `devkit watch <path>` — auto-regenerate**
Stolen from: Caliber's file watcher concept.
Watch `~/.devkit/` for changes. On any modification, re-run generate for all tracked projects. Uses `fsnotify` (MIT, 9k★, Go). This eliminates the "forgot to regenerate" problem entirely for power users.
**Dependency note:** `fsnotify` is MIT and pure Go — passes license check. Add to go.mod only when implementing.

**12. `[agent]` / `[human]` section scoping**
Stolen from: wshobson/agents section-tagged template system.
Most underrated idea in the space. Allow identity/context files to tag sections for specific consumers:
```markdown
<!-- devkit:for agent -->
When writing code, always include error handling.
<!-- devkit:end -->

<!-- devkit:for human -->
This section is for human reference only, not injected into AI context.
<!-- devkit:end -->
```
Sections without tags are included for all. This lets identity files serve dual purpose: AI context AND human documentation. Implementation: strip `human`-tagged sections during composition.

**13. ISA-structured finding template**
Stolen from: danielmiessler's research workflow structure.
Replace `research.tmpl.md` with a 12-section ISA (Issue, Solution, Analysis) template that maps to real engineering work:
- Issue: what broke / what needs to change
- Context: system state, constraints, stakeholders
- Hypotheses: candidate root causes or approaches
- Experiments: what you tried and what happened
- Root cause: confirmed explanation
- Solution: what was actually done
- Verification: how you confirmed it worked
- Outcome: PR link, metrics, before/after
- Lessons: what to carry forward
- References: links, docs, related tickets
The current `research.tmpl.md` is too generic to actually drive a debugging session.

**14. `devkit import <file>` — heuristic import**
Stolen from: observed need when evaluating Caliber and agentfiles.
Many users have existing `.cursorrules`, `CLAUDE.md`, or Copilot instruction files with real content. `devkit import` heuristically parses these and distributes content to the right `~/.devkit/` files — identity-like content to `identity/engineering.md`, constraint-like content to `donts.md`, context-like content to `contexts/<guessed-name>.md`. Outputs a diff before writing so the user can review. Makes adoption from existing tooling frictionless.

---

### Steal List — Tier 3 (Future / v2, defer until pain demands)

**15. `devkit scan <path>` — infer context from repo**
Stolen from: agentsync's repository scanning.
Analyze a project directory and draft a context file: detect languages, frameworks, CI system, dependency managers, test framework. Output as a starting draft in `contexts/<name>.md`. Useful when onboarding to an unfamiliar codebase.

**16. `devkit serve` (MCP mode) — move earlier**
Originally planned as v2. Based on competitive research and SKILL.md ecosystem momentum, this should move to Milestone 3, not v2. Static files will remain primary for 12-18 months, but the MCP server approach is how Anthropic will eventually want devkit to work. ~200 lines of Go (JSON-RPC 2.0 over stdio).

**17. `devkit backup` with age encryption**
As designed. Build when findings > 50 or when multi-machine is needed. `filippo.io/age` Go library embeds cleanly.

---

### Format Stability Assessment (as of 2026-06-18)

What devkit currently generates vs. what it should generate:

| Format | Current | Status | Action |
|--------|---------|--------|--------|
| `CLAUDE.md` | ✓ generated | Stable — Anthropic standard | None |
| `AGENTS.md` | ✓ generated | Stable — Linux Foundation standard, 60k+ repos | None |
| `GEMINI.md` | ✓ generated | Stable — Google official | None |
| `.cursorrules` | ✓ generated | **DEPRECATED** — Cursor moved to `.cursor/rules/*.mdc` | Add `.cursor/rules/devkit-context.mdc` |
| `.windsurfrules` | ✓ generated | At risk — Windsurf acquired by Devin (OpenDevin) | Keep for now, monitor |
| `.github/copilot-instructions.md` | ✓ generated | Stable — GitHub official | None |
| `opencode.json` | ✗ was `opencode.toml` | **FIXED** — OpenCode uses JSONC not TOML | Fixed in this commit |
| `.claude/settings.json` | template-ready | Stable | None |

**New formats to add (Milestone 2.5):**

| Format | Tool | Priority |
|--------|------|----------|
| `.cursor/rules/devkit-context.mdc` | Cursor | HIGH — `.cursorrules` is deprecated |
| `.kiro/steering/identity.md` | AWS Kiro | MEDIUM — new AWS AI IDE |
| `.roorules` | Roo Code (VS Code extension) | MEDIUM — significant user base |
| `.github/instructions/*.instructions.md` | GitHub Copilot (new format) | LOW — additive, not replacement |

**`.cursor/rules/devkit-context.mdc` format:**
```markdown
---
description: devkit identity and context
alwaysApply: true
---

<!-- Generated by devkit. Do not edit. -->
[content here]
```
Keep `.cursorrules` as a compatibility fallback for users on older Cursor versions.

---

### OSS Failure Modes to Avoid

Analysis of why similar tools abandoned/stalled, mapped to devkit's specific exposure:

**Failure 1: "Works for author" — bad first-run experience**
*Exposure: HIGH.*
Current state: user must edit 4+ files before first `devkit generate` produces useful output. If the scaffolded files contain only placeholder text, the output is useless, and the user concludes "this doesn't work."
*Fix:* `devkit wizard` (Tier 2). Short-term: make scaffold templates contain real example content, not TODO placeholders. Every scaffold file should look like a real example that someone would actually use, not a blank form.

**Failure 2: Single-maintainer abandonment**
*Exposure: LOW.*
devkit is explicitly a personal tool. There is no community to disappoint. The tool succeeds if it helps one user (you) for 5+ years. No growth ambitions = no growth failure.

**Failure 3: Scope creep / abstraction overload**
*Exposure: MEDIUM.*
Caliber added plugin systems, server modes, and team features. Most features are unused by 95% of users. devkit's defense: the "Rejected Ideas" section. Add to it aggressively. When in doubt, reject.

**Failure 4: Dependency rot**
*Exposure: LOW (actively mitigated).*
Current deps: cobra, gopkg.in/yaml.v3, testscript. govulncheck + trivy in CI. Dependabot weekly. go-licenses gate. This is already best-in-class for a Go tool of this size.

**Failure 5: Regeneration friction → tool becomes adversary**
*Exposure: HIGH — the most important failure mode.*
If running `devkit generate` becomes something users avoid (because it wipes manual edits, is slow, requires remembering to run it), the tool is dead even if it's installed. Three defenses:
1. Content-block preservation (Tier 1 steal #3) — don't wipe manual additions
2. `devkit watch` (Tier 2 steal #11) — auto-regenerate, never think about it
3. `devkit diff --check` (Tier 1 steal #2) — CI catches forgotten regenerations
**Build at least one of these in Milestone 2.5.**

**Failure 6: Config format churn killing users**
*Exposure: MEDIUM.*
`.cursorrules` is already deprecated. Windsurf's future is uncertain. If devkit writes to a format that disappears, generated files become clutter.
*Fix:* `extra_targets` in workspace.yaml lets users opt out. `devkit doctor` can warn about dead formats. Monitor ecosystem and deprecate stale targets.

**Failure 7: "Too much to fill in" paralysis**
*Exposure: MEDIUM.*
If identity files look like a homework assignment, users skip them. The value of devkit is proportional to the quality of identity/context files. If those are empty, devkit is just a file copier.
*Fix:* wizard (Tier 2). Short-term: scaffold templates should have inline comments explaining what good content looks like, with examples. Not TODOs — examples.

---

### Existential Threats (18-month horizon)

**Threat 1: Anthropic ships native CLAUDE.md management**
*Probability: LOW-MEDIUM. Impact: MEDIUM.*
Claude Code already reads CLAUDE.md. If Anthropic builds a UI for editing it and syncing across projects, devkit's generate flow is redundant for Claude-only users. devkit's defense: AGENTS.md + multi-tool support. Anthropic won't build Cursor/Copilot/Windsurf integrations.

**Threat 2: SKILL.md / Agent Skills becomes universal**
*Probability: HIGH. Impact: HIGH.*
The SKILL.md ecosystem (skills as markdown in `~/.claude/skills/`) is Anthropic-originated and now adopted by 35+ tools. If every AI tool can reference personal skills from `~/.claude/skills/`, and skills include context/identity sections, devkit's value proposition weakens.
*Mitigation:* devkit should generate `~/.claude/skills/devkit-context.md` as part of its output. Not a breaking change — additive. Adds devkit's composed content to the skills ecosystem. Timeline: add in Milestone 2.5.

**Threat 3: IDE-native context management**
*Probability: MEDIUM. Impact: HIGH.*
JetBrains, VS Code, or Cursor could ship a settings UI for managing AI context. Would make file-based approaches feel old. devkit's defense: it's not IDE-specific. The private `~/.devkit/` data model works regardless of IDE.

**Threat 4: MCP kills static files**
*Probability: LOW (18-month horizon). Impact: MEDIUM.*
If AI tools move entirely to live MCP resource queries and stop reading CLAUDE.md, static files become stale. Static files will remain primary for at least 2 years given how many tools are file-based. `devkit serve` is the long-term hedge.

**Threat 5: `.cursorrules` deprecation cascade**
*Probability: HIGH (already happening). Impact: LOW.*
Cursor deprecated `.cursorrules`. Windsurf may follow. Each deprecation = one less reason for a new user to install devkit.
*Mitigation:* add replacement formats as they emerge. `extra_targets` covers long tail. `.cursor/rules/devkit-context.mdc` should be in Milestone 2.5.

---

### Revised Priority Order for Milestone 2.5

Incorporating competitive intelligence into the original Milestone 2.5 plan:

| Priority | Feature | Why |
|----------|---------|-----|
| 1 | `opencode.json` fix | **DONE** — was wrong format (TOML vs JSON) |
| 2 | Atomic writes in generator | Low-cost correctness fix, prevents partial-write corruption |
| 3 | `extra_targets` in workspace.yaml | Enables new tool support without a release; needed for .mdc, .kiro, .roorules |
| 4 | `.cursor/rules/devkit-context.mdc` | `.cursorrules` is deprecated NOW — add replacement before new users hit stale docs |
| 5 | `projects.txt` registry + `--all` | Makes multi-project update a one-liner; required for `devkit status` and `devkit diff` |
| 6 | `devkit status` | Hash-based sync detection — shows which projects need regeneration |
| 7 | `devkit diff <path>` + `--check` | CI integration; solve "forgot to regenerate" problem |
| 8 | Non-destructive `devkit reset` | Safety; one user incident will make you regret this not existing |
| 9 | `devkit context ls` | Quality of life, small scope |
| 10 | `devkit doctor` | Stale detection; leverages projects registry |
| 11 | `identity/telos.md` scaffold | Add to init scaffold; optional but valuable long-term |
| 12 | `devkit lint` | Pre-generate validation; catches bad frontmatter and oversized files |
| 13 | `devkit search --interactive` | fzf + go-fuzzyfinder fallback |
| 14 | `~/.claude/skills/devkit-context.md` output | Hedge against SKILL.md ecosystem threat |
| 15 | `devkit sync` | git pull/push on ~/.devkit/ — multi-machine |

**Not in Milestone 2.5 (defer to 3 or v2):**
- `devkit wizard` — high value but high effort; defer until "first-run problem" is confirmed painful
- `devkit watch` — adds fsnotify dep; build when "forgot to regenerate" is daily pain
- `devkit serve` (MCP) — still a v2 item; static files dominate for 12-18 months
- Content-block preservation — defer until "someone edits CLAUDE.md manually" is real problem
- `devkit import` — nice but edge case; manual file copying is fine
- `devkit scan` — useful but complex; defer to v2
- `devkit backup` — build when findings > 50

---

## Next Steps

1. **Use daily** — Milestones 2.5 and 2.75 are complete. Let real daily use surface friction before building more.
2. **Milestone 3** — `devkit archive` + `devkit backup` with age encryption (when findings > 50)
3. **v2** — `devkit serve` (MCP server mode) + `devkit watch` (fsnotify auto-regen)

---

## Post-Audit Action Items (2026-06-18)

Full engineering audit performed with 7 specialized review agents. Findings below are prioritized for immediate action.

### Critical (Fix Immediately)

| # | Issue | File | Fix |
|---|-------|------|-----|
| 1 | `append(MarkdownTargets, ws.ExtraTargets...)` mutates package-level slice | `generator.go:47`, `generate.go:181` | Use `slices.Concat` or `make([]string, 0, cap) + append` |
| 2 | `extra_targets` path traversal — can write to `../../.ssh/authorized_keys` | `generator.go:48-49` | Validate `filepath.Clean(resolved)` stays within `targetDir` |
| 3 | `active_context` path traversal — can read `../../etc/file.md` | `devctx.go:39` | Validate resolved path stays within `contexts/` dir |
| 4 | `trufflehog@main` in CI — mutable branch = supply chain attack | `ci.yaml:80` | Pin to commit SHA |
| 5 | `trivy-action@master` in CI — same class | `deps.yaml` | Pin to commit SHA |

### High (Fix This Week)

| # | Issue | File | Fix |
|---|-------|------|-----|
| 6 | `--dry-run --all` panics (args[0] accessed before --all check) | `generate.go:78-84` | Add guard or handle combination |
| 7 | No `devkit version` command, no version in binary | `main.go` | Add ldflags `-X main.version={{.Version}}` in goreleaser |
| 8 | `~/.devkit/` created with 0755 (world-readable) | `init.go:33` | Change to 0700; files to 0600 |
| 9 | Missing default targets: `CONVENTIONS.md`, `.claude/rules/devkit-context.md`, `.kiro/steering/identity.md` | `generator.go` | Add to MarkdownTargets |
| 10 | All GitHub Actions pinned to tags not SHA | All workflow files | Pin to commit SHA |
| 11 | `writeSkillsFile` bypasses FS interface | `generate.go:156-172` | Refactor to accept FS parameter |

### Medium (Fix This Month)

| # | Issue | Fix |
|---|-------|-----|
| 12 | Duplicated MDC frontmatter string in generator.go vs diff.go | Extract to shared constant |
| 13 | OsFS atomic write uses fixed temp name (race on concurrent access) | Use `os.CreateTemp` in same dir |
| 14 | MemFS returns zero ModTime (breaks doctor testability) | Add `ModTimes map[string]time.Time` |
| 15 | No cross-platform CI testing (binary ships for 6 platforms, tested on 1) | Add macOS/Windows to matrix |
| 16 | Repeated pipeline wiring across 5 commands | Extract `resolveComposed()` helper |
| 17 | Consolidate `status` and `doctor` into one command | Content comparison is canonical |
| 18 | Add `devkit untrack <path>` | Essential quality-of-life for registry |
| 19 | Add .gitignore check — warn if generated files not ignored | Prevent private context leakage |
| 20 | `trufflehog --only-verified` too lenient | Remove flag or use `verified,unverified` |
| 21 | goreleaser `version: latest` in release workflow | Pin to specific version |
| 22 | Makefile build flags inconsistent with goreleaser | Add `-trimpath -ldflags="-s -w"` |

### Low (Backlog)

| # | Issue | Fix |
|---|-------|-----|
| 23 | No `--verbose` / debug flag for troubleshooting | Add to root command |
| 24 | No test coverage reporting in CI | Add `-coverprofile` + upload |
| 25 | No `mise.toml` for local dev tool versions | Document Go + golangci-lint versions |
| 26 | `tools.md` scaffold will never be filled | Remove from scaffold or auto-populate |
| 27 | `registry.ReadAll` treats permission error as "no file" | Check specific error type |
| 28 | No `Remove` on FS interface | Add when cleanup features are needed |
| 29 | `reset.go` reads `os.Stdin` directly | Use `cmd.InOrStdin()` for testability |
| 30 | Consider per-target `TargetSpec` for format divergence | Future-proof for size limits/filters |

### Architecture Decisions Confirmed by Audit

These decisions were challenged and validated:
- **Flat root package** — correct for current size (~10 commands). Revisit at ~15.
- **FS interface level of abstraction** — appropriate. Minimal surface, enables testing.
- **No plugin system** — correct. `extra_targets` + template system covers extensibility needs.
- **projects.txt plaintext format** — acceptable. Add JSON only if metadata becomes needed.
- **Composition order (identity → context → donts)** — correct. Donts last maximizes LLM attention weight.
- **Single binary with embedded templates** — strong distribution story. No runtime deps.
- **Atomic writes** — correctly implemented (write-then-rename). Fix temp name uniqueness.

### Architecture Decisions Challenged by Audit

These need re-evaluation:
- **"Same content everywhere" for markdown targets** — will break as tools diverge. Plan a `TargetSpec` with optional max-size, filter, prefix/suffix before it becomes urgent.
- **Separate `doctor` and `status` commands** — confusing. Both detect staleness differently (mtime vs content). Consolidate into one canonical approach (content comparison).
- **Regeneration as core workflow** — the fundamental friction source. `devkit watch` (fsnotify) or `devkit serve` (MCP) eliminates it. Prioritize one for Milestone 3.
- **16KB warning threshold** — may never trigger in practice. Most users will have 3-8KB composed output. Consider raising to 24KB or removing the warning (keep 32KB hard limit).

### Test Gaps to Close

Priority test additions (from QA audit):
1. `generate_dry_run_all.txtar` — currently panics
2. `doctor_stale.txtar` — no test for stale detection
3. `lint_errors.txtar` — no test for lint failure path
4. `status_stale.txtar` — no test for stale state
5. `pipeline_full_lifecycle.txtar` — init → generate → edit → status → generate --all
6. Unit test: verify `append` doesn't mutate `MarkdownTargets` global
7. Unit test: path traversal in extra_targets is rejected
8. Unit test: MemFS with custom ModTime for doctor testing
