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
$resourceDir = Join-Path (Join-Path (Split-Path -Parent $Destination) "base") "resource"
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

$tag = ""
$assetUrl = ""
$assetName = ""

if ($Version -eq "latest" -or $Version -eq "nightly" -or $Version -like "nightly-*") {
    $tag = "nightly"
    $apiUrl = Add-Mirror "https://api.github.com/repos/ksqeib/MaaCoreBuild/releases/tags/nightly"
} else {
    $tag = "v" + $Version.TrimStart("v")
    $apiUrl = Add-Mirror "https://api.github.com/repos/ksqeib/MaaCoreBuild/releases/tags/$tag"
}

if ($DryRun) {
    Write-Host "[dry-run] GET $apiUrl"
    Write-Host "[dry-run] destination: $Destination"
    return
}
$release = Invoke-RestMethod -Uri $apiUrl -Headers $headers -UseBasicParsing @proxyArgs
$tag = $release.tag_name
$asset = $release.assets | Where-Object {
    $_.name -match "^MaaCore-sdk-windows-x64-.+\.(zip|7z)$" -and $_.name -notmatch "symbols"
} | Select-Object -First 1
if (-not $asset) { throw "No MaaCore SDK asset found in release $tag." }
$assetName = $asset.name
$assetUrl = Add-Mirror $asset.browser_download_url

Write-Host "Release: $tag"
Write-Host "Asset:   $assetUrl"
Write-Host "Target:  $Destination"
Write-Host "Resource: $resourceDir (unchanged)"

if ($DryRun) {
    Write-Host "[dry-run] no files were changed"
    return
}

New-Item -ItemType Directory -Force $Destination | Out-Null
$tempRoot = Join-Path $env:TEMP ("MaaGO-core-" + [Guid]::NewGuid().ToString("N"))
$archiveExt = [IO.Path]::GetExtension($assetName)
$archivePath = Join-Path $tempRoot ("core" + $archiveExt)
New-Item -ItemType Directory -Force $tempRoot | Out-Null

try {
    Write-Host "Downloading..."
    Invoke-WebRequest -Uri $assetUrl -OutFile $archivePath -UseBasicParsing @proxyArgs
    Write-Host "Extracting..."
    $extractDir = Join-Path $tempRoot "extract"
    if ($archiveExt -eq ".7z") {
        $sevenZip = Get-Command 7z -ErrorAction SilentlyContinue | Select-Object -First 1
        if (-not $sevenZip) {
            foreach ($candidate in @(
                (Join-Path $env:ProgramFiles "7-Zip\7z.exe"),
                (Join-Path ${env:ProgramFiles(x86)} "7-Zip\7z.exe")
            )) {
                if ($candidate -and (Test-Path $candidate)) {
                    $sevenZip = Get-Item $candidate
                    break
                }
            }
        }
        if (-not $sevenZip) {
            throw "This release ships a .7z archive. Install 7-Zip or use the in-app core update."
        }
        & $sevenZip.Source x $archivePath "-o$extractDir" -y | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "7-Zip failed with exit code $LASTEXITCODE." }
    } else {
        Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force
    }

    $coreDll = Get-ChildItem $extractDir -Recurse -File -Filter "MaaCore.dll" | Select-Object -First 1
    if (-not $coreDll) { throw "MaaCore.dll was not found in the archive." }
    $binDir = $coreDll.Directory.FullName
    $dlls = Get-ChildItem $binDir -File -Filter "*.dll"
    if (-not $dlls) { throw "No runtime DLLs were found next to MaaCore.dll." }

    # The SDK archive only contains MaaCore/MaaUtils. Keep third-party runtime
    # DLLs (OpenCV, ONNX Runtime, DirectML, FastDeploy) and control units that
    # are already in the destination instead of deleting the whole directory.
    foreach ($dll in $dlls) {
        Copy-Item $dll.FullName $Destination -Force
        Write-Host "  copied $($dll.Name)"
    }
    $deps = Get-ChildItem $extractDir -Recurse -File -Filter "deps.json" | Select-Object -First 1
    if ($deps) {
        Copy-Item $deps.FullName $Destination -Force
        Write-Host "  copied deps.json"
        $depsData = Get-Content $deps.FullName -Raw | ConvertFrom-Json
        $missing = @()
        if ($depsData.runtimeFiles) {
            foreach ($property in $depsData.runtimeFiles.PSObject.Properties) {
                if (-not (Test-Path (Join-Path $Destination $property.Name))) {
                    $missing += $property.Name
                }
            }
        }
        $runtimeSection = $depsData.runtime.'windows-x64'
        if ($runtimeSection -and $runtimeSection.files) {
            foreach ($property in $runtimeSection.files.PSObject.Properties) {
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
    $marker = $tag
    if ($depsData -and $depsData.maaCore -and $depsData.maaCore.version) {
        $marker = $depsData.maaCore.version
    } elseif ($tag -eq "nightly" -and $release.body -match 'nightly-(\d{8})-([0-9a-fA-F]{7,40})') {
        $marker = "nightly-$($Matches[1])-" + $Matches[2].Substring(0, 7).ToLower()
    }
    Set-Content -Path (Join-Path $Destination ".maago-core-version") -Value $marker -NoNewline -Encoding ascii

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

    # Re-apply the RDP control-unit shim. A fresh SDK contains the official
    # MaaWin32ControlUnit.dll; keep it as the fallback and put the bridge DLL
    # back in place so RDP mode stays headless.
    $bridgeDll = Join-Path (Split-Path -Parent $PSScriptRoot) "bridge\MaaRDPControlUnit\build\Release\MaaWin32ControlUnit.dll"
    $installedControlUnit = Join-Path $Destination "MaaWin32ControlUnit.dll"
    if (Test-Path $bridgeDll) {
        if (Test-Path $installedControlUnit) {
            Copy-Item $installedControlUnit (Join-Path $Destination "MaaWin32ControlUnit.official.dll") -Force
        }
        Copy-Item $bridgeDll $installedControlUnit -Force
        Write-Host "  installed RDP control-unit bridge"
    } elseif (Test-Path $installedControlUnit) {
        Write-Warning "RDP control-unit bridge is not built; run scripts\build-rdp-controlunit.ps1 and reinstall it."
    }

    if (-not (Test-Path (Join-Path $resourceDir "version.json"))) {
        Write-Warning "resource\ is missing at $resourceDir. Run MaaGO once or install the base resource."
    }

    Write-Host ""
    Write-Host "MaaCore updated in $Destination"
    Write-Host "Restart MaaGO, then run Connection Diagnostics."
} finally {
    if ($KeepArchive) {
        Write-Host "Archive kept at $archivePath"
    } else {
        Remove-Item $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
