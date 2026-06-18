# Event-Driven Agent Orchestration — Design Document

**Status:** RESEARCH — Tools evaluated, scenarios brainstormed. Not yet implemented.
**Date:** 2026-06-19
**Goal:** Add event-driven, code-first agent orchestration to the devkit ecosystem. No UI. Webhook in → fan-out sub-agents → work gets done → results posted back.

---

## Tool Selection

### Strategy: Two-Track Approach

We pick **two tools** — one zero-dependency embedded library for lightweight/local use, one with-dependencies engine for production/scale:

| Track | Tool | Deps | Use case |
|-------|------|------|----------|
| **Light** | go-workflows | None (SQLite embedded) | Single binary, local dev, personal automation |
| **Heavy** | Hatchet | Postgres | Production, rate limiting, multi-machine, team use |

Both are Go. Both support fan-out. The light track can graduate to the heavy track by swapping the backend — same workflow logic, different engine.

---

## Full Evaluation Matrix

### All tools evaluated

| Tool | Stars | Language | External Deps | Windows Native | Fan-out | Event Triggers | SDKs | Last Active |
|------|-------|----------|---------------|----------------|---------|----------------|------|-------------|
| **go-workflows** | 494 | Go | None (SQLite/in-memory) | Yes | Temporal-style futures | You build it (library) | Go | 2026-06-05 |
| **Hatchet** | 4,000+ | Go | Postgres | Yes | Native DAG steps + worker groups | First-class event subscriptions | Go, Python, TS | 2026-06 |
| **Tork** | 807 | Go | None (standalone) or Postgres+RabbitMQ | Yes | Native parallel blocks | REST API (POST /jobs) | REST API + Go | 2026-06-18 |
| **Conductor OSS** | 31,960 | Java | Java 21 + optional Postgres | Yes | FORK_JOIN + dynamic fanout | Workers poll + API start | Go, Python, TS, Java, C#, Ruby, Rust | 2026-06-17 |
| **RuleGo** | 1,547 | Go | None (embedded library) | Yes | Parallel rule chains | HTTP/MQTT/Kafka endpoints | Go | 2026-06-11 |
| **Windmill** | 16,800 | Rust | Postgres | Yes (.exe) | Flow composition | Webhooks, Kafka, cron | Python, TS, Go, Bash, Rust | 2026-06-18 |
| **DBOS Transact** | 1,247 | TypeScript | Postgres | Yes (Node) | Concurrent workflow queues | Exactly-once events | TypeScript only | 2026-06-15 |
| **Obelisk** | 708 | Rust | None (SQLite) | No (Linux/Mac) | Structured concurrency | WASI webhook endpoints | WASM languages | 2026-06-06 |
| **Golem** | 1,592 | Rust | Multi-service Docker | Yes | Durable concurrent exec | Worker HTTP handlers | Rust, TS, Scala | 2026-06-16 |
| **BullMQ** | 9,015 | TypeScript | Redis | Yes (Node) | FlowProducer DAG | Programmatic only | Node, Python, Rust | Active |
| **River** | 5,257 | Go | Postgres | Yes | Multi-queue workers | Programmatic only | Go | 2026-06-03 |
| **Temporal** | 12,000+ | Go | Cassandra/Postgres + ES + server | Yes | Child workflows | Signals + schedules | Go, Python, TS, Java, .NET | Active |
| **Inngest** | 5,000+ | Go | Self-host server | Yes | Step functions | Event-driven ("send event → functions react") | TS, Python, Go | Active |
| **Restate** | 2,000+ | Rust | None (single binary) | Yes | Virtual objects | HTTP handlers | TS, Python, Java, Go | Active |

### Rejected (with reasons)

| Tool | Why rejected |
|------|-------------|
| **Temporal** | Overengineered for AI workloads. Replay determinism constraints add friction. Multi-service cluster is ops overhead for personal use. |
| **Conductor OSS** | Java 21 runtime requirement. "Enterprise" operational model. Best fit for large teams, overkill for personal/small-team use. |
| **Inngest** | Clean model but TypeScript-first. Go SDK less mature. Self-hosted server adds a dependency without clear advantage over Hatchet. |
| **Restate** | Fastest runtime (Rust single binary) but youngest ecosystem. Virtual objects paradigm is a different mental model — better for microservices than AI agent orchestration. |
| **Windmill** | Positioned as a full platform (scripts → UIs → webhooks). Overkill — we want an engine, not a platform. |
| **DBOS Transact** | TypeScript only. No Go SDK. |
| **Obelisk** | No Windows support. WASM ecosystem still maturing. AGPL license problematic. |
| **Golem** | Multi-service Docker stack. More complex than Hatchet for same capability tier. |
| **BullMQ** | Redis dependency. No built-in webhook handling. Node.js ecosystem. |
| **River** | Go + Postgres, but no built-in workflow DAG — it's a job queue, not an orchestrator. Would need to build orchestration on top. |
| **RuleGo** | Rule-chain paradigm is unusual. Documentation heavily Chinese-language. Better for IoT/edge than AI agent orchestration. |
| **n8n** | UI-first. Violates "no UI, code-first" requirement. |
| **Prefect/Dagster** | Data pipeline tools. Wrong abstraction for event-driven agent work. |
| **CrewAI/LangGraph** | Framework-locked. Libraries that assume their LLM abstraction layer. Not infrastructure. |

