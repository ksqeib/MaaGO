# MaaGO

MaaGO 是一个使用 **Go + Wails v3 + React** 编写的 MAA（MaaAssistantArknights）桌面客户端，参考 LeviLauncher 的界面结构与视觉风格。

已通过 MaaCore 的 C API（`AsstCaller`）接入真实核心；MaaCore 与依赖 DLL、`resource/`、ADB 全部放在独立的 `core/` 目录，不会散落在程序根目录。

## 文档

- [使用说明（目录、首次运行、任务、日志、排障）](./docs/usage.zh-CN.md)
- [core 运行目录清单](./docs/core-runtime.zh-CN.md)
- [代理与下载说明](./docs/proxy.zh-CN.md)

## 功能

- Wails v3 无边框窗口、自绘标题栏、窗口尺寸持久化
- 侧边栏 + 顶栏导航（圆角联体背景）
- 仪表盘：运行状态、连接检测、定时计划、任务概览、最近日志
- 任务页
  - 6 个核心任务：开始唤醒、理智作战、基建换班、自动公招、信用收支、领取奖励
  - 拖动排序、启用/停用、添加/删除、单任务运行
  - 按任务类型渲染参数表单
  - 实时日志、运行进度、停止任务
- 日志页：级别过滤、搜索、自动滚动、清空、打开日志目录
- 设置页
  - 连接：ADB 地址、自动检测、模拟器端口扫描、ADB 路径、触控模式、截图增强
  - 运行：客户端类型、阻止休眠、屏幕常亮、运行时长上限
  - 定时：每天/指定星期执行、强制定时启动、开始前通知
  - 通知：Windows 系统通知（完成/出错/定时前）
  - 外观：浅色 / 深色 / 跟随系统、简体中文 / English
  - 托盘：最小化到托盘、关闭到托盘
  - 更新：应用内代理、检查更新、下载核心、更新资源、重启安装
- 系统托盘：显示窗口、开始任务、停止任务、退出
- 独立设计的界面与图标，不使用 MAA 官方 logo

## MaaCore 目录

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

详细说明见 [core-runtime.zh-CN.md](./docs/core-runtime.zh-CN.md)。

从官方 MAA 发布包重新生成：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-core.ps1
```

默认从 `D:\tool\MAA-v6.18.0-win-x64` 和 `D:\tool\platform-tools` 复制，可用参数覆盖：

```powershell
.\scripts\install-core.ps1 -Source "D:\path\to\MAA" -AdbSource "D:\path\to\platform-tools"
```

查找顺序：`%MAAGO_CORE_DIR%` → `<exe>/core` → `<exe>/../core` → `<cwd>/core`。
`MaaCore.dll` 使用 `LOAD_WITH_ALTERED_SEARCH_PATH` 加载，依赖 DLL 会从 `core/` 解析。

如果 `core/` 不存在，程序会自动降级为界面演示模式，并在仪表盘/设置中提示。

## 使用真实核心

1. 确认 `core/MaaCore.dll` 与 `core/resource/` 存在。
2. 在“设置 → 连接”填写模拟器 ADB 地址，例如 `127.0.0.1:16384`；留空时会自动扫描常见模拟器端口，并读取 `adb devices` 的结果。
3. 确认 ADB 可用：`core/adb/adb.exe devices` 能看到设备。
4. 回到仪表盘点击“一键开始”。

> 真实运行会操作游戏。首次测试建议只启用“开始唤醒”或“领取奖励”，确认连接和截图正常后再启用完整队列。

## 连接诊断

“测试连接”已改为逐项诊断，不会只给一个“截图失败”：

1. `MaaCore`：DLL 与 `resource/` 是否加载
2. `ADB`：定位 `adb.exe`
3. `ADB 设备`：`adb devices`，必要时先 `adb connect`
4. `MaaCore 连接`：`AsstConnect`
5. `系统信息`：Android 版本、分辨率、DPI、设备 UUID（`adb shell` + `AsstGetUUID`）
6. `截图测试`：`AsstAsyncScreencap` + `AsstGetImage`，保存 PNG 到诊断目录
7. `触控测试`：点击 `(10,10)`，再做一次截图确认界面仍可读

每一步都会显示状态、耗时和具体错误；前置步骤失败时，后续步骤标记为“因上一步未通过而跳过”。

## 开发

要求：

- Windows 10/11
- Go 1.25+
- Node.js 20+
- Wails v3 CLI（建议与 `go.mod` 的 `beta.19` 一致）

```bash
go mod tidy
cd frontend && npm install && cd ..
wails3 task build
wails3 task dev
```

构建产物在 `bin/MaaGO.exe`。

## 数据位置

- 配置：`%APPDATA%\MaaGO\config.json`
- 启动日志：`%APPDATA%\MaaGO\logs\startup.log`
- 任务日志：`%APPDATA%\MaaGO\logs\run.log`
- 核心日志：`%APPDATA%\MaaGO\logs\core\core-YYYY-MM-DD.log`
  - 按日期自动切换文件，保留最近 14 天
  - 每次真实运行前写入会话分隔符、版本、地址、ADB、任务顺序
  - 官方 Release 的 MaaCore 日志写入 `debug/asst.log`，不是 stdout；程序会同时保留 stdout 管道，并把 asst.log 的新增内容实时镜像到按日期滚动的核心日志
- 诊断截图：`%APPDATA%\MaaGO\cache\diagnostics\`
- 更新暂存：`<core 同级>\.maago-updates\`
- 旧核心备份：`<core 同级>\core.backup\`

## 目录结构

```
.
├── build/                  # Wails 构建配置、图标、Windows 清单
├── core/                   # MaaCore 运行时（DLL / adb / resource，二进制不入库）
├── frontend/
│   └── src/
│       ├── assets/locales/ # zh_CN / en_US
│       ├── components/     # 外壳、表单、日志、开关等
│       ├── data/           # 任务元数据、默认配置
│       ├── pages/          # 仪表盘 / 任务 / 日志 / 设置 / 关于
│       ├── pages/settings/ # 设置分区
│       ├── store/          # AppStore：配置、事件、日志、运行状态
│       └── types/
├── internal/
│   ├── app/                # 暴露给前端的 Wails Service
│   ├── apppath/            # 统一路径
│   ├── config/             # 配置读写、迁移、任务模型
│   ├── connection/         # 模拟器端口检测
│   ├── lifecycle/          # 窗口显示/隐藏、退出状态
│   ├── maa/                # MaaCore C API 绑定、任务参数、ADB 解析
│   ├── notify/             # Windows 系统通知
│   ├── runner/             # 任务队列（真实核心 + 演示降级）
│   ├── scheduler/          # 定时执行
│   ├── tray/               # 系统托盘
│   └── version/
├── scripts/install-core.ps1
├── main.go
└── Taskfile.yml
```

## 暂未实现

- 自动战斗（Copilot）
- 小工具（公招识别、仓库识别、干员识别）
- 完成后动作（锁屏/睡眠/关机等，仅保存设置）
- 自动更新与安装包
