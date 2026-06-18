# devkit E2E Testing Guide

Use this guide to manually test every command and feature end-to-end. Run through this in order — each section builds on the previous one.

---

## 0. Prerequisites

### Install the binary

```sh
cd ~/personal/devkit
go install .
devkit version          # should print a version string, not panic
```

### Fill in ~/.devkit/

`devkit init` creates stubs. Fill them in before testing:

**~/.devkit/workspace.yaml**
```yaml
name: "Your Name"
active_context: work
```

**~/.devkit/identity/ai.md** — your AI collaboration preferences  
**~/.devkit/identity/engineering.md** — your engineering principles  
**~/.devkit/contexts/work.md** — your current work context (projects, goals, stack)  
**~/.devkit/donts.md** — things you don't want AI assistants to do

None of these need to be long — a few lines each is enough to verify composition works.

---

## 1. Validate source files

```sh
devkit lint
```

Expected: exits 0, no errors. If you get errors, check workspace.yaml syntax and fix before continuing.

```sh
devkit context ls
```

Expected: lists `work` with size and last-modified date.

---

## 2. First generate

Pick any test project directory (create one if needed):

```sh
mkdir -p /tmp/devkit-test-project
devkit generate --dry-run /tmp/devkit-test-project
```

Expected: prints what would be written, exits 0, nothing written to disk.

```sh
devkit generate /tmp/devkit-test-project
```

Expected: prints `✓ Generated /tmp/devkit-test-project` and registers the project.

### Verify written files

```sh
ls /tmp/devkit-test-project/
```

Expected files:
- `CLAUDE.md`
- `AGENTS.md`
- `GEMINI.md`
- `CONVENTIONS.md`
- `.cursorrules`
- `.cursor/rules/devkit-context.mdc`
- `.windsurfrules`
- `.github/copilot-instructions.md`
- `.claude/rules/devkit-context.md`
- `.kiro/steering/identity.md`

Spot-check one file:
```sh
cat /tmp/devkit-test-project/CLAUDE.md
```

Expected: contains your identity and context content, not empty.

---

## 3. Multi-project + registry

```sh
mkdir -p /tmp/devkit-test-project-2
devkit generate /tmp/devkit-test-project-2
```

```sh
devkit status
```

Expected: both projects show `✓ in-sync`.

```sh
devkit doctor
```

Expected: both projects show `✓ up-to-date`.

---

## 4. Stale detection

Touch a source file to make it newer than the generated outputs:

```sh
touch ~/.devkit/identity/ai.md
```

```sh
devkit doctor
```

Expected: both projects show `✗ stale` (mtime-based — source file is newer than CLAUDE.md).

Now edit identity/ai.md to actually change the content:

```sh
echo "" >> ~/.devkit/identity/ai.md
```

```sh
devkit status
```

Expected: both projects show `✗ stale` (content-comparison — generated output no longer matches).

### Regenerate all

```sh
devkit generate --all
```

Expected: regenerates both projects, both show `✓ Generated`.

```sh
devkit status
devkit doctor
```

Expected: both commands show all projects clean.

---

## 5. Diff and CI flag

Edit an identity file to change the content:

```sh
echo "\n## Addendum\n\nTest content." >> ~/.devkit/identity/engineering.md
```

```sh
devkit diff /tmp/devkit-test-project
```

Expected: shows a diff of what would change in CLAUDE.md (and other targets), exits 0.

```sh
devkit diff --check /tmp/devkit-test-project
echo "Exit code: $?"
```

Expected: exits 1 because files are stale. This is what you'd use in CI.

After regenerating:

```sh
devkit generate /tmp/devkit-test-project
devkit diff --check /tmp/devkit-test-project
echo "Exit code: $?"
```

Expected: exits 0.

---

## 6. Search

```sh
devkit search "engineering"
```

Expected: shows matching lines from your identity and context files.

```sh
devkit search "work"
```

Expected: matches from contexts/work.md (and any other files mentioning "work").

### Interactive search (requires fzf)

```sh
devkit search --interactive "test"
```

Expected: opens fzf UI. Navigate with arrows, Enter to select, Esc to quit.

If fzf is not installed: `Expected: error message saying fzf is not available` (graceful failure, not a panic).

---

## 7. Context switching

