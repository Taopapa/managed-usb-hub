# C2G USB Hub Manager - macOS 环境开发与打包指南

本文档详细介绍了如何在 macOS 环境中配置开发环境、运行项目以及构建分发包（.app, .dmg 和 .pkg）。

## 1. 系统环境准备

### 1.1 安装 Xcode Command Line Tools
首先，确保已安装 Xcode 命令行工具，这是开发的基础。
```bash
xcode-select --install
```

### 1.2 安装 Homebrew (可选，推荐)
Homebrew 是 macOS 上最常用的包管理器，方便安装 Go 和其他依赖。
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 1.3 安装 Go 语言环境
```bash
brew install go
# 或者从官网下载安装包

# 验证安装
go version
# 确保版本 >= 1.21
```

### 1.4 安装 Node.js
Wails 前端构建需要 Node.js。
```bash
brew install node
# 验证安装
node -v
npm -v
```

### 1.5 安装 Wails CLI
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 将 Go 二进制目录添加到 PATH (如果尚未添加)
export PATH=$PATH:$(go env GOPATH)/bin
# 为了持久生效，请将上面这行添加到 ~/.zshrc 或 ~/.bash_profile 中
```

---

## 2. 项目代码获取与运行

### 2.1 克隆代码
```bash
git clone https://github.com/Taopapa/managed-usb-hub.git
cd managed-usb-hub
```

### 2.2 开发模式运行
在开发模式下，Wails 会启动一个本地服务器，支持热重载。
```bash
wails dev
```
*   如果遇到权限问题，可能需要检查终端是否已获得访问 USB 设备的权限（通常不需要额外授权，除非使用了受限 API）。

---

## 3. 构建发布版本

### 3.1 编译 GUI 和 CLI
在根目录下运行：
```bash
# 编译 GUI (将生成 build/bin/C2G USB Hub Manager.app)
wails build -platform darwin/universal

# 编译 CLI (确保与您的应用名对应，如果需要 CLI 的话)
go build -o build/bin/muhcli ./cmd/muhcli
```

---

## 4. 打包分发 (生成安装包)

macOS 下常用的分发格式有 `.dmg` (磁盘镜像) 和 `.pkg` (安装包)。

### 4.1 方案 A：生成 DMG 镜像 (推荐，最常见)
DMG 是 macOS 上分发软件的标准方式，用户只需拖拽即可安装。

#### 步骤 1: 安装 create-dmg
我们需要使用 `create-dmg` 工具来创建漂亮的 DMG 镜像。
```bash
brew install create-dmg
```

#### 步骤 2: 准备应用图标和背景图 (可选)
如果需要自定义 DMG 打开后的背景图，可以准备一张图片。

#### 步骤 3: 运行打包命令
```bash
# 确保先编译了 .app
wails build -platform darwin/universal

# 创建 DMG
create-dmg \
  --volname "Managed USB Hub Installer" \
  --window-pos 200 120 \
  --window-size 800 400 \
  --icon-size 100 \
  --icon "Managed USB Hub.app" 200 190 \
  --hide-extension "Managed USB Hub.app" \
  --app-drop-link 600 185 \
  "Managed-USB-Hub-Installer.dmg" \
  "build/bin/Managed USB Hub.app"
```
生成的 `.dmg` 文件即可直接分发给用户。

### 4.2 方案 B：生成 PKG 安装包 (适合企业部署)
PKG 类似于 Windows 的安装程序，有安装向导。

#### 步骤 1: 使用 pkgbuild
macOS 自带 `pkgbuild` 工具，无需额外安装。

```bash
pkgbuild --root "build/bin/C2G USB Hub Manager.app" \
         --identifier "com.yourcompany.c2g-usb-hub-manager" \
         --version "1.0.4" \
         --install-location "/Applications/C2G USB Hub Manager.app" \
         "C2G-USB-Hub-Manager-Installer.pkg"
```

#### 步骤 2: 签名 (重要)
如果您希望用户在双击安装时不看到“未识别的开发者”警告，必须对 PKG 进行签名。这需要 Apple Developer ID。
```bash
productsign --sign "Developer ID Installer: Your Name (XXXXXXXXXX)" \
            "C2G-USB-Hub-Manager-Installer.pkg" \
            "C2G-USB-Hub-Manager-Installer-Signed.pkg"
```

---

## 5. 常见问题排查

*   **无法打开应用 ("App is damaged")**：
    *   这是因为应用未签名或未经过公证 (Notarization)。
    *   临时解决方法：在终端运行 `xattr -cr /Applications/C2G\ USB\ Hub\ Manager.app` 清除隔离属性。
    *   正式发布解决方法：使用 `gon` 工具对应用进行公证。
*   **串口无法识别**：
    *   macOS 可能会将串口设备挂载为 `/dev/cu.usbserial-xxxx` 或 `/dev/tty.usbserial-xxxx`。
    *   如果使用 `muhcli`，请尝试使用 `ls /dev/cu.*` 查看设备列表。
