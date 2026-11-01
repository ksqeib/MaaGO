# MaaCore runtime

This directory holds the MaaCore runtime and is intentionally kept separate
from the launcher binary:

```
core/
├── MaaCore.dll
├── MaaUtils.dll
├── MaaAdbControlUnit.dll
├── MaaWin32ControlUnit.dll
├── DirectML.dll
├── fastdeploy_ppocr_maa.dll
├── onnxruntime_maa.dll
├── opencv_world4_maa.dll
├── adb/
│   ├── adb.exe
│   ├── AdbWinApi.dll
│   └── AdbWinUsbApi.dll
└── resource/
```

`MaaCore.dll` and the `resource/` tree are large binaries and are ignored by
git. Recreate the directory from an official MAA release with:

```powershell
powershell -ExecutionPolicy Bypass -File ..\scripts\install-core.ps1
```

The launcher looks for `core/` in this order:

1. `%MAAGO_CORE_DIR%`
2. `<exe dir>\core`
3. `<exe dir>\..\core`
4. `<working dir>\core`

MaaCore is loaded with `LOAD_WITH_ALTERED_SEARCH_PATH`, so its sibling DLLs are
resolved from this directory and never need to be copied next to the launcher.

## Updating

```powershell
# Rebuild this directory from a local MAA release
..\scripts\install-core.ps1 -Source "D:\tool\MAA-v6.20.1-win-x64" -AdbSource "D:\tool\platform-tools"

# Download a full release (core + resource)
..\scripts\update-core.ps1 -Version 6.20.1

# Merge MaaResource hot-update data into resource/
..\scripts\update-resource.ps1
```

Proxy and mirror parameters are documented in
[../docs/proxy.zh-CN.md](../docs/proxy.zh-CN.md). The full manifest is in
[../docs/core-runtime.zh-CN.md](../docs/core-runtime.zh-CN.md).
