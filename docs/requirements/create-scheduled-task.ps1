if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) 
{
    Start-Process PowerShell -Verb RunAs -ArgumentList "-File `"$PSCommandPath`""
    exit
}

$exePath = "D:\common\CPA\start-proxy.vbs"
$taskName = "CLIProxyAPIUserTask"
$taskDescription = "CLIProxyAPI Background Service for Current User"

if (-not (Test-Path $exePath)) 
{
    Write-Host "ERROR Program not found at $exePath" -ForegroundColor Red
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

$action = New-ScheduledTaskAction -Execute $exePath -WorkingDirectory "D:\common\CPA"
Write-Host "Action Direct EXE execution" -ForegroundColor Green

$currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
$principal = New-ScheduledTaskPrincipal -UserId $currentUser -LogonType Interactive -RunLevel Limited
Write-Host "Principal Current user $currentUser" -ForegroundColor Green

$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -Hidden
Write-Host "Settings Hidden task runs as current user" -ForegroundColor Green

try 
{
    Register-ScheduledTask -TaskName $taskName -Description $taskDescription -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force

    Write-Host "Task created successfully" -ForegroundColor Green
    Write-Host "Task Name $taskName" -ForegroundColor Cyan
    Write-Host "Executable $exePath" -ForegroundColor Cyan
    Write-Host "User $currentUser" -ForegroundColor Cyan
    
    $test = Read-Host "Test run now Y or N"
    if ($test -eq "Y" -or $test -eq "y") 
    {
        Write-Host "Starting test" -ForegroundColor Yellow
        Start-ScheduledTask -TaskName $taskName
        Start-Sleep -Seconds 3
        
        $proc = Get-Process -Name "cli-proxy-api" -ErrorAction SilentlyContinue
        if ($proc) 
        {
            Write-Host "Test successful PID $($proc.Id)" -ForegroundColor Green
        } 
        else 
        {
            Write-Host "Test started process may be initializing" -ForegroundColor Yellow
        }
    }
}
catch 
{
    Write-Host "Creation failed $_" -ForegroundColor Red
    exit 1
}

Write-Host "Task Status" -ForegroundColor Cyan
$task = Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
if ($task)
{
    Write-Host "Name $($task.TaskName)" -ForegroundColor Gray
    Write-Host "State $($task.State)" -ForegroundColor Gray
    Write-Host "Next Run $($task.NextRunTime)" -ForegroundColor Gray
}