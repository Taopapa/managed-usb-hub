C2G 7-port Managed USB HUB - Linux Installation & Usage Guide

1. INSTALLATION
   Ensure you have all the required files (install.sh, 99-c2ghub.rules, icon.png, and the managed-usb-hub-wails binary) in the same directory.
   Open a terminal in this directory and run the installation script with root privileges:
   
   sudo ./install.sh
   
   IMPORTANT: After the script completes, you MUST log out and log back in (or restart your computer) for the serial port permissions to take effect.

2. USAGE
   - GUI Mode:
     After logging back in, you can launch the application directly from your Linux desktop environment's application menu. Simply search for "C2G USB Hub Manager" and click to open. You do not need to use 'sudo' anymore.

   - CLI Mode:
     You can use the command-line interface by running the executable directly from the installation directory:
     
     /opt/c2g-usb-hub/managed-usb-hub-wails /?
     
     This will display the help menu with all available commands.