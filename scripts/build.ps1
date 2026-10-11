$ErrorActionPreference = 'Stop'
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptDir

Set-Location $repoRoot

Write-Host 'Synchronizing Versions across project...'
& powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $scriptDir 'sync_version.ps1')

Write-Host 'Building BS2PRO-Controller...'

$displayVersion = ''
$wailsJson = Get-Content -LiteralPath 'cmd/bs2pro-controller/wails.json' -Raw
if ($wailsJson -match '"productVersion"\s*:\s*"([^"]+)"') {
    $displayVersion = $Matches[1]
}

if ([string]::IsNullOrWhiteSpace($displayVersion)) {
    Write-Warning 'Could not extract version from wails.json, using dev'
    $displayVersion = 'dev'
}

$coreLdflags = "-X github.com/TIANLI0/BS2PRO-Controller/internal/version.BuildVersion=$displayVersion -s -w"
$monitorLdflags = "$coreLdflags -H windowsgui"
$wailsLdflags = "-X github.com/TIANLI0/BS2PRO-Controller/internal/version.BuildVersion=$displayVersion"

Write-Host 'Building core service...'
if (Test-Path 'cmd/core/winres/winres.json') {
    & go-winres make --in cmd/core/winres/winres.json --out cmd/core/rsrc
}
& go build -trimpath -ldflags $coreLdflags -o 'build/bin/BS2PRO-CoreService.exe' './cmd/core/'

Write-Host 'Building monitor probe...'
if (Test-Path 'cmd/bs2pro-monitor/winres/winres.json') {
    & go-winres make --in cmd/bs2pro-monitor/winres/winres.json --out cmd/bs2pro-monitor/rsrc
}
& go build -trimpath -ldflags $monitorLdflags -o 'build/bin/BS2PRO-Monitor.exe' './cmd/bs2pro-monitor/'

$wailsVersion = (go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2).Trim()
Write-Host "Ensuring Wails CLI $wailsVersion..."
& go install "github.com/wailsapp/wails/v2/cmd/wails@$wailsVersion"
if ($LASTEXITCODE -ne 0) {
    throw "Failed to install Wails CLI $wailsVersion"
}

$env:PATH = "$env:PATH;C:\Program Files (x86)\NSIS\Bin"

$null = New-Item -ItemType Directory -Force -Path 'frontend/out'
if (-not (Get-ChildItem 'frontend/out' -File -Recurse -Force | Where-Object { $_.Name -notmatch '^[._]' })) {
    Set-Content -LiteralPath 'frontend/out/index.html' -Value '<!doctype html><title></title>' -Encoding ascii
}

Write-Host 'Building main application...'
Push-Location 'cmd/bs2pro-controller'
try {
    & wails build -nsis -trimpath -webview2 browser -ldflags $wailsLdflags
    if ($LASTEXITCODE -ne 0) {
        throw 'Wails build failed!'
    }
} finally {
    Pop-Location
}

if (-not (Test-Path 'build/bin/BS2PRO-CoreService.exe')) {
    throw 'Core service build failed!'
}

if (-not (Test-Path 'build/bin/BS2PRO-Monitor.exe')) {
    throw 'Monitor probe build failed!'
}

if (-not (Test-Path 'build/bin/BS2PRO-Controller-amd64-installer.exe')) {
    throw 'NSIS installer build failed!'
}

Write-Host "Build completed successfully with display version $displayVersion"
