---
description: Break work into small verifiable tasks with acceptance criteria and dependency ordering
---

Use internal module `modules/planning-and-task-breakdown/index.md`.

Read the existing spec from `<project_root>/docs/SPEC.md` or equivalent and the relevant codebase sections. Then:

1. Enter plan mode — read only, no code changes
2. Identify the dependency graph between components
3. Slice work vertically (one complete path per task, not horizontal layers)
4. Write tasks with acceptance criteria and verification steps
5. Add checkpoints between phases
6. Present the plan for human review

Save the plan to `<project_root>/docs/plan.md` and the task list to `<project_root>/docs/todo.md`.

If `<project_root>/docs/SPEC.md` is missing, stop before planning, report that the spec could not be found, and still state that any future plan artifacts for this command belong at `<project_root>/docs/plan.md` and `<project_root>/docs/todo.md`.
