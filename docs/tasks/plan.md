# Implementation Plan: Model Price Sync Source Selection And Field Labels

Status: Draft for human review. Implementation not started.

Source spec: `docs/SPEC.md`

## Overview

Update request monitoring model pricing so the UI presents three clear price fields, and one-click sync asks the administrator to choose `embedded` or `model.dev`. The backend keeps the existing model price persistence shape while adding source-aware sync behavior. `model.dev` sync fetches models.dev pricing server-side, matches by provider plus model, applies approved OpenAI fallback rules, maps models.dev costs into the existing three price fields, and preserves unmatched/manual prices.

## Architecture Decisions

- Keep the existing persisted `ModelPrice` fields: `prompt`, `completion`, and `cache`.
- Treat the new labels as UI terminology only: input price, output price, input cache price.
- Keep `POST /v0/management/model-prices/sync` as the sync endpoint and extend its request body with source and provider/model targets.
- Keep `embedded` as a selectable source that returns currently stored prices with `imported = 0` until a real embedded price table is approved.
- Implement `model.dev` fetching in the backend, not the browser.
- Use standard Go `net/http`; do not add a dependency for external fetching.
- Match models.dev by normalized provider plus model.
- Normalize Codex provider/source and `gpt-*` model IDs to OpenAI for matching.
- Map models.dev `cost.input` to `prompt`, `cost.output` to `completion`, and `cost.cache_read` to `cache`.
- If `cost.cache_read` is absent, set `cache` to `cost.input / 10`.
- Preserve existing prices for unmatched models and for models not included in a sync request.

## Dependency Graph

```text
docs/SPEC.md
  -> Backend sync request/response contract in internal/httpapi/info.go
    -> models.dev price fetch/parse/match helpers
      -> backend route tests in internal/httpapi/info_test.go
        -> frontend API types in web/src/services/api/usageService.ts
          -> usage data hook sync signature in web/src/features/monitoring/hooks/useUsageData.ts
            -> MonitoringCenterPage sync modal and provider/model target collection
              -> i18n labels and focused frontend tests
```

Cross-reference dependencies:

```text
internal/store/store.go
  -> ModelPrice persistence and validation remain unchanged unless tests expose a gap

web/src/utils/usage.ts and internal/store/usage_charts.go
  -> cost formula remains unchanged, but labels describe fields differently
```

## Task List

### Phase 1: Backend Contract And Embedded Source

## Task 1: Extend Model Price Sync Request Contract

**Description:** Add a typed sync request body that accepts `source` and provider/model targets while preserving current `embedded` behavior.

**Acceptance criteria:**
- [ ] `POST /v0/management/model-prices/sync` accepts `source: "embedded"` and `models: [{ provider, model }]`.
- [ ] Missing source defaults to `embedded` for compatibility with current frontend calls.
- [ ] `embedded` response still returns stored prices, `imported = 0`, and `skipped = 0`.
- [ ] Invalid source returns a structured bad request error.

**Verification:**
- [ ] `go test ./internal/httpapi`

**Dependencies:** None

**Files likely touched:**
- `internal/httpapi/info.go`
- `internal/httpapi/info_test.go`

**Estimated scope:** Small, 2 files

### Checkpoint: Backend Contract

- [ ] Existing sync clients still work with an empty request body.
- [ ] The route contract can carry provider/model targets for later `model.dev` matching.

### Phase 2: Backend models.dev Sync

## Task 2: Add models.dev Price Fetch And Parse Helpers

**Description:** Implement backend helpers to fetch and parse `https://models.dev/api.json` into a provider/model pricing lookup.

**Acceptance criteria:**
- [ ] External response parsing validates provider IDs, model IDs, and numeric costs.
- [ ] The parser maps `cost.input`, `cost.output`, and `cost.cache_read` into existing `ModelPrice` fields.
- [ ] Missing `cost.cache_read` maps to `cost.input / 10`.
- [ ] Unit tests use injected JSON or a test server; no test calls the live models.dev endpoint.

**Verification:**
- [ ] `go test ./internal/httpapi`

**Dependencies:** Task 1

**Files likely touched:**
- `internal/httpapi/info.go` or a new small helper file under `internal/httpapi`
- `internal/httpapi/info_test.go` or a new focused test file

**Estimated scope:** Medium, 2-3 files

## Task 3: Match And Import models.dev Prices

**Description:** Add `model.dev` sync source handling that matches request targets to parsed models.dev prices and saves imported prices without deleting unmatched/manual entries.

**Acceptance criteria:**
- [ ] Matching uses normalized provider plus model.
- [ ] Codex provider/source and `gpt-*` model IDs normalize to OpenAI.
- [ ] Imported prices set `source = "model.dev"`, a source model identifier, and sync metadata when available in the existing model price shape.
- [ ] Existing prices for unmatched targets remain unchanged.
- [ ] Existing prices for models not included in the sync request remain unchanged.
- [ ] Response `imported` and `skipped` counts reflect matched and unmatched requested targets.

**Verification:**
- [ ] `go test ./internal/httpapi ./internal/store`

**Dependencies:** Task 2

**Files likely touched:**
- `internal/httpapi/info.go` or helper file
- `internal/httpapi/info_test.go` or focused test file
- `internal/store/store.go` only if metadata preservation requires a minimal validation adjustment

**Estimated scope:** Medium, 2-3 files

### Checkpoint: Backend Sync

