param(
    [string]$OpenCVRoot = "D:\self\maa\MaaDeps\vcpkg\installed\maa-x64-windows",
    [string]$Configuration = "Release"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$source = Join-Path $root "bridge\MaaRDPControlUnit"
$build = Join-Path $source "build"

if (-not (Test-Path (Join-Path $OpenCVRoot "lib\opencv_world4.lib"))) {
    throw "OpenCV import library not found under $OpenCVRoot. Pass -OpenCVRoot <MaaDeps prefix>."
}

cmake -S $source -B $build -G "Visual Studio 18 2026" -A x64 -DOPENCV_ROOT="$OpenCVRoot"
if ($LASTEXITCODE -ne 0) {
    throw "CMake configure failed."
}

cmake --build $build --config $Configuration
if ($LASTEXITCODE -ne 0) {
    throw "CMake build failed."
}

$dll = Join-Path $build "$Configuration\MaaWin32ControlUnit.dll"
if (-not (Test-Path $dll)) {
    throw "Build finished but $dll was not produced."
}
Write-Host "Built: $dll"