---

## Track 1: go-workflows (Zero Dependencies)

### What it is

A Go library that gives you Temporal's programming model without Temporal's infrastructure. No server, no daemon — your binary IS the orchestrator.

**GitHub:** github.com/cschleiden/go-workflows (494 stars, active, last release 2026-06-05)

### Why this is the light track

| Property | Value |
|----------|-------|
| External deps | None. SQLite embedded, or in-memory for tests |
| Binary count | 1 (your app) |
| RAM usage | ~30MB (your process + SQLite) |
| Windows | Yes (pure Go, cross-compiles) |
| Startup time | Instant |
| Backup | Copy one SQLite file |
| Upgrade path | Swap SQLite backend for Postgres/MySQL/Redis — same workflow code |

### Fan-out pattern

```go
func ResearchWorkflow(ctx workflow.Context, ticket JiraTicket) (Result, error) {
    // Fan-out: launch parallel sub-workflows
    codeSearch := workflow.CreateSubWorkflowInstance[SearchResult](ctx,
        workflow.DefaultSubWorkflowOptions, SearchCodeWorkflow, ticket.Key)
    
    slackSearch := workflow.CreateSubWorkflowInstance[SearchResult](ctx,
        workflow.DefaultSubWorkflowOptions, SearchSlackWorkflow, ticket.Summary)
    
    gitHistory := workflow.CreateSubWorkflowInstance[SearchResult](ctx,
        workflow.DefaultSubWorkflowOptions, GitHistoryWorkflow, ticket.AffectedRepos)
    
    prSearch := workflow.CreateSubWorkflowInstance[SearchResult](ctx,
        workflow.DefaultSubWorkflowOptions, RelatedPRsWorkflow, ticket.Key)

    // Await all results
    code, _ := codeSearch.Get(ctx)
    slack, _ := slackSearch.Get(ctx)
    git, _ := gitHistory.Get(ctx)
    prs, _ := prSearch.Get(ctx)

    // Aggregate and synthesize (call Claude API)
    plan := synthesize(ctx, ticket, code, slack, git, prs)
    
    // Post results
    workflow.ExecuteActivity[any](ctx, PostToJira, ticket.Key, plan)
    workflow.ExecuteActivity[any](ctx, PostToSlack, plan.Summary)
    
    return plan, nil
}
```

### Webhook integration

You build a standard Go HTTP server alongside the workflow engine — they live in the same binary:

```go
func main() {
    // Initialize workflow engine with SQLite backend
    b := sqlite.NewSqliteBackend("workflows.db")
    engine := workflow.NewEngine(b)
    
    // Register workflows
    engine.RegisterWorkflow(ResearchWorkflow)
    engine.RegisterWorkflow(CodeReviewWorkflow)
    
    // Register activities (the actual work)
    engine.RegisterActivity(SearchCode)
    engine.RegisterActivity(SearchSlack)
    engine.RegisterActivity(CallClaude)
    engine.RegisterActivity(PostToJira)
    engine.RegisterActivity(PostToSlack)
    
    // Start engine (processes workflows in background)
    engine.Start(ctx)
    
    // HTTP server for webhooks — same binary, same process
    http.HandleFunc("/webhook/jira", func(w http.ResponseWriter, r *http.Request) {
        ticket := parseJiraWebhook(r)
        engine.StartWorkflow(ctx, ResearchWorkflow, ticket)
        w.WriteHeader(200)
    })
    
    http.HandleFunc("/webhook/slack", func(w http.ResponseWriter, r *http.Request) {
        msg := parseSlackEvent(r)
        engine.StartWorkflow(ctx, DeepResearchWorkflow, msg)
        w.WriteHeader(200)
    })
    
    http.HandleFunc("/webhook/github", func(w http.ResponseWriter, r *http.Request) {
        pr := parseGitHubWebhook(r)
        engine.StartWorkflow(ctx, CodeReviewWorkflow, pr)
        w.WriteHeader(200)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### Architecture (single binary)

```
┌──────────────────────────────────────────────────────┐
│                YOUR SINGLE GO BINARY                   │
│                                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │
│  │ HTTP Server  │  │  Workflow    │  │  Activity  │  │
│  │ (webhooks)   │→ │  Engine     │→ │  Workers   │  │
│  └──────────────┘  └──────────────┘  └────────────┘  │
│                           │                            │
│                    ┌──────┴──────┐                     │
│                    │   SQLite    │                     │
│                    │ (embedded)  │                     │
│                    └─────────────┘                     │
└──────────────────────────────────────────────────────┘
         ↑                                    ↓
    Jira/Slack/GitHub                  Slack/Jira/GitHub
    webhooks IN                        API calls OUT
