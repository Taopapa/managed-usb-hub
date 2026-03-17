# Managed USB Hub - Linux 环境开发与打包指南

本文档详细介绍了如何在 Linux 环境（以 Ubuntu 为例）中配置开发环境、运行项目以及构建分发包（.deb 和 .AppImage）。

## 1. 系统环境准备

### 1.1 基础工具安装
打开终端，运行以下命令安装 Go 语言环境和基础构建工具：

```bash
# 更新软件包列表
sudo apt update

# 安装构建工具和依赖库
sudo apt install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev npm

# 安装 Go (如果系统源版本过低，建议去官网下载最新版)
# 检查当前版本
go version
# 确保版本 >= 1.21
```

### 1.2 安装 Wails CLI
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 将 Go 二进制目录添加到 PATH (如果尚未添加)
export PATH=$PATH:$(go env GOPATH)/bin
# 为了持久生效，请将上面这行添加到 ~/.bashrc 或 ~/.zshrc 中
```

### 1.3 串口权限设置 (重要)
Linux 下访问 USB 串口设备需要特定权限。
```bash
# 将当前用户加入 dialout 组
sudo usermod -a -G dialout $USER

# 注意：设置完成后，必须注销并重新登录，或者重启系统才能生效！
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
*   如果遇到 `webkit2gtk` 相关错误，请检查是否安装了 `libwebkit2gtk-4.1-dev`。

---

## 3. 构建发布版本

### 3.1 编译二进制文件
构建生产环境的二进制文件：
```bash
# 使用 webkit2_41 标签（适配 Ubuntu 24.04+）
wails build -platform linux/amd64 -tags webkit2_41
```
构建产物位于：`build/bin/managed-usb-hub-wails`

### 3.2 构建 CLI 工具
```bash
go build -o build/bin/hub-cli ./cmd/hub-cli
```

---

## 4. 打包分发 (生成安装包)

Linux 下常用的分发格式有 `.deb` (Debian/Ubuntu) 和 `.AppImage` (通用)。

### 4.1 方案 A：生成 .deb 安装包 (推荐)
我们需要使用 `nfpm` 工具来创建 .deb 包。

#### 步骤 1: 安装 nfpm
```bash
go install -u github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
```

#### 步骤 2: 创建配置文件 `nfpm.yaml`
在项目根目录下创建一个名为 `nfpm.yaml` 的文件，内容如下：

```yaml
name: "managed-usb-hub"
arch: "amd64"
platform: "linux"
version: "1.0.4"
section: "default"
priority: "extra"
replaces:
- managed-usb-hub
provides:
- managed-usb-hub
maintainer: "Your Name <you@example.com>"
description: "Managed USB Hub Control Software"
vendor: "Your Company"
homepage: "https://github.com/Taopapa/managed-usb-hub"
license: "MIT"
contents:
- src: "build/bin/managed-usb-hub-wails"
  dst: "/usr/bin/managed-usb-hub"
- src: "build/bin/hub-cli"
  dst: "/usr/bin/hub-cli"
- src: "build/appicon.png"
  dst: "/usr/share/icons/hicolor/512x512/apps/managed-usb-hub.png"
- src: "linux/managed-usb-hub.desktop"
  dst: "/usr/share/applications/managed-usb-hub.desktop"
```

#### 步骤 3: 创建桌面快捷方式文件
在项目根目录创建 `linux` 文件夹，并在其中创建 `managed-usb-hub.desktop` 文件：
```ini
[Desktop Entry]
Type=Application
Name=Managed USB Hub
Comment=Control your Managed USB Hub
Exec=/usr/bin/managed-usb-hub
Icon=managed-usb-hub
Terminal=false
Categories=Utility;HardwareSettings;
```

#### 步骤 4: 生成 .deb 包
```bash
# 确保先编译了二进制文件
wails build -platform linux/amd64 -tags webkit2_41
go build -o build/bin/hub-cli ./cmd/hub-cli

# 运行 nfpm 打包
nfpm package --packager deb --target "build/bin/managed-usb-hub_1.0.4_amd64.deb"
```
生成的 `.deb` 文件可以直接通过 `sudo dpkg -i ...` 安装。

### 4.2 方案 B：生成 AppImage (通用)
AppImage 不需要安装，赋予执行权限即可运行。

#### 步骤 1: 安装 linuxdeploy
```bash
# 下载 linuxdeploy
wget https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage
chmod +x linuxdeploy-x86_64.AppImage
sudo mv linuxdeploy-x86_64.AppImage /usr/local/bin/linuxdeploy
```

#### 步骤 2: 准备 AppDir 结构
```bash
mkdir -p AppDir/usr/bin
mkdir -p AppDir/usr/share/applications
mkdir -p AppDir/usr/share/icons/hicolor/256x256/apps/

# 复制二进制文件
cp build/bin/managed-usb-hub-wails AppDir/usr/bin/managed-usb-hub
cp build/bin/hub-cli AppDir/usr/bin/hub-cli

# 复制图标和桌面文件
cp build/appicon.png AppDir/usr/share/icons/hicolor/256x256/apps/managed-usb-hub.png
cp linux/managed-usb-hub.desktop AppDir/usr/share/applications/
```

#### 步骤 3: 生成 AppImage
```bash
# 设置环境变量指向主程序
export VERSION=1.0.4
linuxdeploy --appdir AppDir --output appimage --icon-file build/appicon.png --desktop-file linux/managed-usb-hub.desktop --executable AppDir/usr/bin/managed-usb-hub
```
生成的 `Managed_USB_Hub-1.0.4-x86_64.AppImage` 文件即为最终产物。

---

## 5. 常见问题排查

*   **无法打开串口**：
    *   检查是否已将用户加入 `dialout` 组。
    *   检查 `/dev/ttyUSB*` 或 `/dev/ttyACM*` 文件是否存在。
*   **Webview 空白或报错**：
    *   确保系统已安装 `libwebkit2gtk-4.1-dev`。
    *   尝试设置环境变量 `WEBKIT_DISABLE_COMPOSITING_MODE=1` 运行。
