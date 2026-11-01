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
