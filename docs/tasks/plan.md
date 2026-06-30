# Plan: CPA SDK 7.2.20 Upgrade And Upstream Management Diff

Status: Implemented and verified.

Spec: `docs/SPEC.md`

## Goal

Upgrade CPA-PC to CLIProxyAPI SDK `v7.2.20`, sync relevant current-tree official management UI differences into CPA-PC, rebuild generated management assets when needed, and document both SDK and UI differences.

## Implementation Order

1. Baseline inspection

   Verify worktree state, current module dependency, current upstream SDK tags/history in `../CLIProxyApi`, and official management UI current tree availability.

   Verification: record baseline versions and source paths in the SDK/UI reports.

2. SDK change analysis

   Compare `../CLIProxyApi` from `v7.1.39` to `v7.2.20`, focusing on public SDK APIs and CPA-PC integration points: app startup, config loading, management routes, auth, static serving, usage plugin hooks, logging, runtime paths, and dependencies.

   Verification: draft `docs/reports/cpa-sdk-7.2.20-upgrade.md` with relevant changes, compatibility notes, impact, additions, modifications, and breaking-change status.

3. SDK upgrade

   Run `go get github.com/router-for-me/CLIProxyAPI/v7@v7.2.20`, then `go mod tidy`. Fix compile errors with minimal CPA-PC integration changes.

   Verification: `go test ./...` passes or failures are documented with root cause.

4. Official management UI comparison

   Compare current official UI tree at `D:\develop\source-workspace\Cli-Proxy-API-Management-Center` to CPA-PC `web/`, focusing on routes, API clients, auth, config management, model/API-key pages, request monitoring, usage pages, i18n, build setup, and shared styling.

   Verification: draft `docs/reports/management-ui-upstream-diff.md` with adopt/preserve/conflict notes, impact, additions, modifications, and breaking-change status.

5. UI sync

   Directly sync relevant official management UI changes into CPA-PC `web/` while preserving CPA-PC-specific behavior: hash routing, usage pages, CPA-PC info, API-key aliases, local model prices, i18n, and single-file build output.

   Verification: `npm --prefix web run type-check` and `npm --prefix web run lint` pass after frontend changes.

6. Generated asset rebuild

   If `web/` changes, run `npm --prefix web run build` to regenerate `static/management.html` through the existing build pipeline.

   Verification: generated file exists and no manual edits are made to `static/management.html`.

7. Final verification and report update

   Re-run required backend/frontend checks, update both reports with final decisions and test results, then summarize changed files, impact, additions, modifications, breaking-change status, and residual risks.

   Verification: reports include commands run, pass/fail status, impact, additions, modifications, breaking-change status, and any follow-up recommendations.

## Dependencies

- SDK upgrade depends on `../CLIProxyApi` history being usable for comparison and on Go module resolution for `v7.2.20`.
- UI sync depends on official UI current tree being readable at `D:\develop\source-workspace\Cli-Proxy-API-Management-Center`.
- `static/management.html` rebuild depends on successful `npm --prefix web run build`.
- Packaging smoke is optional unless build outputs or packaging-sensitive files change beyond `static/management.html`.

## Parallelizable Work

- SDK diff analysis and official UI current-tree inventory can be gathered independently.
- Report drafting can happen alongside compatibility fixes as findings are discovered.

## Sequential Work

- Confirm spec before plan.
- Confirm plan before task breakdown.
- Complete task breakdown before implementation.
- Upgrade SDK before final backend verification.
- Apply frontend sync before rebuilding `static/management.html`.
- Run final verification after all code and generated assets are updated.

## Risks And Mitigations

- SDK API breakage: keep fixes minimal and scoped to CPA-PC integration points; prefer compile-guided changes.
- Breaking-change ambiguity: classify every meaningful change as non-breaking, potentially breaking, or breaking, and explain the affected users or maintainers.
- Hidden route conflict: inspect management route ownership and avoid duplicate `/management.html` registration.
- Upstream UI overwrites CPA-PC custom behavior: apply targeted sync only; preserve usage pages, aliases, hash router, and local persistence.
- Generated asset churn: rebuild only through `npm --prefix web run build` and include it only when `web/` changes.
- External reference path drift: record exact source path and git revision/status in reports when available.
- Test flakiness or environment constraints: document exact failing command, failure excerpt, and root cause if a required check cannot pass.

## Verification Checkpoints

1. After baseline inspection: current CPA SDK version and reference source status documented.
2. After SDK upgrade: `go.mod` contains `v7.2.20` and `go test ./...` is run.
3. After UI sync: `npm --prefix web run type-check` and `npm --prefix web run lint` are run.
4. After generated asset rebuild: `npm --prefix web run build` is run and `static/management.html` is regenerated by script.
5. Before final handoff: both reports include final changes, impact, additions, modifications, breaking-change status, commands, results, and residual risks.

## Plan Acceptance Criteria

- The plan preserves all boundaries in `docs/SPEC.md`.
- The plan explicitly includes SDK upgrade, SDK diff report, official UI diff report, direct UI sync, generated asset rebuild, and verification.
- The plan requires a final summary of impact, new additions, modifications, and breaking-change status.
- The implementation order avoids syncing UI blindly before documenting differences.
- No implementation starts until this plan is confirmed.
