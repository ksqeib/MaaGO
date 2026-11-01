# MaaGO

MaaGO 是一个使用 **Go + Wails v3 + React** 编写的 MAA（MaaAssistantArknights）桌面客户端，参考 LeviLauncher 的界面结构与视觉风格。

> 当前尚未接入 MaaCore 调用层。任务运行是**界面演示模式**：会完整走一遍任务队列、状态与日志流程，但不会操作游戏。

## 已实现

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
  - 连接：ADB 地址、自动检测、常见模拟器端口扫描、ADB 路径、触控模式、截图增强
  - 运行：客户端类型、阻止休眠、屏幕常亮、运行时长上限、完成后动作（仅保存）
  - 定时：每天/指定星期执行、强制定时启动、开始前通知
  - 通知：Windows 系统通知（完成/出错/定时前）
  - 外观：浅色 / 深色 / 跟随系统、简体中文 / English
  - 托盘：最小化到托盘、关闭到托盘
- 系统托盘：显示窗口、开始任务、停止任务、退出
- 无需 MAA 官方素材：界面与图标均为独立设计

## 开发

要求：

- Windows 10/11
- Go 1.25+
- Node.js 20+
- Wails v3 CLI（建议与本项目 `go.mod` 的 `beta.19` 保持一致）

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

## 目录结构

```
.
├── build/                  # Wails 构建配置、图标、Windows 清单
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
│   ├── notify/             # Windows 系统通知
│   ├── runner/             # 任务队列与演示执行器
│   ├── scheduler/          # 定时执行
│   ├── tray/               # 系统托盘
│   └── version/
├── main.go
└── Taskfile.yml
```

## 说明

- 自动战斗、小工具、成就、远程控制、第三方推送等暂未实现。
- 自动更新暂未实现。
- 接入 MaaCore 后，`internal/runner` 的演示执行器可以替换为真实调用，前端事件契约保持不变。
