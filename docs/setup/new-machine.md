# Setting Up devkit on a New Machine

This guide covers installing devkit and getting your identity and context files onto a new machine.

---

## 1. Install Go

### With mise (recommended — cross-platform, version-pinned)

```bash
# Install mise
curl https://mise.run | sh

# Install Go
mise use --global go@latest
```

### With Homebrew (macOS)

```bash
brew install go
```

Verify:

```bash
go version
```

---

## 2. Install devkit

```bash
go install github.com/dklinux7/devkit@latest
```

Verify the binary is on your PATH:

```bash
devkit --help
```

If `devkit` is not found, add Go's bin directory to your PATH:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Add this to your `~/.zshrc` or `~/.bashrc` to make it permanent.

---

## 3. First-time setup (new machine with no existing `~/.devkit/`)

```bash
devkit init
```

This scaffolds `~/.devkit/` with starter template files. Then edit:

```
~/.devkit/identity/ai.md          ← how AI should behave with you
~/.devkit/identity/engineering.md  ← your coding style, git workflow, preferences
~/.devkit/donts.md                ← things AI must never do
~/.devkit/contexts/work.md        ← your company, repos, tools, team
```

Then generate AI config for a project:

```bash
devkit generate ~/projects/my-app
```

---

## 4. Multi-machine sync with a private git repo

The simplest way to keep `~/.devkit/` in sync across machines is a private git repository.

### Initial setup (on first machine)

```bash
# Initialize git in your devkit data directory
git init ~/.devkit/
cd ~/.devkit/

# Create a .gitignore if needed
echo ".devkit-tmp*" > .gitignore

# Create a private repo on GitHub (or GitLab, Bitbucket, etc.)
# Then add the remote:
git remote add origin git@github.com:<your-username>/devkit-data.git

# Commit and push
git add -A
git commit -m "init: devkit identity and context"
git push -u origin main
```

### On a new machine

```bash
# Clone to ~/.devkit/
git clone git@github.com:<your-username>/devkit-data.git ~/.devkit/

# Verify
devkit context ls
```

### Day-to-day sync

After editing files in `~/.devkit/`, sync across machines with:

```bash
devkit sync
```

This runs `git pull --rebase && git push` on `~/.devkit/`. Requires the private git repo to be set up as described above.

---

## 5. Generate AI config files for your projects

Once `~/.devkit/` is populated, generate config files for any project:

```bash
devkit generate ~/projects/my-app
```

This writes:
- `CLAUDE.md` — Claude Code
- `AGENTS.md` — all AGENTS.md-compatible tools
- `GEMINI.md` — Gemini CLI
- `CONVENTIONS.md` — Aider
- `.cursorrules` — Cursor (legacy)
- `.cursor/rules/devkit-context.mdc` — Cursor (current)
- `.windsurfrules` — Windsurf
- `.github/copilot-instructions.md` — GitHub Copilot
- `.claude/rules/devkit-context.md` — Claude Code scoped rules
- `.kiro/steering/identity.md` — AWS Kiro

### Regenerate all tracked projects at once

After updating your identity or context files:

```bash
devkit generate --all
```

---

## 6. Useful commands

```bash
devkit status              # content comparison: in-sync vs stale
devkit doctor              # mtime-based stale detection (faster than status)
devkit diff ~/project      # see what generate would change; --check exits 1 if anything would change
devkit lint                # validate your ~/.devkit/ source files
devkit context ls          # list contexts with size and date
devkit search "query"      # search across all ~/.devkit/ markdown; --interactive for fzf UI
devkit sync                # sync ~/.devkit/ across machines (git pull --rebase + push)
devkit untrack ~/project   # remove a project from the tracking registry
devkit generate --all      # regenerate all tracked projects at once
devkit version             # print current version
```
