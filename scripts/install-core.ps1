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
$resourceDir = Join-Path (Split-Path -Parent $Destination) "resource"
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

Get-ChildItem $Destination -File -Filter "*.dll" -ErrorAction SilentlyContinue | Remove-Item -Force
foreach ($dll in $dlls) {
    Copy-Item $dll.FullName $Destination -Force
}
if ($Version) {
    Set-Content -Path (Join-Path $Destination ".maago-core-version") -Value $Version -NoNewline -Encoding ascii
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
