# win-webapp-gen

将 Web 应用打包为 Windows 原生桌面程序（`.exe`），基于 [WebView2](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) 渲染，开箱即用。

## 原理

项目由两个组件构成：

| 组件        | 路径           | 作用                                                              |
| ----------- | -------------- | ----------------------------------------------------------------- |
| **loader**  | `cmd/loader/`  | 运行时载体，负责读取配置、启动 WebView2 窗口并加载目标页面        |
| **creator** | `cmd/creator/` | 打包工具（主程序），将 loader 模板与应用配置拼接为一个独立 `.exe` |

打包流程：

```
loader.exe 模板 + PNG 图标 + JSON 配置 → 目标 .exe
```

生成的 `.exe` 末尾附加了一段 JSON 配置数据（带魔数标记），运行时由 loader 自行读取并解析。

## 构建

需要 **Go 1.26+** 和 Windows 环境。

```bash
# 使用 Task（推荐）
task

# 或手动构建
go build -trimpath -ldflags '-s -w -H=windowsgui' -o cmd/creator/win-webapp-loader.exe ./cmd/loader/
go build -trimpath -ldflags '-s -w' -o bin/win-webapp-gen.exe ./cmd/creator/
```

构建完成后在 `bin/` 目录下得到 `win-webapp-gen.exe`。

## 使用

```bash
win-webapp-gen.exe [flags]
```

### 参数

| 标志             | 简写 | 说明                                      | 示例                                                   |
| ---------------- | ---- | ----------------------------------------- | ------------------------------------------------------ |
| `--url`          | `-u` | 目标 Web 地址（必填）                     | `-u https://example.com`                               |
| `--name`         | `-n` | 应用名称（默认取 URL 主机名或目录名）     | `-n MyApp`                                             |
| `--width`        | `-w` | 窗口宽度                                  | `-w 1200`                                              |
| `--height`       | `-h` | 窗口高度                                  | `-h 800`                                               |
| `--icon`         |      | PNG 图标路径（自动转换为 ICO 并注入）     | `--icon logo.png`                                      |
| `--target-dir`   | `-d` | 输出目录                                  | `-d ./dist`                                            |
| `--proxy-server` | `-P` | SOCKS 代理（`ip:端口`）                   | `-P 127.0.0.1:1080`                                    |
| `--insecure`     | `-i` | 允许不安全内容（自签名证书等）            | `-i`                                                   |
| `--debug`        | `-D` | 调试模式（启用右键菜单、开发者工具）      | `-D`                                                   |
| `--backend`      | `-b` | 后端反向代理配置                          | `-b "route=/api,url=http://127.0.0.1:7993,strip=/api"` |
| `--portable`     | `-p` | 便携模式（数据存储在 exe 旁而非用户目录） | `-p`                                                   |

### 示例

**打包远程 Web 应用**

```bash
win-webapp-gen.exe --url https://dashboard.example.com --name Dashboard --width 1400 --height 900 --icon icon.png
```

**打包本地静态站点**

```bash
win-webapp-gen.exe --url file://D:/my-site/dist --name MySite --target-dir D:/output
```

**开发调试（带代理和后端转发）**

```bash
win-webapp-gen.exe --url http://localhost:5173 -D --backend "route=/api,url=http://127.0.0.1:8080,strip=/api" -p
```

## 运行时特性

- **本地文件托管**：`file://` 协议的 URL 会自动在本地起一个临时 HTTP 服务
- **后端反向代理**：`-b` 参数支持将指定路由转发到后端服务，格式 `route=路径,url=地址,strip=路径前缀`
- **便携模式**：启用后应用数据存储在 exe 同级目录，方便 U 盘携带
- **图标注入**：提供 PNG 图标时自动转换为多尺寸 ICO 并写入生成的 exe

## 系统要求

- Windows 10 / 11（需已安装 [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)）
