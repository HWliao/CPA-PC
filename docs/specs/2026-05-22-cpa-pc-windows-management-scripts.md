# Spec: CPA-PC Windows Management Scripts

Status: Approved and implemented.

## Objective

Build Windows management scripts for the packaged `cpa-pc.exe` release. The scripts are for operations/admin users and packaging automation, and should mirror the behavior of the existing examples under `docs/specs/` while removing hard-coded `D:\common\CPA` paths.

Target users:

- Operators/admins who need to register, remove, start, stop, and inspect the CPA-PC scheduled task.
- The Windows packaging flow, which should place the management scripts beside the final `cpa-pc.exe`.

Acceptance criteria:

- Add Windows scripts under `scripts/win/`.
- Provide a PowerShell manager with actions for `create`, `remove`, `start`, `stop`, `status`, and interactive `menu`.
- Provide a hidden launcher compatible with Task Scheduler so `cpa-pc.exe` starts without a visible console window.
- Register the scheduled task using the same strategy as `docs/specs/create-scheduled-task.ps1`: current-user logon trigger, hidden task, interactive current-user principal, limited run level, battery-friendly settings.
- Resolve `cpa-pc.exe` and helper scripts relative to the script/release directory, not from an absolute machine-specific path.
- Update Windows packaging so the scripts are copied directly into the same release directory as `cpa-pc.exe`.
- Do not implement extra log cleanup, config editing, Windows service installation, or dependency installation.

## Tech Stack

- PowerShell compatible with Windows PowerShell 5.1 or newer.
- VBScript via Windows Script Host for hidden process launch, matching the existing `docs/specs/start-proxy.vbs` pattern.
- Node.js packaging script at `scripts/package-windows.ts` copies release assets.
- Go application binary built from `./cmd/cpa-pc` as `cpa-pc.exe`.

## Commands

Development checks:

```powershell
go test ./...
go build ./cmd/cpa-pc
npm --prefix web test -- --run
npm --prefix web run lint
npm --prefix web run build
```

Package Windows release without rebuilding the frontend:

```powershell
npm run package:windows -- --version dev
```

Package Windows release and rebuild the frontend:

```powershell
npm run package:windows -- --version dev --build-frontend
```

Manual script smoke checks after implementation:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\manage-cpa-pc.ps1 -Action status
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\manage-cpa-pc.ps1 -Action create
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\manage-cpa-pc.ps1 -Action start
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\manage-cpa-pc.ps1 -Action stop
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\manage-cpa-pc.ps1 -Action remove
```

Manual packaged-release smoke check:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\dist\cpa-pc_dev_windows_amd64\manage-cpa-pc.ps1 -Action status
```

## Project Structure

Relevant existing structure:

```text
cmd/cpa-pc/                 Go entrypoint for cpa-pc.exe
internal/                   Go application packages and tests
web/                        React/Vite management UI
static/                     Built management.html copied into releases
docs/specs/                 Existing reference ps1/vbs scripts
scripts/package-windows.ts  Windows release packaging script
dist/                       Generated release output
```

New/changed structure:

```text
scripts/win/manage-cpa-pc.ps1   PowerShell management script
scripts/win/start-cpa-pc.vbs    Hidden launcher used by Task Scheduler
docs/specs/2026-05-22-cpa-pc-windows-management-scripts.md
                                Archived implementation spec
dist/cpa-pc_<version>_windows_amd64/
  cpa-pc.exe
  manage-cpa-pc.ps1
  start-cpa-pc.vbs
  config.example.yaml
  static/management.html
  data/
  logs/
```

## Code Style

PowerShell should stay close to the existing scripts: simple top-level constants, small action functions, `Write-Host` status messages with colors, and `Get-ScheduledTask -ErrorAction SilentlyContinue` for idempotent checks. Paths must be derived from `$PSScriptRoot`.

Example style:

```powershell
param(
    [ValidateSet("create", "remove", "start", "stop", "status", "menu")]
    [string]$Action = "menu"
)

$taskName = "CPAPCTask"
$appDir = $PSScriptRoot
$exePath = Join-Path $appDir "cpa-pc.exe"
$launcherPath = Join-Path $appDir "start-cpa-pc.vbs"

function Get-AppStatus
{
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    $proc = Get-Process -Name "cpa-pc" -ErrorAction SilentlyContinue

    if ($task)
    {
        Write-Host "Scheduled Task Created" -ForegroundColor Green
    }
    else
    {
        Write-Host "Scheduled Task Not created" -ForegroundColor Red
    }
}
```