```

### Limitations (why you'd graduate to Track 2)

- No rate limiting built-in (you build it yourself or use a semaphore)
- No dashboard/visibility into running workflows (you add logging)
- SQLite = single machine only (no distributed workers)
- No event subscription model (you wire webhooks → workflow starts manually)
- Go only — no Python/TypeScript workers

---

## Track 2: Hatchet (With Dependencies, Production-Grade)

### What it is

Purpose-built workflow engine for AI/LLM workloads. Event-driven, with native rate limiting, concurrency controls, and fan-out.

**GitHub:** github.com/hatchet-dev/hatchet (4,000+ stars, YC-backed, very active)

### Why this is the heavy track

| Property | Value |
|----------|-------|
| External deps | Postgres |
| Services | Hatchet engine + Postgres (docker-compose) |
| RAM usage | ~768MB total (engine 256MB + Postgres 512MB) |
| Windows | Yes (Go binary + Postgres) |
| Distributed workers | Yes — workers can run on different machines |
| Rate limiting | Built-in per workflow type, per step, per API provider |
| Event model | First-class: emit event → subscribed workflows fire |
| Dashboard | Built-in web UI (optional, can ignore) |

### Fan-out pattern

```go
// Hatchet workflow definition
func ResearchWorkflow(ctx worker.HatchetContext) error {
    ticket := ctx.Input()
    
    // Fan-out: spawn child workflows in parallel
    results := ctx.SpawnWorkflows([]worker.SpawnWorkflowOpts{
        {Workflow: "search-code", Input: ticket},
        {Workflow: "search-slack", Input: ticket},
        {Workflow: "git-history", Input: ticket},
        {Workflow: "related-prs", Input: ticket},
    })
    
    // Await all
    allResults := results.AwaitAll()
    
    // Synthesize (Claude API call with rate limiting handled by Hatchet)
    plan := ctx.RunStep("synthesize", func() (Plan, error) {
        return callClaude(allResults)
    })
    
    // Post results
    ctx.RunStep("post-jira", func() error {
        return postToJira(ticket.Key, plan)
    })
    
    return nil
}
```

### Event-driven triggers (no custom webhook server needed)

```go
// Hatchet handles the event routing — you just declare subscriptions
worker.On("jira:ticket:assigned", ResearchWorkflow)
worker.On("slack:message:mention", DeepResearchWorkflow)
worker.On("github:pr:opened", CodeReviewWorkflow)
worker.On("cron:daily:8am", MorningBriefingWorkflow)

// Your webhook receiver is minimal — just parse and push events
http.HandleFunc("/webhook/jira", func(w http.ResponseWriter, r *http.Request) {
    event := parseJiraWebhook(r)
    hatchet.Event().Push("jira:ticket:assigned", event)
    w.WriteHeader(200)
})
```

### Architecture (multi-service)

```
┌─────────────────────────────────────────────────────────────────┐
│                        HATCHET STACK                              │
│                                                                   │
│  ┌────────────────┐     ┌─────────────────┐     ┌───────────┐  │
│  │ Webhook Server │     │  Hatchet Engine  │     │ Postgres  │  │
│  │ (your Go code) │────→│  (orchestrator)  │←───→│  (state)  │  │
│  └────────────────┘     └────────┬────────┘     └───────────┘  │
│                                  │                               │
│              ┌───────────────────┼───────────────────┐          │
│              ▼                   ▼                   ▼          │
│     ┌──────────────┐   ┌──────────────┐   ┌──────────────┐    │
│     │  Worker A    │   │  Worker B    │   │  Worker C    │    │
│     │  (machine 1) │   │  (machine 2) │   │  (machine 3) │    │
│     └──────────────┘   └──────────────┘   └──────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### What Hatchet adds over go-workflows

