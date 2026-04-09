# C2G USB Hub Manager - Windows 环境开发与打包指南

本文档详细介绍了如何在 Windows 环境中配置开发环境、运行项目以及构建包含 GUI 和 CLI 的 Setup 安装包。

## 1. 系统环境准备

### 1.1 安装 Go 语言环境
1.  下载并安装 Go (建议 1.21 或更高版本): [https://go.dev/dl/](https://go.dev/dl/)
2.  验证安装：打开 PowerShell 运行 `go version`。

### 1.2 安装 Node.js
1.  下载并安装 Node.js (LTS 版本): [https://nodejs.org/](https://nodejs.org/)
2.  验证安装：运行 `node -v` 和 `npm -v`。

### 1.3 安装 NSIS (用于生成安装包)
Wails 使用 NSIS 来生成 Windows 安装程序。
1.  下载 NSIS: [https://nsis.sourceforge.io/Download](https://nsis.sourceforge.io/Download)
2.  安装时保持默认选项即可。
3.  **重要**：确保 NSIS 安装路径添加到系统环境变量 PATH 中（通常安装程序会自动处理，如果没有，请手动添加，例如 `C:\Program Files (x86)\NSIS`）。

### 1.4 安装 Wails CLI
```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

---

## 2. 项目代码获取与运行

### 2.1 克隆代码
```powershell
git clone https://github.com/Taopapa/managed-usb-hub.git
cd managed-usb-hub
```

### 2.2 开发模式运行
在开发模式下，Wails 会启动一个本地服务器，支持热重载。
```powershell
wails dev
```

---

## 3. 构建 Setup 安装包

本项目配置了自定义的 NSIS 脚本 (`build/windows/installer/project.nsi`)，它会将 GUI 主程序和 CLI 工具一同打包。

### 步骤 1: 构建 CLI 工具
首先，我们需要编译 CLI 工具并将其放置在 `build/bin` 目录下，以便 NSIS 打包时能找到它。

```powershell
# 确保 build/bin 目录存在
mkdir build/bin -ErrorAction SilentlyContinue

# 编译 CLI 工具
go build -o "build/bin/muhcli.exe" ./cmd/muhcli
```

### 步骤 2: 构建 GUI 并生成安装包
使用 Wails 的 `-nsis` 参数来构建 GUI 并生成安装程序。

```powershell
wails build -platform windows/amd64 -nsis
```

### 步骤 3: 获取安装包
构建完成后，安装包将生成在 `build/bin` 目录下。

*   **安装包文件**: `build/bin/C2G USB Hub Manager-amd64-installer.exe`
*   **主程序文件**: `build/bin/C2G USB Hub Manager.exe`
*   **CLI 工具**: `build/bin/muhcli.exe`

---

## 4. 安装包自定义说明

NSIS 脚本位于 `build/windows/installer/project.nsi`。如果您需要修改安装逻辑（例如修改安装目录、添加注册表项等），请编辑该文件。

目前该脚本已配置为：
1.  安装 `C2G USB Hub Manager.exe` (GUI)。
2.  安装 `muhcli.exe` (CLI)。
3.  创建桌面快捷方式和开始菜单快捷方式。
4.  提供卸载程序。

---

## 5. 常见问题排查

*   **NSIS 错误**: 如果提示找不到 `makensis`，请检查 NSIS 是否安装正确，并将其路径添加到系统 PATH 环境变量中。
*   **缺少文件**: 如果安装后发现缺少 `muhcli.exe`，请确保在运行 `wails build` 之前先执行了**步骤 1** 中的 CLI 构建命令。