VBScript should be minimal and derive the executable path from its own directory:

```vbscript
Set shell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")
appDir = fso.GetParentFolderName(WScript.ScriptFullName)
shell.CurrentDirectory = appDir
shell.Run Chr(34) & fso.BuildPath(appDir, "cpa-pc.exe") & Chr(34), 0, False
```

Conventions:

- Use ASCII in new scripts unless needed for existing localized output.
- Keep file names explicit: `manage-cpa-pc.ps1` and `start-cpa-pc.vbs`.
- Do not introduce third-party dependencies or modules.
- Keep behavior idempotent: creating a task should replace or update the existing CPA-PC task; removing a missing task should print a warning, not fail.

## Testing Strategy

Automated checks:

- Run `go test ./...` to ensure backend changes were not broken.
- Run `npm run package:windows -- --version dev` to ensure release packaging succeeds and includes the Windows scripts.
- If frontend assets are rebuilt, run `npm --prefix web test -- --run`, `npm --prefix web run lint`, and `npm --prefix web run build`.

Manual Windows checks:

- From the repo root, run `manage-cpa-pc.ps1 -Action status` and verify missing task/program is reported cleanly.
- From a packaged release directory, run `manage-cpa-pc.ps1 -Action create` and verify Task Scheduler contains the expected task name.
- Run `manage-cpa-pc.ps1 -Action start` and verify a `cpa-pc` process starts.
- Run `manage-cpa-pc.ps1 -Action stop` and verify the `cpa-pc` process stops.
- Run `manage-cpa-pc.ps1 -Action remove` and verify the scheduled task is removed.

Coverage expectations:

- No new unit-test framework is required for PowerShell/VBScript.
- Packaging behavior should be verified by inspecting the generated `dist/cpa-pc_<version>_windows_amd64/` directory.
- Any TypeScript packaging logic changes should remain small and covered by the packaging command.

## Boundaries

Always:

- Resolve release paths relative to the script location so the scripts work beside `cpa-pc.exe`.
- Preserve the existing Task Scheduler strategy from `docs/specs`: logon trigger, current user, hidden task, interactive limited principal.
- Keep scripts compatible with Windows PowerShell 5.1 and default Windows Script Host.
- Keep implementation minimal and focused on task registration plus program start/stop/status.
- Update packaging so the management scripts are copied into the final executable directory.

Ask first:

- Changing the scheduled task trigger away from current-user logon.
- Requiring administrator elevation beyond what the existing reference pattern uses.
- Adding Windows service support, installer behavior, config mutation, log cleanup, or external dependencies.
- Renaming the task to something other than `CPAPCTask`.
- Changing `cpa-pc.exe` startup arguments, including forcing `-config`.

Never:

- Hard-code local absolute paths such as `D:\common\CPA`.
- Delete data, logs, configs, auth files, or unrelated scheduled tasks.
- Modify files under `dist/` as source of truth.
- Commit secrets or generated local credentials.
- Replace the app runtime with the old standalone `CLIProxyAPI.exe` model.

## Success Criteria

- The spec was reviewed and approved before script implementation began.
- `scripts/win/manage-cpa-pc.ps1` and `scripts/win/start-cpa-pc.vbs` were created after approval.
- `npm run package:windows -- --version dev` creates a release directory containing `cpa-pc.exe`, `manage-cpa-pc.ps1`, and `start-cpa-pc.vbs` in the same directory.
- The PowerShell manager supports non-interactive actions and an interactive menu.
- Manual `status` checks work from both the source script path and packaged release path without third-party dependencies.
- Manual `create`, `start`, `stop`, and `remove` remain destructive/local-environment checks and were intentionally not run during implementation.

## Verification Results

Passed:

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\manage-cpa-pc.ps1 -Action status`
- `go test ./...`
- `npm run package:windows -- --version dev`
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\dist\cpa-pc_dev_windows_amd64\manage-cpa-pc.ps1 -Action status`
- `git diff --check`

Package inspection confirmed `dist/cpa-pc_dev_windows_amd64/` contains `manage-cpa-pc.ps1` and `start-cpa-pc.vbs` beside `cpa-pc.exe`.

## Confirmed Decisions

- The scheduled task name is `CPAPCTask`.
- The launcher starts `cpa-pc.exe` with no explicit arguments, relying on the app's default config resolution.
- The implementation includes both `manage-cpa-pc.ps1` and `start-cpa-pc.vbs` under `scripts/win/`.
