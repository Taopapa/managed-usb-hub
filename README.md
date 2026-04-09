# C2G USB Hub Manager

C2G USB Hub Manager 是一个基于 [Wails](https://wails.io/) 框架开发的跨平台桌面应用程序及命令行工具，专门用于管理和配置 C2G 品牌及其兼容型号的智能 USB Hub（控制端口开关、重置、修改设备名称及密码等）。

## 目录
- [环境搭建](#环境搭建)
- [本地开发与运行](#本地开发与运行)
- [编译与构建](#编译与构建)
- [Linux 专属权限问题](#linux-专属权限问题)

---

## 环境搭建

由于本项目基于 Wails (Go + Vue/JS) 开发，你需要先在本地安装相关开发环境：

### 1. 安装 Go
- 下载并安装 [Go 1.20 或以上版本](https://go.dev/dl/)。
- 确保 `go` 命令已加入系统环境变量 (`PATH`)。

### 2. 安装 Node.js
- 下载并安装 [Node.js 18 或以上版本](https://nodejs.org/) (推荐安装 LTS 版本)。
- 安装完成后，确保 `npm` (或 `yarn`/`pnpm`) 命令可以正常运行。

### 3. 安装 Wails CLI
打开终端，运行以下命令全局安装 Wails 命令行工具：
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```
> **注意**：如果安装后提示找不到 `wails` 命令，请确保你的 `GOPATH/bin` (如 `%USERPROFILE%\go\bin` 或 `~/go/bin`) 已经添加到了系统环境变量中。你可以运行 `wails doctor` 检查环境是否全部就绪。

### 4. Linux 专属环境依赖
如果你使用的是 Linux (如 Ubuntu/Debian)，还需要安装构建 WebKit 视窗相关的依赖库：
```bash
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev build-essential pkg-config
```

---

## 本地开发与运行

### 运行开发服务器 (Live Reload)
Wails 提供了非常方便的开发者模式。在该模式下，当你修改前端 Vue 代码或后端 Go 代码时，程序会自动热重载（Hot Reload），无需手动重启。

在项目根目录下运行：
```bash
wails dev
```
首次运行会自动安装前端的 NPM 依赖并启动 Vue 的开发服务器。

---

## 编译与构建

你可以将项目编译为适用于不同操作系统的独立可执行文件（GUI 桌面端和 CLI 命令行端）。

### 1. Windows 下一键构建
项目中提供了一个专为 Windows 设计的打包脚本。双击运行或在命令行执行：
```cmd
package-windows.bat
```
该脚本会自动：
- 安装 Wails CLI (若缺失)
- 编译生成 `muhcli.exe` 命令行工具
- 编译生成 `C2G USB Hub Manager.exe` 图形化界面程序
- 将产物收集到 `build/bin/` 目录下

### 2. 手动构建 GUI 桌面端
如果你需要在当前系统下构建桌面端应用：
```bash
wails build
```
*(这将在 `build/bin/` 目录下生成对应系统的可执行文件，Mac 下为 `.app`)*

### 3. 手动构建 CLI 命令行工具
CLI 工具 (`muhcli`) 是用纯 Go 编写的。在根目录运行：
```bash
# 生成到 build/bin/ 目录
go build -o build/bin/muhcli ./cmd/muhcli
```

### 4. 使用 GitHub Actions 云端构建
本项目配置了 GitHub Actions 流水线。当你在 Git 仓库中打上 `v` 开头的 Tag 并推送时（例如 `git tag v1.0.0 && git push origin v1.0.0`），云端流水线会自动编译：
- Linux AMD64 (GUI + CLI)
- Linux ARM64 (仅 CLI)
- Linux ARM32 (仅 CLI)

你可以在 GitHub 仓库的 **Releases** 页面下载这些打包好的 `.tar.gz` 压缩包。

---

## Linux 专属权限问题

Linux 系统出于安全原因，普通用户默认**没有执行下载程序的权限**，也没有**读写串口 (`/dev/ttyUSB*`) 的权限**。

如果你下载了针对 Linux 编译的压缩包，解压后请不要直接双击，而是运行里面附带的 `install.sh` 脚本：
```bash
sudo bash install.sh
```
该脚本会自动帮你：
1. 赋予 `C2G USB Hub Manager` 和 `muhcli` 可执行权限。
2. 将程序安装到全局命令路径 (`/usr/local/bin`)。
3. 创建桌面快捷方式，方便你在启动器中打开。
4. 自动配置 `udev` 规则，**一劳永逸地解决串口的读写权限（Access Denied）问题**。
