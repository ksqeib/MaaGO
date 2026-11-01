# 代理与下载说明

MaaGO 已经内置更新器：

- “设置 → 更新”可以启用 HTTP 代理、测试代理、检查更新、下载核心、更新资源。
- 应用内代理只用于更新检查与下载。
- 命令行脚本也支持同样的代理参数，适合批量/离线维护。

## 1. 代理地址格式

### 应用内

“设置 → 更新 → 代理与下载”：

1. 打开“启用 HTTP 代理”
2. 填写 `http://127.0.0.1:7890`
3. 点“测试代理”
4. 再执行“检查更新 / 下载并安装核心 / 更新资源”

### 命令行脚本

常见本机代理端口：

| 客户端 | 常见 HTTP 代理 |
|---|---|
| Clash / Clash Verge | `http://127.0.0.1:7890` |
| v2rayN | `http://127.0.0.1:10809` |
| Mihomo / Clash Meta | `http://127.0.0.1:7890` |
| 其他 | 以软件显示的 HTTP 端口为准 |

MaaGO 的 PowerShell 脚本使用 `-Proxy`，支持 HTTP/HTTPS 代理：

```powershell
.\scripts\update-core.ps1 -Version 6.20.1 -Proxy "http://127.0.0.1:7890"
```

> PowerShell 的 `-Proxy` 不直接支持 SOCKS5。若只有 SOCKS5，请让代理软件同时开启 HTTP 端口，或使用支持 SOCKS5 的 Git 配置。

## 2. 环境变量

脚本在没有传 `-Proxy` 时会读取：

```powershell
$env:HTTPS_PROXY
$env:HTTP_PROXY
```

示例：

```powershell
$env:HTTPS_PROXY = "http://127.0.0.1:7890"
$env:HTTP_PROXY  = $env:HTTPS_PROXY
.\scripts\update-core.ps1 -Version 6.20.1
```

只在当前 PowerShell 窗口生效，不影响系统全局设置。

Git 使用：

```powershell
git config --global http.proxy "http://127.0.0.1:7890"
git config --global https.proxy "http://127.0.0.1:7890"

# 取消
git config --global --unset http.proxy
git config --global --unset https.proxy
```

如果 Git 使用 SOCKS5：

```powershell
git config --global http.proxy "socks5h://127.0.0.1:1080"
git config --global https.proxy "socks5h://127.0.0.1:1080"
```

## 3. 系统代理

Windows 设置 → 网络和 Internet → 代理，可以设置系统级 HTTP/HTTPS 代理。

注意：

- 不同工具对系统代理的遵循程度不同。
- PowerShell 脚本显式传 `-Proxy` 最可靠。
- MaaCore 自身的网络请求（如企鹅物流/一图流上传）当前没有在 MaaGO 中启用，不保证遵循 MaaGO 的任何设置。

## 4. 下载镜像

如果直连 GitHub 很慢，可以使用可信的 GitHub 镜像前缀：

```powershell
.\scripts\update-core.ps1 `
  -Version 6.20.1 `
  -Mirror "https://your-mirror.example/"
```

脚本会拼成：

```text
https://your-mirror.example/https://github.com/MaaAssistantArknights/...
```

资源更新：

```powershell
.\scripts\update-resource.ps1 -Mirror "https://your-mirror.example/"
```

安全提示：

- 只使用你信任的镜像。
- 镜像可能缓存旧版本或篡改文件；下载后确认核心版本和 `resource/version.json`。
- 不要为了省事使用来源不明的“整合包”。

## 5. 先验证代理是否可用

```powershell
Invoke-WebRequest "https://api.github.com" `
  -Proxy "http://127.0.0.1:7890" `
  -UseBasicParsing `
  -Headers @{ "User-Agent" = "MaaGO" }
```

能返回 JSON 就说明代理可用。

再验证下载域名：

```powershell
Invoke-WebRequest "https://github.com" -Proxy "http://127.0.0.1:7890" -UseBasicParsing
```

## 6. 常见代理问题

| 现象 | 处理 |
|---|---|
| `407 Proxy Authentication Required` | 代理需要认证；把用户名密码写进 URL，例如 `http://user:pass@127.0.0.1:7890` |
| `SSL/TLS` 错误 | 检查系统时间、代理证书、TLS 1.2；换官方源或可信镜像 |
| 连接超时 | 确认代理端口、软件是否运行、是否允许局域网/本机连接 |
| 镜像返回 404 | 镜像不缓存 Release 资产；换 `-Version` 或直接使用 GitHub |
| Git 仍走代理 | 检查 `git config --global --get http.proxy`；也可能是系统代理 |
| 下载到一半失败 | 重新执行脚本；脚本使用临时目录，不会破坏已有 `resource` 之外的文件 |

## 7. NO_PROXY

PowerShell 脚本没有实现复杂的 `NO_PROXY` 规则。需要绕过时，直接不传 `-Proxy`，或使用不走代理的网络环境。

Git 可以单独配置排除：

```powershell
git config --global http.https://github.com.proxy ""
```

## 8. 与 MaaCore 的关系

- MaaGO 通过 C API 加载 MaaCore，应用内代理只注入到更新器的 Go HTTP 客户端。
- 当前没有启用企鹅物流、一图流等上传功能，所以核心不需要代理。
- 如果以后启用核心网络功能，需要单独确认 MaaCore 是否读取系统代理/环境变量；不能假设它会自动使用更新器代理。

## 9. 安全建议

- 不要把代理账号密码写进仓库、截图或 issue。
- 不要在公开日志中保留完整代理 URL。
- 下载完成后检查：
  - `core/MaaCore.dll` 版本
  - `core/resource/version.json`
  - `core/resource/tasks/` 是否存在
- 重要更新前备份 `core/`。
