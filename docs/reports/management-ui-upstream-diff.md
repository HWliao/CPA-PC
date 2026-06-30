# Management UI Upstream Diff Report

Status: Current-tree comparison complete. Relevant plugin UI sync applied and verified.

Spec: `docs/SPEC.md`

Plan: `docs/tasks/plan.md`

Tasks: `docs/tasks/todo.md`

## Baseline

- CPA-PC UI source: `D:\develop\workspace\CPA-PC\web`
- Official UI reference source: `D:\develop\source-workspace\Cli-Proxy-API-Management-Center`
- Official UI comparison mode: current-tree comparison
- Official UI reference branch: `main`
- Official UI reference head: `32699c9`
- Official UI reference status: clean
- Generated CPA-PC management asset: `D:\develop\workspace\CPA-PC\static\management.html`

## Scope

Compare and directly sync relevant official management UI differences while preserving CPA-PC-specific behavior:

- Routing and page structure
- API clients and endpoint paths
- Authorization handling
- Config editor behavior
- Model/API-key management pages
- Request monitoring and model pricing
- CPA-PC usage pages and local usage persistence integration
- i18n strings
- Styling and build setup
- Generated single-file management asset implications

## Preserve CPA-PC Behavior

- Hash router management UI
- CPA-PC usage pages and request monitoring additions
- `/cpa-pc/info`, `/usage-service/*`, `/v0/management/usage*`, model prices, and API-key aliases
- Local SQLite usage/model-price persistence
- Single-file `static/management.html` build output
- Management-key `Authorization: Bearer <key>` flow

## Impact Summary

- Current-tree comparison found `312` files in CPA-PC `web/`, `264` files in the official UI, `199` common files, `109` changed common files, `113` CPA-PC-only files, and `65` official-only files.
- Both projects still use `createHashRouter`, so CPA-PC's server-side static route assumptions remain valid.
- CPA-PC-only UI remains significant and is preserved: usage service setup, request monitoring, monitoring charts, model prices, API-key aliases, CPA-PC info, single-file packaging script, and local tests.
- Official UI adds SDK `7.2.x` plugin management surfaces that align with the upgraded SDK: plugin list/config, plugin store/install, plugin resource pages, plugin support header detection, and plugin config examples.
- The sync is targeted: plugin management support was added without replacing CPA-PC's existing AI provider pages or monitoring pages.

## Additions

- Added plugin API client and plugin types:
  - `web/src/services/api/plugins.ts`
  - `web/src/types/plugin.ts`
- Added plugin pages and supporting code from the official UI:
  - `web/src/features/plugins/PluginsPage.tsx`
  - `web/src/features/plugins/PluginStorePage.tsx`
  - `web/src/features/plugins/PluginResourcePage.tsx`
  - `web/src/features/plugins/pluginResources.ts`
  - `web/src/features/plugins/pluginPolling.ts`
  - `web/src/features/plugins/components/PluginInstallGateModal.tsx`
  - related SCSS module files.
- Added reusable upstream UI support needed by plugin pages:
  - `web/src/components/ui/Sheet/*`
  - `web/src/components/ui/scrollLock.ts`
- Added plugin navigation/routes gated by SDK plugin support:
  - `/plugins`
  - `/plugin-store`
  - `/plugin-pages/:pluginId/:menuIndex`
- Added runtime/plugin capability detection from official UI:
  - Home/CPA version header precedence.
  - `x-cpa-support-plugin` boolean header support.
  - Runtime kind detection via `/nodes` fallback.
- Added plugin translation keys to `en`, `zh-CN`, `zh-TW`, and `ru` locale files.

## Modifications

