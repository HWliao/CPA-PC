# Implementation Plan: Monitoring Charts Account Dimension Adjustment

Status: Draft for human review. No implementation has started.

Source spec: `docs/SPEC.md`

## Overview

Adjust the existing `/monitoring/charts` feature so chart dimensions and filters use account instead of provider, while caller-key terminology replaces user-facing API-key wording on the chart page. The work changes the chart API contract, SQLite aggregation, frontend API types, chart filter state, page rendering, labels, and unit tests. Request monitoring behavior is used as the account semantics reference but should not be refactored.

## Architecture Decisions

- Replace chart provider contract fields with account-oriented fields for this feature rather than adding compatibility aliases, unless an external consumer is discovered and approved.
- Keep existing chart endpoint path: `GET /v0/management/usage/charts`.
- Keep existing chart metrics, ranges, granularity, pricing formulas, and chart rendering mechanics unchanged.
- Keep technical caller-key identifiers such as `apiKeyHash`; only user-facing chart labels change to `Caller key` and `调用方密钥`.
- Match request monitoring account semantics where the chart data path can support it: auth-file metadata account/email/label when available, then event account snapshot, auth label/file snapshot, source display name, auth index, then auth label/source grouping fallback. Provider must not be an account fallback.
- Do not add dependencies, database schema changes, new services, or request monitoring refactors.

## Dependency Graph

```text
docs/SPEC.md
  -> Backend chart contract in internal/usage/charts.go
    -> HTTP route query parsing in internal/httpapi/info.go
      -> Store aggregation in internal/store/usage_charts.go
        -> Frontend API types/client in web/src/services/api/usageService.ts
          -> Frontend chart filter helpers in web/src/features/monitoring/charts/filters.ts
            -> MonitoringChartsPage controls and series selection
              -> i18n labels and page tests
```

Cross-reference dependency:

```text
web/src/features/monitoring/hooks/useMonitoringData.ts
  -> account fallback semantics reference only
  -> no planned request monitoring behavior change
```

## Task List

### Phase 1: Backend Contract

## Task 1: Switch Chart API Contract To Account

**Description:** Update the backend chart query and response contract so the route accepts `account`, exposes account filters/options/series, and no longer exposes provider as the chart dimension/filter contract.

**Acceptance criteria:**
- [ ] `usage.ParseChartQuery` reads `account` and no longer normalizes provider/auth-index filters for chart UI behavior.
- [ ] `usage.ChartsResponse` exposes `filters.account`, `options.accounts`, and `byAccount`.
- [ ] `usage.EmptyChartsResponse` returns empty account options and series.
- [ ] Route tests prove `/v0/management/usage/charts?account=...&apiKeyHash=...&model=...` passes the account filter to the store.

**Verification:**
- [ ] Tests pass: `go test ./internal/usage ./internal/httpapi`

**Dependencies:** None

**Files likely touched:**
- `internal/usage/charts.go`
- `internal/httpapi/info_test.go`
- `internal/httpapi/info.go`
- `internal/usage/charts_test.go` if focused parser tests are added

**Estimated scope:** Medium, 3-4 files

### Checkpoint: Backend Contract

- [ ] Human-readable contract matches `docs/SPEC.md`.
- [ ] Route still uses `/v0/management/usage/charts`.
- [ ] Empty store response remains valid.
- [ ] Focused backend contract tests pass.

### Phase 2: Backend Data Path

## Task 2: Aggregate Account Series From Usage Events

**Description:** Replace provider aggregation in the SQLite chart store with account aggregation, preserving global, caller-key, model, cost, TPM, missing-price, and bucket behavior.

**Acceptance criteria:**
- [ ] Store returns account options and `byAccount.series` using account fallback semantics available from usage events.
- [ ] Account filtering constrains global buckets, account series, caller-key series, and model series.
- [ ] Caller-key and model dimensions continue to work after provider removal.
- [ ] Missing-price model reporting and cost calculation remain unchanged.

**Verification:**
- [ ] Tests pass: `go test ./internal/store`
- [ ] Backend focused tests pass: `go test ./internal/usage ./internal/store ./internal/httpapi`

**Dependencies:** Task 1

**Files likely touched:**
- `internal/store/usage_charts.go`
- `internal/store/usage_charts_test.go`

**Estimated scope:** Small, 2 files

### Checkpoint: Backend Data Path

- [ ] Account series are populated from representative account snapshot and auth label/file snapshot rows.
- [ ] No provider series/options remain in chart store tests.
- [ ] Global totals match previous behavior for the same events.
- [ ] Backend focused test command passes.

### Phase 3: Frontend Query And Rendering

## Task 3: Update Frontend Chart Types And Filter Query Path

**Description:** Update frontend chart API types and pure filter helpers so account replaces provider and caller-key remains keyed by `apiKeyHash`.

**Acceptance criteria:**
- [ ] `UsageChartsQueryParams` uses `account` instead of `provider`.
- [ ] `UsageChartsResponse` uses `options.accounts` and `byAccount` instead of provider fields.
- [ ] `UsageChartsDimension` is exactly `global | account | apiKey | model`.
- [ ] Filter helper tests prove account is included in query params and omitted when account is the active dimension.