| Feature | go-workflows | Hatchet |
|---------|--------------|---------|
| Rate limiting per LLM provider | Build yourself | Built-in |
| Concurrency limits | Build yourself | Built-in (per workflow, per step) |
| Distributed workers | No (single process) | Yes (workers anywhere) |
| Event subscriptions | No (manual wiring) | First-class |
| Retry with backoff | Basic | Configurable per step |
| Bulk cancellation | No | Yes |
| Visibility/debugging | Logs only | Web dashboard |
| Cost tracking | No | Track per-workflow resource usage |
| Cron schedules | Build yourself | Built-in |
| Multi-language workers | No (Go only) | Go, Python, TypeScript |

---

## Track 3: Tork (Middle Ground — YAML Workflows)

### What it is

Single Go binary that runs workflows defined in YAML. Zero deps in standalone mode, Postgres+RabbitMQ for production.

**GitHub:** github.com/runabol/tork (807 stars, daily releases, very active)

### Why it's worth noting

It sits between go-workflows (library, code-only) and Hatchet (full platform). If you want declarative YAML workflows without writing Go orchestration code:

```yaml
name: research-on-jira-ticket
inputs:
  ticket_key: string
  summary: string

tasks:
  - name: fan-out-research
    parallel:
      tasks:
        - name: search-codebase
          run: |
            ./agents/search-code --ticket={{inputs.ticket_key}}
          
        - name: search-slack
          run: |
            ./agents/search-slack --query="{{inputs.summary}}"
          
        - name: git-history
          run: |
            ./agents/git-history --ticket={{inputs.ticket_key}}
          
        - name: related-prs
          run: |
            ./agents/related-prs --ticket={{inputs.ticket_key}}

  - name: synthesize
    run: |
      ./agents/synthesize \
        --code="{{tasks.search-codebase.result}}" \
        --slack="{{tasks.search-slack.result}}" \
        --git="{{tasks.git-history.result}}" \
        --prs="{{tasks.related-prs.result}}"

  - name: post-results
    run: |
      ./agents/post-to-jira --ticket={{inputs.ticket_key}} --plan="{{tasks.synthesize.result}}"
```

### When to pick Tork over the other two

| If you want... | Pick |
|----------------|------|
| YAML-defined workflows, each agent is a separate binary/script | Tork |
| Everything in one Go binary, programmatic control | go-workflows |
| Production-grade with rate limiting, distributed workers | Hatchet |

---

## Decision Matrix

### Pick go-workflows (Track 1) when:

- You want one binary with zero external dependencies
- Running on a personal machine (Mac/Windows/Linux laptop)
- Workflows are relatively simple (< 10 concurrent)
- You want maximum portability (copy binary + SQLite file = done)
- Budget: $0 infrastructure

### Pick Hatchet (Track 2) when:

- You need rate limiting per LLM provider (Claude, OpenAI, etc.)
- Workers run on multiple machines (work laptop + home server)
- You want event subscriptions without custom routing code
- You need visibility into running workflows (dashboard)
- You're hitting scale (dozens of concurrent workflows)
- Budget: Postgres hosting (~$5-15/month or local Docker)

### Pick both (recommended path):

1. **Start with go-workflows** — build the webhook server + workflows in a single Go binary. Validate the patterns, prove the scenarios work.
2. **Graduate to Hatchet** when you hit one of: need distributed workers, need rate limiting, need visibility, workflows exceed what a single process handles.
3. **Workflow logic is portable** — the sub-agent pattern (fan-out → await → aggregate) is the same in both. Only the engine plumbing changes.

---

## Comparison: go-workflows vs Hatchet vs Tork

| | go-workflows | Hatchet | Tork |
|---|---|---|---|
| **Philosophy** | Library — you own everything | Platform — it handles orchestration | Engine — YAML workflows, REST API |
| **Binary count** | 1 (your app) | 3+ (engine, postgres, your workers) | 1 (standalone) or 3 (engine, postgres, rabbitmq) |
| **Language** | Go only | Go, Python, TypeScript | Language-agnostic (shell/Docker) |
| **Workflow definition** | Go code | Go/Python/TS code | YAML |
| **State storage** | SQLite/Postgres/MySQL/Redis/Memory | Postgres | Memory/Postgres+RabbitMQ |
| **Fan-out** | Futures (spawn N, await all) | SpawnWorkflows + DAG steps | `parallel:` block in YAML |
| **Rate limiting** | DIY | Built-in per provider | DIY |
| **Distributed** | No (single process) | Yes | Yes (with Postgres+RabbitMQ) |
| **Windows** | Yes | Yes | Yes |
| **Learning curve** | Know Go + Temporal concepts | Know Go + Hatchet API | Know YAML + REST |
| **Ideal for** | Personal single-machine automation | Production multi-machine orchestration | Polyglot teams, declarative workflows |
| **Stars** | 494 | 4,000+ | 807 |
| **Maturity** | 2 years | 2 years (YC-backed) | 2 years (daily releases) |

