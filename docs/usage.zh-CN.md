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
```

`core/` 只放 MaaCore SDK 二进制和 ADB；`resource/` 单独放资源。升级 core 不会动资源目录，更新资源也不会动 core。

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

首次安装如果没有 `resource/`，MaaGO 会在“更新资源”时下载官方 MAA 完整发布包并只提取 `resource/`。

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

MaaGO 使用 [ksqeib/MaaCore](https://github.com/ksqeib/MaaCore/releases/) 的 SDK 发布包，只包含 `bin/` 下的 DLL 和依赖，不包含资源。

### 方式 A：应用内更新（推荐）

“设置 → 更新”：

1. 需要代理时先启用 HTTP 代理并填写地址，点“测试代理”。
2. “核心更新”可留空目标版本使用最新 Release，或填写 `0.1.1`。
3. 点“检查更新”确认最新版本。
4. 点“下载并安装核心”，等待进度完成。
5. 点“重启并安装”；程序重启时会在加载 MaaCore 前替换 `core/` 下的 DLL。

资源更新也在同一页：点“更新资源”后重启生效。

更新过程会：

- 下载 SDK 包到 `.maago-updates/` 暂存目录
- 校验 `MaaCore.dll` 和 `MaaUtils.dll`
- 重启时把旧 `core/` 改名为 `core.backup/`
- 把新 DLL 移到 `core/`，写入 `.maago-core-version`
- 资源目录 `resource/` 完全不受影响

### 方式 B：从本地 SDK 包复制

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-core.ps1 `
  -Source "D:\tool\MaaCore-sdk" `
  -AdbSource "D:\tool\platform-tools" `
  -Version "v0.1.1"
```

默认会复制 `bin/` 下的全部 DLL，不复制 `include/`、PDB 和资源。

### 方式 C：命令行下载

```powershell
# 指定版本
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 -Version 0.1.1

# 自动取 latest
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 -Version latest

# 带代理
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 `
  -Version 0.1.1 `
  -Proxy "http://127.0.0.1:7890"

# 使用 GitHub 镜像前缀
powershell -ExecutionPolicy Bypass -File .\scripts\update-core.ps1 `
  -Version 0.1.1 `
  -Mirror "https://your-mirror.example/"
```

脚本会：

1. 下载 `MaaCore-sdk-windows-x64-<版本>.zip`
2. 提取 `bin/` 下的全部 DLL
3. 替换 `core/` 下的 DLL，保留 `adb/`
4. 写入 `.maago-core-version`
5. 不修改 `resource/`

`-DryRun` 只打印 URL 和目标路径，不下载、不修改文件。

### 更新资源（不更新核心）

MAA 有独立的动态资源仓库 `MaaResource`，用于新活动关卡、公招数据等。它只包含“与核心版本无关”的动态文件，**必须已有基础 `resource/`**。

- 已有 `resource/`：应用内“更新资源”会下载 MaaResource 并合并到 `resource/`。
- 没有 `resource/`（首次安装）：会下载官方 MAA 完整发布包，只提取其中的 `resource/` 作为基础资源。
- 下载量：动态更新几 MB；首次安装基础资源约 300 MB。

应用内：“设置 → 更新 → 更新资源”，重启后生效。

命令行：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1

# 带代理
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1 -Proxy "http://127.0.0.1:7890"

# 只预览
powershell -ExecutionPolicy Bypass -File .\scripts\update-resource.ps1 -DryRun
```

脚本会把 `MaaResource` 的 `resource/` 内容合并覆盖到 `resource/`，不会删除基础资源。

### 游戏界面更新与核心版本

游戏大版本更新会改动主界面按钮、主题皮肤等界面元素，而这类**模板图在基础 `resource/` 里，不在 `MaaResource` 动态更新里，也不在 MaaCore SDK 包里**。如果游戏比资源新，可能出现所有任务都在第一步识别失败。

例如 2026-11-01 的国服界面更新（新主题「天想」、新版 UI 尺寸）需要包含 `Default/Task-90.png` 等新模板的 `resource/`。遇到这种情况：

- 在“设置 → 更新”点“更新资源”。已有基础资源时只更新动态数据；没有基础资源时会下载官方 MAA 完整发布包并提取 `resource/`。
- 确认 `resource/tasks/UiTheme/Task.json` 里有 `Task-90.png`。
- 如果界面又发生较大变化，等待 MAA 发布新资源；不要在本地手工维护模板副本。

MaaCore SDK 本身不携带资源，更新 SDK 不会修改 `resource/`；两者需要分别更新。

### 版本兼容与回滚

- MaaCore SDK 和 `resource/` 分开更新、分开回滚。
- 只更新 `MaaResource` 动态资源通常没问题；如果任务报资源/模板错误，改用官方 MAA 完整发布包的 `resource/` 覆盖。
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
- 检查更新、下载核心、更新资源都会使用这个代理

命令行脚本也支持同样的代理参数：

```powershell
.\scripts\update-core.ps1 -Version 0.1.1 -Proxy "http://127.0.0.1:7890"
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
| 游戏更新后所有任务都无法识别 | 界面模板在 `resource/` 里；在“设置 → 更新”更新资源，或安装包含新模板的官方 MAA 完整资源 |
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
