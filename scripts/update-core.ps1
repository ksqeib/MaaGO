param(
    [string]$Version = "latest",
    [string]$Proxy = "",
    [string]$Mirror = "",
    [string]$Destination = "",
    [string]$AdbSource = "",
    [switch]$KeepArchive,
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

if (-not $Destination) {
    $Destination = Join-Path (Split-Path -Parent $PSScriptRoot) "core"
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
$headers = @{ "User-Agent" = "MaaGO-UpdateCore" }

function Add-Mirror([string]$url) {
    if (-not $Mirror) { return $url }
    return ($Mirror.TrimEnd("/") + "/" + $url)
}

$release = $null
$assetUrl = $null
$tag = ""

if ($Version -eq "latest") {
    $apiUrl = Add-Mirror "https://api.github.com/repos/MaaAssistantArknights/MaaAssistantArknights/releases/latest"
    if ($DryRun) {
        Write-Host "[dry-run] GET $apiUrl"
        Write-Host "[dry-run] destination: $Destination"
        return
    }
    $release = Invoke-RestMethod -Uri $apiUrl -Headers $headers -UseBasicParsing @proxyArgs
    $tag = $release.tag_name
    $asset = $release.assets | Where-Object { $_.name -match "^MAA-v.+-win-x64\.zip$" } | Select-Object -First 1
    if (-not $asset) { throw "No win-x64 asset found in the latest release." }
    $assetUrl = Add-Mirror $asset.browser_download_url
} else {
    $tag = "v" + $Version.TrimStart("v")
    $versionNumber = $tag.TrimStart("v")
    $assetName = "MAA-v$versionNumber-win-x64.zip"
    $assetUrl = Add-Mirror "https://github.com/MaaAssistantArknights/MaaAssistantArknights/releases/download/$tag/$assetName"
}

Write-Host "Release: $tag"
Write-Host "Asset:   $assetUrl"
Write-Host "Target:  $Destination"

if ($DryRun) {
    Write-Host "[dry-run] no files were changed"
    return
}

New-Item -ItemType Directory -Force $Destination | Out-Null
$tempRoot = Join-Path $env:TEMP ("MaaGO-core-" + [Guid]::NewGuid().ToString("N"))
$zipPath = Join-Path $tempRoot "core.zip"
New-Item -ItemType Directory -Force $tempRoot | Out-Null

try {
    Write-Host "Downloading..."
    Invoke-WebRequest -Uri $assetUrl -OutFile $zipPath -UseBasicParsing @proxyArgs
    Write-Host "Extracting..."
    $extractDir = Join-Path $tempRoot "extract"
    Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force

    $coreDll = Get-ChildItem $extractDir -Recurse -File -Filter "MaaCore.dll" | Select-Object -First 1
    if (-not $coreDll) { throw "MaaCore.dll was not found in the archive." }
    $sourceRoot = $coreDll.Directory.FullName

    $required = @(
        "MaaCore.dll",
        "MaaUtils.dll",
        "MaaAdbControlUnit.dll",
        "MaaWin32ControlUnit.dll",
        "DirectML.dll",
        "fastdeploy_ppocr_maa.dll",
        "onnxruntime_maa.dll",
        "opencv_world4_maa.dll"
    )
    foreach ($file in $required) {
        $source = Join-Path $sourceRoot $file
        if (Test-Path $source) {
            Copy-Item $source $Destination -Force
            Write-Host "  copied $file"
        } else {
            Write-Warning "  missing $file in this release"
        }
    }

    $resourceSource = Join-Path $sourceRoot "resource"
    if (-not (Test-Path $resourceSource)) {
        throw "resource/ was not found next to MaaCore.dll."
    }
    $resourceTarget = Join-Path $Destination "resource"
    if (Test-Path $resourceTarget) {
        Write-Host "Replacing resource/ ..."
        Remove-Item $resourceTarget -Recurse -Force
    }
    Copy-Item $resourceSource $resourceTarget -Recurse -Force
    Write-Host "  copied resource/"

    if ($AdbSource) {
        $adbTarget = Join-Path $Destination "adb"
        New-Item -ItemType Directory -Force $adbTarget | Out-Null
        foreach ($file in @("adb.exe", "AdbWinApi.dll", "AdbWinUsbApi.dll")) {
            $source = Join-Path $AdbSource $file
            if (Test-Path $source) {
                Copy-Item $source $adbTarget -Force
                Write-Host "  copied adb/$file"
            }
        }
    } elseif (-not (Test-Path (Join-Path $Destination "adb\adb.exe"))) {
        Write-Warning "adb\adb.exe is not present. Pass -AdbSource <platform-tools dir> or copy ADB manually."
    }

    Write-Host ""
    Write-Host "MaaCore updated in $Destination"
    Write-Host "Restart MaaGO, then run Connection Diagnostics."
} finally {
    if ($KeepArchive) {
        Write-Host "Archive kept at $zipPath"
    } else {
        Remove-Item $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
