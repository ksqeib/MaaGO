param(
    [string]$OpenCVRoot = "D:\self\maa\MaaDeps\vcpkg\installed\maa-x64-windows",
    [string]$Configuration = "Release",
    [string]$Generator = "Visual Studio 18 2026",
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$source = Join-Path $root "bridge\MaaRDPControlUnit"
$build = Join-Path $source "build"

if (-not (Test-Path (Join-Path $OpenCVRoot "lib\opencv_world4.lib"))) {
    throw "OpenCV import library not found under $OpenCVRoot. Pass -OpenCVRoot <MaaDeps prefix>."
}

$configureArgs = @(
    "-S", $source,
    "-B", $build,
    "-G", $Generator,
    "-A", "x64",
    "-DOPENCV_ROOT=$OpenCVRoot"
)
if ($SkipTests) {
    $configureArgs += "-DMAA_BUILD_CONTROLUNIT_TESTS=OFF"
} else {
    $configureArgs += "-DMAA_BUILD_CONTROLUNIT_TESTS=ON"
}
cmake @configureArgs
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
