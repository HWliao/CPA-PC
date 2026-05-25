# Implementation Plan: CPA-PC Request Monitoring Charts

Status: Archived after implementation. All tasks were completed before archiving.

Source spec: `docs/specs/2026-05-25-cpa-pc-request-monitoring-charts.md`.

## Overview

Add a new `/monitoring/charts` page that is reachable from the existing Request Monitoring page beside Codex Account Inspection. The page uses one new protected read-only endpoint, `GET /v0/management/usage/charts`, to render ECharts line charts for token structure, cost, and TPM across global, provider/auth-file, API-key, and model dimensions. The implementation keeps CPA-PC single-process, uses the existing local SQLite store, and does not change existing `/v0/management/usage` behavior.

## Architecture Decisions

- No SQLite schema migration: aggregate from existing `usage_events`, `model_prices`, and `api_key_aliases` tables.
- Backend owns time bucket aggregation, cost calculation, filter option generation, missing-price reporting, and auth-safe labels.
- Frontend owns range/filter UI, loading/error/empty states, and ECharts rendering.
- Series are not capped or aggregated; all matching provider/auth-file, API-key, and model series are returned and rendered.
- Use direct `echarts` dependency, not a React wrapper, unless the user approves a change later.
- Layout should follow `MonitoringCenterPage` and `CodexInspectionPage`: page header, status/action panel rhythm, card/panel spacing, responsive behavior, and visual tone.

## Dependency Graph

```text
Existing SQLite tables
  usage_events
  model_prices
  api_key_aliases
        |
        v
internal/usage chart query and response contract
        |
        v
internal/store UsageCharts aggregation
        |
        v
internal/httpapi protected route and validation
        |
        v
web usageService API client and data loader
        |
        v
chart option helpers and ECharts host component
        |
        v
MonitoringChartsPage route, filters, panels, and states
        |
        v
Request Monitoring navigation entry and generated static asset
```

Sequential dependencies:

- Backend route depends on chart query/response types.
- Store aggregation depends on existing SQLite schema and pricing formula.
- Frontend data loading depends on backend API contract.
- ECharts page rendering depends on the API client and chart option helpers.
- Final build/static generation depends on all frontend source changes.

Parallelization opportunities after contracts are stable:

- Store aggregation tests and frontend chart option helper tests can be developed independently after `UsageChartsResponse` is finalized.
- UI styling and i18n labels can be polished after the page shell exists.
- Route tests and store tests can run in parallel once `UsageCharts` is added to the store interface.

## Task List

### Phase 1: Backend Chart API

- [ ] Task 1: Add global chart endpoint slice
- [ ] Task 2: Add dimension series, filters, aliases, and missing-price reporting

### Checkpoint: Backend API

- [ ] `go test ./internal/store ./internal/httpapi ./internal/usage` passes.
- [ ] `GET /v0/management/usage/charts` is protected by existing management auth.
- [ ] Empty store response is valid and does not affect `/v0/management/usage`.
- [ ] User review confirms backend response shape before frontend work proceeds.

### Phase 2: First Frontend Path

- [ ] Task 3: Add frontend chart API client and data loader
- [ ] Task 4: Add chart page shell, route, and monitoring-page entry
- [ ] Task 5: Add ECharts dependency and chart infrastructure
- [ ] Task 6: Render global ECharts line charts

### Checkpoint: First Visible Flow

- [ ] User can navigate from `/monitoring` to `/monitoring/charts`.
- [ ] Page shows loading, error, empty, and global chart data states.
- [ ] `npm --prefix web test -- src/features/monitoring/charts` passes for added helpers.
- [ ] `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx` passes for page state coverage.

### Phase 3: Full Chart Page

- [ ] Task 7: Add filter controls and range/granularity reload behavior
- [ ] Task 8: Render provider/auth-file, API-key, and model chart sections
- [ ] Task 9: Add localization, responsive polish, and final static build

### Checkpoint: Complete

- [ ] `go test ./...` passes.
- [ ] `npm --prefix web run lint` passes.
- [ ] `npm --prefix web run type-check` passes.
- [ ] `npm --prefix web run build` passes and regenerates `static/management.html`.
- [ ] Existing request monitoring tables, import/export, model price settings, account overview, API-key summary, and Codex inspection behavior remain unchanged except for the new navigation entry.
- [ ] Human review approves the implemented feature before integration testing.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Time bucket tests become flaky because current time moves | Medium | Keep bucket normalization deterministic in helpers and use fixed `nowMs` in unit tests where possible. |
| Cost totals differ from existing frontend expectations | Medium | Mirror the existing `calculateCost` formula: prompt `max(input - cached, 0)`, cache `cached`, completion `output`, all per 1M tokens. |
| Chart page becomes visually inconsistent | Medium | Reuse layout patterns from `MonitoringCenterPage.module.scss` and `CodexInspectionPage.module.scss`; avoid a new dashboard visual language. |
| ECharts is difficult to test in Vitest/server rendering | Medium | Keep option builders pure and unit-tested; keep the chart host component thin. |
| Rendering all series clutters charts if data grows unexpectedly | Low | Follow the confirmed requirement now; if needed later, ask before adding capping or grouping. |
| Existing `UsageStore` test fake breaks when the interface changes | Low | Add the new method to `fakeUsageStore` in the same backend route task. |

## Open Questions

- None. The spec is confirmed; implementation still waits for the user's explicit approval.
