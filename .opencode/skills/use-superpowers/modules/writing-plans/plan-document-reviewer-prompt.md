# Plan Document Reviewer Prompt Template

Use this template when dispatching a plan document reviewer task.

**Purpose:** Verify the plan is complete, matches the spec, and has proper task decomposition.

**Dispatch after:** The complete plan is written.

```
Task-dispatch payload (plan-document-reviewer or general-purpose):
  description: "Review plan document"
  prompt: |
    You are a plan document reviewer. Verify this plan is complete and ready for implementation.

    **Plan index to review:** [PLAN_INDEX_PATH]
    **Task files to review:** [TASK_FILE_PATHS]
    **Spec for reference:** [SPEC_FILE_PATH]

    ## What to Check

    | Category | What to Look For |
    |----------|------------------|
    | Completeness | TODOs, placeholders, incomplete tasks, missing steps |
    | Spec Alignment | Plan covers spec requirements, no major scope creep |
    | Index Structure | `index.md` has header, `## Task List`, and `## Reference Files` |
    | Task Files | Every task row points to a real, self-contained task detail file |
    | Task State | Initial statuses use only `pending`, `processing`, `done` and new tasks start as `pending` |
    | Dependencies | `Blocked By` references are valid, justified, and form a DAG |
    | Task Decomposition | Tasks have clear boundaries, steps are actionable |
    | Buildability | Could an engineer follow this plan without getting stuck? |

    ## Calibration

    **Only flag issues that would cause real problems during implementation.**
    An implementer building the wrong thing or getting stuck is an issue.
    Minor wording, stylistic preferences, and "nice to have" suggestions are not.

    Approve unless there are serious gaps — missing requirements from the spec,
    contradictory steps, placeholder content, invalid task dependencies, missing task files,
    or tasks so vague they can't be acted on.

    ## Output Format

    ## Plan Review

    **Status:** Approved | Issues Found

    **Issues (if any):**
    - [Task X / index.md / task file]: [specific issue] - [why it matters for implementation]

    **Recommendations (advisory, do not block approval):**
    - [suggestions for improvement]
```

**Reviewer returns:** Status, Issues (if any), Recommendations