---

## Webhook Integration (Applies to All Tracks)

None of these tools have built-in Jira/Slack/GitHub connectors. You always write a thin webhook receiver. This is ~50 lines of Go per integration:

### How each tool ingests events

| Tool | How to trigger a workflow |
|------|-------------------------|
| go-workflows | `engine.StartWorkflow(ctx, MyWorkflow, input)` |
| Hatchet | `client.Event().Push("event.name", payload)` |
| Tork | `POST /jobs` with YAML reference + inputs |

### Webhook receiver (shared across all tracks)

```go
// This is the same regardless of which engine you pick
mux := http.NewServeMux()

mux.HandleFunc("POST /webhook/jira", func(w http.ResponseWriter, r *http.Request) {
    if !verifyJiraSignature(r) {
        http.Error(w, "unauthorized", 401)
        return
    }
    event := parseJiraWebhook(r.Body)
    
    switch event.Type {
    case "issue_assigned":
        triggerWorkflow("research", event.Issue)
    case "issue_commented":
        triggerWorkflow("respond-to-comment", event.Issue)
    }
    w.WriteHeader(200)
})

mux.HandleFunc("POST /webhook/slack", func(w http.ResponseWriter, r *http.Request) {
    if !verifySlackSignature(r) {
        http.Error(w, "unauthorized", 401)
        return
    }
    event := parseSlackEvent(r.Body)
    
    switch event.Type {
    case "app_mention":
        triggerWorkflow("deep-research", event.Message)
    case "message":
        if isQuestion(event.Message) {
            triggerWorkflow("answer-question", event.Message)
        }
    }
    w.WriteHeader(200)
})

mux.HandleFunc("POST /webhook/github", func(w http.ResponseWriter, r *http.Request) {
    if !verifyGitHubSignature(r) {
        http.Error(w, "unauthorized", 401)
        return
    }
    event := parseGitHubWebhook(r)
    
    switch r.Header.Get("X-GitHub-Event") {
    case "pull_request":
        triggerWorkflow("code-review", event.PR)
    case "push":
        triggerWorkflow("impact-analysis", event.Commits)
    }
    w.WriteHeader(200)
})
```

### Setting up webhooks in each tool

| Tool | Where to configure | URL to point at |
|------|-------------------|-----------------|
| **Jira** | Project Settings → Webhooks → Add | `https://your-server:8080/webhook/jira` |
| **Slack** | api.slack.com → Event Subscriptions | `https://your-server:8080/webhook/slack` |
| **GitHub** | Repo Settings → Webhooks | `https://your-server:8080/webhook/github` |
| **PagerDuty** | Service → Integrations → Webhooks | `https://your-server:8080/webhook/pagerduty` |

**Exposing to internet:** Use ngrok (dev), Cloudflare Tunnel (free, production), or a VPS with a public IP.

---

## Architecture (Combined Approach)

### Phase 1: Single binary (go-workflows)

```
┌──────────────────────────────────────────────────────────────┐
│                   devkit-agent (single Go binary)              │
│                                                                │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │ Webhook      │  │  go-workflows │  │  Agent Workers     │  │
│  │ Receiver     │→ │  Engine       │→ │  (Claude API calls) │  │
│  │ (HTTP)       │  │  (SQLite)     │  │  (code search)     │  │
│  └──────────────┘  └──────────────┘  │  (git operations)   │  │
│                                       └────────────────────┘  │
│         ↕                                       ↕              │
│  Jira/Slack/GitHub                     Jira/Slack/GitHub      │
│  (webhooks IN)                         (API calls OUT)        │
└──────────────────────────────────────────────────────────────┘
            One process. One file (workflows.db). Done.
```

