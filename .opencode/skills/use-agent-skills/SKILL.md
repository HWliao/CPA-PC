---
name: use-agent-skills
description: Use ONLY when the user explicitly asks to use use-agent-skills, agent-skills, this internal skill system, or one of its internal commands such as /spec, /plan, /build, /test, /review, /code-simplify, or /ship. Once activated, announce entry into the agent skills flow, route the task to the appropriate internal module or command template, and stop responding to internal modules or commands after the flow exits until the user explicitly re-enters.
---

# Use Agent Skills

## Purpose

This is a user-activated engineering workflow skill. It integrates the `agent-skills` workflows into one entry point, then routes the active task to the appropriate internal module, internal command template, or child-agent prompt template.

Do not treat files under `modules/` as separate OpenCode skills. They are internal modules of this skill.

## Activation Rule

Use this skill only when the user explicitly asks for it, for example:

- "use use-agent-skills"
- "use agent-skills"
- "use the agent skills flow"
- "enter the internal /spec command"
- "use the internal review command"

Do not enter this skill for ordinary coding, testing, debugging, reviewing, or shipping requests unless the user names `use-agent-skills`, `agent-skills`, the internal skill system, or an internal command.

## Flow State

When entering this skill flow, explicitly tell the user:

```text
Entered.
```

Stay in the agent skills flow while working on the activated task and directly related follow-ups.

Exit the flow when either condition is met:

- The activated task is complete.
- The user asks to exit, stop using agent skills, leave the flow, or return to normal mode.

When exiting, explicitly tell the user:

```text
Exited.
```

After exit, internal module names and internal commands are no longer active instructions. Treat `/spec`, `/plan`, `/build`, `/test`, `/review`, `/code-simplify`, `/ship`, and internal module names as plain text unless the user explicitly re-enters `use-agent-skills` or `agent-skills`.

If the user invokes an internal command after exit without re-entering, say that the agent skills flow is not active and ask whether they want to re-enter it.

## Routing Rule

After activation, route in this order:

1. If the user names an internal command, read and follow `commands/<command>.md`.
2. If the user names an internal module, read and follow `modules/<module>/index.md`.
3. If the user gives a task without naming a module, identify the development phase and choose the smallest useful module or module sequence.
4. For compound work, use a short sequence instead of loading everything.
5. If the route is ambiguous, ask one necessary clarifying question.

Internal modules are workflows, not suggestions. After selecting a module, follow its steps and do not skip its verification requirements.

## Task-To-Module Map

```text
Task arrives inside active agent skills flow
    │
    ├── Don't know what the user wants yet? ──────→ interview-me
    ├── Rough concept, need variants? ────────────→ idea-refine
    ├── New project/feature/change? ──────────────→ spec-driven-development
    ├── Have a spec, need tasks? ─────────────────→ planning-and-task-breakdown
    ├── Implementing code? ───────────────────────→ incremental-implementation
    │   ├── UI work? ─────────────────────────────→ frontend-ui-engineering
    │   ├── API work? ────────────────────────────→ api-and-interface-design
    │   ├── Need better context? ─────────────────→ context-engineering
    │   ├── Need source-verified decisions? ──────→ source-driven-development
    │   └── Stakes high / unfamiliar code? ───────→ doubt-driven-development
    ├── Writing/running tests? ───────────────────→ test-driven-development
    │   └── Browser-based? ───────────────────────→ browser-testing-with-devtools
    ├── Something broke? ─────────────────────────→ debugging-and-error-recovery
    ├── Reviewing code? ──────────────────────────→ code-review-and-quality
    │   ├── Too complex? ─────────────────────────→ code-simplification
    │   ├── Security concerns? ───────────────────→ security-and-hardening
    │   └── Performance concerns? ────────────────→ performance-optimization
    ├── Committing/branching? ────────────────────→ git-workflow-and-versioning
    ├── CI/CD pipeline work? ─────────────────────→ ci-cd-and-automation
    ├── Deprecating/migrating? ───────────────────→ deprecation-and-migration
    ├── Writing docs/ADRs? ───────────────────────→ documentation-and-adrs
    └── Deploying/launching? ─────────────────────→ shipping-and-launch
```

## Internal Commands

These command templates are internal to this skill. They are not OpenCode slash commands.

