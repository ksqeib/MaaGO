# MaaGO 使用说明

本文面向实际使用者和维护者，说明程序需要什么、`core/` 里应该有什么、如何更新核心与资源、如何配置 ADB，以及代理和日志的使用方式。

## 1. 运行要求

### 必需

| 项目 | 要求 |
|---|---|
| 系统 | Windows 10/11 x64 |
| WebView2 Runtime | Windows 11 通常自带；Windows 10 需要安装 Microsoft Edge WebView2 Runtime |
| Visual C++ 运行库 | Microsoft Visual C++ 2015-2022 Redistributable (x64)，MaaCore 和 ONNX/OpenCV 依赖 |
| MaaCore | `core/MaaCore.dll` 及同目录依赖 DLL（ksqeib/MaaCore SDK） |
| MAA 资源 | `resource/`，与 core 目录分开存放 |
| ADB | `core/adb/adb.exe`，或设置中指定其他 `adb.exe` |
| 模拟器/客户端 | MuMu、雷电、蓝叠、夜神、逍遥、WSA 等，或 MAA 支持的 PC 客户端 |
| 磁盘空间 | `core/` 约 100 MB，`resource/` 约 300–400 MB；日志和截图额外占用 |

### 推荐

- 模拟器分辨率 1280×720 或 1920×1080。
- 关闭模拟器的“应用保活”或按 MAA 文档调整截图/触控增强。
- 首次使用只启用一个任务测试，不要直接跑完整队列。

## 2. 目录结构

```text
MaaGO/
├── MaaGO.exe
├── config.json                 # 配置（和程序放一起，方便复制）
├── core/                       # MaaCore SDK（只放二进制）
│   ├── MaaCore.dll
│   ├── MaaUtils.dll
│   ├── DirectML.dll
│   ├── fastdeploy_ppocr_maa.dll
│   ├── onnxruntime_maa.dll
│   ├── opencv_world4_maa.dll
│   ├── .maago-core-version     # 安装的 SDK 版本标记
│   └── adb/
│       ├── adb.exe
│       ├── AdbWinApi.dll
│       └── AdbWinUsbApi.dll
├── resource/                   # MAA 资源（与 core 分开）
│   ├── version.json
│   ├── tasks/
│   ├── template/
│   ├── onnx/
│   ├── global/
│   │   ├── YoStarEN/
│   │   ├── YoStarJP/
│   │   ├── YoStarKR/
│   │   └── txwy/
│   └── ...
├── .maago-updates\             # 更新暂存（下载/解压后等待重启）
└── core.backup\                # 上一版 core 备份
```

用户数据（日志/缓存）仍然在 `%APPDATA%`：

```text
%APPDATA%\MaaGO\
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
```

`core/` 只放 MaaCore SDK 二进制和 ADB；`resource/` 单独放资源。升级 core 不会动资源目录，更新底包或覆盖也不会动 core。

> 旧版本的 `core/resource/` 会在启动时自动迁移到程序目录下的 `resource/`。

## 3. core 目录清单

详细的依赖说明见 [core-runtime.zh-CN.md](./core-runtime.zh-CN.md)。这里先给结论：

### core 必须有

以 `ksqeib/MaaCore` SDK 发布包 `bin/` 为准，目前是：

- `MaaCore.dll`
- `MaaUtils.dll`
- `DirectML.dll`
- `fastdeploy_ppocr_maa.dll`
- `onnxruntime_maa.dll`
- `opencv_world4_maa.dll`

不同 SDK 版本可能增减依赖 DLL，**整个 `bin/` 一起替换**即可，不要混用版本。资源不放在 `core/`。

### resource 必须有

- `resource/version.json`
- `resource/tasks/`
- `resource/template/`
- OCR/ONNX 模型和 `global/` 等其余内容

首次安装如果没有 `resource/`，在“资源底包更新”里下载官方 MAA 完整发布包并只提取 `resource/`。

### 强烈建议有

- `core/adb/adb.exe`
- `core/adb/AdbWinApi.dll`
- `core/adb/AdbWinUsbApi.dll`

### 会自动生成，不需要手动放

