# AGENTS.md

## Repository Map
- CPA-PC is a Windows-first single-process Go app: `cmd/cpa-pc` starts embedded CLIProxyAPI, CPA-PC usage persistence, and the management UI routes in one process. Do not reintroduce a separate `CLIProxyAPI.exe` launcher.
- `internal/app` wires CLIProxyAPI SDK startup, config pass-through, `MANAGEMENT_STATIC_PATH`, usage plugin registration, and file logging.
- `internal/httpapi` owns CPA-PC extra routes such as `/cpa-pc/info`, `/usage-service/*`, `/v0/management/usage*`, model prices, and API-key aliases. CPA core routes and most `/v0/management/*` behavior come from the embedded SDK.
- `internal/store` is the local SQLite store using `modernc.org/sqlite`; no external SQLite service is required.
- `web/` is the React/Vite management UI source. `static/management.html` is the generated single-file release asset and packaging input; rebuild it from `web/` instead of editing it by hand.
- Generated/local directories are `dist/`, `data/`, `logs/`, `web/dist/`, and `node_modules/`.

## Commands
- Use npm lockfiles, not yarn/pnpm. Root npm installs only packaging deps: `npm install`; frontend reproducible install: `npm --prefix web ci`.
- Backend full check: `go test ./...`; focused package/test: `go test ./internal/config` or `go test ./cmd/cpa-pc -run TestRunVersionDoesNotRequireConfig`.
- Backend build/run: `go build ./cmd/cpa-pc`; `go run ./cmd/cpa-pc -config .\config.example.yaml`.
- Frontend focused test: `npm --prefix web test -- src/utils/apiKeyHash.test.ts`; full web tests: `npm --prefix web test`.
- Frontend checks: `npm --prefix web run lint`, `npm --prefix web run type-check`, `npm --prefix web run build`.
- `npm --prefix web run build` runs `tsc && vite build && node scripts/copy-management-html.mjs`, then writes root `static/management.html`.
- Package existing static asset: `npm run package:windows -- --version dev`; rebuild web first: `npm run package:windows -- --version dev --build-frontend`; options: `npm run package:windows -- --help`.
- Run `scripts/package-windows.ts` only through `npm run package:windows`; direct `node ...` fails when `npm_execpath` is absent.

## Config And Runtime
- `config.example.yaml` is both CPA-PC config and upstream CLIProxyAPI example. `internal/config.Load` reads CPA-PC fields, while `internal/app.loadCPAConfig` loads the same file through CLIProxyAPI, so unknown YAML keys can be meaningful pass-through settings.
- Preserve CPA-PC-specific config when merging upstream examples: `data-dir`, `logs-dir`, `static-dir`, `usage`, and `remote-management.secret-key: "123456"`.
- Runtime paths `data-dir`, `logs-dir`, `static-dir`, and `usage.db-path` resolve relative to the config file. With no `-config`, `cpa-pc.exe` looks for `config.yaml` beside the executable.
- `internal/app` sets `MANAGEMENT_STATIC_PATH` from `static-dir` unless it is already set; deliberate env overrides can change where `/management.html` is served from.
- Management calls use `Authorization: Bearer <management key>`; the example local key is `123456`.

## Frontend Notes
- Vite uses `vite-plugin-singlefile`, inlines assets, disables CSS/code splitting, and aliases `@` to `web/src`.
- SCSS modules export camelCase names and automatically load `@use "@/styles/variables.scss" as *;`.
- Routing uses `createHashRouter` in `web/src/App.tsx`; do not switch to browser-router URLs without adding server-side route support.
- The API client rewrites deprecated `/generative-language-api-key` requests to `/gemini-api-key`; preserve this compatibility unless all callers/tests are updated.

## Packaging And Windows Scripts
- `scripts/package-windows.ts` builds `GOOS=windows`, `GOARCH=amd64`, `CGO_ENABLED=0`, then creates `dist/cpa-pc_<version>_windows_amd64/` and the same-name `.zip`.
- Packaging requires root `static/management.html`, `config.example.yaml`, `scripts/win/manage-cpa-pc.ps1`, and `scripts/win/start-cpa-pc.vbs`.
- `scripts/win/manage-cpa-pc.ps1 -Action status` is the safe smoke check. `create`, `start`, `stop`, and `remove` alter the local `CPAPCTask` scheduled task or local `cpa-pc` processes.
- The documented release target is Windows amd64; ask before adding other OS targets, installers, or Windows service behavior.

## Constraints From Specs
- Do not add `../CLIProxyAPI` or `../CPA-Manager` as subdirectories, submodules, or long-term build/runtime dependencies; specs only allow them as reference sources.
- Do not register a duplicate `GET /management.html` route in CPA-PC; the embedded CLIProxyAPI SDK owns that route and serves the external static file.
- Usage data should flow from the CLIProxyAPI SDK usage plugin into local SQLite, not from an external HTTP/RESP usage queue collector.
- Specs under `docs/specs/` are approved implementation records; read the relevant spec before changing config merging, Windows scripts, or product topology.
- There are no `.github/workflows` or pre-commit hooks in this repo; local verification commands are the source of truth.

## Agent Rules

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

### 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

### 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

### 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

### 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" -> "Write tests for invalid inputs, then make them pass"
- "Fix the bug" -> "Write a test that reproduces it, then make it pass"
- "Refactor X" -> "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.
