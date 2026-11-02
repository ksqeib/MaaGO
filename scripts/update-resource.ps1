param(
    [string]$Proxy = "",
    [string]$Mirror = "",
    [string]$Destination = "",
    [string]$Branch = "main",
    [switch]$KeepArchive,
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

if (-not $Destination) {
    $Destination = Join-Path (Split-Path -Parent $PSScriptRoot) "resource"
}
$Destination = [IO.Path]::GetFullPath($Destination)
if (-not $Proxy) {
    $Proxy = $env:HTTPS_PROXY
    if (-not $Proxy) { $Proxy = $env:HTTP_PROXY }
}

$proxyArgs = @{}
if ($Proxy) {
    $proxyArgs["Proxy"] = $Proxy
    Write-Host "Using proxy: $Proxy"
}

$url = "https://codeload.github.com/MaaAssistantArknights/MaaResource/zip/refs/heads/$Branch"
if ($Mirror) {
    $url = $Mirror.TrimEnd("/") + "/" + $url
}

Write-Host "Resource archive: $url"
Write-Host "Target:           $Destination"

if (-not (Test-Path (Join-Path $Destination "version.json"))) {
    Write-Warning "Base resource is missing. Install a full MAA release first; this script only applies hot-update files."
}

if ($DryRun) {
    Write-Host "[dry-run] no files were changed"
    return
}

New-Item -ItemType Directory -Force $Destination | Out-Null
$tempRoot = Join-Path $env:TEMP ("MaaGO-resource-" + [Guid]::NewGuid().ToString("N"))
$zipPath = Join-Path $tempRoot "resource.zip"
New-Item -ItemType Directory -Force $tempRoot | Out-Null

try {
    Write-Host "Downloading..."
    Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing @proxyArgs
    Write-Host "Extracting..."
    $extractDir = Join-Path $tempRoot "extract"
    Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force

    $resourceDir = Get-ChildItem $extractDir -Recurse -Directory -Filter "resource" |
        Where-Object { Test-Path (Join-Path $_.FullName "version.json") } |
        Select-Object -First 1
    if (-not $resourceDir) { throw "resource/version.json was not found in the downloaded archive." }

    Write-Host "Merging hot-update resource..."
    Copy-Item (Join-Path $resourceDir.FullName "*") $Destination -Recurse -Force
    Write-Host "Resource updated in $Destination"
    Write-Host "Restart MaaGO before the next run if a task is active."
} finally {
    if ($KeepArchive) {
        Write-Host "Archive kept at $zipPath"
    } else {
        Remove-Item $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
