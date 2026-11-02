# MaaGO 使用说明

本文面向实际使用者和维护者，说明程序需要什么、`core/` 里应该有什么、如何更新核心与资源、如何配置 ADB，以及代理和日志的使用方式。

## 1. 运行要求

### 必需

| 项目 | 要求 |
|---|---|
| 系统 | Windows 10/11 x64 |
| WebView2 Runtime | Windows 11 通常自带；Windows 10 需要安装 Microsoft Edge WebView2 Runtime |
| Visual C++ 运行库 | Microsoft Visual C++ 2015-2022 Redistributable (x64)，MaaCore 和 ONNX/OpenCV 依赖 |
| MaaCore | `core/MaaCore.dll` 及同目录依赖 DLL |
| MAA 资源 | `core/resource/`，必须与核心配套使用 |
| ADB | `core/adb/adb.exe`，或设置中指定其他 `adb.exe` |
| 模拟器/客户端 | MuMu、雷电、蓝叠、夜神、逍遥、WSA 等，或 MAA 支持的 PC 客户端 |
| 磁盘空间 | `core/` 约 300–400 MB（主要是 `resource/`）；日志和截图额外占用 |

### 推荐

- 模拟器分辨率 1280×720 或 1920×1080。
- 关闭模拟器的“应用保活”或按 MAA 文档调整截图/触控增强。
- 首次使用只启用一个任务测试，不要直接跑完整队列。

## 2. 目录结构

```text
MaaGO/
├── MaaGO.exe
├── core/
│   ├── MaaCore.dll
│   ├── MaaUtils.dll
│   ├── MaaAdbControlUnit.dll
│   ├── MaaWin32ControlUnit.dll
│   ├── DirectML.dll
│   ├── fastdeploy_ppocr_maa.dll
│   ├── onnxruntime_maa.dll
│   ├── opencv_world4_maa.dll
│   ├── adb/
│   │   ├── adb.exe
│   │   ├── AdbWinApi.dll
│   │   └── AdbWinUsbApi.dll
│   └── resource/
│       ├── version.json
│       ├── tasks/
│       ├── template/
│       ├── onnx/
│       ├── global/
│       │   ├── YoStarEN/
│       │   ├── YoStarJP/
│       │   ├── YoStarKR/
│       │   └── txwy/
│       └── ...
└── ...
```

用户数据不在程序目录：

```text
%APPDATA%\MaaGO\
├── config.json
├── core\                    # MaaCore 的 UserDir（原始日志和缓存）
│   ├── debug\asst.log
│   └── cache\
├── logs\
│   ├── startup.log
│   ├── run.log
│   └── core\
│       └── core-YYYY-MM-DD.log
└── cache\
    └── diagnostics\
        └── diagnostic-YYYYMMDD-HHMMSS.png

core 同级目录还会出现：
├── .maago-updates\        # 更新暂存（下载/解压后等待重启）
└── core.backup\           # 上一版核心备份
```

`core/` 只放运行需要的二进制和资源，不放用户日志，不放 `MAA.exe`/WPF 文件。

## 3. core 目录清单

详细的依赖说明见 [core-runtime.zh-CN.md](./core-runtime.zh-CN.md)。这里先给结论：

### 必须有

- `MaaCore.dll`
- `MaaUtils.dll`
- `MaaAdbControlUnit.dll`
- `MaaWin32ControlUnit.dll`
- `DirectML.dll`
- `fastdeploy_ppocr_maa.dll`
- `onnxruntime_maa.dll`
- `opencv_world4_maa.dll`
- `resource/`

### 强烈建议有

- `adb/adb.exe`
- `adb/AdbWinApi.dll`
- `adb/AdbWinUsbApi.dll`

### 会自动生成，不需要手动放

