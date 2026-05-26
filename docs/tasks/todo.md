# Task List: Monitoring Charts Account Dimension Adjustment

Status: Draft for human review. Do not implement until approved.

## Phase 1: Backend Contract

- [x] Task 1: Switch chart API contract to account
  - Acceptance: `account` query parsing works; chart response has `filters.account`, `options.accounts`, and `byAccount`; empty responses include empty account options/series; route tests pass account through to store.
  - Verify: `go test ./internal/usage ./internal/httpapi`
  - Dependencies: None
  - Files: `internal/usage/charts.go`, `internal/httpapi/info.go`, `internal/httpapi/info_test.go`, optional `internal/usage/charts_test.go`

## Checkpoint: Backend Contract

- [x] Endpoint path remains `/v0/management/usage/charts`.
- [x] Empty store response remains valid.
- [x] Focused backend contract tests pass.

## Phase 2: Backend Data Path

- [x] Task 2: Aggregate account series from usage events
  - Acceptance: store returns account options and `byAccount.series`; account filter constrains global/account/caller-key/model series; caller-key/model behavior, cost, TPM, and missing-price behavior are preserved.
  - Verify: `go test ./internal/store` and `go test ./internal/usage ./internal/store ./internal/httpapi`
  - Dependencies: Task 1
  - Files: `internal/store/usage_charts.go`, `internal/store/usage_charts_test.go`

## Checkpoint: Backend Data Path

- [x] Account series tests cover account snapshot and auth label/file snapshot fallback.
- [x] Provider series/options expectations are removed from chart store tests.
- [x] Backend focused tests pass.

## Phase 3: Frontend Query And Rendering

- [x] Task 3: Update frontend chart types and filter query path
  - Acceptance: frontend chart query params use `account`; response types use `options.accounts` and `byAccount`; dimension type is `global | account | apiKey | model`; filter tests cover account inclusion/omission.
  - Verify: `npm --prefix web test -- src/features/monitoring/charts/filters.test.ts`
  - Dependencies: Task 1
  - Files: `web/src/services/api/usageService.ts`, `web/src/features/monitoring/charts/filters.ts`, `web/src/features/monitoring/charts/filters.test.ts`

- [x] Task 4: Render account dimension and account filter in chart page
  - Acceptance: provider select is gone; account select appears except when account is active dimension; account dimension renders account series; caller-key/model dimensions still work.
  - Verify: `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx`
  - Dependencies: Task 3
  - Files: `web/src/pages/MonitoringChartsPage.tsx`, `web/src/pages/MonitoringChartsPage.test.tsx`

- [ ] Task 5: Update chart labels and remove provider copy
  - Acceptance: English chart labels use `Account` and `Caller key`; Simplified Chinese chart labels use `账号` and `调用方密钥`; Traditional Chinese and Russian chart labels are updated consistently; page tests no longer depend on stale provider/API-key labels.
  - Verify: `npm --prefix web test -- src/features/monitoring/charts/filters.test.ts src/pages/MonitoringChartsPage.test.tsx` and `npm --prefix web run type-check`
  - Dependencies: Task 4
  - Files: `web/src/i18n/locales/en.json`, `web/src/i18n/locales/zh-CN.json`, `web/src/i18n/locales/zh-TW.json`, `web/src/i18n/locales/ru.json`, optional `web/src/pages/MonitoringChartsPage.test.tsx`

## Checkpoint: Frontend Chart UI

- [ ] No provider chart dimension or filter appears in tests.
- [ ] Account, caller-key, and model filters hide correctly when active.
- [ ] Caller-key wording appears in chart controls and related chart copy.
- [ ] Focused frontend tests and type check pass.

## Phase 4: Final Verification

- [ ] Task 6: Run final checks and review diff
  - Acceptance: focused backend tests pass; focused frontend tests pass; type check passes; lint passes if frontend files changed; generated static asset is not touched unless approved and rebuilt via `npm --prefix web run build`.
  - Verify: `go test ./internal/usage ./internal/store ./internal/httpapi`; `npm --prefix web test -- src/features/monitoring/charts/filters.test.ts src/pages/MonitoringChartsPage.test.tsx`; `npm --prefix web run type-check`; `npm --prefix web run lint`; `go test ./...` if full backend verification is needed.
  - Dependencies: Tasks 1-5
  - Files: None unless generated asset rebuild is approved

## Checkpoint: Complete

- [ ] All `docs/SPEC.md` acceptance criteria are satisfied.
- [ ] No unrelated monitoring, quota, Codex inspection, packaging, or config code was changed.
- [ ] Human review approves the completed implementation.
