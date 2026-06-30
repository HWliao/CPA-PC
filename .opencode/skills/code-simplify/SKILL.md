---
name: code-simplify
description: "Use this skill whenever the user wants to simplify, refactor, clean up, or humanize code while preserving behavior, especially AI-generated-looking code. Trigger for Chinese and English requests such as 代码简化, 去除AI味道, 去AI痕迹, 去机器味, 精简AI生成代码, humanize code, de-AI, remove AI-looking code, remove AI slop, simplify generated code, clean up generated code, or make code look reviewed by a senior engineer. Do not use for broad redesigns, feature implementation, performance tuning, formatting-only requests, security audits, or code review findings unless the user also asks to simplify/refactor the code."
metadata:
  pattern: reviewer-pipeline
  domains: code-quality, refactoring, ai-generated-code-cleanup
---

# Code Simplify

Use this skill to simplify AI-generated-looking code and remove mechanical patterns while preserving exact behavior. The goal is code that looks like it has been reviewed by a pragmatic senior engineer, not code that is merely shorter.

## Core Principle

Prefer the smallest safe improvement. False negatives are acceptable; behavior regressions are not.

## Workflow

1. Determine scope.
2. Read project context.
3. Identify AI-looking or overcomplicated patterns.
4. Delegate suitable independent subtasks to subagents when this reduces main-session context pressure.
5. Apply minimal safe edits.
6. Review the diff for behavior risk.
7. Run targeted verification when feasible.
8. Report files changed, skipped risks, and verification.

## Scope Selection

Choose the narrowest useful scope:

1. If the user names files, directories, symbols, or pasted code, process only that target.
2. If no target is named but the current conversation modified files, process only those touched files.
3. If the user explicitly asks to clean up the current branch, inspect branch-changed files with git and process relevant source/test files.
4. Ask before repository-wide cleanup, broad architectural refactors, generated output changes, or bulk removal of a repeated pattern that may be intentional.

Skip generated, vendored, minified, lock, snapshot, binary, and migration files unless the user explicitly asks to edit them.

## Subagent Delegation

Use subagents when the task is broad enough that loading all context in the main conversation would be wasteful. Keep single-file or small local simplifications in the main conversation.

Good delegation targets:

- Branch-level cleanup with multiple changed files.
- Several independent files or components named by the user.
- Large files where a read-only pass can identify candidate line ranges before editing.
- Independent test files that need duplicate-boilerplate cleanup.

Delegation rules:

- Use read-only exploration subagents to map candidate files, suspicious patterns, and relevant project conventions before reading large amounts of code in the main session.
- Use editing subagents only for narrow, independent file-level cleanup where changes cannot conflict with other active edits.
- Give each subagent one file or one tightly related file group, plus the behavior-preservation rules from this skill.
- Ask subagents to return a concise report: files inspected, changes made, risky candidates skipped, and verification performed.
- Launch independent subagents in parallel when the runtime supports it.
- Do not delegate the final safety decision. The main session must review the combined diff and own the final response.
- Do not use subagents for tiny edits, uncertain cross-file refactors, public API changes, or changes that require one shared design decision.

## Context Priority

Resolve conflicts in this order:

1. User's explicit instruction.
2. Repository instructions such as `AGENTS.md`.
3. Local style in nearby code.
4. Project docs such as README, contributing guides, or `CLAUDE.md` if present.
5. This skill's default rules.

If a requested simplification conflicts with a higher-priority instruction, explain the conflict and keep the safer behavior.

## What To Remove Or Simplify

Treat these as candidates, not automatic edits:

- Obvious comments that restate code, section banners, commented-out code, vague `TODO`, vague `Note`, and trivial docstrings.
- Over-defensive code for values proven non-null or already validated in the same scope.
- Meaningless `try/catch` or fallback defaults around code that cannot fail in context.
- One-off wrappers, helpers, abstractions, or re-exports that add indirection without reuse or clarity.
- Deep nested conditionals that can become guard clauses or clearer branches.
- Nested ternaries and dense one-liners that make control flow harder to read.
- Repeated boilerplate branches or tests that can be collapsed without hiding important cases.
- Generic names such as `data`, `result`, `item`, or `temp` when nearby context supports a clearer name.

## What To Keep

Preserve code that explains or protects real behavior:

- Comments that explain why, business rules, edge cases, workarounds, regexes, non-obvious algorithms, or issue links.
- Boundary validation for user input, files, network calls, external APIs, database rows, environment variables, and untrusted data.
- Error handling for I/O, network, file, process, concurrency, transaction, and cleanup operations.
- Public API signatures, exported names, serialized shapes, persisted data formats, migrations, and config semantics.
- Type hints, annotations, explicit interface contracts, and test assertions that document expected behavior.
- Existing BDD or test-structure comments such as `given`, `when`, and `then` when they match project style.
- Helpful abstractions that separate concerns or make future changes safer.

## Safety Check Before Each Edit

For every candidate change, ask:

- Could this change return values, side effects, thrown errors, logs, metrics, timing, retries, cleanup, or ordering?
- Is the removed code guarding a system boundary or unreliable dependency?
- Is the pattern required by framework conventions, generated contracts, tests, or public consumers?
- Would a reader lose useful intent or business context?
- Is this edit outside the requested scope?

If the answer is yes or unclear, skip the edit and mention it in the final report when relevant.

## Editing Rules

- Make the smallest behavior-preserving edits.
- Prefer deleting noise over rewriting working code.
- Keep changes local unless a small extraction or rename clearly improves readability.
- Do not chase line-count reduction; clarity beats compactness.
- Do not introduce new helper names unless they remove duplicated logic or clarify a real concept.
- Do not add compatibility shims unless there is persisted data, shipped behavior, external consumers, or explicit user need.
- Avoid formatting-only churn outside the edited lines.

## Branch-Level Cleanup

Only use branch-level cleanup when the user asks for it. Prefer explicit targets over branch scans.

When cleaning a branch:

1. Inspect changed files with git.
2. Filter to relevant source and test files.
3. Use subagents for independent file-level analysis or cleanup when there are multiple relevant files.
4. Process files directly in the main session when the target set is small or cross-file reasoning is required.
5. After edits, review the combined diff for accidental behavior changes.
6. Never use destructive rollback commands that discard user or branch work.

## Verification

After editing, verify at the narrowest reliable level available:

- Review the diff yourself.
- Run targeted tests for touched code when commands are discoverable and affordable.
- Run typecheck, lint, build, or syntax checks when they are the natural project verification.
- If verification is skipped, say why.

## Final Report

Keep the final response concise. Include:

- Files changed.
- Main simplification categories applied.
- Risky-looking items deliberately preserved, if any.
- Verification performed or skipped.

Avoid long before/after dumps unless the user requests them.