### Phase 2: Graduate to Hatchet (when needed)

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                       │
│  ┌────────────────┐     ┌──────────────────┐     ┌──────────────┐  │
│  │ Webhook Server │     │  Hatchet Engine   │     │   Postgres   │  │
│  │ (same Go code) │────→│  (orchestrator)   │←───→│   (state)    │  │
│  └────────────────┘     └────────┬─────────┘     └──────────────┘  │
│                                  │                                    │
│         ┌────────────────────────┼────────────────────────┐         │
│         ▼                        ▼                        ▼         │
│  ┌─────────────┐         ┌─────────────┐         ┌─────────────┐  │
│  │ Worker:     │         │ Worker:     │         │ Worker:     │  │
│  │ Research    │         │ Code Review │         │ Comms       │  │
│  │ (home Mac)  │         │ (work laptop)│         │ (VPS)       │  │
│  └─────────────┘         └─────────────┘         └─────────────┘  │
│                                                                       │
│  Rate limits: Claude 40 req/min | GitHub 5000/hr | Slack 50/min     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Scenarios for the devkit Ecosystem

### Tier 1 — High Value, Clear Trigger, Build First

#### 1. Jira Ticket Assigned → Research & Plan

**Trigger:** Jira webhook — issue assigned to you
**Workflow:**
1. Parse ticket (summary, description, acceptance criteria, linked issues)
2. Fan-out:
   - Search codebase for relevant files (using ~/dev/readonly/ analysis copies)
   - Check related PRs and recent commits in affected repos
   - Search Slack for prior discussion about this topic
   - Look up similar past tickets and their resolutions
3. Aggregate into implementation plan
4. Post plan as Jira comment + Slack DM with summary

**Why first:** This is the highest-friction daily task. Reading a ticket, understanding context, finding code, checking history — 30-60 minutes of manual work that's highly automatable.

#### 2. Slack Question → Deep Research Response

**Trigger:** Slack message mentioning you (or in a monitored channel) asking a technical question
**Workflow:**
1. Parse question intent
2. Fan-out:
   - Search codebase (grep, AST analysis)
   - Search internal docs (Confluence/wiki)
   - Check git history for relevant changes
   - Search Slack history for prior answers
3. Synthesize answer with citations (file paths, commit SHAs, doc links)
4. Post response as Slack thread reply

**Why:** Answering "where is X defined?" or "how does Y work?" burns significant context-switching time. Agent does the legwork.

#### 3. PR Opened → Multi-Agent Review

