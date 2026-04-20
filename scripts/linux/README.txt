C2G 7-port Managed USB HUB - Linux Installation & Usage Guide

1. INSTALLATION
   Ensure you have all the required files (install.sh, appicon.png, and the binaries) in the same directory.
   Open a terminal in this directory and run the installation script with root privileges:
   
   sudo bash ./install.sh
   
   IMPORTANT: After the script completes, you MUST unplug and re-plug your USB Hub (or restart your computer) for the serial port permissions to take effect.

2. USAGE
   - GUI Mode:
     After installation, you can launch the application directly from your Linux desktop environment's application menu. Simply search for "C2G USB Hub Manager" and click to open. You do not need to use 'sudo' anymore.

   - CLI Mode:
     You can use the command-line interface directly from any terminal window:
     
     muhcli /?
     
     This will display the help menu with all available commands.