| Command | Template | Main Module | Use When |
| --- | --- | --- | --- |
| `/spec` | `commands/spec.md` | `spec-driven-development` | Define what to build before implementation. |
| `/plan` | `commands/plan.md` | `planning-and-task-breakdown` | Break an existing spec into ordered tasks. |
| `/build` | `commands/build.md` | `incremental-implementation` | Implement in small verified slices. |
| `/test` | `commands/test.md` | `test-driven-development` | Prove behavior with tests or reproduce a bug first. |
| `/review` | `commands/review.md` | `code-review-and-quality` | Review code before merge or handoff. |
| `/code-simplify` | `commands/code-simplify.md` | `code-simplification` | Reduce complexity without changing behavior. |
| `/ship` | `commands/ship.md` | `shipping-and-launch` | Prepare release, rollout, monitoring, and rollback. |

## Internal Module Quick Reference

| Phase | Module | Path | Use When |
| --- | --- | --- | --- |
| Define | `interview-me` | `modules/interview-me/index.md` | Surface what the user actually wants before any plan, spec, or code exists. |
| Define | `idea-refine` | `modules/idea-refine/index.md` | Refine rough ideas through divergent and convergent thinking. |
| Define | `spec-driven-development` | `modules/spec-driven-development/index.md` | Requirements and acceptance criteria before code. |
| Plan | `planning-and-task-breakdown` | `modules/planning-and-task-breakdown/index.md` | Decompose a spec into small, verifiable tasks. |
| Build | `incremental-implementation` | `modules/incremental-implementation/index.md` | Build in thin vertical slices and verify each slice. |
| Build | `context-engineering` | `modules/context-engineering/index.md` | Load the right context at the right time. |
| Build | `source-driven-development` | `modules/source-driven-development/index.md` | Verify framework/library facts against authoritative sources. |
| Build | `doubt-driven-development` | `modules/doubt-driven-development/index.md` | Cross-examine non-trivial decisions in-flight. |
| Build | `frontend-ui-engineering` | `modules/frontend-ui-engineering/index.md` | Build production-quality UI with accessibility. |
| Build | `api-and-interface-design` | `modules/api-and-interface-design/index.md` | Design stable APIs, contracts, and boundaries. |
| Verify | `test-driven-development` | `modules/test-driven-development/index.md` | Write proof through tests, especially behavior changes and bug fixes. |
| Verify | `browser-testing-with-devtools` | `modules/browser-testing-with-devtools/index.md` | Verify browser behavior with runtime data. |
| Verify | `debugging-and-error-recovery` | `modules/debugging-and-error-recovery/index.md` | Reproduce, localize, fix, and guard failures. |
| Review | `code-review-and-quality` | `modules/code-review-and-quality/index.md` | Review correctness, readability, architecture, security, and performance. |
| Review | `code-simplification` | `modules/code-simplification/index.md` | Preserve behavior while reducing unnecessary complexity. |
| Review | `security-and-hardening` | `modules/security-and-hardening/index.md` | Harden input handling, auth, storage, dependencies, and integrations. |
| Review | `performance-optimization` | `modules/performance-optimization/index.md` | Measure first and optimize only what matters. |
| Ship | `git-workflow-and-versioning` | `modules/git-workflow-and-versioning/index.md` | Keep commits, branches, and version history clean. |
| Ship | `ci-cd-and-automation` | `modules/ci-cd-and-automation/index.md` | Automate quality gates and deployment pipelines. |
| Ship | `deprecation-and-migration` | `modules/deprecation-and-migration/index.md` | Remove old systems and migrate users safely. |
| Ship | `documentation-and-adrs` | `modules/documentation-and-adrs/index.md` | Document the why, especially ADRs and durable project docs. |
| Ship | `shipping-and-launch` | `modules/shipping-and-launch/index.md` | Run pre-launch checks, monitoring, rollout, and rollback planning. |

## Common Sequences

Do not mechanically apply the full lifecycle. Use the shortest sequence that fits the task.