- [ ] `embedded` and `model.dev` source paths are both tested.
- [ ] No live network calls happen in unit tests.
- [ ] Manual prices are preserved for skipped/unrequested models.

### Phase 3: Frontend API And Sync Flow

## Task 4: Update Frontend Sync Types And Hook Signature

**Description:** Extend the frontend usage-service API client and monitoring hook so sync can send a selected source and provider/model targets.

**Acceptance criteria:**
- [ ] Frontend has a `ModelPriceSyncSource` type with `embedded | model.dev`.
- [ ] Frontend sync targets include provider and model.
- [ ] `usageServiceApi.syncModelPrices` sends `{ source, models }`.
- [ ] `useUsageData.syncModelPrices` exposes the new signature without changing unrelated usage loading behavior.

**Verification:**
- [ ] `npm --prefix web run type-check`

**Dependencies:** Task 1

**Files likely touched:**
- `web/src/services/api/usageService.ts`
- `web/src/features/monitoring/hooks/useUsageData.ts`

**Estimated scope:** Small, 2 files

## Task 5: Add Source Selection Modal To Request Monitoring

**Description:** Change one-click sync so it opens a secondary modal where the administrator chooses `embedded` or `model.dev`, then runs sync with provider/model targets.

**Acceptance criteria:**
- [ ] Clicking one-click sync opens a source selection modal and does not immediately call the sync API.
- [ ] Source modal lists `embedded` and `model.dev`.
- [ ] Confirming a source calls sync once with selected source and collected targets.
- [ ] Targets include model names and best available provider/source metadata.
- [ ] Codex and `gpt-*` matching requirements are supported by backend normalization; frontend does not need broad provider alias logic.
- [ ] Success and failure notifications remain clear.

**Verification:**
- [ ] `npm --prefix web test -- src/pages/MonitoringCenterPage.test.tsx` if focused tests are practical
- [ ] `npm --prefix web run type-check`

**Dependencies:** Task 4

**Files likely touched:**
- `web/src/pages/MonitoringCenterPage.tsx`
- `web/src/pages/MonitoringCenterPage.test.tsx` if tests are added or updated

**Estimated scope:** Medium, 1-2 files

## Task 6: Rename Price Labels In UI Locales

**Description:** Update model price labels from prompt/completion/cache wording to input/output/input-cache wording in supported locales.

**Acceptance criteria:**
- [ ] Simplified Chinese uses `输入价格`, `输出价格`, and `输入缓存价格`.
- [ ] English uses `Input price`, `Output price`, and `Input cache price`.
- [ ] Traditional Chinese and Russian are updated consistently enough to avoid stale prompt/completion wording.
- [ ] The editor and saved-price table still show exactly three price fields.

**Verification:**
- [ ] `npm --prefix web run type-check`
- [ ] `npm --prefix web run lint`

**Dependencies:** None, but should land with Task 5 for coherent UI review

**Files likely touched:**
- `web/src/i18n/locales/en.json`
- `web/src/i18n/locales/zh-CN.json`
- `web/src/i18n/locales/zh-TW.json`
- `web/src/i18n/locales/ru.json`

**Estimated scope:** Small, 4 files

### Checkpoint: Frontend UI

- [ ] Price field labels are clear in the model price modal.
- [ ] Sync source selection is required before sync starts.
- [ ] Frontend type check passes.

### Phase 4: Final Verification

## Task 7: Run Final Checks And Review Diff

**Description:** Run focused verification, inspect the diff for scope drift, and decide whether a generated static asset rebuild is needed.

**Acceptance criteria:**
- [ ] Focused backend tests pass.
- [ ] Frontend type check passes.
- [ ] Frontend lint passes if frontend files changed.
- [ ] Relevant frontend tests pass if added or updated.
- [ ] `static/management.html` is not touched unless explicitly rebuilt through the frontend build command.

**Verification:**
- [ ] `go test ./internal/httpapi ./internal/store`
- [ ] `npm --prefix web run type-check`
- [ ] `npm --prefix web run lint`
- [ ] `npm --prefix web test -- src/pages/MonitoringCenterPage.test.tsx` if focused tests exist
- [ ] `go test ./...` if backend route or helper changes warrant full-suite verification

**Dependencies:** Tasks 1-6

**Files likely touched:** None unless verification exposes a defect

**Estimated scope:** Small, verification only

## Risks And Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Provider metadata may be incomplete in current monitoring rows. | Medium | Backend applies explicit OpenAI fallback for Codex and `gpt-*`; unmatched models are skipped without deleting existing prices. |
| models.dev JSON shape may change. | Medium | Parse defensively and validate numeric fields before import. |
| Live network dependency can make sync fail. | Medium | Return a clear sync failure error and leave existing prices unchanged. |
| Adding source selection could make sync feel slower. | Low | Keep modal small and default/recommend the most likely source if UX allows. |
| Locale changes might leave stale terminology in secondary copy. | Low | Search all model price label keys and update all supported locales. |

## Parallelization Opportunities

- Task 6 label updates can run independently of backend tasks.
- Task 4 frontend API types can start after Task 1 contract is stable.
- Tasks 2 and 3 are sequential because import depends on parsing/matching helpers.
- Task 5 depends on Task 4 but can be reviewed separately from backend implementation with mocked hook behavior.

## Open Questions

- None currently blocking. User confirmed `model.dev` UI label, `cache_read` fallback as `input / 10`, and keeping `embedded` visible with current behavior.
