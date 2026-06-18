# devkit — Personal Dev Workspace Design Document

**Status:** ACTIVE — Milestone 2 complete, Milestone 2.5 in progress.
**Date started:** 2026-06-18
**Design locked:** 2026-06-18
**Last updated:** 2026-06-18 (AI tool landscape research, plug-and-play decisions, roadmap expanded)
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
.cursorrules
.windsurfrules
copilot-instructions.md   (written to .github/copilot-instructions.md)
GEMINI.md
```

### Default structured targets (template-rendered)

```
opencode.json
.mcp.json                  ← project-level MCP server config (Milestone 2.5)
.claude/settings.json
```

### AI tool config file reference

| Tool | Files read | Format |
|------|-----------|--------|
| Claude Code | `CLAUDE.md`, `.claude/rules/*.md`, `AGENTS.md` | Markdown |
| GitHub Copilot | `.github/copilot-instructions.md`, `AGENTS.md` | Markdown |
| Cursor | `.cursorrules`, `.cursor/rules/*.mdc` | Markdown + YAML frontmatter |
| Windsurf | `.windsurfrules` | Markdown |
| Gemini CLI | `GEMINI.md`, `~/.gemini/settings.json` | Markdown + JSON |
| OpenCode | `opencode.json`, `AGENTS.md` | JSONC |
| Aider | `.aider.conf.yml`, `CONVENTIONS.md` | YAML + Markdown |
| Continue.dev | `~/.continue/config.yaml` | YAML |
| Zed | `settings.json` (`context_servers` key) | JSON |
| Devin | `AGENTS.md`, `.devin/rules/*.md` | Markdown |
| Warp | `AGENTS.md` | Markdown |
| JetBrains Junie | `AGENTS.md` | Markdown |

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
defaultTargets := []string{"CLAUDE.md", "AGENTS.md", "GEMINI.md", ".cursorrules", ".windsurfrules",
    ".github/copilot-instructions.md"}
allTargets := append(defaultTargets, workspace.ExtraTargets...)
for _, name := range allTargets {
    writeFile(filepath.Join(target, name), content)
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
- **Path traversal**: `active_context` is trusted user input (user owns ~/.devkit/). No privilege escalation possible.
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
internal/
├── fs/           ← filesystem interface (testable with in-memory)
├── config/       ← loads workspace.yaml, resolves data dir via os.UserHomeDir()
├── devctx/       ← loads identity/*, contexts/<active>, donts.md (NOT "context" — avoids stdlib shadow)
├── composer/     ← composition order, frontmatter stripping, \n\n separators, size enforcement
├── generator/    ← write markdown targets + render template targets to target path
├── search/       ← ripgrep with Go-native fallback
├── registry/     ← read/write ~/.devkit/projects.txt (Milestone 2.5)
└── mcp/          ← parse mcp_servers from context frontmatter, render .mcp.json (Milestone 2.5)

cmd/devkit/
├── main.go       ← root command (cobra)
├── init.go       ← scaffolds ~/.devkit/
├── reset.go      ← delete + re-initialize with confirmation
├── generate.go   ← the core command (--dry-run, --include-lessons, --force, --all, --git-context)
├── search.go     ← search entry point (--interactive flag)
├── context.go    ← devkit context ls (Milestone 2.5)
├── doctor.go     ← stale detection (Milestone 2.5)
├── sync.go       ← git pull/push on ~/.devkit/ (Milestone 2.5)
└── serve.go      ← MCP server mode (v2)
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

Note: `Glob` needed for `identity/*` file discovery. `embed` directives use `//go:embed all:templates` (the `all:` prefix is required to capture dotfiles like `.cursorrules` in scaffold templates).

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
- [ ] `extra_targets` in workspace.yaml — user-defined additional output files, merged with defaults in generator
- [ ] `~/.devkit/projects.txt` registry — append on every `devkit generate`, read by `--all`
- [ ] `devkit generate --all` — regenerate all tracked project paths
- [ ] `devkit doctor` — compare mtime of identity/context sources vs generated files, print stale projects
- [ ] `devkit context ls` — list contexts/ with size + last-modified date
- [ ] `devkit search --interactive` — fzf via exec.LookPath, go-fuzzyfinder as fallback
- [ ] `devkit sync` — git pull/push wrapper on ~/.devkit/
- [ ] `.mcp.json` generation — parse `mcp_servers` from context frontmatter, render to project
- [ ] `GEMINI.md` added to default markdown targets
- [ ] `docs/setup/new-machine.md` — private git repo for ~/.devkit/ + mise setup guide

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

## Resolved Design Questions (from final review)

| Question | Resolution |
|----------|-----------|
| macOS path vs os.UserConfigDir() | Use `os.UserHomeDir()` + ".devkit" everywhere. Drop UserConfigDir. |
| opencode.toml gets markdown | Separate markdown targets (direct write) from structured targets (template render). |
| No separators between files | `\n\n` between every composed section. Strip frontmatter from sources. |
| 16KB too tight | Warn at 16KB, fail at 32KB. |
| donts.md ordering | Moved to LAST position (LLMs weight end-of-prompt more). |
| Overwrite without warning | Print which files are overwritten. Header is the contract. |
| ripgrep runtime dep | Go-native fallback (WalkDir + regexp); rg used when available for speed. |
| `context` package shadows stdlib | Renamed to `devctx`. |
| embed misses dotfiles | Use `//go:embed all:templates`. |
| $DEVKIT_HOME vs XDG priority | $DEVKIT_HOME wins unconditionally on all platforms. |
| License enforcement | `go-licenses` in CI. |
| ripgrep flag injection | `--` separator before query arg. |
| Private data leakage | Generated file header warns "contains private context". |
| Prompts "first-class" but unused | Demoted to "reference directory". v2 candidate for CLI integration. |
| findings/analyzed orphaned in v1 | Manual-use-only. No command manages them in v1. |

---

## Next Steps

1. **Pre-work** — create repo, Go module, CI, templates
2. **Q&A** — fill identity/ and donts.md content
3. **Milestone 1** — `devkit generate` working end-to-end
4. **Use daily for a week** — let real friction guide what's next
5. **Milestone 2-3** — only when pain demands
