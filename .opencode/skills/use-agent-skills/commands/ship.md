---
description: Run the pre-launch checklist via parallel fan-out to child-agent prompt templates, then synthesize a go/no-go decision
---

Use internal module `modules/shipping-and-launch/index.md`.

`/ship` is a **fan-out orchestrator**. It runs three child-agent prompt templates in parallel against the current change, then merges their reports into a single go/no-go decision with a rollback plan. The child agents operate independently — no shared state, no ordering — which is what makes parallel execution safe and useful here.

## Phase A — Parallel fan-out

Spawn three subagents concurrently using the Agent tool. **Issue all three Agent tool calls in a single assistant turn so they execute in parallel** — sequential calls defeat the purpose of this command.

Use the templates under `agents/` when dispatching child agents:

Explicitly tell the user that `agents/` files are child-agent prompt templates used by this skill, not user-selectable roles or external dependencies.

1. **`code-reviewer`** — Run a five-axis review (correctness, readability, architecture, security, performance) on the staged changes or recent commits. Output the standard review template.
2. **`security-auditor`** — Run a vulnerability and threat-model pass. Check OWASP Top 10, secrets handling, auth/authz, dependency CVEs. Output the standard audit report.
3. **`test-engineer`** — Analyze test coverage for the change. Identify gaps in happy path, edge cases, error paths, and concurrency scenarios. Output the standard coverage analysis.

In harnesses without an Agent tool, use each child-agent prompt template sequentially and treat their outputs as if returned in parallel — the merge phase still works.

Constraints:
- Subagents cannot spawn other subagents — do not let one child agent delegate to another.
- Each subagent gets its own context window and returns only its report to this main session.
- If you need teammates that talk to each other instead of just reporting back, see `references/orchestration-patterns.md`.

## Phase B — Merge in main context

Once all three reports are back, the main agent synthesizes them:

1. **Code Quality** — Aggregate Critical/Important findings from `code-reviewer` and any failing tests, lint, or build output. Resolve duplicates between reviewers.
2. **Security** — Promote any Critical/High `security-auditor` findings to launch blockers. Cross-reference with `code-reviewer`'s security axis.
3. **Performance** — Pull from `code-reviewer`'s performance axis; cross-check Core Web Vitals if applicable.
4. **Accessibility** — Verify keyboard nav, screen reader support, and contrast directly.
5. **Infrastructure** — Env vars, migrations, monitoring, feature flags. Verify directly.
6. **Documentation** — README, ADRs, changelog. Verify directly.

## Phase C — Decision and rollback

Produce a single output:

```markdown
## Ship Decision: GO | NO-GO

### Blockers (must fix before ship)
- [Source agent: Critical finding + file:line]

### Recommended fixes (should fix before ship)
- [Source agent: Important finding + file:line]

### Acknowledged risks (shipping anyway)
- [Risk + mitigation]

### Rollback plan
- Trigger conditions: [what signals would prompt rollback]
- Rollback procedure: [exact steps]
- Recovery time objective: [target]

### Specialist reports (full)
- [code-reviewer report]
- [security-auditor report]
- [test-engineer report]
```

## Rules

1. The three Phase A child agents run in parallel when the runtime supports it.
2. Child agents do not call each other. The main agent merges in Phase B.
3. The rollback plan is mandatory before any GO decision.
4. If any child agent returns a Critical finding, the default verdict is NO-GO unless the user explicitly accepts the risk.
5. **Skip the fan-out only if all of the following are true:** the change touches 2 files or fewer, the diff is under 50 lines, and it does not touch auth, payments, data access, or config/env. Otherwise, default to fan-out. `/ship` is designed for production-bound changes — when the blast radius is non-trivial, run the parallel review even if the diff looks small.
