---
name: writing-plans
description: Use when you have a spec or requirements for a multi-step task, before touching code
---

# Writing Plans

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, code, testing, docs they might need to check, how to test it. Give them the whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Announce at start:** "I'm routing into the `writing-plans` internal module to create the implementation plan. After the plan is written, execution must route through `subagent-driven-development` (default) or `executing-plans` (if the user prefers inline execution)."

**Context:** This can be run in a dedicated worktree if the workflow chooses workspace isolation.

**Save plans to:** `docs/superpowers/plans/YYYY-MM-DD-<feature-name>/`
- (User preferences for plan location override this default)
- Every plan is a directory, not one large `plan.md` file.
- The directory MUST contain `index.md` plus one task detail file per task.

## Module Dependency

- **Expected upstream module:** `brainstorming` or an already-approved spec/requirements source
- **Default downstream module:** `subagent-driven-development`
- **Alternate downstream module:** `executing-plans` when inline execution is the better fit
- **Decision rule:** the agent chooses the execution module; it is not chosen only because the user explicitly asked for worktree or session style

## Scope Check

If the spec covers multiple independent subsystems, it should have been broken into sub-project specs during brainstorming. If it wasn't, suggest breaking this into separate plans — one per subsystem. Each plan should produce working, testable software on its own.

## File Structure

Before defining tasks, map out which files will be created or modified and what each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are more reliable when files are focused. Prefer smaller, focused files over large ones that do too much.
- Files that change together should live together. Split by responsibility, not by technical layer.
- In existing codebases, follow established patterns. If the codebase uses large files, don't unilaterally restructure - but if a file you're modifying has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce self-contained changes that make sense independently.

## Plan Directory Layout

Write plans as a directory so long plans do not collapse into one huge file:

```text
docs/superpowers/plans/YYYY-MM-DD-<feature-name>/
├── index.md
├── task-01-<task-name>.md
├── task-02-<task-name>.md
└── task-03-<task-name>.md
```

`index.md` is the controller document. It contains the plan header, task graph, and references. It does not contain every implementation step.

Each `task-<NN>-<task-name>.md` file contains the full task details using the Task Structure below. Task files must be independently readable because execution agents may receive exactly one task file at a time.

Task file naming rules:

- Use two-digit task numbers: `task-01`, `task-02`, `task-03`.
- Use a short English task name, less than 20 characters, in kebab case.
- The task file path stored in `index.md` is relative to `index.md`, for example `task-01-cli-export.md`.

## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Commit" - step

## Plan Document Header

**Every `index.md` MUST start with this header:**

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** Default execution route: `subagent-driven-development`. Use `executing-plans` only if the user explicitly prefers inline execution. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

---
```

## Index Task List

After the header, `index.md` MUST contain a `## Task List` section with this exact table shape:

```markdown
## Task List

| # | Task Title | Task File | Task Status | Blocked By |
|---|------------|-----------|-------------|------------|
| 1 | cli-export | `task-01-cli-export.md` | `pending` | `none` |
| 2 | summary-api | `task-02-summary-api.md` | `pending` | `1` |
```

Task list field rules:

- `#`: sequential integer starting at `1`.
- `Task Title`: short English name, less than 20 characters.
- `Task File`: task detail file path relative to `index.md`.
- `Task Status`: one of `pending`, `processing`, or `done`. New plans start with every task as `pending`.
- `Blocked By`: `none` or a comma-separated list of task numbers that must be `done` first.

When planning dependencies, treat the task list as a directed acyclic graph. Think through input/output relationships, shared files, resource conflicts, and whether work can be isolated before marking a task independent. Never create circular dependencies.

After the task list, add a `## Reference Files` section with the spec path and important code/docs references:

```markdown
## Reference Files

- Spec: `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`
- Existing code: `src/path/file.ts`
- Tests: `tests/path/file.test.ts`
```

## Task Structure

**Each task file `task-<NN>-<task-name>.md` MUST be written according to the following structure:**

````markdown
# Task N: [Task Name]

> **Status source:** Update this task's row in `index.md`; do not invent task status elsewhere.

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
```
````

## No Placeholders

Every step must contain the actual content an engineer needs. These are **plan failures** — never write them:
- "TBD", "TODO", "implement later", "fill in details"
- "Add appropriate error handling" / "add validation" / "handle edge cases"
- "Write tests for the above" (without actual test code)
- "Similar to Task N" (repeat the code — the engineer may be reading tasks out of order)
- Steps that describe what to do without showing how (code blocks required for code steps)
- References to types, functions, or methods not defined in any task
- A single large `plan.md` when the workflow calls for an `index.md` plus task files
- A task file that depends on unstated context from another task file instead of naming the dependency in `Blocked By`

## Remember
- Exact file paths always
- Complete code in every step — if a step changes code, show the code
- Exact commands with expected output
- `index.md` tracks task graph and status; task files hold detailed execution steps
- Task dependencies must form a DAG and reflect real implementation ordering
- DRY, YAGNI, TDD, frequent commits

## Precautions

When you have finished thinking and are ready to output the plan documents (`index.md`, `task-01`, `task-02`, ...), **write them one by one**. Do not output the entire plan document at once, to avoid interruption or freezing due to excessive output size.

## Self-Review

After writing the complete plan, look at the spec with fresh eyes and check the plan against it. This is a checklist you run yourself — not a subagent dispatch.

**1. Spec coverage:** Skim each section/requirement in the spec. Can you point to a task that implements it? List any gaps.

**2. Placeholder scan:** Search your plan for red flags — any of the patterns from the "No Placeholders" section above. Fix them.

**3. Type consistency:** Do the types, method signatures, and property names you used in later tasks match what you defined in earlier tasks? A function called `clearLayers()` in Task 3 but `clearFullLayers()` in Task 7 is a bug.

**4. Plan directory integrity:** Does `index.md` include the required header, `## Task List`, and `## Reference Files` sections? Does every task list row point to an existing task file? Is every task file self-contained?

**5. Task state and dependency check:** Are all initial statuses `pending`? Is every `Blocked By` reference valid? Does the dependency graph avoid cycles? Are independent tasks truly independent, without hidden file/resource conflicts?

If you find issues, fix them inline. No need to re-review — just fix and move on. If you find a spec requirement with no task, add the task.

## Execution Handoff

After saving the plan, offer execution choice:

**"Plan complete and saved to `<plan-directory>/index.md`. Two execution options:**

**1. `subagent-driven-development` (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. `executing-plans` (inline execution)** - Execute tasks in this session using `executing-plans`, batch execution with checkpoints

**Which approach?"**

**If `subagent-driven-development` chosen:**
- **REQUIRED INTERNAL MODULE:** Route into `subagent-driven-development`
- Fresh subagent per task + two-stage review

**If `executing-plans` chosen:**
- **REQUIRED INTERNAL MODULE:** Route into `executing-plans`
- Batch execution with checkpoints for review