- `%APPDATA%\MaaGO\core\debug\asst.log`
- `%APPDATA%\MaaGO\core\cache\`

### 不要复制

从 MAA 发布包里只需要“核心运行时 + resource”。以下文件不属于 MaaGO：

- `MAA.exe`、`MAA.dll`
- `MAA.deps.json`、`MAA.runtimeconfig.json`
- `hostfxr.dll`、`hostpolicy.dll`
- WPF/HandyControl 等 .NET UI DLL
- `Res/`、`Python/`、`config/`、`data/`、`externals/`
- 旧版 `debug/`、`cache/`

## 4. 安装或更新 core

### 方式 A：应用内更新（推荐）

“设置 → 更新”：

1. 需要代理时先启用 HTTP 代理并填写地址，点“测试代理”。
2. “核心更新”可留空目标版本使用最新 Release，或填写 `6.20.1`。
3. 点“检查更新”确认最新版本。
4. 点“下载并安装核心”，等待进度完成。
5. 点“重启并安装”；程序重启时会在加载 MaaCore 前替换核心。

资源更新也在同一页：点“更新资源”后重启生效。

更新过程会：

- 下载到 `core/` 同级的 `.maago-updates/` 暂存目录
- 校验 `MaaCore.dll` 和 `resource/version.json`
- 重启时把旧 `core/` 改名为 `core.backup/`
- 把新核心移动到 `core/`
- 保留旧核心用于手动回滚

### 方式 B：从本地 MAA 发布包复制

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-core.ps1 `
  -Source "D:\tool\MAA-v6.20.1-win-x64" `
  -AdbSource "D:\tool\platform-tools"
```

默认会复制：

- 8 个核心 DLL
- 整个 `resource/`
- ADB 三个文件

### 方式 C：命令行下载

```powershell
# 指定版本，最稳定
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 -Version 6.20.1

# 自动取 latest
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 -Version latest

# 带代理
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 `
  -Version 6.20.1 `
  -Proxy "http://127.0.0.1:7890"

# 使用 GitHub 镜像前缀
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 `
  -Version 6.20.1 `
  -Mirror "https://your-mirror.example/"
```

脚本会：

1. 下载 `MAA-v<版本>-win-x64.zip`
2. 提取到临时目录
3. 只复制核心 DLL 和 `resource/`
4. 可选复制 ADB
5. 校验关键文件

`-DryRun` 只打印 URL 和目标路径，不下载、不修改文件。

### 更新资源（不更新核心）

MAA 有独立的动态资源仓库 `MaaResource`，用于新活动关卡、公招数据等。它只包含“与核心版本无关”的动态文件，**必须已有基础 `resource/`**。

应用内：“设置 → 更新 → 更新资源”，重启后生效。

命令行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1

# 带代理
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1 -Proxy "http://127.0.0.1:7890"

