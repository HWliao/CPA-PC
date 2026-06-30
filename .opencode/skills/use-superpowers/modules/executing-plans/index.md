---
name: executing-plans
description: Use when you have a written implementation plan to execute in a separate session with review checkpoints
---

# Executing Plans

## Overview

Load the plan index, review critically, execute tasks in dependency order, maintain task status in the plan index, and report when complete.

**Announce at start:** "I'm routing into the executing-plans module to implement this plan."

**Note:** Tell your human partner that Superpowers works much better with access to subagents. The quality of its work will be significantly higher if run on a platform with subagent support (such as Claude Code or Codex). If subagents are available, route into `subagent-driven-development` instead of this module.

## Module Dependency

- **Expected upstream module:** `writing-plans`
- **Preferred sibling module when available:** `subagent-driven-development`
- **Optional supporting modules:** `using-git-worktrees`, `test-driven-development`, `systematic-debugging`, `verification-before-completion`
- **Review handoff:** `requesting-code-review` / `receiving-code-review`
- **Terminal downstream module:** `finishing-a-development-branch`

## The Process

### Step 1: Load and Review Plan
1. Read the plan `index.md`, usually at `docs/superpowers/plans/YYYY-MM-DD-<topic>/index.md`.
2. Parse the `## Task List` table and identify each task's number, title, task file, status, and blockers.
3. Read the task detail files referenced by `Task File` before starting execution.
4. Review critically - identify missing task files, invalid statuses, circular dependencies, unclear blockers, or task instructions that cannot be followed.
5. If concerns: Raise them with your human partner before starting.
6. If no concerns: Create a task-tracking checklist from `index.md` and proceed.

### Step 2: Maintain Plan Task Status

The plan index is the source of truth for execution state.

- Allowed statuses are only `pending`, `processing`, and `done`.
- Before starting a task, update that task's `Task Status` in `index.md` from `pending` to `processing`.
- After the task is implemented and verification passes, update its `Task Status` from `processing` to `done`.
- If a task blocks or verification repeatedly fails, stop and ask for help. Do not invent a new status; leave the current status visible in `index.md` and explain the blocker.
- Do not mark a task `done` until its required verification has passed.

### Step 3: Execute Tasks

For each task:
1. Choose a task whose status is `pending` and whose `Blocked By` tasks are all `done`.
2. Update the task row in `index.md` to `processing`.
3. Read the referenced task file if it is not already loaded.
4. Follow each step exactly (task files have bite-sized steps).
5. Run verifications as specified.
6. Update the task row in `index.md` to `done` only after verification passes.

Respect `Blocked By` as a directed acyclic graph. If no `pending` task is eligible but tasks remain unfinished, the plan has a dependency or status problem; stop and ask rather than guessing.

### Step 4: Complete Development

After all tasks in `index.md` are `done` and verified:
- Announce: "I'm routing into the finishing-a-development-branch module to complete this work."
- **REQUIRED INTERNAL MODULE:** Route into `finishing-a-development-branch`
- Follow that module to verify tests, present options, execute choice

## When to Stop and Ask for Help

**STOP executing immediately when:**
- Hit a blocker (missing dependency, test fails, instruction unclear)
- Plan has critical gaps preventing starting
- `index.md` is missing, malformed, or points to missing task files
- The task graph has circular dependencies or no eligible pending task
- You don't understand an instruction
- Verification fails repeatedly

**Ask for clarification rather than guessing.**

## When to Revisit Earlier Steps

**Return to Review (Step 1) when:**
- Partner updates the plan based on your feedback
- Fundamental approach needs rethinking

**Don't force through blockers** - stop and ask.

## Remember
- Review plan critically first
- Treat `index.md` as the source of truth for task status
- Follow plan steps exactly
- Don't skip verifications
- Update task status before starting and after successful verification
- Reference internal modules when the plan says to
- Stop when blocked, don't guess
- Never start implementation on main/master branch without explicit user consent

## Integration

**Integrated workflow modules:**
- **`writing-plans`** - Creates the plan this module executes
- **`finishing-a-development-branch`** - Complete development after all tasks

**Optional workspace isolation:**
- **`using-git-worktrees`** - Load this when the agent decides an isolated workspace will reduce execution risk or coordination conflicts
