param(
    [string]$Source = "D:\tool\MAA-v6.18.0-win-x64",
    [string]$AdbSource = "D:\tool\platform-tools"
)

$ErrorActionPreference = "Stop"

$coreDir = Join-Path (Split-Path -Parent $PSScriptRoot) "core"
$adbDir = Join-Path $coreDir "adb"
New-Item -ItemType Directory -Force $coreDir, $adbDir | Out-Null

$dlls = @(
    "MaaCore.dll",
    "MaaUtils.dll",
    "MaaAdbControlUnit.dll",
    "MaaWin32ControlUnit.dll",
    "DirectML.dll",
    "fastdeploy_ppocr_maa.dll",
    "onnxruntime_maa.dll",
    "opencv_world4_maa.dll"
)

foreach ($dll in $dlls) {
    $path = Join-Path $Source $dll
    if (-not (Test-Path $path)) {
        throw "Missing $path; pass -Source <MAA release directory>"
    }
    Copy-Item $path $coreDir -Force
}

$resourceSrc = Join-Path $Source "resource"
if (-not (Test-Path $resourceSrc)) {
    throw "Missing $resourceSrc"
}
$resourceDst = Join-Path $coreDir "resource"
if (-not (Test-Path $resourceDst)) {
    Copy-Item $resourceSrc $resourceDst -Recurse -Force
}

$adbFiles = @("adb.exe", "AdbWinApi.dll", "AdbWinUsbApi.dll")
foreach ($file in $adbFiles) {
    $path = Join-Path $AdbSource $file
    if (Test-Path $path) {
        Copy-Item $path $adbDir -Force
    }
}

Write-Output "MaaCore runtime installed in $coreDir"
