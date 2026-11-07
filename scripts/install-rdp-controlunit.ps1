param(
    [string]$CoreDir = "",
    [string]$DllPath = "",
    [switch]$Force
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
if (-not $CoreDir) {
    $CoreDir = Join-Path $root "core"
}
if (-not $DllPath) {
    $DllPath = Join-Path $root "bridge\MaaRDPControlUnit\build\Release\MaaWin32ControlUnit.dll"
}
$CoreDir = [IO.Path]::GetFullPath($CoreDir)
$DllPath = [IO.Path]::GetFullPath($DllPath)

if (-not (Test-Path $DllPath)) {
    throw "RDP control unit not built: $DllPath. Run scripts\build-rdp-controlunit.ps1 first."
}
if (-not (Test-Path (Join-Path $CoreDir "MaaCore.dll"))) {
    throw "MaaCore.dll not found in $CoreDir."
}

$target = Join-Path $CoreDir "MaaWin32ControlUnit.dll"
$official = Join-Path $CoreDir "MaaWin32ControlUnit.official.dll"

if (-not (Test-Path $official)) {
    if (-not (Test-Path $target)) {
        throw "Neither $target nor $official exists. Install the official MAA core first."
    }
    Copy-Item $target $official -Force
    Write-Host "Preserved official control unit: $official"
}

try {
    Copy-Item $DllPath $target -Force
}
catch {
    if (-not $Force) {
        throw "Could not replace $target (MaaGO may still be running). Close MaaGO, then retry. $($_.Exception.Message)"
    }
    throw
}

Write-Host "Installed RDP control unit: $target"
Write-Host "Official control unit kept at: $official"