Create a personal context:

**~/.devkit/contexts/personal.md** — add a line or two about personal projects

Switch context:

```sh
# Edit ~/.devkit/workspace.yaml: change active_context to "personal"
devkit context ls
```

Expected: shows `personal` as the active context (or both contexts listed).

```sh
devkit generate --all
```

Expected: regenerates all projects with personal context content.

Spot-check:

```sh
grep -i "personal" /tmp/devkit-test-project/CLAUDE.md
```

Expected: your personal context content appears.

Switch back:

```sh
# Edit ~/.devkit/workspace.yaml: change active_context back to "work"
devkit generate --all
```

---

## 8. Extra targets

Add extra targets to workspace.yaml:

```yaml
name: "Your Name"
active_context: work
extra_targets:
  - MY_AI_RULES.md
```

```sh
devkit generate /tmp/devkit-test-project
ls /tmp/devkit-test-project/MY_AI_RULES.md
```

Expected: file exists with the same composed content.

Remove `extra_targets` from workspace.yaml when done.

---

## 9. MCP servers from context frontmatter

Add frontmatter to ~/.devkit/contexts/work.md:

```markdown
---
mcp_servers:
  filesystem:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
---

Your existing work context content here.
```

```sh
devkit generate /tmp/devkit-test-project
cat /tmp/devkit-test-project/.mcp.json
```

Expected: valid JSON with the `filesystem` server configured.

Remove the frontmatter when done (or keep if you actually use MCP).

---

## 10. Registry management (untrack)

```sh
devkit status
```

Note both projects are tracked. Now untrack one:

```sh
devkit untrack /tmp/devkit-test-project-2
devkit status
```

Expected: only `/tmp/devkit-test-project` appears.

Re-track by generating:

```sh
devkit generate /tmp/devkit-test-project-2
devkit status
```

Expected: both appear again.

---

## 11. Reset (non-destructive)

Add a custom file to ~/.devkit/:

```sh
echo "my custom file" > ~/.devkit/identity/custom.md
```

Run reset without --hard:

```sh
devkit reset
```

Expected: exits 0, does NOT overwrite custom.md, only creates any missing stub files.

Verify your custom file is untouched:

```sh
cat ~/.devkit/identity/custom.md
```

Expected: `my custom file` still there.

---

## 12. Sync (requires git remote)

This test requires ~/.devkit/ to be a git repo with a remote.

```sh
ls ~/.devkit/.git
```

If `.git` does not exist, skip this section (expected error: `~/.devkit is not a git repository`).

If it exists:

```sh
devkit sync
```

Expected: pulls then pushes, exits 0.

---

## 13. Verbose flag

```sh
devkit --verbose generate /tmp/devkit-test-project
```

Expected: stderr shows `[debug]` lines including:
- data dir path
- active context name
- composed size in bytes

```sh
devkit -v status
```

Expected: same debug output before the status table.

The `--verbose` / `-v` flag works globally on all commands.

---

## Cleanup

```sh
rm -rf /tmp/devkit-test-project /tmp/devkit-test-project-2
```

---

## Pass criteria

| # | Command | Pass condition |
|---|---------|----------------|
| 1 | `devkit lint` | exits 0 |
| 1 | `devkit context ls` | lists active context |
| 2 | `devkit generate --dry-run` | no files written |
| 2 | `devkit generate` | all 10 target files present |
| 3 | `devkit status` | both projects in-sync |
| 3 | `devkit doctor` | both projects up-to-date |
| 4 | `devkit doctor` after touch | shows stale |
| 4 | `devkit status` after edit | shows stale |
| 4 | `devkit generate --all` | regenerates everything |
| 5 | `devkit diff` | shows diff |
| 5 | `devkit diff --check` | exits 1 when stale, 0 when clean |
| 6 | `devkit search` | returns matches |
| 7 | context switch + generate | new context content in output files |
| 8 | extra_targets | extra file written |
| 9 | mcp_servers frontmatter | .mcp.json written |
| 10 | `devkit untrack` | project removed from status |
| 11 | `devkit reset` | non-destructive (custom files preserved) |
| 12 | `devkit sync` | pull + push (if git remote exists) |
| 13 | `devkit --verbose` | [debug] lines on stderr |