# 只预览
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1 -DryRun
```

脚本会把 `MaaResource` 的 `resource/` 内容合并覆盖到 `core/resource/`，不会删除基础资源。

### 游戏界面更新与核心版本

游戏大版本更新会改动主界面按钮、主题皮肤等界面元素，而这类**模板图随 MaaCore 发布，不随 `MaaResource` 动态资源更新**。如果游戏比 MaaCore 新，可能出现所有任务都在第一步识别失败。

例如 2026-11-01 的国服界面更新（新主题「天想」、新版 UI 尺寸）需要 **MaaCore v6.21.1 或更高版本**。遇到这种情况：

- 在“设置 → 更新”检查并安装最新核心，然后重启。
- 确认 `logs\startup.log` 中的核心版本已经是新版本。
- 不要把旧核心的模板单独拷贝到新核心里，也不建议手工维护模板副本；直接更新完整核心即可。

MaaCore 更新通常会在游戏大版本更新后的几天内发布；暂时无法更新时，可以先在游戏内把主题切回默认/日间主题再运行。

### 版本兼容与回滚

- 核心 DLL 和 `resource/` 最好来自同一个 MAA 发布包。
- 只更新 `MaaResource` 动态资源通常没问题；如果任务报资源/模板错误，改用完整发布包覆盖。
- 更新前建议备份：

```powershell
Copy-Item .\core ..\core-backup-$(Get-Date -Format yyyyMMdd-HHmmss) -Recurse
```

- 更新后重启 MaaGO，并在“设置 → 连接”重新跑一次连接诊断。

## 5. ADB 与模拟器准备

### 常见端口

| 模拟器 | 常见 ADB 地址 |
|---|---|
| MuMu 12 | `127.0.0.1:16384` |
| MuMu 多开 | `127.0.0.1:16385`、`127.0.0.1:16416` |
| 雷电 | `127.0.0.1:5555` |
| 雷电多开 | `127.0.0.1:5556`、`127.0.0.1:5557` |
| 蓝叠 | `127.0.0.1:5555` / `127.0.0.1:5565` |
| 夜神 | `127.0.0.1:62001` |
| 逍遥 | `127.0.0.1:21503` |
| WSA | `127.0.0.1:58526` |

### 检查

```powershell
.\core\adb\adb.exe devices
```

出现 `device` 才算正常；`unauthorized` 需要在模拟器里点“允许 USB 调试”。

网络 ADB：

```powershell
.\core\adb\adb.exe connect 192.168.10.160:5555
```

连接失败时：

```powershell
.\core\adb\adb.exe kill-server
.\core\adb\adb.exe start-server
.\core\adb\adb.exe devices
```

### MaaGO 设置

“设置 → 连接”：

- **连接方式**：ADB
- **自动检测连接**：建议开启
- **连接地址**：留空时会扫描常见端口，并读取 `adb devices`
- **模拟器**：自动，或按实际选择
- **触控模式**：默认 `ADB`，兼容性最好；MuMu/雷电可尝试 `minitouch` / `MaaTouch`
- **截图增强**：MuMu/LD 可尝试开启

## 6. 首次使用流程

1. 确认 `core/MaaCore.dll` 和 `core/resource/version.json` 存在。
2. 启动 `MaaGO.exe`。
3. 设置 → 连接，点击“连接诊断”。
4. 七项全部通过后再开始任务：
   - MaaCore
   - ADB 可执行文件
   - ADB 设备
   - MaaCore 连接
   - 系统信息
   - 截图测试
   - 触控测试
5. 回到仪表盘，只启用“开始唤醒”或“领取奖励”试跑。
6. 确认游戏被正确识别后，再启用完整 6 个任务。

> 连接诊断的触控测试会在 `(10,10)` 点一下，用于确认输入链路。它不是只做截图。

## 7. 任务说明

当前保留 6 个核心任务：

| 任务 | 说明 |
|---|---|
| 开始唤醒 | 连接设备、启动客户端、进入主界面 |
| 理智作战 | 关卡、次数、普通理智药（数量）、临期理智药（N 天内过期）、源石（数量）、代理倍率 |
| 基建换班 | 模式、设施、无人机、心情阈值、会客室/线索 |
| 自动公招 | 刷新、Tag 等级、确认等级、次数、加急、保留词条 |
| 信用收支 | 访问好友、购物、优先/黑名单、折扣、信用低于 300 停止、OF-1 |
| 领取奖励 | 每日/每周、邮件、免费单抽、幸运墙、开采、月卡、签到 |

暂不包含自动战斗（Copilot）、公招识别、仓库识别、干员识别等小工具。

> 理智作战默认**只使用临期理智药**（默认临期窗口 1 天，即 24 小时内），普通理智药默认关闭。需要吃普通药时，再打开“使用普通理智药”并填写数量。
>
> MaaCore 会先消耗普通理智药，再应用临期窗口；如果同时开启两者，普通药会先被用掉。只想吃临期药时，保持“使用普通理智药”关闭即可。

## 8. 日志与诊断

### 核心日志

```text
%APPDATA%\MaaGO\logs\core\core-YYYY-MM-DD.log
```

- 按日期滚动，保留 14 天。
- 每次真实运行前写入分隔符、版本、地址、ADB、任务顺序。
- 运行结束写入 `session ended`。
- 内容来自 MaaCore 的 `asst.log` 镜像。
- 子任务识别失败只在候选变化时记录一次：`子任务未识别：A / B（来自 C）`，不会重复刷屏。
- 某个任务最终失败时会补充 `最后识别失败：...` 和 `失败截图：<png 路径>`，可直接打开截图查看当时的界面。

> 官方 Release 的 MaaCore 在非 DEBUG 构建下不写 stdout，只写自己的 `debug/asst.log`。MaaGO 保留了 stdout 管道，同时实时镜像 `asst.log`，所以最终仍是一份完整的、按日期滚动的核心日志。

原始文件：

```text
%APPDATA%\MaaGO\core\debug\asst.log
```

它由 MaaCore 自己按大小滚动，通常不需要手动查看；排障时以 `logs\core\core-YYYY-MM-DD.log` 为主。

### 其他日志

| 文件 | 内容 |
|---|---|
| `logs\startup.log` | 启动、MaaCore 加载、WebView2 错误 |
| `logs\run.log` | 任务链回调、连接信息、任务状态 |
| `cache\diagnostics\*.png` | 连接诊断截图 |

界面入口：

- 日志页右上角“核心日志”“失败截图”
- 设置 → 连接 → 核心状态卡片“核心日志”“截图目录”

## 9. 代理

代理已经做进应用本体：

- “设置 → 更新 → 代理与下载”
- 启用 HTTP 代理，填写 `http://127.0.0.1:7890`
- 点“测试代理”确认 GitHub API 可达
- 检查更新、下载核心、更新资源都会使用这个代理

