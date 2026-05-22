param(
    [string]$Version = "dev",
    [string]$OutputRoot = "dist",
    [switch]$BuildFrontend
)

$ErrorActionPreference = "Stop"

function Invoke-Native {
    param(
        [string]$File,
        [string[]]$Arguments
    )

    & $File @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$File $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$safeVersion = $Version -replace '[\\/:*?"<>|]', '-'
$packageName = "cpa-pc_${safeVersion}_windows_amd64"
$outputRootPath = Join-Path $repoRoot $OutputRoot
$packagePath = Join-Path $outputRootPath $packageName
$staticSource = Join-Path $repoRoot "static\management.html"
$configSource = Join-Path $repoRoot "config.example.yaml"

if ($BuildFrontend) {
    $webRoot = Join-Path $repoRoot "web"
    $previousVersion = $env:VERSION
    try {
        $env:VERSION = $Version
        Invoke-Native -File "npm" -Arguments @("--prefix", $webRoot, "run", "build")
    } finally {
        $env:VERSION = $previousVersion
    }
}

if (-not (Test-Path -LiteralPath $staticSource)) {
    throw "static/management.html not found; run npm --prefix web run build or pass -BuildFrontend"
}

if (-not (Test-Path -LiteralPath $configSource)) {
    throw "config.example.yaml not found"
}

New-Item -ItemType Directory -Force -Path $outputRootPath | Out-Null
if (Test-Path -LiteralPath $packagePath) {
    Remove-Item -LiteralPath $packagePath -Recurse -Force
}

$staticTargetDir = Join-Path $packagePath "static"
$dataTargetDir = Join-Path $packagePath "data"
$logsTargetDir = Join-Path $packagePath "logs"
New-Item -ItemType Directory -Force -Path $staticTargetDir, $dataTargetDir, $logsTargetDir | Out-Null

$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$previousCGO = $env:CGO_ENABLED
try {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"

    Push-Location $repoRoot
    try {
        Invoke-Native -File "go" -Arguments @(
            "build",
            "-trimpath",
            "-ldflags", "-s -w -X main.version=$Version",
            "-o", (Join-Path $packagePath "cpa-pc.exe"),
            "./cmd/cpa-pc"
        )
    } finally {
        Pop-Location
    }
} finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
    $env:CGO_ENABLED = $previousCGO
}

Copy-Item -LiteralPath $configSource -Destination $packagePath
Copy-Item -LiteralPath $staticSource -Destination $staticTargetDir

foreach ($optionalFile in @("README.md", "LICENSE")) {
    $sourcePath = Join-Path $repoRoot $optionalFile
    if (Test-Path -LiteralPath $sourcePath) {
        Copy-Item -LiteralPath $sourcePath -Destination $packagePath
    }
}

"Package created: $packagePath"
