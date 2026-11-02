# MaaCore runtime

This directory holds the MaaCore SDK binaries only. Resource data lives in the
sibling `resource/` directory, so core updates and resource updates are
independent:

```
core/                          # ksqeib/MaaCore SDK
├── MaaCore.dll
├── MaaUtils.dll
├── DirectML.dll
├── fastdeploy_ppocr_maa.dll
├── onnxruntime_maa.dll
├── opencv_world4_maa.dll
├── .maago-core-version
└── adb/
    ├── adb.exe
    ├── AdbWinApi.dll
    └── AdbWinUsbApi.dll

resource/                      # MAA resource (sibling of core)
├── version.json
├── tasks/
├── template/
├── onnx/
└── ...
```

`MaaCore.dll` and the runtime DLLs are large binaries and are ignored by git.
Recreate them from a local SDK package:

```powershell
powershell -ExecutionPolicy Bypass -File ..\scripts\install-core.ps1 `
  -Source "D:\tool\MaaCore-sdk" `
  -AdbSource "D:\tool\platform-tools" `
  -Version "v0.1.1"
```

The launcher looks for `core/` in this order:

1. `%MAAGO_CORE_DIR%`
2. `<exe dir>\core`
3. `<exe dir>\..\core`
4. `<working dir>\core`

MaaCore is loaded with `LOAD_WITH_ALTERED_SEARCH_PATH`, so its sibling DLLs are
resolved from this directory. `AsstLoadResource` is pointed at the parent of
`resource/`, not at `core/`.

## Updating

```powershell
# Rebuild core from a local SDK package (does not touch resource/)
..\scripts\install-core.ps1 -Source "D:\tool\MaaCore-sdk" -Version "v0.1.1"

# Download the latest SDK release (does not touch resource/)
..\scripts\update-core.ps1 -Version latest

# Merge MaaResource hot-update data into resource/
..\scripts\update-resource.ps1
```

Proxy and mirror parameters are documented in
[../docs/proxy.zh-CN.md](../docs/proxy.zh-CN.md). The full manifest is in
[../docs/core-runtime.zh-CN.md](../docs/core-runtime.zh-CN.md).
