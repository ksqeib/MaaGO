# MaaCore 运行目录说明

本文说明 `core/` 和 `resource/` 的职责，以及从 `ksqeib/MaaCore` SDK 和 MAA 资源中应该复制什么、不要复制什么。

## 1. 目录总览

```text
core/                          # 只放 MaaCore SDK 二进制
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

resource/                      # MAA 资源，与 core 分开
├── version.json
├── tasks/
├── template/
├── onnx/
├── global/
└── ...
```

所有文件都是 x64。MaaGO 通过 `LoadLibraryEx(..., LOAD_WITH_ALTERED_SEARCH_PATH)` 加载 `core/MaaCore.dll`，依赖 DLL 会优先从 `core/` 解析；资源则通过 `AsstLoadResource` 加载程序目录下的 `resource/`。

## 2. 必需文件

### `MaaCore.dll`

主核心。提供 `AsstCaller` C API、任务调度、图像识别和控制器管理。

### `MaaUtils.dll`

MaaCore 的公共工具库，包含日志、路径、平台等基础能力。缺少时 `MaaCore.dll` 加载会失败。

### `DirectML.dll`

DirectML 运行时，ONNX/OCR 推理在 Windows 上使用。

### `onnxruntime_maa.dll`

ONNX Runtime 的 MAA 定制版，用于干员识别、OCR 等模型推理。

### `fastdeploy_ppocr_maa.dll`

PP-OCR 推理组件。

### `opencv_world4_maa.dll`

OpenCV 的 MAA 定制版，用于图像处理、模板匹配。

### `resource/`（与 core 同级）

MAA 的任务定义、模板图、OCR 模型、关卡数据、活动数据、多服务器覆盖资源等。没有它，MaaCore 会初始化失败或任务无法识别。它不在 `core/` 里面：

```text
MaaGO/
├── core/        # MaaCore SDK
└── resource/    # MAA 资源
```

关键子目录：

| 路径 | 作用 |
|---|---|
| `resource/tasks/` | 任务定义和识别流程 |
| `resource/template/` | 界面模板图 |
| `resource/onnx/` | 干员/OCR 模型 |
| `resource/global/` | 外服覆盖资源 |
| `resource/platform_diff/` | 平台差异资源 |
| `resource/version.json` | 动态资源版本 |

`resource/version.json` 必须存在，连接诊断和启动检查会用它判断基础资源是否完整。

## 3. ADB（强烈建议）

```text
core/adb/
├── adb.exe
├── AdbWinApi.dll
└── AdbWinUsbApi.dll
```

MaaGO 的查找顺序：

1. 设置中的“ADB 路径”
2. `core/adb/adb.exe`
3. 常见模拟器安装目录
4. `PATH` 中的 `adb`

官方 MAA 发布包里通常不带 ADB；可以从 Android SDK Platform-Tools 复制以上三个文件，或使用模拟器自带的 `adb.exe` 并在设置中指定。

## 4. 自动生成的文件

MaaGO 会把 MaaCore 的 UserDir 设置到：

```text
%APPDATA%\MaaGO\core
```

因此以下文件由 MaaCore 自动生成，不在 `core/`：

```text
%APPDATA%\MaaGO\core\debug\asst.log    # MaaCore 原始日志
%APPDATA%\MaaGO\core\cache\            # 识别缓存
```

MaaGO 会把 `asst.log` 的新增内容镜像到：

```text
%APPDATA%\MaaGO\logs\core\core-YYYY-MM-DD.log
```

所以不需要手动在 `core/` 里创建 `debug/` 或 `cache/`。

## 5. 不要复制的内容

从 `ksqeib/MaaCore` SDK 包中只需要 `bin/*.dll`。以下内容不要放进 `core/`：

| 文件/目录 | 原因 |
|---|---|
| `include/` | 集成头文件，运行不需要 |
| `MaaCore.pdb`、`MaaUtils.pdb` | 调试符号 |
| `README.md`、`LICENSE.md` | 文档 |
| `MAA.exe`、`MAA.dll` | 官方 WPF 前端，MaaGO 不使用 |
| `resource/` | 应放到程序目录的 `resource/`，不要放进 `core/` |
| 旧 `debug/`、`cache/` | 旧日志和缓存 |

## 6. 版本匹配

- `MaaCore.dll` 和同目录 DLL 必须来自同一个 SDK Release，整体替换。
- 资源是独立分发的；更新 core 不会修改 `resource/`。
- 只更新 `MaaResource` 动态资源时，基础 `resource/` 必须已经存在。
- 如果连接诊断提示资源加载失败，用“更新资源”补装基础资源，或检查 `resource/version.json`。
- MaaCore 版本可在“设置 → 连接 → 核心状态”或 `logs/startup.log` 查看；SDK 的 `DEBUG_VERSION` 会由 `core/.maago-core-version` 中的发布标签替代显示。
- 游戏客户端比 MaaCore 新时，主界面模板可能对不上；应更新基础 `resource/` 到包含该界面适配的版本（2026-11 国服界面需要带 `Task-90` 的新资源），不要手工拷贝旧模板。

## 7. 安装/更新命令

应用内：设置 → 更新，可启用代理、检查版本、下载核心、更新资源，重启后自动安装。

命令行：

```powershell
# 从本地 SDK 包复制 bin/ 下的 DLL
.\scripts\install-core.ps1 -Source "D:\tool\MaaCore-sdk" -AdbSource "D:\tool\platform-tools" -Version "v0.1.1"

# 从 GitHub 下载指定 SDK 版本
.\scripts\update-core.ps1 -Version 0.1.1

# 更新动态资源（或首次安装基础资源）
.\scripts\update-resource.ps1
```

代理用法见 [proxy.zh-CN.md](./proxy.zh-CN.md)。

## 8. 最小自检

更新完成后执行：

```powershell
Test-Path .\core\MaaCore.dll
Test-Path .\resource\version.json
Test-Path .\core\adb\adb.exe
.\core\adb\adb.exe devices
```

然后在 MaaGO 中运行“连接诊断”，确认：

- MaaCore 加载成功
- ADB 可执行文件存在
- ADB 设备为 `device`
- AsstConnect 成功
- 系统信息、截图、触控全部通过
