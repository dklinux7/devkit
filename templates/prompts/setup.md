# devkit setup

I need to fill in my `~/.devkit/` identity and context files so you have accurate context about how I work.

Interview me section by section using the questions below. After all sections are done, write the files.

---

## Section 1 — identity/ai.md (how AI should behave with me)

Ask me:
1. How do you prefer AI explanations — terse (just the answer) or contextual (answer + reasoning)?
2. What's your experience level and primary stack? Where are you less familiar?
3. When something is ambiguous, should AI ask you first, state an assumption and proceed, or give you options?
4. What actions require your explicit approval before AI takes them? (commits, deletes, PRs, shell commands, etc.)
5. Anything else about tone, communication style, or collaboration preferences?

---

## Section 2 — identity/engineering.md (how I write code and work)

Ask me:
1. How do you approach code style — minimal comments, strong naming, specific formatting rules?
2. What's your git workflow — commit format, branching strategy, PR size preference?
3. What's your architecture philosophy — when do you add abstraction? interfaces? packages?
4. How do you approach testing — unit vs integration, mocks vs real, coverage expectations?
5. How do you debug — what's your process when something breaks?
6. Any strong opinions about error handling, security, or dependencies?

---

## Section 3 — donts.md (hard constraints)

Ask me:
1. What should AI never do without your explicit confirmation?
2. What should AI never do, period — regardless of context?
3. Any code quality rules that are non-negotiable?

---

## Section 4 — contexts/work.md (current company and team)

Ask me:
1. What company do you work at and what does it do?
2. What is your role and which team are you on?
3. What are the key repos you work in and what does each do?
4. What is your tech stack — languages, databases, messaging, cloud, CI/CD, monitoring?
5. What are your team's conventions — PR process, deploy process, on-call, naming?
6. What are you actively working on right now?

---

## After the interview

Write all four files with the answers. Use this format:
- Plain markdown, no HTML
- No placeholder comments left behind
- Sections the user didn't answer: omit entirely (don't leave empty headers)
- identity/ai.md and identity/engineering.md: bullet lists under clear headings
- donts.md: grouped by "never without confirmation" vs "never period"
- contexts/work.md: prose + bullet lists, structured by the sections above

Files to write:
- `~/.devkit/identity/ai.md`
- `~/.devkit/identity/engineering.md`
- `~/.devkit/donts.md`
- `~/.devkit/contexts/work.md`