**Trigger:** GitHub webhook — PR opened/updated
**Workflow:**
1. Fetch diff and PR description
2. Fan-out:
   - Security review agent (OWASP patterns, secrets, injection vectors)
   - Architecture impact agent (does this change cross module boundaries? break contracts?)
   - Test coverage agent (are new code paths tested? what's missing?)
   - Style/convention agent (uses devkit identity/engineering.md as the standard)
3. Aggregate into single review comment with sections
4. Post as GitHub PR review (or Slack summary if you're the author)

**Why:** Manual review is slow and inconsistent. Fan-out catches different categories in parallel.

---

### Tier 2 — Medium Value, Clear Trigger, Build Second

#### 4. Incident Alert → Root Cause Analysis

**Trigger:** PagerDuty/OpsGenie webhook (or Slack #incidents channel message)
**Workflow:**
1. Parse alert (service, error, severity)
2. Fan-out:
   - Recent deploys to affected service (git log, CD pipeline)
   - Error patterns in logs (Datadog/Grafana query)
   - Git blame on recently changed files in affected service
   - Search for similar past incidents
3. Draft incident report (timeline, suspected cause, suggested next steps)
4. Post to incident Slack channel

**Why:** First 15 minutes of an incident are the most stressful and least productive. Having an agent gather context while you get oriented is high-leverage.

#### 5. New Repo Onboarded → Auto-Analysis

**Trigger:** Manual event (`devkit onboard <repo-url>`) or webhook on repo clone
**Workflow:**
1. Clone repo to ~/dev/readonly/
2. Fan-out:
   - Language/framework detection
   - Entry point and architecture analysis (using devkit's analysis.tmpl.md structure)
   - Dependency audit (licenses, vulnerabilities)
   - CI/CD pipeline analysis
   - Test coverage assessment
3. Generate findings file (findings/<repo-name>.md)
4. Update active context file with repo description
5. Run `devkit generate --all` to propagate context

**Why:** Onboarding to a new codebase takes days. Automated source analysis gives you a head start.

#### 6. Context Drift → Auto-Regeneration

**Trigger:** File watcher on ~/.devkit/ (fsnotify) or git push to ~/.devkit/ repo
**Workflow:**
1. Detect which source files changed (identity, context, donts)
2. Run `devkit generate --all`
3. Report which projects were regenerated
4. If any project's .gitignore doesn't cover generated files, warn via notification

**Why:** Eliminates the #1 friction source identified in the design doc — forgetting to regenerate.

---

### Tier 3 — High Value, Complex, Build When Pain Demands

#### 7. Daily Briefing (Scheduled)

**Trigger:** Cron — every morning at 8am
**Workflow:**
1. Fan-out:
   - Yesterday's git commits across all tracked repos
   - Open PRs needing your review
   - Jira tickets in your sprint (status changes, new comments)
   - Slack mentions you haven't responded to
   - Failed CI runs on your branches
2. Synthesize into daily briefing
3. Post to personal Slack channel or DM

**Why:** Replaces 10 minutes of morning context-gathering across 5 tools.

#### 8. Cross-Repo Impact Analysis

**Trigger:** Manual event or webhook on shared library PR merge
**Workflow:**
1. Identify changed exports/interfaces in shared library
2. Fan-out across all downstream repos in ~/dev/readonly/:
   - Grep for usage of changed symbols
   - Analyze if changes are breaking
   - Check if downstream tests would still pass
3. Generate impact report (which repos affected, severity, required changes)
4. Create Jira tickets for required downstream updates (or post to relevant Slack channels)

**Why:** Shared library changes have hidden blast radius. Automated impact analysis catches what humans miss.

#### 9. Knowledge Capture Pipeline

**Trigger:** PR merged + Jira ticket closed (compound event)
**Workflow:**
1. Gather: PR description, review comments, Jira ticket context, Slack discussion
2. Extract lessons: what was surprising, what was hard, what would you do differently
3. Draft findings file (findings/<ticket-id>.md using research template)
4. If findings/ count > 50, trigger archive suggestion

**Why:** Knowledge is lost the moment a ticket closes. Automated capture while context is fresh.

#### 10. Security Scanning Pipeline (Scheduled + Event)

**Trigger:** Weekly cron + on any Dependabot PR opened
**Workflow:**
1. Fan-out across all tracked repos:
   - govulncheck
   - trivy container scan
   - trufflehog secrets scan
   - License compliance check
2. Aggregate findings, deduplicate, severity-rank
3. Create Jira tickets for critical/high findings
4. Post summary to security Slack channel

**Why:** Security hygiene is the easiest thing to let slip. Automation keeps it continuous.

#### 11. Sprint Planning Assistant

**Trigger:** Manual event or calendar-triggered (sprint start)
**Workflow:**
1. Pull tickets from sprint backlog
2. Fan-out per ticket:
   - Estimate complexity based on codebase analysis
   - Identify dependencies between tickets
   - Flag tickets that need cross-team coordination
3. Suggest sprint ordering (dependency graph)
4. Post to Slack or update Jira sprint board

#### 12. Documentation Staleness Detection (Scheduled)

**Trigger:** Weekly cron
**Workflow:**
1. For each findings/ and analyzed/ file:
   - Check if referenced files still exist
   - Check if referenced code has changed significantly since analysis date
   - Check if referenced tickets are still open
2. Flag stale documentation
3. Post report (which analyses need refresh, which findings are outdated)

---

## Integration with devkit

### What devkit provides to the orchestration layer

| What | How |
|------|-----|
| Identity/context for agents | Agents read generated CLAUDE.md/AGENTS.md from project dirs |
| Analysis templates | Workflows use `analysis.tmpl.md` and `research.tmpl.md` as output structure |
| Project registry | `~/.devkit/projects.txt` tells orchestration which repos to scan |
| Findings storage | Workflows write output to `~/.devkit/findings/` |
| Context updates | Workflows can append to context files (with human review gate) |

### What the orchestration layer provides to devkit

| What | How |
|------|-----|
| Auto-regeneration | Watches ~/.devkit/ → triggers `devkit generate --all` |
| Findings generation | Populates findings/ with structured research output |
| Context enrichment | Suggests context file updates based on repo analysis |
| Staleness detection | Supplements `devkit doctor` with deeper drift analysis |

### Boundary: devkit vs orchestration

```
devkit (static, portable)              Orchestration (dynamic, event-driven)
─────────────────────────              ─────────────────────────────────────
Generates AI config files              Reacts to external events
Composes identity + context            Fans out sub-agents for research
Validates source files                 Integrates with Slack/Jira/GitHub APIs
Tracks projects                        Runs LLM calls with rate limiting
Single binary, no daemon               Runs as service (or embedded in one binary)
```

**Rule:** devkit remains a static generator with zero runtime dependencies. The orchestration layer is a separate service (or binary) that _uses_ devkit (calls `devkit generate`, reads its output, writes to its directories) but does not _become_ devkit.

---

## Implementation Plan

### Phase 1: Single Binary (go-workflows)

1. Create new repo: `devkit-agent` (Go binary)
2. Embed go-workflows with SQLite backend
3. Build webhook receiver (Jira, Slack, GitHub signature validation)
4. Implement first workflow: Slack mention → search codebase → reply in thread
5. Sub-agent pattern: activities that call Claude API with devkit-generated context
6. Deploy: single binary on home Mac or VPS + Cloudflare Tunnel for webhook ingress

### Phase 2: Core Workflows

7. Jira ticket assigned → research & plan
8. PR opened → multi-agent review
9. Context drift → auto-regeneration (fsnotify activity)
10. Daily briefing (cron via goroutine scheduler)

### Phase 3: Graduate to Hatchet (when needed)

11. Stand up Hatchet engine + Postgres (docker-compose)
12. Port workflows from go-workflows → Hatchet SDK (same logic, different engine)
13. Distribute workers across machines (work laptop, home server)
14. Enable rate limiting per LLM provider
15. Add concurrency controls for expensive workflows

### Phase 4: Advanced Workflows

16. Incident response pipeline
17. Cross-repo impact analysis
18. Knowledge capture
19. Security scanning

### Infrastructure Requirements

#### Phase 1 (go-workflows — minimal)

| Component | Purpose | Resource |
|-----------|---------|----------|
| devkit-agent binary | Everything | ~50MB RAM, one process |
| SQLite file | Workflow state | ~1MB disk |
| Cloudflare Tunnel | Expose webhooks to internet | Free tier |
| Claude API key | LLM calls | Anthropic API |

**Total: One binary + one tunnel. Runs on any machine.**

#### Phase 3 (Hatchet — production)

| Component | Purpose | Resource |
|-----------|---------|----------|
| Hatchet engine | Workflow orchestration | ~256MB RAM |
| Postgres | Workflow state, event log | ~512MB RAM |
| devkit-agent binary | Webhook receiver + workers | ~50MB RAM |
| Cloudflare Tunnel | Expose webhooks | Free tier |
| Claude API key | LLM calls | Anthropic API |

**Total: docker-compose up + one binary. Runs on Mac Mini, home server, or $5/month VPS.**

---

## Platform Support

### devkit-agent binary

| Platform | Support | Notes |
|----------|---------|-------|
| macOS (arm64/amd64) | Native | Primary dev platform |
| Linux (arm64/amd64) | Native | VPS/server deployment |
| Windows (amd64) | Native | No WSL, no Docker required |

Pure Go binary. Cross-compiles with `GOOS=windows GOARCH=amd64 go build`. SQLite via modernc.org/sqlite (pure Go, no CGO).

### Hatchet engine

| Platform | Support | Notes |
|----------|---------|-------|
| Any (Docker) | Yes | docker-compose on any platform |
| Linux | Native binary | Primary deployment target |
| macOS | Native binary | Dev/home server |
| Windows | Native binary | With native Postgres or Docker |

---

## Open Questions

1. **Repo structure** — Separate `devkit-agent` repo or monorepo with devkit?
2. **Worker language** — Go (consistent with devkit) or Python (richer LLM tooling)?
3. **Agent framework within workers** — Raw Claude API calls or thin wrapper?
4. **State persistence** — Workflow state vs filesystem for sharing context between steps?
5. **Human-in-the-loop gates** — Which workflows need approval before acting (posting, creating tickets)?
6. **Cost management** — Rate limiting per workflow type? Daily budget caps on Claude API?
7. **Local vs cloud** — Run on local machine (always-on Mac) or small VPS?
8. **go-workflows SQLite backend** — modernc.org/sqlite (pure Go) or mattn/go-sqlite3 (CGO)?

---

## References

- go-workflows: https://github.com/cschleiden/go-workflows
- Hatchet: https://github.com/hatchet-dev/hatchet
- Tork: https://github.com/runabol/tork
- Conductor OSS: https://github.com/conductor-oss/conductor
- Temporal: https://github.com/temporalio/temporal
- Cloudflare Tunnel: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/
- modernc.org/sqlite: https://pkg.go.dev/modernc.org/sqlite (pure Go SQLite, no CGO)
- devkit design doc: `devkit-workspace-design.md`
- devkit project registry: `~/.devkit/projects.txt`
