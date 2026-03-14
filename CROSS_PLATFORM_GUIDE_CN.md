# Managed USB Hub - 跨平台开发与部署指南

本指南详细说明了如何扩展 Managed USB Hub 应用程序（Wails GUI + CLI），使其除了现有的 Windows 支持外，还能兼容 Linux 和 macOS 系统。

## 1. 环境准备

### 通用要求
*   **Go**: v1.21+ (确保 `go` 命令在 PATH 环境变量中)
*   **Node.js**: v16+ 及 **npm**
*   **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Linux (Ubuntu/Debian)
*   安装系统依赖库：
    ```bash
    sudo apt update
    sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev build-essential pkg-config
    ```
*   **串口权限设置**：
    当前用户必须加入 `dialout` 用户组，才能在不使用 sudo 的情况下访问串口。
    ```bash
    sudo usermod -a -G dialout $USER
    # 设置完成后需要注销并重新登录才能生效
    ```

### macOS
*   **Xcode 命令行工具**：
    ```bash
    xcode-select --install
    ```
*   **Apple Silicon (M1/M2)**: Go 语言支持交叉编译，但强烈建议在真机上进行测试以确保兼容性。

## 2. 代码兼容性注意事项

### 串口处理 (`pkg/hubmanager` & `pkg/hubcli`)
本项目使用的 `go.bug.st/serial` 库本身支持 Windows, Linux 和 macOS。主要差异在于设备路径的命名规则：

*   **Windows**: `COM1`, `COM2`, ...
*   **Linux**: `/dev/ttyUSB0`, `/dev/ttyACM0`, ...
*   **macOS**: `/dev/tty.usbserial-...`, `/dev/cu.usbmodem...`

**需关注项：**
1.  **路径枚举**：`serial.GetPortsList()` 是跨平台的，但在 Linux/Mac 上过滤端口的逻辑可能需要调整（如果之前的逻辑依赖于 Windows 特有的驱动名称）。目前的逻辑主要通过发送 `?Q` 命令查询设备 ID，这在所有平台上都是安全的。
2.  **CLI 参数解析**：
    *   CLI 目前可以处理 `COM6` 这样的路径。
    *   在 Linux/Mac 上，路径通常是 `/dev/ttyUSB0`。
    *   当前的参数解析逻辑（`cmd/hub-cli/main.go`）支持 `/` 和 `-` 前缀。
    *   使用 `-S:/dev/ttyUSB0` 是可行的，因为代码是按第一个冒号 `:` 分割命令和参数的。

### GUI (前端)
*   **窗口控制**：如果在 Wails 中使用了自定义标题栏，在 macOS 上表现可能会有所不同（例如红绿灯按钮的位置）。Wails 默认的原生窗口模式通常兼容性很好。
*   **系统托盘**：如果实现了托盘图标，请在不同的 Linux 桌面环境（GNOME, KDE）下测试图标的可见性。

## 3. 构建步骤

### 构建 GUI 应用
在目标操作系统上运行以下命令：
```bash
wails build
```
*   **Linux 输出**: `build/bin/managed-usb-hub-wails`
*   **macOS 输出**: `build/bin/Managed USB Hub.app` (应用程序包)

### 构建 CLI 工具
```bash
# Linux 环境
go build -o build/bin/hub-cli ./cmd/hub-cli

# macOS 环境
go build -o build/bin/hub-cli ./cmd/hub-cli
```

### 交叉编译 (从 Windows 构建)
您可以从 Windows 编译其他平台的二进制文件，但**无法**构建完整的 Wails GUI 应用包（因为涉及到 CGO 和 WebView 依赖）。**强烈建议在目标系统上构建 GUI，或使用 Docker/CI。**

**仅 CLI 工具 (支持交叉编译):**
```bash
# 编译 Linux AMD64 版本
$Env:GOOS = "linux"; $Env:GOARCH = "amd64"; go build -o build/bin/linux/hub-cli ./cmd/hub-cli

# 编译 macOS ARM64 (M1/M2) 版本
$Env:GOOS = "darwin"; $Env:GOARCH = "arm64"; go build -o build/bin/mac/hub-cli ./cmd/hub-cli
```

## 4. 打包与分发

### Linux
*   **Debian 包 (.deb)**: 使用 `wails build -platform linux/amd64` (需要安装 `nfpm` 工具)。
*   **AppImage / Flatpak**: 推荐用于更广泛的发行版兼容性。
*   **CLI 安装**: 通常只需将二进制文件复制到 `/usr/local/bin/` 即可。

### macOS
*   **DMG / App Bundle**: Wails 会生成 `.app` 包。您可以使用 `create-dmg` 工具将其封装为 `.dmg` 镜像文件。
*   **签名与公证 (Notarization)**: macOS 要求应用程序必须经过签名和公证才能在其他电脑上无警告运行。这需要 Apple 开发者账号。
    ```bash
    gon -log-level=info gon.json # 使用 gon 工具进行签名
    ```

## 5. 测试清单

| 功能点 | Windows | Linux | macOS |
| :--- | :---: | :---: | :---: |
| **GUI 启动** | ✅ | [ ] | [ ] |
| **设备自动搜索** | ✅ | [ ] | [ ] |
| **端口开关控制 (GUI)** | ✅ | [ ] | [ ] |
| **CLI 查询 (`-Q`)** | ✅ | [ ] | [ ] |
| **CLI 设置状态 (`-S`)** | ✅ | [ ] | [ ] |
| **串口权限** | N/A | 检查 `dialout` 组 | 检查隐私权限 |
| **热插拔检测** | ✅ | [ ] | [ ] |

## 6. 已知问题与注意事项
1.  **换行符**: 串口通信目前处理的是 `\r\n`。请确保硬件设备在所有平台上的响应格式一致。
2.  **路径分隔符**: 在 Go 代码中操作文件路径时，请务必使用 `filepath.Join`，避免硬编码 `\` 或 `/`。
3.  **权限问题**: Linux 用户经常忘记将自己添加到 `dialout` 组，这会导致“拒绝访问 (Access Denied)”错误。

## 7. CI/CD 自动化构建 (推荐)
建议使用 GitHub Actions 自动构建多平台版本。

```yaml
name: Build
on: [push]
jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - uses: dAppWiz/wails-action@v1
        with:
          build-name: ManagedUSBHub
          build-platform: ${{ matrix.os }}
```
