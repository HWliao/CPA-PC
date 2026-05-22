param(
    [ValidateSet("create", "remove", "start", "stop", "status", "menu")]
    [string]$Action = "menu"
)

$taskName = "CPAPCTask"
$taskDescription = "CPA-PC Background Service for Current User"
$appDir = $PSScriptRoot
$exePath = Join-Path $appDir "cpa-pc.exe"
$launcherPath = Join-Path $appDir "start-cpa-pc.vbs"

function Test-IsAdministrator
{
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")
}

function Ensure-Administrator
{
    if (Test-IsAdministrator)
    {
        return
    }

    Write-Host "Requesting administrator privileges" -ForegroundColor Yellow
    $arguments = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "`"$PSCommandPath`"", "-Action", $Action)
    Start-Process -FilePath "powershell.exe" -Verb RunAs -ArgumentList $arguments
    exit
}

function Test-AppFiles
{
    if (-not (Test-Path -LiteralPath $exePath -PathType Leaf))
    {
        Write-Host "ERROR Program not found at $exePath" -ForegroundColor Red
        return $false
    }

    if (-not (Test-Path -LiteralPath $launcherPath -PathType Leaf))
    {
        Write-Host "ERROR Launcher not found at $launcherPath" -ForegroundColor Red
        return $false
    }

    return $true
}

function Show-Menu
{
    Clear-Host
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host "CPA-PC Task Scheduler Manager" -ForegroundColor White -BackgroundColor DarkBlue
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Task Name $taskName" -ForegroundColor Gray
    Write-Host "App Dir $appDir" -ForegroundColor Gray
    Write-Host "Runs as Current User" -ForegroundColor Gray
    Write-Host ""
    Write-Host "1 Create Task at Logon" -ForegroundColor Green
    Write-Host "2 Delete Task" -ForegroundColor Red
    Write-Host "3 Run Now" -ForegroundColor Magenta
    Write-Host "4 Stop Program" -ForegroundColor DarkRed
    Write-Host "5 Check Status" -ForegroundColor Cyan
    Write-Host "0 Exit" -ForegroundColor Gray
    Write-Host ""
}

function Get-AppStatus
{
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    $proc = Get-Process -Name "cpa-pc" -ErrorAction SilentlyContinue

    Write-Host "Status Report" -ForegroundColor Cyan

    if ($task)
    {
        Write-Host "Scheduled Task Created" -ForegroundColor Green
        Write-Host "State $($task.State)" -ForegroundColor Gray
        Write-Host "User $($task.Principal.UserId)" -ForegroundColor Gray

        $taskInfo = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction SilentlyContinue
        if ($taskInfo)
        {
            Write-Host "Next Run $($taskInfo.NextRunTime)" -ForegroundColor Gray
            Write-Host "Last Run $($taskInfo.LastRunTime)" -ForegroundColor Gray
            Write-Host "Last Result $($taskInfo.LastTaskResult)" -ForegroundColor Gray
        }
    }
    else
    {
        Write-Host "Scheduled Task Not created" -ForegroundColor Red
    }

    if ($proc)
    {
        Write-Host "Process Running PID $($proc.Id -join ', ')" -ForegroundColor Green
    }
    else
    {
        Write-Host "Process Not running" -ForegroundColor Red
    }
}

function New-AppTask
{
    Ensure-Administrator

    if (-not (Test-AppFiles))
    {
        exit 1
    }

    Write-Host "Creating Hidden Scheduled Task for Current User" -ForegroundColor Cyan

    $existing = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($existing)
    {
        Write-Host "Existing task found removing" -ForegroundColor Yellow
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
    }

    $trigger = New-ScheduledTaskTrigger -AtLogOn
    Write-Host "Trigger At user logon" -ForegroundColor Green

    $action = New-ScheduledTaskAction -Execute "wscript.exe" -Argument "`"$launcherPath`"" -WorkingDirectory $appDir
    Write-Host "Action Hidden launcher execution" -ForegroundColor Green

    $currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
    $principal = New-ScheduledTaskPrincipal -UserId $currentUser -LogonType Interactive -RunLevel Limited
    Write-Host "Principal Current user $currentUser" -ForegroundColor Green

    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -Hidden
    Write-Host "Settings Hidden task runs as current user" -ForegroundColor Green

    try
    {
        Register-ScheduledTask -TaskName $taskName -Description $taskDescription -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force | Out-Null

        Write-Host "Task created successfully" -ForegroundColor Green
        Write-Host "Task Name $taskName" -ForegroundColor Cyan
        Write-Host "Executable $exePath" -ForegroundColor Cyan
        Write-Host "Launcher $launcherPath" -ForegroundColor Cyan
        Write-Host "User $currentUser" -ForegroundColor Cyan
    }
    catch
    {
        Write-Host "Creation failed $_" -ForegroundColor Red
        exit 1
    }

    Get-AppStatus
}

function Remove-AppTask
{
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($task)
    {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        Write-Host "Task deleted" -ForegroundColor Green
    }
    else
    {
        Write-Host "Task not found" -ForegroundColor Yellow
    }
}

function Start-TaskNow
{
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    if ($task)
    {
        Start-ScheduledTask -TaskName $taskName
        Write-Host "Task triggered" -ForegroundColor Green
    }
    else
    {
        Write-Host "Task not found create it first" -ForegroundColor Red
    }
}

function Stop-Program
{
    $proc = Get-Process -Name "cpa-pc" -ErrorAction SilentlyContinue
    if ($proc)
    {
        $proc | Stop-Process -Force
        Write-Host "Program stopped" -ForegroundColor Green
    }
    else
    {
        Write-Host "Program not running" -ForegroundColor Yellow
    }
}

if ($Action -ne "menu")
{
    switch ($Action)
    {
        "create" { New-AppTask }
        "remove" { Remove-AppTask }
        "start" { Start-TaskNow }
        "stop" { Stop-Program }
        "status" { Get-AppStatus }
    }
}
else
{
    do
    {
        Show-Menu
        $choice = Read-Host "Enter choice 0 to 5"

        switch ($choice)
        {
            "1" { New-AppTask; Read-Host "Press Enter to continue" }
            "2" { Remove-AppTask; Read-Host "Press Enter to continue" }
            "3" { Start-TaskNow; Read-Host "Press Enter to continue" }
            "4" { Stop-Program; Read-Host "Press Enter to continue" }
            "5" { Get-AppStatus; Read-Host "Press Enter to continue" }
            "0" { Write-Host "Goodbye" -ForegroundColor Green }
            default { Write-Host "Invalid option" -ForegroundColor Red; Read-Host "Press Enter to continue" }
        }
    } while ($choice -ne "0")
}