- `%APPDATA%\MaaGO\core\debug\asst.log`
- `%APPDATA%\MaaGO\core\cache\`
- `core/.maago-core-version`

### 不要复制

SDK 包里只需要 `bin/*.dll`。以下文件不属于 MaaGO：

- `include/` 头文件
- `MaaCore.pdb`、`MaaUtils.pdb`
- `README.md`、`LICENSE.md`
- `MAA.exe`、`MAA.dll` 等官方启动器文件
- `resource/`（放到程序目录的 `resource/`，不要塞进 `core/`）

## 4. 安装或更新 core

MaaGO 使用 [ksqeib/MaaCore](https://github.com/ksqeib/MaaCore/releases/) 的 SDK 发布包，只包含 `bin/` 下的 `MaaCore.dll`、`MaaUtils.dll` 和 `deps.json`，不包含第三方运行库和资源。第三方 DLL（OpenCV、ONNX Runtime、DirectML、FastDeploy）以及 `MaaWin32ControlUnit.dll` 会从旧 `core/` 保留，不会因为换核心而丢失。

### 方式 A：应用内更新（推荐）

“设置 → 更新”分成三个独立区域，互不影响：

| 区域 | 操作 | 数据源 | 应用方式 |
|---|---|---|---|
| 核心更新 | 检查 / 下载并安装核心 | `ksqeib/MaaCore` Release（`.zip` 或 `.7z`） | 替换 `core/` 下的 DLL，保留旧 core 的第三方 DLL、控制组件和 `adb/` |
| 资源底包更新 | 检查 / 下载并安装底包 | 官方 MAA Release 的 `MAA-v*-win-x64.zip` | **整体替换** `resource/`，写入 `.maago-base-version` |
| 资源覆盖更新 | 检查 / 下载并安装覆盖 | `MaaResource` 主分支 | **合并**到 `resource/`，写入 `.maago-overlay-version` |

1. 需要代理时先启用 HTTP 代理并填写地址，点“测试代理”。
2. “核心更新”可留空目标版本使用最新 Release，或填写 `0.1.4`；点“检查更新”确认版本。
3. 点“下载并安装核心”，等待进度完成；点“重启并安装”后程序会在加载 MaaCore 前替换 DLL。
4. 资源底包和资源覆盖分别在各自区域检查和下载。
5. **一次只能暂存一项更新**：底包或覆盖下载完成后必须先重启应用，才能继续下一项。

核心更新过程会：

- 下载 SDK 包到 `.maago-updates/` 暂存目录（支持 `.zip` 和 `.7z`）
- 校验 `MaaCore.dll` 和 `MaaUtils.dll`，并把 `deps.json` 一起放进 `core/`
- 重启时把旧 `core/` 改名为 `core.backup/`
- 把新 DLL 移到 `core/`，写入 `.maago-core-version`
- 从旧 core 保留 OpenCV、ONNX Runtime、DirectML、FastDeploy、`MaaWin32ControlUnit.dll`、`adb/` 等 SDK 未包含的文件
- 资源目录 `resource/` 完全不受影响

> SDK 包里的 `deps.json` 记录了 OpenCV、ONNX Runtime、DirectML、FastDeploy 的目标版本。MaaGO 会读取它并检查 `core/` 下的前置库：
>
> - 更新核心时，旧 core 的前置库会被保留并校验文件大小；缺少或不匹配时，自动下载 MaaGO 内置的固定运行库包（当前为 MaaCore v0.1.3 完整 SDK，只提取其中的运行库 DLL）。
> - 也可以在“核心更新 → 前置运行库”点“下载前置库”，单独修复当前 core，重启后生效。
> - 如果新核心要求的版本没有对应的运行库包，MaaGO 会拒绝暂存该核心并给出提示，避免装出一个无法加载的核心。
>
> 运行库包只包含 `opencv_world4_maa.dll`、`onnxruntime_maa.dll`、`DirectML.dll`、`fastdeploy_ppocr_maa.dll`，不会覆盖 `MaaCore.dll`、`MaaUtils.dll` 或 `MaaWin32ControlUnit.dll`。
>
> 来源说明：该包取自 `ksqeib/MaaCore` 的 `v0.1.3` 完整 SDK Release 资产（MaaGO 使用的核心也来自同一项目），不是 MAA 官方发布包。DirectML / FastDeploy 与 MAA 官方发布包中的文件哈希一致；OpenCV / ONNX Runtime 是相同版本但由 MaaCore 项目另行构建，文件大小和哈希与 MAA 官方发布包中的版本不同。MaaGO 以核心提供方的 `deps.json` 和固定哈希为准。

### 方式 B：从本地 SDK 包复制

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-core.ps1 `
  -Source "D:\tool\MaaCore-sdk" `
  -AdbSource "D:\tool\platform-tools" `
  -Version "v0.1.4"
```

默认会复制 `bin/` 下的全部 DLL 和 `deps.json`，保留目标目录已有的第三方 DLL，不复制 `include/`、PDB 和资源。

### 方式 C：命令行下载

```powershell
# 指定版本
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 -Version 0.1.4

# 自动取 latest
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 -Version latest

# 带代理
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 `
  -Version 0.1.4 `
  -Proxy "http://127.0.0.1:7890"

# 使用 GitHub 镜像前缀
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 `
  -Version 0.1.4 `
  -Mirror "https://your-mirror.example/"
```

脚本会：

1. 下载 `MaaCore-sdk-windows-x64-<版本>.zip` 或 `.7z`（新版本通常只提供 `.7z`）
2. 提取 `bin/` 下的全部 DLL 和 `deps.json`
3. 覆盖 `core/` 下的 `MaaCore.dll`、`MaaUtils.dll`，**保留**旧的第三方 DLL、控制组件和 `adb/`
4. 写入 `.maago-core-version`
5. 不修改 `resource/`

`.7z` 需要本机安装 [7-Zip](https://www.7-zip.org/)（脚本会自动查找 `7z` 或默认安装目录）；没有 7-Zip 时请使用应用内更新。

`-DryRun` 只打印 URL 和目标路径，不下载、不修改文件。

### 资源底包与资源覆盖

资源更新拆成两项，分别对应“设置 → 更新”里的两个区域。

**资源底包**是官方 MAA 完整发布包里的 `resource/`，包含模板图、`platform_diff/PC` 和所有静态资源：

- 点“下载并安装底包”会下载约 300 MB 的 `MAA-v*-win-x64.zip`，只提取其中的 `resource/`。
- 重启后**整体替换** `resource/`，并写入 `.maago-base-version`。
- 底包适合游戏大版本更新后模板图缺失、PC 平台差异文件丢失等情况。
- 替换底包会清除 `.maago-overlay-version`，之后可以再叠加覆盖。

**资源覆盖**来自 `MaaResource` 主分支，只包含与核心版本无关的动态数据：

- 点“下载并安装覆盖”会下载几 MB 的 `main` 分支压缩包。
- 重启后**合并**到 `resource/`，不会删除底包里的 `platform_diff/PC`，并写入 `.maago-overlay-version`。
- 适合新活动、公招、关卡数据的小幅更新。

应用内进入“设置 → 更新”，在对应区域操作；两项都只在重启后生效，且一次只能暂存一项。命令行仍可使用：

```powershell
# 只更新动态资源（合并到 resource/）
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1

# 带代理
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1 -Proxy "http://127.0.0.1:7890"

# 只预览
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1 -DryRun
```

脚本会把 `MaaResource` 的 `resource/` 内容合并覆盖到 `resource/`，不会删除基础资源；它对应应用内的“资源覆盖更新”，不会替换底包。

### 游戏界面更新与核心版本

游戏大版本更新会改动主界面按钮、主题皮肤等界面元素，而这类**模板图在基础 `resource/` 里，不在 `MaaResource` 动态更新里，也不在 MaaCore SDK 包里**。如果游戏比资源新，可能出现所有任务都在第一步识别失败。

例如 2026-11-01 的国服界面更新（新主题「天想」、新版 UI 尺寸）需要包含 `Default/Task-90.png` 等新模板的 `resource/`。遇到这种情况：

- 在“设置 → 更新”点“下载并安装底包”，让 `resource/` 回到包含最新模板的官方状态；如果需要最新动态数据，再单独更新资源覆盖。
- 确认 `resource/tasks/UiTheme/Task.json` 里有 `Task-90.png`。
- 如果界面又发生较大变化，等待 MAA 发布新资源；不要在本地手工维护模板副本。

MaaCore SDK 本身不携带资源，更新 SDK 不会修改 `resource/`；核心、底包、覆盖需要分别更新。

### 版本兼容与回滚

- 核心、资源底包、资源覆盖分开更新、分开回滚。
- 只更新资源覆盖通常没问题；如果任务报资源/模板错误，改用资源底包恢复官方 `resource/`。
- 底包会整体替换 `resource/`，覆盖只合并；底包更新后再叠加覆盖。
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

### PC 客户端（Win32 窗口）

MaaGO 也可以直接控制明日方舟 PC 客户端，不需要 ADB：

**使用前检查（最重要）：**

1. 游戏内 `设置 → 游戏 → UI 缩放` 改为 **100%**。PC 客户端默认是 90%，不改会导致主页/公招/基建等模板大面积匹配失败。
2. 游戏保持 **16:9 全屏 / 无边框全屏**，推荐 `1920×1080` 或 `1280×720`；不要最小化，不要把窗口拉成非 16:9。
3. Windows 显示缩放建议 100%，避免 DPI 拉伸。
4. 鼠标尽量停在游戏窗口中央。PC 客户端有鼠标视差；MaaGO 的 RDP 模式会在截图前自动归中。
5. 主题尽量使用默认/浅色主题；未知主题可能导致主页入口识别失败。
6. 游戏和 MaaGO 都建议以管理员身份运行。
7. 建议禁用 PRTS 软件光标（见下方小节），避免光标遮挡模板。

操作步骤：

1. 先手动启动 PC 客户端，进入游戏主界面。
2. “设置 → 连接 → 连接方式”选择 **Win32 窗口**。
3. 窗口标题留空即可（官服/B服/台服会自动匹配 `明日方舟`；国际服按客户端类型匹配）。
4. 截图/鼠标/键盘方式保持默认：
   - 截图：`PrintWindow`
   - 鼠标：`PostMessageWithWindowPos`
   - 键盘：`SendMessage`
5. 点击“连接诊断”，确认窗口绑定、截图、点击测试通过。

注意事项：

- 需要 `core/MaaWin32ControlUnit.dll`（PC 窗口控制组件）。它不在 MaaCore SDK 包里，MaaGO 会从旧 core 保留；如果缺失，请从官方 MAA 发布包的 `bin/` 或根目录复制到 `core/`。
- PC 客户端如果以管理员身份运行，MaaGO 也建议以管理员身份运行，否则截图/输入可能被 Windows 拒绝（错误码 5）。设置页检测到未提权时会显示“以管理员身份重启”按钮，点击后走 UAC。
- 游戏窗口需要保持 16:9 分辨率（如 1280×720、1920×1080）；窗口被拉成其他比例时 MaaCore 会拒绝绑定。
- 游戏内 UI 缩放必须保持 100%（PC 默认 90%），鼠标尽量停在窗口中央。PC 客户端的主界面是鼠标布局，和 Android 主界面不同，不能直接套用全部 Android 模板。
- MaaGO 会在 `resource/platform_diff/PC` 中补充 PC 确认弹窗模板（`PopupConfirmPC.png` / `PopupCancelPC.png`）；其他界面如果识别失败，需要从 PC 截图重新裁剪模板，或改用 OCR/多锚点。
- PC 模式下“开始唤醒”不会自动启动游戏（只负责进入主界面），请先手动启动客户端。
- 切换 ADB / Win32 连接方式后建议重启 MaaGO，避免平台资源叠加影响。

#### 禁用 PRTS 软件光标

PC 客户端会渲染独立软件光标，在 RDP/`PrintWindow` 截图里可能遮挡模板。可使用第三方工具 [assfafa/arknights-system-cursor](https://github.com/assfafa/arknights-system-cursor)（MIT，非官方）：

1. 完全退出游戏。
2. 下载 `ArknightsSystemCursor-v1.0.0-win-portable.zip`，解压。
3. 运行 `ArknightsSystemCursor.exe`，选择 `Arknights.exe`，点“启用系统指针”；或使用 CLI：

```powershell
$tool = ".\ArknightsSystemCursor-v1.0.0-win-portable\ArknightsSystemCursor.Cli.exe"
$game = "D:\Hypergryph Launcher\games\Arknights"
& $tool status --game-dir $game --json
& $tool enable --game-dir $game --json
& $tool restore --game-dir $game --json
```

工具通过资源包内的 `dyn/mouse/pc_mouse_mgr.prefab` 识别鼠标资源，备份到 `%LOCALAPPDATA%\ArknightsSystemCursor\installations\`，再把原文件改名为 `.system-cursor-disabled-<sha256前缀>`。游戏更新后如果光标恢复，重新执行 `enable`；如果资源加载异常，执行 `restore`。

### RDP 远程机器

MaaGO 可以通过 RDP 控制另一台 Windows 机器上的明日方舟 PC 客户端；MaaGO 仍运行在本机，远端不需要安装 MaaGO。

> RDP 模式同样要先完成上面的 PC 客户端检查：**UI 缩放 100%**、16:9 全屏、鼠标居中；建议同时禁用 PRTS 软件光标。

1. 远端机器开启“远程桌面”，并确认账户已登录、游戏在主界面且处于前台。
2. 本机“设置 → 连接 → 连接方式”选择 **RDP 远程机器**。
3. 填写主机、端口（3389）、用户名、密码；分辨率建议与远端桌面一致（推荐 1920×1080），并让游戏保持全屏，避免连接时改变分辨率导致游戏变成窗口模式。
4. 点击“连接诊断”。MaaGO 会：
   - 建立 RDP 连接（NLA）
   - 如果远端处于锁屏，发送 Enter 唤醒（必要时使用保存的密码）
   - 在仪表盘显示 **RDP 预览卡片**
   - 让 MaaCore 绑定隐藏的代理窗口；截图从 RDP 画面读取，点击/按键转发到远端
5. 诊断通过后即可运行任务。MaaCore 的任务逻辑仍在本机执行。

注意事项：

- 保持 MaaGO 运行：RDP 会话由 MaaGO 持有；关闭 MaaGO 会断开远端会话（远端会锁屏）。
- RDP 预览卡片是内嵌小窗，默认只预览；点击卡片或“放大”进入大窗，打开 **操作模式** 后点击/按键才会发送到远端。关闭操作模式或大窗即回到只预览，不会误操作。
- 预览卡片显示未连接时，可以点“连接”只建立 RDP 会话（不执行任务）；“连接诊断”也会建立会话。
- 远端桌面/游戏需要保持 16:9；如果远端游戏变成窗口模式，请先让它全屏。
- 密码保存在本机 `config.json`，不写入日志；请自行保护该文件。
- 不要同时用 mstsc 连接同一个远端会话，否则会接管并断开 MaaGO 的 RDP 连接。
- RDP 模式使用隐藏代理窗口和 PrintWindow 钩子，截图不依赖 MaaGO 前台；但本机 MaaGO 必须保持运行。
- PC 客户端会根据鼠标位置产生界面偏转（类似手机陀螺仪）。MaaGO 在每次自动点击后、下一次截图前会把远端鼠标移回窗口中央，等界面稳定后再截图，避免“点击后指针停在按钮上导致下一帧识别失败”。
- 不建议在截图前把鼠标放到左下角：那样会重新引入最大幅度的界面偏转，模板分数通常会更差。如果担心鼠标光标遮挡，优先考虑禁用客户端的 PRTS 软件光标（使用系统光标），或把光标停在中线附近、没有按钮的空白区域。

## 6. 首次使用流程

1. 确认 `core/MaaCore.dll` 和 `resource/version.json` 存在。
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
| 理智作战 | 关卡弹窗多选与排序（主线/资源/活动/剿灭）、次数、普通理智药（数量）、临期理智药（N 天内过期）、源石（数量）、代理倍率 |
| 基建换班 | 模式、设施、无人机、心情阈值、会客室/线索 |
| 自动公招 | 刷新、Tag 等级、确认等级、次数、加急、保留词条 |
| 信用收支 | 访问好友、购物、优先/黑名单、折扣、信用低于 300 停止、OF-1 |
| 领取奖励 | 每日/每周、邮件、免费单抽、幸运墙、开采、月卡、签到 |

暂不包含自动战斗（Copilot）、公招识别、仓库识别、干员识别等小工具。

> 理智作战默认**只使用临期理智药**（默认临期窗口 1 天，即 24 小时内），普通理智药默认关闭。需要吃普通药时，再打开“使用普通理智药”并填写数量。
>
> MaaCore 会先消耗普通理智药，再应用临期窗口；如果同时开启两者，普通药会先被用掉。只想吃临期药时，保持“使用普通理智药”关闭即可。

### 关卡选择

理智作战的“关卡顺序”是一个弹窗选择器，不是下拉框：

- 顶部按类型分页：主线、资源、活动、剿灭、其他。
- 可以勾选多个关卡；右侧“执行顺序”按 1、2、3… 排列，支持上移、下移、删除。
- 运行时会从第 1 个开始，自动跳过当前未开放的关卡（资源本按星期、活动按起止时间判断）；全部不可用时按第一个执行，并在日志中提示。
- 不选任何关卡时使用游戏内“当前/上次”关卡。
- 关卡数据来自本地 `resource/stages.json`；活动名称和开放时间来自 MAA 的 `StageActivityV2.json`，首次打开或超过 6 小时会通过“设置 → 更新”的代理下载并缓存到 `%APPDATA%\MaaGO\cache\`。
- “代理倍率”默认是 **自动**：MaaCore 会根据当前理智和关卡消耗调整连战次数（不是固定 1 次一打），日志里会显示“本次连战 N 次”。

### 定时执行

“设置 → 定时”支持多个每天执行的时间点：

- 右上角“添加时间点”可以添加任意多个时间。
- 每个时间点可以单独设置：
  - 时间（HH:mm）
  - 星期（至少选一天）
  - 是否启用
  - 当前有任务运行时是否强制停止并重新开始
- “开始前发送系统通知”和提前分钟数是全局设置。
- 仪表盘/设置里的“下次运行”取所有启用时间点中最早的一个。
- 定时只在 MaaGO 运行期间生效；应用退出后不会触发。

### 性能与 GPU 加速

“设置 → 性能”控制 MaaCore 的 OCR 推理设备：

- **默认 CPU**：兼容性最好。
- **GPU 加速 OCR**：通过 DirectML 在显卡上运行 OCR 推理，降低 CPU 占用；需要显卡/驱动支持 DirectML。
- **GPU 设备**：通常填 `0`；多显卡可以填设备序号，或 `luid:十六进制适配器 ID`。
- GPU 只影响 OCR，不改变截图、模板匹配和触控方式。
- OCR 提供者只能在资源加载前设置，所以修改后需要重启 MaaGO。
- 如果 GPU 不可用，MaaCore 会记录错误并自动回退 CPU；可以看 `logs\core\core-YYYY-MM-DD.log` 里的 `gpu_active` / `falling back to CPU`。
- 启动成功时 `logs\startup.log` 会出现 `OCR GPU acceleration enabled (device 0)`。

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
- MaaCore 的结构化事件会先解析成中文再写入日志，例如：
  - `会客室 #1 当前产品：综合`
  - `公招标签：狙击干员 / 群攻`
  - `关卡 AP-5 掉落：龙门币×360、声望×360（3★）`
  - `截图耗时：平均 215ms，最大 238ms，最小 185ms`
  - 未知事件会按 `字段：值` 的格式展开；完全无法解析时才回退显示原始 JSON。
> 日志文件本身仍保留 MaaCore 原始内容，方便排查；界面日志和 `run.log` 里是解析后的版本。

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
- 检查更新、下载核心、资源底包和资源覆盖都会使用这个代理

命令行脚本也支持同样的代理参数：

```powershell
.\scripts\update-core.ps1 -Version 0.1.4 -Proxy "http://127.0.0.1:7890"
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
| 显示“MaaCore 资源加载失败” | `resource/` 缺失或损坏；重新执行 `update-core.ps1` |
| ADB 可执行文件失败 | 放 `core/adb/adb.exe`，或在设置中填 ADB 路径 |
| ADB 设备失败 | `adb devices` 是否 `device`；`unauthorized` 需在模拟器允许调试；网络 ADB 先 `adb connect` |
| MaaCore 连接失败 | 先把触控模式切到 `ADB`；检查地址、模拟器配置、重启 ADB |
| 系统信息失败 | `adb shell getprop ro.build.version.release` 和 `wm size` 是否正常 |
| 截图测试失败 | 换截图增强/关闭增强；重启模拟器；确认分辨率正常 |
| 触控测试失败 | 换触控模式；检查模拟器输入权限；MuMu 关闭应用保活 |
| 任务一开始就失败 | 先看 `logs\core\core-YYYY-MM-DD.log` 和 `logs\run.log` 的具体错误 |
| PC 端所有任务都识别失败 | 先检查游戏内 `设置 → 游戏 → UI 缩放` 是否为 **100%**（PC 默认 90%），再确认 16:9 全屏、鼠标居中、PRTS 软件光标已禁用 |
| 游戏更新后所有任务都无法识别 | 界面模板在 `resource/` 里；先在“设置 → 更新”更新资源底包，必要时再叠加资源覆盖 |
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
