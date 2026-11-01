# MaaGO

MaaGO 是一个使用 **Go + Wails v3 + React** 编写的 MAA（MaaAssistantArknights）桌面客户端。

当前仓库只包含基础外壳，功能页面会在确认保留范围后逐步接入。

## 已包含

- Wails v3 无边框窗口、自绘标题栏与窗口控制
- 侧边栏 + 顶栏导航（圆角联体背景）
- 浅色 / 深色 / 跟随系统主题
- 简体中文 / English 多语言
- 配置文件持久化（`%APPDATA%\MaaGO\config.json`）
- 启动日志（`%APPDATA%\MaaGO\logs\startup.log`）
- 前端构建、Go 构建、Wails 开发模式的任务脚本

## 开发

要求：

- Windows 10/11
- Go 1.25+
- Node.js 20+
- Wails v3 CLI（与本项目 `go.mod` 中的版本保持一致）

```bash
go mod tidy
cd frontend && npm install && cd ..
wails3 task build
wails3 task dev
```

## 目录结构

```
.
├── build/                  # Wails 构建配置、图标、Windows 清单
├── frontend/               # React + Vite + HeroUI + Tailwind
│   └── src/
│       ├── assets/         # 字体与语言包
│       ├── components/     # 外壳组件（导航、标题栏、页面容器）
│       ├── constants/      # 路由与布局常量
│       ├── icons/
│       └── pages/          # 页面（当前为占位页）
├── internal/
│   ├── app/                # 暴露给前端的 Wails 服务
│   ├── apppath/            # 统一路径管理
│   ├── config/             # 配置读写
│   └── version/            # 构建版本信息
├── main.go                 # Wails 应用入口
└── Taskfile.yml
```
