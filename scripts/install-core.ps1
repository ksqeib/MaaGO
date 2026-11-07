param(
    [string]$Source = "",
    [string]$AdbSource = "",
    [string]$Version = "",
    [string]$Destination = ""
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
if (-not $Destination) {
    $Destination = Join-Path $root "core"
}
if (-not $Source) {
    throw "Pass -Source <MaaCore SDK directory containing bin\\MaaCore.dll or the DLLs directly>."
}
$Source = [IO.Path]::GetFullPath($Source)
$Destination = [IO.Path]::GetFullPath($Destination)
$resourceDir = Join-Path (Join-Path (Split-Path -Parent $Destination) "base") "resource"
New-Item -ItemType Directory -Force $Destination | Out-Null

$coreDll = Get-ChildItem $Source -Recurse -File -Filter "MaaCore.dll" | Select-Object -First 1
if (-not $coreDll) {
    throw "MaaCore.dll was not found under $Source."
}
$binDir = $coreDll.Directory.FullName
$dlls = Get-ChildItem $binDir -File -Filter "*.dll"
if (-not $dlls) {
    throw "No runtime DLLs were found next to MaaCore.dll."
}

# The SDK package only contains MaaCore/MaaUtils. Do not delete existing DLLs:
# the destination keeps its OpenCV/ONNX/DirectML/FastDeploy runtimes and
# MaaWin32ControlUnit.dll across core updates.
foreach ($dll in $dlls) {
    Copy-Item $dll.FullName $Destination -Force
}
$deps = Get-ChildItem $Source -Recurse -File -Filter "deps.json" | Select-Object -First 1
if ($deps) {
    Copy-Item $deps.FullName $Destination -Force
    $depsData = Get-Content $deps.FullName -Raw | ConvertFrom-Json
    $missing = @()
    if ($depsData.runtimeFiles) {
        foreach ($property in $depsData.runtimeFiles.PSObject.Properties) {
            if (-not (Test-Path (Join-Path $Destination $property.Name))) {
                $missing += $property.Name
            }
        }
    }
    if ($missing.Count -gt 0) {
        Write-Warning "Missing runtime DLLs: $($missing -join ', ')"
        Write-Warning "Open MaaGO -> Settings -> Update -> Download runtime, or copy them from a full MAA release."
    }
}
if ($Version) {
    Set-Content -Path (Join-Path $Destination ".maago-core-version") -Value $Version -NoNewline -Encoding ascii
}

# Keep the official control unit as a fallback and apply MaaGO's RDP bridge.
$bridgeDll = Join-Path (Split-Path -Parent $PSScriptRoot) "bridge\MaaRDPControlUnit\build\Release\MaaWin32ControlUnit.dll"
$installedControlUnit = Join-Path $Destination "MaaWin32ControlUnit.dll"
if (Test-Path $bridgeDll) {
    if (Test-Path $installedControlUnit) {
        Copy-Item $installedControlUnit (Join-Path $Destination "MaaWin32ControlUnit.official.dll") -Force
    }
    Copy-Item $bridgeDll $installedControlUnit -Force
    Write-Output "RDP control-unit bridge installed."
} elseif (Test-Path $installedControlUnit) {
    Write-Warning "RDP control-unit bridge is not built; run scripts\build-rdp-controlunit.ps1 and reinstall it."
}

if ($AdbSource) {
    $adbTarget = Join-Path $Destination "adb"
    New-Item -ItemType Directory -Force $adbTarget | Out-Null
    foreach ($file in @("adb.exe", "AdbWinApi.dll", "AdbWinUsbApi.dll")) {
        $source = Join-Path $AdbSource $file
        if (Test-Path $source) {
            Copy-Item $source $adbTarget -Force
        }
    }
} elseif (-not (Test-Path (Join-Path $Destination "adb\adb.exe"))) {
    Write-Warning "adb\adb.exe is not present. Pass -AdbSource <platform-tools dir> or copy ADB manually."
}

if (-not (Test-Path (Join-Path $resourceDir "version.json"))) {
    Write-Warning "resource\ is missing at $resourceDir. Run MaaGO once to install the base resource."
}

Write-Output "MaaCore runtime installed in $Destination"
Write-Output "Resource directory: $resourceDir (not modified)"
