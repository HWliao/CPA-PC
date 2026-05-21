param(
    [ValidateSet("create", "remove", "start", "stop", "status", "menu")]
    [string]$Action = "menu"
)

$taskName = "CLIProxyAPIUserTask"
$exePath = "D:\common\CPA\start-proxy.vbs"

function Show-Menu 
{
    Clear-Host
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host "CLIProxyAPI Task Scheduler Manager" -ForegroundColor White -BackgroundColor DarkBlue
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Task Name $taskName" -ForegroundColor Gray
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

function Get-TaskStatus 
{
    $task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
    $proc = Get-Process -Name "cli-proxy-api" -ErrorAction SilentlyContinue
    
    Write-Host "Status Report" -ForegroundColor Cyan
    
    if ($task) 
    {
        Write-Host "Scheduled Task Created" -ForegroundColor Green
        Write-Host "State $($task.State)" -ForegroundColor Gray
        Write-Host "Next Run $($task.NextRunTime)" -ForegroundColor Gray
        Write-Host "User $($task.Principal.UserId)" -ForegroundColor Gray
    } 
    else 
    {
        Write-Host "Scheduled Task Not created" -ForegroundColor Red
    }
    
    if ($proc) 
    {
        Write-Host "Process Running PID $($proc.Id)" -ForegroundColor Green
    } 
    else 
    {
        Write-Host "Process Not running" -ForegroundColor Red
    }
}

function Remove-Task 
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
    $proc = Get-Process -Name "cli-proxy-api" -ErrorAction SilentlyContinue
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
        "create" { & "$PSScriptRoot\create-scheduled-task.ps1" }
        "remove" { Remove-Task }
        "start" { Start-TaskNow }
        "stop" { Stop-Program }
        "status" { Get-TaskStatus }
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
            "1" { & "$PSScriptRoot\create-scheduled-task.ps1"; Read-Host "Press Enter to continue" }
            "2" { Remove-Task; Read-Host "Press Enter to continue" }
            "3" { Start-TaskNow; Read-Host "Press Enter to continue" }
            "4" { Stop-Program; Read-Host "Press Enter to continue" }
            "5" { Get-TaskStatus; Read-Host "Press Enter to continue" }
            "0" { Write-Host "Goodbye" -ForegroundColor Green }
            default { Write-Host "Invalid option" -ForegroundColor Red; Read-Host "Press Enter to continue" }
        }
    } while ($choice -ne "0")
}