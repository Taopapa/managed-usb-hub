# Windows Packaging Guide for Managed USB Hub

This guide describes how to build the Managed USB Hub application (GUI & CLI) and create a Windows Installer (`Setup.exe`).

## Prerequisites

1.  **Go Programming Language**: Ensure Go 1.21+ is installed.
2.  **Wails Framework**: Ensure `wails` CLI is installed (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`).
3.  **NSIS (Nullsoft Scriptable Install System)**:
    *   Download from [https://nsis.sourceforge.io/Download](https://nsis.sourceforge.io/Download).
    *   Install it to the default location.

## Step 1: Build the Applications

You need to build both the GUI application and the CLI tool separately.

### 1. Build GUI Application
Open a terminal in the project root and run:
```powershell
wails build
```
This will create `build/bin/Managed USB Hub.exe`.

### 2. Build CLI Tool
In the same terminal, run:
```powershell
go build -o hub-cli.exe ./cmd/hub-cli
```
This will create `hub-cli.exe` in the project root.

## Step 2: Create the Installer

We use NSIS to package both executables into a single setup file.

### 1. Locate the Script
The installer script is located at: `installer/installer.nsi`

### 2. Compile the Script
**Method A: Right-Click (Recommended)**
1.  Navigate to the `installer` folder in File Explorer.
2.  Right-click on `installer.nsi`.
3.  Select **"Compile NSIS Script"**.

**Method B: Command Line**
Run the following command from the project root:
```powershell
"C:\Program Files (x86)\NSIS\makensis.exe" installer\installer.nsi
```

## Step 3: Verify

After compilation is complete, you will find the installer file in the **project root directory**:

**File Name**: `ManagedUSBHub_Setup.exe`

### Testing the Installer
1.  Run `ManagedUSBHub_Setup.exe`.
2.  Follow the installation wizard (License -> Location -> Install).
3.  Verify that shortcuts are created on the Desktop and Start Menu.
4.  Run the application to ensure it works.
5.  Check "Add or Remove Programs" to verify uninstallation works.

## Troubleshooting

*   **"File not found" error during NSIS compilation**:
    *   Ensure you ran **both** build commands in Step 1.
    *   Check if `build/bin/Managed USB Hub.exe` exists.
    *   Check if `hub-cli.exe` exists in the root.

*   **Application doesn't start after install**:
    *   Ensure the `build/windows/icon.ico` exists if referenced.
    *   Check antivirus logs (sometimes unsigned exes are flagged).