- `web/src/services/api/client.ts` now reads CPA/Home version headers and dispatches plugin-support events while preserving deprecated Gemini endpoint rewriting.
- `web/src/stores/useAuthStore.ts` now tracks `serverRuntimeKind` and `supportsPlugin`; plugin UI is hidden until the backend reports support.
- `web/src/router/MainRoutes.tsx` adds plugin routes behind `supportsPlugin` and redirects plugin routes back to `/` when unsupported.
- `web/src/components/layout/MainLayout.tsx` adds plugin and plugin-store sidebar entries only when supported, preserving CPA-PC monitoring/log/system entries.
- `web/src/components/ui/icons.tsx` adds plugin-related icons used by the upstream pages.
- `web/src/utils/constants.ts`, `web/src/utils/helpers.ts`, `web/src/types/index.ts`, and `web/src/services/api/index.ts` export the official UI support needed by the plugin feature.
- Locale JSON files were structurally merged with upstream plugin/nav keys while preserving CPA-PC-only strings.

## Breaking Changes

- No intentional CPA-PC UI breaking change was introduced.
- Plugin pages are feature-gated by `x-cpa-support-plugin`; older or unsupported backends redirect plugin routes to `/` instead of showing broken pages.
- Potentially breaking upstream differences not synced yet:
  - The official UI has a newer provider workbench that differs from CPA-PC's existing provider edit pages. It was not synced to avoid overwriting CPA-PC-specific provider and monitoring behavior.
  - The official UI removed Amp-oriented surfaces; CPA-PC still has Amp UI files pending a separate decision because removing them would be a visible breaking change for existing CPA-PC users.

Classification will use:

- Non-breaking: no behavior or integration contract break for CPA-PC users/maintainers.
- Potentially breaking: could affect routes, auth, build output, management workflows, or packaging depending on deployment.
- Breaking: requires user/admin action or changes an existing CPA-PC management UI contract.

## Current-Tree Diff Findings

Comparison summary:

- Official-only relevant files include plugin pages, plugin store/resource support, `Sheet`, `scrollLock`, config search index, provider workbench files, and additional UI primitives.
- CPA-PC-only relevant files include monitoring center/charts, usage service API/store, model price utilities, CPA-PC info API, API-key alias APIs, packaging copy script, and related tests.
- Package/build differences:
  - CPA-PC keeps `npm --prefix web run build` as `tsc && vite build && node scripts/copy-management-html.mjs` for root `static/management.html` generation.
  - Official UI build remains `tsc && vite build`.
  - CPA-PC keeps Vitest/test dependencies and ECharts for monitoring charts.

Adopted now:

- Plugin management/store/resource UI.
- Plugin capability detection and route gating.
- Plugin i18n strings.
- Required shared UI primitives for plugin pages.

Preserved for CPA-PC:

- Hash router.
- Existing provider pages.
- Usage/request-monitoring pages.
- CPA-PC API-key alias/model-price/usage-service APIs.
- Single-file packaging pipeline.

Ask first / not synced:

- Full official provider workbench replacement.
- Broad dependency upgrades to official UI versions (`typescript`, `i18next`, `axios`, etc.).
- Removing CPA-PC Amp UI/services.

## Synced UI Changes

Applied targeted plugin-management sync as listed in Additions and Modifications.

## Verification

Commands run:

```powershell
npm --prefix web run type-check
npm --prefix web run lint
npm --prefix web run build
```

Results:

- `npm --prefix web run type-check`: passed after adding missing plugin icon exports.
- `npm --prefix web run lint`: passed.
- `npm --prefix web run build`: passed; Vite built the single-file app and `scripts/copy-management-html.mjs` wrote `static\management.html`.
- `npm run package:windows -- --build-frontend`: passed after frontend version bump to `1.1.8`; packaging rebuilt `static\management.html` and produced `dist\cpa-pc_0.2.11_windows_amd64.zip`.

Expected commands if `web/` changes:

```powershell
npm --prefix web run type-check
npm --prefix web run lint
npm --prefix web run build
```

## Residual Risks

- Plugin management UI is gated by the SDK support header. If a backend does not expose plugin endpoints or does not send `x-cpa-support-plugin`, users will not see plugin navigation.
- The official provider workbench was not synced because it would replace substantial CPA-PC provider UI and could disturb CPA-PC monitoring/model-price workflows.
- CPA-PC still contains Amp UI/source files. Upstream SDK removed Amp support, but removing CPA-PC's visible Amp entry would be a separate breaking-change decision.
