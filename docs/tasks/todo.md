# Tasks: CPA SDK 7.2.20 Upgrade And Upstream Management Diff

Status: Implemented and verified.

Spec: `docs/SPEC.md`

Plan: `docs/tasks/plan.md`

## Task List

- [x] Task 1: Capture baseline and create report scaffolds
  - Acceptance: `docs/reports/cpa-sdk-7.2.20-upgrade.md` and `docs/reports/management-ui-upstream-diff.md` exist with baseline sections for current CPA-PC SDK version, reference source paths, source revision/status when available, and report structure for impact/additions/modifications/breaking changes.
  - Verify: confirm both report files exist and include baseline, scope, and pending-verification sections.
  - Files: `docs/reports/cpa-sdk-7.2.20-upgrade.md`, `docs/reports/management-ui-upstream-diff.md`.
  - Depends on: confirmed plan.

- [x] Task 2: Analyze SDK changes from `v7.1.39` to `v7.2.20`
  - Acceptance: SDK report summarizes relevant upstream changes affecting CPA-PC integration points: startup, config loading, management routes, auth, static serving, usage plugin hooks, logging, runtime paths, public APIs, and dependency shifts.
  - Verify: report includes exact comparison basis from `../CLIProxyApi`, plus impact/additions/modifications/breaking-change classification.
  - Files: `docs/reports/cpa-sdk-7.2.20-upgrade.md`.
  - Depends on: Task 1.

- [x] Task 3: Upgrade CLIProxyAPI SDK dependency
  - Acceptance: root `go.mod` requires `github.com/router-for-me/CLIProxyAPI/v7 v7.2.20`, `go.sum` is updated consistently, and any compile-time compatibility fixes are minimal and scoped to CPA-PC integration.
  - Verify: run `go get github.com/router-for-me/CLIProxyAPI/v7@v7.2.20`, `go mod tidy`, and `go test ./...`; record results in the SDK report.
  - Files: `go.mod`, `go.sum`, possible focused files under `cmd/cpa-pc/`, `internal/app/`, `internal/config/`, `internal/httpapi/` only if required by compile/test failures.
  - Depends on: Task 2.

- [x] Task 4: Compare official management UI current tree with CPA-PC `web/`
  - Acceptance: UI report identifies relevant current-tree differences for routes, API clients, auth handling, config editor, model/API-key pages, request monitoring, usage pages, i18n, build setup, styling, and generated asset implications.
  - Verify: report includes adopt/preserve/conflict decisions and impact/additions/modifications/breaking-change classification.
  - Files: `docs/reports/management-ui-upstream-diff.md`.
  - Depends on: Task 1.

- [x] Task 5: Sync relevant official management UI changes into CPA-PC
  - Acceptance: approved relevant upstream UI differences are applied to `web/` without removing CPA-PC-specific behavior: hash router, usage pages, CPA-PC info, API-key aliases, local model prices, i18n, and single-file build setup.
  - Verify: run `npm --prefix web run type-check` and `npm --prefix web run lint`; record results in the UI report.
  - Files: targeted files under `web/` only, plus `docs/reports/management-ui-upstream-diff.md`.
  - Depends on: Task 4.

- [x] Task 6: Rebuild generated management asset after frontend changes
  - Acceptance: if Task 5 changes `web/`, root `static/management.html` is regenerated through the existing build script and not edited by hand.
  - Verify: run `npm --prefix web run build`; confirm `static/management.html` is updated only by the build pipeline and record result in the UI report.
  - Files: `static/management.html`, possible `web/dist/` transient output ignored by repository rules, `docs/reports/management-ui-upstream-diff.md`.
  - Depends on: Task 5.

- [x] Task 7: Final verification and summary reporting
  - Acceptance: final reports summarize actual impact, additions, modifications, breaking-change status, commands run, pass/fail outcomes, changed files, and residual risks.
  - Verify: run required final checks based on touched files: `go test ./...`; if `web/` changed, `npm --prefix web run type-check`, `npm --prefix web run lint`, and `npm --prefix web run build` must have results recorded.
  - Files: `docs/reports/cpa-sdk-7.2.20-upgrade.md`, `docs/reports/management-ui-upstream-diff.md`.
  - Depends on: Tasks 3 and 6 when frontend changes are made; Tasks 3 and 4 if no frontend changes are applied.

## Implementation Rules

- Complete tasks in dependency order unless a task is explicitly independent.
- Keep code changes minimal and tied to the task acceptance criteria.
- Do not replace the entire `web/` app wholesale.
- Do not hand-edit `static/management.html`.
- Do not introduce `../CLIProxyApi` or the official management UI checkout as a dependency.
- Stop and ask before any change that violates `docs/SPEC.md` boundaries.

## Confirmation Gate

Implementation completed after user confirmation.
