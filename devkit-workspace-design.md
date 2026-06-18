# devkit — Personal Dev Workspace Design Document

**Status:** DESIGN LOCKED — implementation ready.
**Date started:** 2026-06-18
**Design locked:** 2026-06-18
**Final review:** 2026-06-18 (all issues from analysis resolved below)
**Goal:** Make you + AI maximally productive. Not portable, not generic, not elegant — productive.

---

## Problem Statement

Current workspace is tightly coupled to one employer. Need a personal workspace that:
- Works with any AI coding tool (Claude Code, OpenCode, Cursor, Copilot, Windsurf)
- Works at any company (swap context, not rebuild)
- Works on macOS, Linux, Windows
- Philosophy: install binary → `devkit init` → `devkit generate ~/project` → done

---

## Core Principles

> 1. Canonical format is markdown. All AI tools read it.
> 2. Zero known vulnerabilities. Clean deps at all times.
> 3. Public repo = the tool. User data = private, separate.
> 4. workspace.yaml = 2 fields. Everything else in markdown.
> 5. Hard output limit: >16KB warns, >32KB fails (unless --force).
> 6. If it doesn't help solve a production problem, understand a codebase, or use AI better within 30 days — don't build it in v1.

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

## workspace.yaml — Final Schema

```yaml
name: "Your Name"
active_context: work
```

That's it. 2 fields.

- `name` — used in templates
- `active_context` — which context file/folder to load from `contexts/`

**What's NOT here:** ai_tools (generate everything — disk is free), integrations (belong in context prose), role/languages (in identity/ai.md).

**Switching context:** Edit `active_context`, run `devkit generate`. One line edit.

---

## Go CLI — `devkit`

### v1 Commands (all that's needed)

| Command | Purpose |
|---------|---------|
| `devkit init` | Creates ~/.devkit/, scaffolds from templates |
| `devkit generate <path>` | Reads identity + context + donts → writes CLAUDE.md, AGENTS.md, .cursorrules, etc. directly to target |
| `devkit generate --dry-run` | Show what would be generated |
| `devkit generate --include-lessons` | Append lessons at end of output |
| `devkit generate --force` | Bypass 32KB hard limit |
| `devkit search <query>` | Search across all ~/.devkit/ markdown (uses ripgrep if available, Go-native fallback otherwise) |

### v2+ Commands (build only when pain demands)

| Command | Purpose |
|---------|---------|
| `devkit archive` | AI-summarize findings → lessons, compress originals |
| `devkit backup` | Encrypted tarball of ~/.devkit/ |

### Removed commands (not needed for solo user)

| Command | Why |
|---------|-----|
| `devkit switch` | Edit one YAML line + generate. Not worth a command. |
| `devkit validate` | You know if your tools work. |
| `devkit doctor` | Clear error messages suffice. |
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

// Markdown targets — write content directly
for _, name := range []string{"CLAUDE.md", "AGENTS.md", ".cursorrules", ".windsurfrules", "copilot-instructions.md"} {
    writeFile(filepath.Join(target, name), content)
}

// Structured targets — render through templates (text/template)
for _, tmpl := range []string{"opencode.toml", ".claude/settings.json"} {
    if templateExists(tmpl) {
        rendered := renderTemplate(tmpl, workspace, content)
        writeFile(filepath.Join(target, tmpl), rendered)
    }
}
```

Every generated file starts with a header line so you never wonder "should I edit this directly?" The answer is always: edit the source in `~/.devkit/`, then regenerate.

**Overwrite behavior:** If a target file exists and differs from what would be generated, print which files are being overwritten. The header ("do not edit") is the contract — if you edited it manually, regeneration replaces it.

Tool-specific configs (Claude's settings.json, OpenCode's opencode.toml) are Go `text/template` files in the public repo's `templates/` dir. `devkit generate` renders them from workspace.yaml + composed content. No interface, no adapter pattern — just template files + a render loop.

If a new AI tool appears: add one template file + one filename to the loop. 5 minutes of work.

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
├── config/       ← loads workspace.yaml (2 fields), resolves data dir via os.UserHomeDir()
├── devctx/       ← loads identity/*, contexts/<active>, donts.md (NOT "context" — avoids stdlib shadow)
├── composer/     ← composition order, frontmatter stripping, \n\n separators, size enforcement
├── generator/    ← write markdown targets + render template targets to target path
└── search/       ← ripgrep with Go-native fallback

cmd/devkit/
├── main.go       ← root command (cobra)
├── init.go       ← scaffolds ~/.devkit/
├── generate.go   ← the core command
└── search.go     ← search entry point
```

No model/ package needed — the types are simple enough to live in each package. No adapter interface — it's a for-loop over filenames. No validator, no doctor, no archive in v1.

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

### Pre-work: Repo setup
- [ ] Create GitHub repo `devkit` (PUBLIC, no license)
- [ ] Initialize Go module (go 1.22+)
- [ ] Set up CI (lint, test, govulncheck, trivy, go-licenses)
- [ ] Configure Dependabot
- [ ] Add trufflehog check
- [ ] Write scaffold templates (identity, context, prompts, tools, research, analysis) — use `//go:embed all:templates` for dotfiles
- [ ] Public repo .gitignore

### Milestone 1: `devkit generate` working (USE IT FOR A WEEK)
- [ ] `internal/fs` — interface (with Glob) + real implementation
- [ ] `internal/config` — load workspace.yaml, resolve data dir via `os.UserHomeDir()` + ".devkit", `$DEVKIT_HOME` override
- [ ] `internal/devctx` — load identity/*.md + contexts/<active> + donts.md (strip frontmatter from all sources)
- [ ] `internal/composer` — composition order (identity → context → donts), `\n\n` separators, size check (16KB warn, 32KB fail)
- [ ] `internal/generator` — markdown targets (direct write) + structured targets (template render), overwrite reporting
- [ ] `cmd/devkit/init.go` — scaffold ~/.devkit/ from embedded templates
- [ ] `cmd/devkit/generate.go` — the core command (--dry-run, --include-lessons, --force)
- [ ] `cmd/devkit/main.go` — cobra root
- [ ] **USE IT DAILY. Let friction guide what to build next.**

### Milestone 2: Search + polish
- [ ] `internal/search` — Go-native fallback (filepath.WalkDir + regexp), ripgrep when available (exec.Command with `--` separator)
- [ ] `cmd/devkit/search.go` — search entry point
- [ ] Binary releases (goreleaser OSS — macOS/Linux/Windows/arm64)
- [ ] README

### Milestone 3: Archive (when findings > 50)
- [ ] Summarizer interface (Manual + Claude implementations)
- [ ] Compression to archive/
- [ ] Encryption via `age`
- [ ] `devkit archive` command

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
