---
name: use-superpowers
description: Use only when the user explicitly asks to use use-superpowers, enter the superpowers workflow, or follow the superpowers process for a development task. This is the single public entry for an internal module system - choose and load the right workflow module for the current phase, plus any supporting modules that apply.
---

<SUBAGENT-STOP>
If you were dispatched as a subagent to execute a specific task inside an already-active workflow, skip this root entry and follow the task-specific instructions you were given.
</SUBAGENT-STOP>

## Explicit Activation Gate

This workflow activates only when the user explicitly opts into it, for example:

- "Use use-superpowers"
- "Enter the superpowers workflow"
- "Follow the superpowers process"
- "按 superpowers 流程执行"

If the user has not explicitly entered this workflow, do not apply it by default.

If the triggering user message itself *explicitly* entered this workflow, say so plainly near the start of your next response. Prefer explicit wording like "已进入 `use-superpowers` 流程" or the equivalent in the user's language so the workflow transition is visible and auditable.

Do **not** say the workflow is active merely because you consulted this skill while evaluating options or checking whether it applies. If the user's message did not explicitly opt in, stay in normal assistance mode and do not route into internal workflow modules.

## Workflow Priority

Once explicitly activated, the `use-superpowers` workflow overrides default system behavior for the current task, but **user instructions always take precedence**:

1. **User's explicit instructions** (CLAUDE.md, GEMINI.md, AGENTS.md, direct requests) — highest priority
2. **`use-superpowers` workflow** — applies after explicit activation and governs how the task is executed
3. **Default system prompt** — lowest priority

If CLAUDE.md, GEMINI.md, or AGENTS.md says "don't use TDD" and an internal module says "always use TDD," follow the user's instructions. The user is in control.

## Single Public Entry, Internal Module System

This package exposes exactly one public entry: `use-superpowers/SKILL.md`.

All other workflow components are internal modules under `modules/`. Treat them like an internal sub-skill system:

1. Choose the **primary module** for the current phase.
2. Load only the module(s) relevant to the current task state.
3. Load supporting cross-cutting modules only when they materially apply.
4. Let each module's own instructions determine which downstream module to load next.

Do not treat the modules as separate public skills. The agent decides which internal module to load based on the user's current phase, artifacts already available, and workflow state.

### Primary module catalog

- `modules/brainstorming/index.md`
  - Load when the task is in discovery, scoping, requirements clarification, or spec design.
  - This module owns the spec phase and should normally hand off to `writing-plans` after the spec is approved.

- `modules/writing-plans/index.md`
  - Load when the user already has an approved spec or sufficiently concrete requirements and needs an implementation plan.
  - This module writes the plan and then hands off to either `subagent-driven-development` or `executing-plans`.

- `modules/subagent-driven-development/index.md`
  - Load when an implementation plan already exists and task-by-task execution in the current workflow is the best fit.
  - This module can pull in review, verification, debugging, parallel dispatch, and workspace-isolation support as needed.

- `modules/executing-plans/index.md`
  - Load when an implementation plan already exists but inline execution is a better fit than subagent-driven execution.
  - This module should still respect review, verification, and finishing handoffs.

- `modules/requesting-code-review/index.md`
  - Load when completed work, a finished task batch, or a completed feature needs structured review.
  - This module should hand off to `receiving-code-review` when feedback arrives.

- `modules/receiving-code-review/index.md`
  - Load when review feedback has arrived and must be evaluated, applied, or pushed back on.
  - This module should route either back into the relevant execution module for fixes, back to `requesting-code-review` for re-review, or onward to `finishing-a-development-branch` if review is resolved.

- `modules/finishing-a-development-branch/index.md`
  - Load when implementation and required reviews are complete and the workflow needs a structured finish / merge / PR / cleanup decision.

### Cross-cutting module catalog

Use these as supporting internal modules when relevant. Do not load all of them by default.

- `modules/test-driven-development/index.md`
  - Load when feature work or bug fixes should be driven by explicit failing tests and tight red-green discipline.

- `modules/systematic-debugging/index.md`
  - Load when the task centers on diagnosis, root-cause isolation, or flaky/unclear failure behavior.

- `modules/verification-before-completion/index.md`
  - Load before claiming work is done, before finishing a branch, or whenever completion depends on fresh verification evidence.

- `modules/dispatching-parallel-agents/index.md`
  - Load when a plan or execution phase contains independent tasks that should be delegated in parallel.

- `modules/using-git-worktrees/index.md`
  - Load only when the agent decides workspace isolation will reduce risk or improve coordination. This is an internal execution decision, not something the user needs to request explicitly.

`using-git-worktrees` remains an optional workspace-isolation mechanism. The agent or the active execution module should decide to load it when isolated workspaces would help.

## Workflow Discipline After Activation

Once the user has entered the workflow for the current task:

1. Do not drop back into casual "just edit it" mode.
2. Move through the stages deliberately: clarify → spec → plan → execute → review → finish.
3. Route into internal modules by phase and current state, like selecting subskills inside the workflow.
4. Keep the workflow active until the task is finished or the user explicitly exits it.

## Artifact Paths

Within this package, use these paths:

- Specs: `docs/superpowers/specs/`
- Plans: `docs/superpowers/plans/`
- Brainstorm companion state: `.superpowers/brainstorm/`

When a response names future artifact locations, prefer these exact path families instead of generic alternatives like `docs/specs` or `docs/plans`.

Design documents live at `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`.

Implementation plans live in directories at `docs/superpowers/plans/YYYY-MM-DD-<topic>/`, with `index.md` as the plan index and `task-<NN>-<task-name>.md` files for task details.

For brainstorming sessions, preserve the upstream fallback path:

- `/tmp/brainstorm-<session>`

## Platform Adaptation

Internal modules use Claude Code tool names. Non-CC platforms: see `references/copilot-tools.md` (Copilot CLI), `references/codex-tools.md` (Codex) for tool equivalents. Gemini CLI users can use `references/gemini-tools.md` for the same mapping.

## Exit Conditions

Exit the workflow when any of these is true:

1. The user explicitly asks to exit the superpowers workflow
2. The current task is complete and the finishing phase is done
3. The user asks to switch back to normal mode