命令行脚本也支持同样的代理参数：

```powershell
.\scripts\update-core.ps1 -Version 6.20.1 -Proxy "http://127.0.0.1:7890"
```

完整说明见 [proxy.zh-CN.md](./proxy.zh-CN.md)。

注意：

- 应用内代理用于更新检查与下载。
- MaaCore 自身的网络功能（企鹅物流/一图流等）当前未启用，不保证遵循应用内代理。
- 第三方镜像只应从可信来源使用，并自行校验版本和文件。

## 10. 常见问题

| 现象 | 处理 |
|---|---|
| 设置里显示“MaaCore 未找到” | 检查 `core/MaaCore.dll` 是否存在；确认是 x64；确认 VC++ 运行库 |
| 显示“MaaCore 资源加载失败” | `core/resource/` 缺失或损坏；重新执行 `update-core.ps1` |
| ADB 可执行文件失败 | 放 `core/adb/adb.exe`，或在设置中填 ADB 路径 |
| ADB 设备失败 | `adb devices` 是否 `device`；`unauthorized` 需在模拟器允许调试；网络 ADB 先 `adb connect` |
| MaaCore 连接失败 | 先把触控模式切到 `ADB`；检查地址、模拟器配置、重启 ADB |
| 系统信息失败 | `adb shell getprop ro.build.version.release` 和 `wm size` 是否正常 |
| 截图测试失败 | 换截图增强/关闭增强；重启模拟器；确认分辨率正常 |
| 触控测试失败 | 换触控模式；检查模拟器输入权限；MuMu 关闭应用保活 |
| 任务一开始就失败 | 先看 `logs\core\core-YYYY-MM-DD.log` 和 `logs\run.log` 的具体错误 |
| 游戏更新后所有任务都无法识别 | 游戏界面模板随 MaaCore 发布；在“设置 → 更新”安装最新核心并重启（2026-11 国服界面需要 v6.21.1 或更高版本） |
| 核心日志里只有“子任务未识别” | 表示 MaaCore 在当前截图里找不到下一步按钮；看紧随其后的“失败截图”，检查游戏是否停在弹窗、加载页或主题编辑页 |
| 主界面主题不是默认主题 | 未知主题可能导致入口按钮模板不匹配；可在游戏内切回默认/日间主题，或等待 MaaCore 适配 |
| 日志中文乱码 | 用 UTF-8 编辑器打开；核心日志是 UTF-8 |
| 代理 407 | 代理需要认证；把用户名密码写进代理 URL，或换本机免认证端口 |
| 代理 SSL 错误 | 检查代理证书、系统时间、TLS 1.2；换官方源或可信镜像 |
| 更新下载失败（拒绝访问） | 程序目录不可写（如 Program Files）；移动到可写目录或以管理员身份运行 |
| 更新后版本没变 | 必须重启；检查 `<core 同级>\\.maago-updates\\pending.json` 和 `core.backup` |

## 11. 安全与注意事项

- 代理地址、账号密码不要提交到仓库，也不要直接发到公开 issue。
- 诊断截图和核心日志可能包含设备 UUID、路径、游戏界面信息，分享前先脱敏。
- 只从官方 MAA 发布页或可信镜像下载核心和资源。
- 本项目是非官方客户端；真实运行会操作游戏，请自行评估账号风险。
- 自动更新、安装包、完成后动作（关机/睡眠等）暂未实现。
