# devkit

Personal dev workspace generator. Composes your identity, constraints, and project context into AI config files for any coding tool.

## Install

```sh
go install github.com/dklinux7/devkit@latest
```

## Usage

**Setup**
```sh
devkit init                    # scaffold ~/.devkit/ with starter templates
devkit reset                   # scaffold missing files (non-destructive); --hard to wipe and reinit
devkit sync                    # git pull + push on ~/.devkit/ (multi-machine sync)
```

**Generate**
```sh
devkit generate <path>         # write AI config files to a project
devkit generate --all          # regenerate all tracked projects
  # flags: --dry-run, --include-lessons, --force, --quiet
devkit diff <path>             # show what generate would change
  # --check exits 1 if anything would change (for CI)
```

**Inspect**
```sh
devkit status                  # content comparison: in-sync vs stale
devkit doctor                  # mtime comparison: faster stale detection
devkit lint                    # validate ~/.devkit/ source files
devkit context ls              # list contexts with size and last-modified date
devkit search "query"          # search across all ~/.devkit/ markdown; --interactive for fzf UI
```

**Manage**
```sh
devkit untrack <path>          # remove a project from the tracking registry
devkit version                 # print version
```

Global flag: `--verbose` / `-v` — print debug info (data dir, active context, composed size).

Run `devkit help` for full details.

## What it does

You maintain markdown files describing how you work. devkit composes them and writes config files that Claude Code, Cursor, Copilot, Windsurf, Gemini CLI, OpenCode, Aider, and AWS Kiro all understand.

One source of truth → every AI tool gets the same context.

Default targets written to every project:
- `CLAUDE.md` — Claude Code
- `AGENTS.md` — universal (Linux Foundation standard, 60k+ projects)
- `GEMINI.md` — Gemini CLI
- `CONVENTIONS.md` — Aider
- `.cursorrules` — Cursor (legacy compat)
- `.cursor/rules/devkit-context.mdc` — Cursor (current format)
- `.windsurfrules` — Windsurf
- `.github/copilot-instructions.md` — GitHub Copilot
- `.claude/rules/devkit-context.md` — Claude Code scoped rules
- `.kiro/steering/identity.md` — AWS Kiro

Also generates `opencode.json` and `.claude/settings.json` from templates when present. If context frontmatter includes `mcp_servers`, also generates `.mcp.json`.

## Not licensed

This project is not licensed for use, modification, or distribution.
Source code is publicly visible for transparency and reference only.
All rights reserved.
