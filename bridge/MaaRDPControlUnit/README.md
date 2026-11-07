# MaaGO RDP control unit

Source/project name: `MaaRDPControlUnit`. The built artifact is still named
`MaaWin32ControlUnit.dll` because MaaCore's `Win32Controller` hardcodes that
file name and the `MaaWin32ControlUnit*` export names. We keep the legacy name
as a drop-in compatibility layer, not because the implementation uses Win32.

This DLL replaces `MaaWin32ControlUnit.dll` in RDP mode. MaaCore still sees the
same `MaaFwControlUnitAPI` ABI, but:

- `screencap` receives the latest RDP frame over a loopback TCP bridge;
- `click` / `swipe` / `touch_*` / `key_*` / `input_text` are forwarded to the
  remote RDP session by MaaGO;
- no local proxy window or Win32 input injection is used.

The environment variable `MAAGO_RDP_BRIDGE_ADDR` (for example
`127.0.0.1:49738`) selects bridge mode. Without it, the DLL loads
`MaaWin32ControlUnit.official.dll` from the same directory and delegates every
call, so local Win32 mode keeps working.

## Build

```powershell
powershell -ExecutionPolicy Bypass -File ..\..\scripts\build-rdp-controlunit.ps1
```

## Test

The ABI/protocol test includes MaaCore's real `MaaFwControlUnitInterface.h`:

```powershell
cd build\Release
.\MaaRDPControlUnitTest.exe .\MaaWin32ControlUnit.dll
```

For the official-DLL delegation path, put the official DLL next to the test as
`MaaWin32ControlUnit.official.dll` (plus its `MaaUtils.dll` / OpenCV
dependencies) and run:

```powershell
.\MaaRDPControlUnitTest.exe .\MaaWin32ControlUnit.dll delegate
```

## Install

```powershell
powershell -ExecutionPolicy Bypass -File ..\..\scripts\install-rdp-controlunit.ps1 -CoreDir <core dir>
```
