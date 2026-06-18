# GitHub Multi-Account Setup on macOS

Use two GitHub accounts (e.g. work and personal) on one machine via SSH host aliases and git config conditionals.

---

## Assumptions

| Slot | Value (fill in yours) |
|------|----------------------|
| `<work-username>` | your work GitHub username |
| `<work-email>` | your work email |
| `<personal-username>` | your personal GitHub username |
| `<personal-email>` | your personal email |
| `<personal-dir>` | local directory for personal repos, e.g. `~/personal/` |

---

## Step 1 — Check for an existing SSH key

```sh
ls ~/.ssh/
```

If you see `id_ed25519` (or `id_rsa`), check which account it belongs to:

```sh
cat ~/.ssh/id_ed25519.pub
# email at the end tells you which account it's for
```

- If it belongs to your **work** account: reuse it, skip generating a new work key.
- If it belongs to your **personal** account: reuse it as `id_personal`, skip generating a new personal key.
- If neither: generate both below.

---

## Step 2 — Generate missing SSH key(s)

Only run what you need:

```sh
# Work key (skip if id_ed25519 is already your work key)
ssh-keygen -t ed25519 -f ~/.ssh/id_work -C "<work-email>"

# Personal key
ssh-keygen -t ed25519 -f ~/.ssh/id_personal -C "<personal-email>"
```

Hit Enter for no passphrase, or set one for extra security.

---

## Step 3 — Configure ~/.ssh/config

Open (or create) `~/.ssh/config` and add:

```
# Work - <work-username>
Host github.com
  HostName github.com
  User git
  AddKeysToAgent yes
  UseKeychain yes
  IdentityFile ~/.ssh/id_ed25519        # or id_work if you generated a new one

# Personal - <personal-username>
Host github.com-personal
  HostName github.com
  User git
  AddKeysToAgent yes
  UseKeychain yes
  IdentityFile ~/.ssh/id_personal
  IdentitiesOnly yes                    # critical: prevents agent offering wrong key
```

> `IdentitiesOnly yes` on the personal block forces SSH to use only `id_personal`,
> ignoring any keys already loaded in the ssh-agent.

---

## Step 4 — Add public keys to GitHub

Print each public key and add it to the corresponding GitHub account:

```sh
# Work key → github.com/<work-username> → Settings → SSH keys → New SSH key
cat ~/.ssh/id_ed25519.pub

# Personal key → github.com/<personal-username> → Settings → SSH keys → New SSH key
cat ~/.ssh/id_personal.pub
```

---

## Step 5 — Configure ~/.gitconfig

Set work as the global default, override for personal directories:

```ini
[user]
    name = <work-username>
    email = <work-email>

[includeIf "gitdir:<personal-dir>"]
    path = ~/.gitconfig-personal
```

---

## Step 6 — Create ~/.gitconfig-personal

```ini
[user]
    name = <personal-username>
    email = <personal-email>
```

Git automatically uses this file for any repo under `<personal-dir>`.

---

## Step 7 — Verify both connections

```sh
ssh -T git@github.com            # → Hi <work-username>!
ssh -T git@github.com-personal   # → Hi <personal-username>!
```

If the personal one still shows the wrong account, run the verbose test to see which key is being offered:

```sh
ssh -vT git@github.com-personal 2>&1 | grep "Offering\|Server accepts"
```

Most common fix: `IdentitiesOnly yes` missing from the personal host block (Step 3).

---

## Step 8 — Clone repos with the right alias

```sh
# Work repo (default host)
git clone git@github.com:<work-username>/repo.git

# Personal repo (use the alias)
git clone git@github.com-personal:<personal-username>/repo.git
```

For an existing local repo, set the remote:

```sh
git remote add origin git@github.com-personal:<personal-username>/repo.git
# or update an existing remote:
git remote set-url origin git@github.com-personal:<personal-username>/repo.git
```

---

## Commit email verification

Inside a personal repo:

```sh
git config user.email   # should print <personal-email>
```

Inside a work repo:

```sh
git config user.email   # should print <work-email>
```
