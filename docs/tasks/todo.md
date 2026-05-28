# Task List: Model Price Sync Source Selection And Field Labels

Status: Draft for human review. Do not implement until approved.

## Phase 1: Backend Contract And Embedded Source

- [ ] Task 1: Extend model price sync request contract
  - Acceptance: `source: "embedded"` and provider/model targets are accepted; missing source defaults to `embedded`; embedded returns stored prices with `imported = 0`, `skipped = 0`; invalid source returns bad request.
  - Verify: `go test ./internal/httpapi`
  - Dependencies: None
  - Files: `internal/httpapi/info.go`, `internal/httpapi/info_test.go`

## Checkpoint: Backend Contract

- [ ] Existing empty-body sync clients still work.
- [ ] Route contract can carry provider/model targets.

## Phase 2: Backend models.dev Sync

- [ ] Task 2: Add models.dev price fetch and parse helpers
  - Acceptance: parser validates provider/model IDs and numeric costs; maps `input -> prompt`, `output -> completion`, `cache_read -> cache`; missing `cache_read` maps to `input / 10`; tests do not call live models.dev.
  - Verify: `go test ./internal/httpapi`
  - Dependencies: Task 1
  - Files: `internal/httpapi/info.go` or helper file, `internal/httpapi/info_test.go` or focused test file

- [ ] Task 3: Match and import models.dev prices
  - Acceptance: matching uses provider+model; Codex and `gpt-*` normalize to OpenAI; imported prices include `source = "model.dev"`; skipped/unrequested models preserve existing prices; imported/skipped counts are accurate.
  - Verify: `go test ./internal/httpapi ./internal/store`
  - Dependencies: Task 2
  - Files: `internal/httpapi/info.go` or helper file, `internal/httpapi/info_test.go` or focused test file, optional `internal/store/store.go`

## Checkpoint: Backend Sync

- [ ] `embedded` and `model.dev` source paths are both tested.
- [ ] Unit tests make no live network calls.
- [ ] Manual prices are preserved for skipped/unrequested models.

## Phase 3: Frontend API And Sync Flow

- [ ] Task 4: Update frontend sync types and hook signature
  - Acceptance: frontend has `ModelPriceSyncSource`; sync targets include provider/model; API sends `{ source, models }`; hook exposes new signature without unrelated behavior changes.
  - Verify: `npm --prefix web run type-check`
  - Dependencies: Task 1
  - Files: `web/src/services/api/usageService.ts`, `web/src/features/monitoring/hooks/useUsageData.ts`

- [ ] Task 5: Add source selection modal to request monitoring
  - Acceptance: one-click sync opens source modal before API call; modal lists `embedded` and `model.dev`; confirming sync sends selected source and targets; notifications remain clear.
  - Verify: `npm --prefix web test -- src/pages/MonitoringCenterPage.test.tsx` if focused tests are practical; `npm --prefix web run type-check`
  - Dependencies: Task 4
  - Files: `web/src/pages/MonitoringCenterPage.tsx`, optional `web/src/pages/MonitoringCenterPage.test.tsx`

- [ ] Task 6: Rename price labels in UI locales
  - Acceptance: zh-CN uses `输入价格`, `输出价格`, `输入缓存价格`; English uses `Input price`, `Output price`, `Input cache price`; zh-TW and ru avoid stale prompt/completion wording; UI still has exactly three price fields.
  - Verify: `npm --prefix web run type-check`; `npm --prefix web run lint`
  - Dependencies: None, should land with Task 5
  - Files: `web/src/i18n/locales/en.json`, `web/src/i18n/locales/zh-CN.json`, `web/src/i18n/locales/zh-TW.json`, `web/src/i18n/locales/ru.json`

## Checkpoint: Frontend UI

- [ ] Price labels are clear in the model price modal.
- [ ] Sync source selection is required before sync starts.
- [ ] Frontend type check passes.

## Phase 4: Final Verification

- [ ] Task 7: Run final checks and review diff
  - Acceptance: focused backend tests pass; frontend type check passes; lint passes if frontend files changed; relevant frontend tests pass if added/updated; generated static asset is not touched unless rebuilt through `npm --prefix web run build`.
  - Verify: `go test ./internal/httpapi ./internal/store`; `npm --prefix web run type-check`; `npm --prefix web run lint`; `npm --prefix web test -- src/pages/MonitoringCenterPage.test.tsx` if focused tests exist; `go test ./...` if warranted.
  - Dependencies: Tasks 1-6
  - Files: None unless verification exposes a defect

## Checkpoint: Complete

- [ ] All `docs/SPEC.md` acceptance criteria are satisfied.
- [ ] No unrelated monitoring charts, quota panels, config merging, packaging, or Windows script code was changed.
- [ ] Human review approves the completed implementation.