- Vague request to spec: `interview-me` -> `idea-refine` -> `spec-driven-development`
- Existing spec to tasks: `planning-and-task-breakdown`
- Existing tasks to implementation: `incremental-implementation` -> `test-driven-development`
- Bug fix: `debugging-and-error-recovery` -> `test-driven-development` -> `code-review-and-quality`
- UI change: `frontend-ui-engineering` -> `browser-testing-with-devtools` -> `code-review-and-quality`
- API change: `api-and-interface-design` -> `test-driven-development` -> `documentation-and-adrs`
- Pre-launch check: `shipping-and-launch`, plus `security-and-hardening` or `performance-optimization` when needed

## Child-Agent Prompt Templates

Files under `agents/` are prompt templates for child-agent dispatch. They are not user-selectable roles and do not need to be registered as OpenCode agents.

Use them only when an internal command or module needs an isolated review pass:

- `agents/code-reviewer.md`: code quality review.
- `agents/test-engineer.md`: test strategy and coverage review.
- `agents/security-auditor.md`: security audit and threat review.

When dispatching a child agent, provide the task context, files or diff to inspect, and the expected return format.

## Reference Checklists

Files under `references/` are checklists loaded on demand. Read them only when the selected module or command points to them.

- `references/accessibility-checklist.md`
- `references/orchestration-patterns.md`
- `references/performance-checklist.md`
- `references/security-checklist.md`
- `references/testing-patterns.md`

## Runtime Documentation Outputs

If an internal module or command produces project documentation, write it under the current project's `<project_root>/docs/`. Do not write generated project documentation inside this skill directory.

- Spec: `<project_root>/docs/SPEC.md`
- Plan: `<project_root>/docs/plan.md`
- Task list: `<project_root>/docs/todo.md`
- ADRs: `<project_root>/docs/decisions/`
- Review reports: `<project_root>/docs/reviews/`
- Launch notes: `<project_root>/docs/launch/`

If later modules or commands depend on these outputs, read them from `<project_root>/docs/`.

## Core Operating Behaviors

These rules apply throughout the active agent skills flow.

### 1. Surface Assumptions

Before non-trivial implementation, explicitly state your assumptions:

```text
ASSUMPTIONS I'M MAKING:
1. [assumption about requirements]
2. [assumption about architecture]
3. [assumption about scope]
→ Correct me now or I'll proceed with these.
```

Do not silently fill in ambiguous requirements. Wrong assumptions are cheaper to fix when surfaced early.

### 2. Manage Confusion Actively

When you encounter inconsistencies, conflicting requirements, or unclear specifications:

1. Stop guessing.
2. Name the specific confusion.
3. Present the tradeoff or ask the clarifying question.
4. Wait for resolution before continuing.

### 3. Push Back When Warranted

You are not a yes-machine. When an approach has clear problems:

- Point out the issue directly.
- Explain the concrete downside, quantified when possible.
- Propose an alternative.
- Accept the user's decision if they override with full information.

### 4. Enforce Simplicity

Before and after implementation, check:

- Can this be done in fewer lines?
- Are these abstractions earning their complexity?
- Would a senior engineer ask, "Why not just do it directly?"

Prefer boring, direct, correct solutions. Clever complexity is expensive.

### 5. Maintain Scope Discipline

Touch only what the task requires.

Do not:

- Remove comments you do not understand.
- Clean up unrelated code.
- Refactor adjacent systems as a side effect.
- Delete code that only appears unused without explicit approval.
- Add features outside the spec.

### 6. Verify With Evidence

A task is not complete until verification passes. "Looks right" is not enough. Provide tests, build output, runtime data, review output, or other concrete evidence.

## Failure Modes To Avoid

Avoid these behaviors that look productive but create problems:

1. Making wrong assumptions without checking.
2. Continuing while confused.
3. Not surfacing inconsistencies you notice.
4. Not presenting tradeoffs on non-obvious decisions.
5. Being sycophantic toward approaches with clear problems.
6. Overcomplicating code and APIs.
7. Modifying code or comments outside the task scope.
8. Removing things you do not fully understand.
9. Building without a spec because "it's obvious."
10. Skipping verification because "it looks right."

## Completion Check

Before exiting the agent skills flow, confirm:

- You used the smallest suitable module or command.
- You followed the selected module's steps and verification requirements.
- Generated project documentation went to `<project_root>/docs/`.
- Assumptions, confusion, tradeoffs, and risks were surfaced to the user.
- Completion evidence exists.
- You displayed `Exited.`
