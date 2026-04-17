#!/bin/bash
# C2G 7-port Managed USB HUB - Linux Installation & Setup Script
# This script configures permissions (udev), adds the user to the dialout group,
# and creates a desktop shortcut for the GUI application.

set -e

# Colors for output
GREEN='\032[0;32m'
YELLOW='\032[1;33m'
RED='\032[0;31m'
NC='\032[0m'

echo -e "${GREEN}Starting C2G USB Hub Manager Setup...${NC}"

# Check if script is run as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run this script with sudo or as root.${NC}"
    echo -e "Usage: sudo ./install.sh"
    exit 1
fi

# Get the real user who ran sudo
REAL_USER=${SUDO_USER:-$USER}

# 1. Add user to dialout group (for serial port access)
echo -e "\n${YELLOW}[1/4] Configuring serial port permissions...${NC}"
if grep -q "^dialout:" /etc/group; then
    usermod -a -G dialout "$REAL_USER"
    echo -e "Added user '$REAL_USER' to the 'dialout' group."
else
    # Fallback for some Arch/Manjaro systems
    usermod -a -G uucp "$REAL_USER"
    echo -e "Added user '$REAL_USER' to the 'uucp' group."
fi

# 2. Install Udev Rules
echo -e "\n${YELLOW}[2/4] Installing udev rules...${NC}"
cp 99-c2ghub.rules /etc/udev/rules.d/
chmod 644 /etc/udev/rules.d/99-c2ghub.rules
udevadm control --reload-rules
udevadm trigger
echo -e "Udev rules installed. USB Hubs will now have correct read/write permissions."

# 3. Install Application and Desktop Shortcut (Optional, if GUI binary exists)
echo -e "\n${YELLOW}[3/4] Installing GUI application shortcut...${NC}"

# Define installation paths
INSTALL_DIR="/opt/c2g-usb-hub"
BIN_NAME="managed-usb-hub-wails" # Replace with actual GUI binary name if different

if [ -f "./$BIN_NAME" ]; then
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Copy binary
    cp "./$BIN_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BIN_NAME"

    # Copy README if exists
    if [ -f "./README.txt" ]; then
        cp "./README.txt" "$INSTALL_DIR/"
    fi
    
    # Copy icon if exists
    if [ -f "./icon.png" ]; then
        cp "./icon.png" "$INSTALL_DIR/"
        ICON_PATH="$INSTALL_DIR/icon.png"
    else
        ICON_PATH="utilities-terminal" # Fallback generic icon
    fi

    # Create .desktop file
    DESKTOP_FILE="/usr/share/applications/c2ghub.desktop"
    cat > "$DESKTOP_FILE" << EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=C2G USB Hub Manager
Comment=Manage and monitor C2G 7-port USB Hubs
Exec=$INSTALL_DIR/$BIN_NAME
Icon=$ICON_PATH
Terminal=false
Categories=Utility;HardwareSettings;
Keywords=usb;hub;manager;c2g;
EOF
    
    chmod 644 "$DESKTOP_FILE"
    echo -e "Application installed to $INSTALL_DIR"
    echo -e "Desktop shortcut created. You can now find 'C2G USB Hub Manager' in your application launcher."
else
    echo -e "GUI binary ('$BIN_NAME') not found in the current directory. Skipping desktop shortcut creation."
    echo -e "Note: To install the GUI, ensure the binary is in the same folder as this script and named '$BIN_NAME'."
fi

# 4. Final Instructions
echo -e "\n${GREEN}Setup Complete!${NC}"
echo -e "${YELLOW}IMPORTANT:${NC} For the serial port permissions to take effect, the user '$REAL_USER' must ${RED}log out and log back in${NC} (or restart the computer)."
echo -e "After logging back in, you will be able to run the C2G USB Hub Manager GUI without using 'sudo'."
echo -e "A README file has been installed to ${INSTALL_DIR}/README.txt with additional information."