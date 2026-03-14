# Managed USB Hub - Cross-Platform Development & Deployment Guide

This guide outlines the steps and considerations for extending the Managed USB Hub application (Wails GUI + CLI) to support Linux and macOS, in addition to the existing Windows support.

## 1. Prerequisites

### Common
*   **Go**: v1.21+ (Ensure `go` is in your PATH)
*   **Node.js**: v16+ & **npm**
*   **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Linux (Ubuntu/Debian)
*   Install system dependencies:
    ```bash
    sudo apt update
    sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev build-essential pkg-config
    ```
*   **Serial Port Permissions**:
    Current user must be in the `dialout` group to access serial ports without sudo.
    ```bash
    sudo usermod -a -G dialout $USER
    # Logout and login required to take effect
    ```

### macOS
*   **Xcode Command Line Tools**:
    ```bash
    xcode-select --install
    ```
*   **Apple Silicon (M1/M2)**: Go handles cross-compilation, but testing on actual hardware is recommended.

## 2. Code Compatibility

### Serial Port Handling (`pkg/hubmanager` & `pkg/hubcli`)
The project uses `go.bug.st/serial`, which supports Windows, Linux, and macOS. However, device path naming differs:

*   **Windows**: `COM1`, `COM2`, ...
*   **Linux**: `/dev/ttyUSB0`, `/dev/ttyACM0`, ...
*   **macOS**: `/dev/tty.usbserial-...`, `/dev/cu.usbmodem...`

**Action Items:**
1.  **Path Enumeration**: `serial.GetPortsList()` works cross-platform, but you may need to filter relevant ports differently on Linux/Mac if the "CENTOS" string logic relies on Windows-specific driver behavior (though usually, the ID check is done via serial command `?Q`, which is safe).
2.  **CLI Argument Parsing**: The CLI currently handles paths like `COM6` or `/dev/ttyUSB0`. The `/S:COM6` syntax works, but on Linux/Mac, users might prefer standard flags `-p /dev/ttyUSB0`. The current CLI parser supports `/` and `-` prefixes, so `-S:/dev/ttyUSB0` should theoretically work, but `/` inside the path (e.g., `/dev/...`) might conflict with the command parser if not careful.
    *   *Current Logic Check*: `cmd/hub-cli/main.go` splits by `:`, so `-S:/dev/ttyUSB0` might be parsed as Command=`-S`, Port=`/dev/ttyUSB0`. This should work.

### GUI (Frontend)
*   **Window Controls**: Custom title bars (if used) behave differently on macOS. Wails handles native windows well.
*   **System Tray**: If implemented, verify icon visibility on different Linux Desktop Environments (GNOME, KDE).

## 3. Building

### Build GUI
Run the following command on the target OS:
```bash
wails build
```
*   **Linux Output**: `build/bin/managed-usb-hub-wails`
*   **macOS Output**: `build/bin/Managed USB Hub.app` (Application Bundle)

### Build CLI
```bash
# Linux
go build -o build/bin/hub-cli ./cmd/hub-cli

# macOS
go build -o build/bin/hub-cli ./cmd/hub-cli
```

### Cross-Compilation (from Windows)
You can build binaries for other platforms from Windows, but you cannot build the full Wails GUI app bundle (due to CGO/WebView dependencies). **It is highly recommended to build on the target OS or use Docker/CI.**

**CLI Only (Cross-compile possible):**
```bash
# Build for Linux AMD64
$Env:GOOS = "linux"; $Env:GOARCH = "amd64"; go build -o build/bin/linux/hub-cli ./cmd/hub-cli

# Build for macOS ARM64 (M1/M2)
$Env:GOOS = "darwin"; $Env:GOARCH = "arm64"; go build -o build/bin/mac/hub-cli ./cmd/hub-cli
```

## 4. Packaging & Distribution

### Linux
*   **Debian Package (.deb)**: Use `wails build -platform linux/amd64` (requires `nfpm`).
*   **AppImage / Flatpak**: Alternative formats for broader compatibility.
*   **CLI Installation**: Usually copied to `/usr/local/bin/`.

### macOS
*   **DMG / App Bundle**: Wails generates `.app`. You can use `create-dmg` tool to wrap it.
*   **Signing & Notarization**: macOS requires apps to be signed and notarized to run without warnings. This requires an Apple Developer Account.
    ```bash
    gon -log-level=info gon.json # Tool for signing
    ```

## 5. Testing Checklist

| Feature | Windows | Linux | macOS |
| :--- | :---: | :---: | :---: |
| **GUI Launch** | ✅ | [ ] | [ ] |
| **Device Auto-Search** | ✅ | [ ] | [ ] |
| **Port Toggle (GUI)** | ✅ | [ ] | [ ] |
| **CLI Query (`-Q`)** | ✅ | [ ] | [ ] |
| **CLI Set State (`-S`)** | ✅ | [ ] | [ ] |
| **Serial Permission** | N/A | Check `dialout` | Check Privacy |
| **Hot Plug Detection** | ✅ | [ ] | [ ] |

## 6. Known Issues / Considerations
1.  **Line Endings**: Serial communication currently handles `\r\n`. Ensure devices behave consistently across platforms.
2.  **Path Separators**: Use `filepath.Join` in Go code instead of hardcoded `\` or `/`.
3.  **Permissions**: Linux users often forget to add themselves to `dialout` group, leading to "Access Denied" errors.

## 7. CI/CD Pipeline (Recommended)
Use GitHub Actions to build for all platforms automatically.

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