**Verification:**
- [ ] Tests pass: `npm --prefix web test -- src/features/monitoring/charts/filters.test.ts`

**Dependencies:** Task 1

**Files likely touched:**
- `web/src/services/api/usageService.ts`
- `web/src/features/monitoring/charts/filters.ts`
- `web/src/features/monitoring/charts/filters.test.ts`

**Estimated scope:** Small, 3 files

## Task 4: Render Account Dimension And Account Filter In Chart Page

**Description:** Update `MonitoringChartsPage` so it displays account options, uses account series, hides the account filter when account is the active dimension, and still renders global, caller-key, and model charts.

**Acceptance criteria:**
- [ ] Provider select is removed from the chart page.
- [ ] Account select appears when account is not the active dimension.
- [ ] Account select is hidden and omitted from loader params when account is the active dimension.
- [ ] Chart panels render account series when the account dimension is selected.
- [ ] Caller-key and model dimensions still render their series.

**Verification:**
- [ ] Tests pass: `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx`

**Dependencies:** Task 3

**Files likely touched:**
- `web/src/pages/MonitoringChartsPage.tsx`
- `web/src/pages/MonitoringChartsPage.test.tsx`

**Estimated scope:** Small, 2 files

## Task 5: Update Chart Labels And Remove Provider Copy

**Description:** Update chart-specific locale strings so provider copy is no longer used for dimensions/filters and caller-key terminology is consistent.

**Acceptance criteria:**
- [ ] English chart labels use `Account` and `Caller key`.
- [ ] Simplified Chinese chart labels use `账号` and `调用方密钥`.
- [ ] Traditional Chinese and Russian locale chart labels are updated consistently instead of leaving stale provider/API-key wording.
- [ ] Page tests do not depend on stale `Provider` or `API key` labels for chart controls.

**Verification:**
- [ ] Tests pass: `npm --prefix web test -- src/features/monitoring/charts/filters.test.ts src/pages/MonitoringChartsPage.test.tsx`
- [ ] Type check passes: `npm --prefix web run type-check`

**Dependencies:** Task 4

**Files likely touched:**
- `web/src/i18n/locales/en.json`
- `web/src/i18n/locales/zh-CN.json`
- `web/src/i18n/locales/zh-TW.json`
- `web/src/i18n/locales/ru.json`
- `web/src/pages/MonitoringChartsPage.test.tsx` if label assertions need final alignment

**Estimated scope:** Medium, 4-5 files

### Checkpoint: Frontend Chart UI

- [ ] No provider chart dimension or filter appears in tests.
- [ ] Account, caller-key, and model filters hide correctly when used as the active dimension.
- [ ] Caller-key wording is visible in chart controls and related empty/series copy.
- [ ] Focused frontend tests and type check pass.

### Phase 4: Final Verification

## Task 6: Run Final Checks And Review Diff

**Description:** Run the agreed verification commands, inspect the diff for accidental unrelated changes, and decide whether a generated static asset rebuild is required.

**Acceptance criteria:**
- [ ] Focused backend tests pass.
- [ ] Focused frontend tests pass.
- [ ] Frontend type check passes.
- [ ] Frontend lint passes if frontend files changed.
- [ ] `static/management.html` is only updated through `npm --prefix web run build` if explicitly needed.

**Verification:**
- [ ] `go test ./internal/usage ./internal/store ./internal/httpapi`
- [ ] `npm --prefix web test -- src/features/monitoring/charts/filters.test.ts src/pages/MonitoringChartsPage.test.tsx`
- [ ] `npm --prefix web run type-check`
- [ ] `npm --prefix web run lint`
- [ ] `go test ./...` if backend contract changes need full-suite verification

**Dependencies:** Tasks 1-5

**Files likely touched:** None unless generated asset rebuild is approved

**Estimated scope:** Small, verification only

### Checkpoint: Complete

- [ ] All `docs/SPEC.md` acceptance criteria are satisfied.
- [ ] No unrelated monitoring, quota, Codex inspection, packaging, or config code was changed.
- [ ] Human review confirms the plan's scope was followed.

## Risks And Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Chart backend may not have live auth-file metadata that request monitoring uses before snapshots. | Medium | Use persisted event snapshots and auth labels for chart account grouping; pause and ask before adding schema or passing auth-file metadata into chart aggregation. |
| Removing provider response fields may break hidden consumers. | Medium | Follow the spec and do not add compatibility by default; ask first if external consumers are discovered. |
| Locale updates may leave stale provider/API-key strings in non-primary locales. | Low | Search chart locale keys and update all locale files consistently. |
| Frontend tests use label text and may become brittle after terminology changes. | Low | Update tests to assert intended account/caller-key behavior, not implementation details. |

## Parallelization Opportunities

- Tasks 1 and 2 must be sequential because store aggregation depends on the backend contract.
- Task 3 can start after Task 1 and does not require Task 2 to be complete if the response contract is stable.
- Task 5 can be prepared after Task 3 but should be finalized after Task 4 confirms exact labels used by the page.
- Task 6 must be last.

## Open Questions

- None blocking for planning. During implementation, pause if exact request-monitoring account precedence requires live auth-file metadata unavailable to the chart store without broader architecture changes.
