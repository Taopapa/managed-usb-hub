#!/bin/bash

echo "==============================================="
echo "   C2G USB Hub Manager - Linux Installer"
echo "==============================================="

# 确保以 root 权限运行
if [ "$EUID" -ne 0 ]; then
  echo "[!] Please run this script with sudo:"
  echo "    sudo bash install.sh"
  exit 1
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"

# 1. 赋予执行权限
echo "[*] Setting executable permissions..."
chmod +x "C2G USB Hub Manager" 2>/dev/null || true
chmod +x "muhcli" 2>/dev/null || true

# 2. 安装二进制文件到全局目录
echo "[*] Installing binaries to /usr/local/bin..."
cp "C2G USB Hub Manager" /usr/local/bin/c2g-usb-hub-manager 2>/dev/null || true
cp "muhcli" /usr/local/bin/muhcli 2>/dev/null || true

# 尝试拷贝 README (如果包里有的话)
if [ -f "README.txt" ]; then
  mkdir -p /usr/local/share/doc/c2g-usb-hub-manager 2>/dev/null || true
  cp "README.txt" /usr/local/share/doc/c2g-usb-hub-manager/ 2>/dev/null || true
  echo "[*] Copied README.txt to /usr/local/share/doc/c2g-usb-hub-manager/"
fi

# 3. 创建桌面快捷方式 (仅当存在 GUI 文件时)
if [ -f "C2G USB Hub Manager" ]; then
  echo "[*] Creating Desktop entry..."
  
  # 尝试拷贝图标 (如果包里有的话)
  if [ -f "appicon.png" ]; then
    cp "appicon.png" /usr/share/pixmaps/c2g-usb-hub-manager.png 2>/dev/null || true
    ICON_LINE="Icon=c2g-usb-hub-manager"
  else
    ICON_LINE="Icon=utilities-terminal"
  fi

  cat <<EOF > /usr/share/applications/c2g-usb-hub-manager.desktop
[Desktop Entry]
Name=C2G USB Hub Manager
Comment=Manage your C2G USB Hubs easily
Exec=/usr/local/bin/c2g-usb-hub-manager
$ICON_LINE
Terminal=false
Type=Application
Categories=Utility;HardwareSettings;
EOF
fi

# 4. 解决串口 (Serial Port) 访问权限问题
# Linux 默认普通用户没有读取 /dev/ttyUSB* 的权限，这是很多串口软件报错的元凶
echo "[*] Configuring USB Serial Port permissions (udev rules)..."
cat <<EOF > /etc/udev/rules.d/99-usb-serial-c2g.rules
# Allow all users to read/write USB serial devices
KERNEL=="ttyUSB[0-9]*", MODE="0666"
KERNEL=="ttyACM[0-9]*", MODE="0666"
EOF

# 重新加载 udev 规则
udevadm control --reload-rules
udevadm trigger

echo "==============================================="
echo "   Installation Completed Successfully!"
echo "==============================================="
echo "- You can now run the app from your application launcher."
echo "- Or type 'c2g-usb-hub-manager' / 'muhcli' in the terminal."
echo "- Note: If your USB Hub is currently plugged in, please UNPLUG and RE-PLUG it to apply the serial port permissions."
echo "==============================================="
