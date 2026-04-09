package main

import (
	"bufio"
	"context"
	"fmt"
	"managed-usb-hub-wails/pkg/hubmanager"
	"managed-usb-hub-wails/pkg/logger"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	mu               sync.Mutex
	logDir           string
	hubManager       *hubmanager.HubManager
	sessionPasswords map[string]string
	passMu           sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{
		sessionPasswords: make(map[string]string),
	}
	// Use short-lived serial connections so the GUI does not hold the COM port
	// between commands and block the CLI or other tools from using it.
	a.hubManager = hubmanager.NewHubManager(false)

	return a
}

// OpenSystemTerminal opens a system terminal window in the application directory
func (a *App) OpenSystemTerminal() error {
	var cmd *exec.Cmd
	var args []string

	// Get current working directory or executable directory
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	if stdruntime.GOOS == "windows" {
		// On Windows, start cmd.exe
		// /c start cmd.exe opens a new window
		// Or just start cmd.exe directly
		// To open in current directory: start cmd.exe /k "cd /d <dir>"
		args = []string{"/c", "start", "cmd.exe", "/k", fmt.Sprintf("cd /d %s", dir)}
		cmd = exec.Command("cmd", args...)
	} else if stdruntime.GOOS == "darwin" {
		// On macOS, open Terminal
		cmd = exec.Command("open", "-a", "Terminal", dir)
	} else {
		// On Linux, try common terminals
		// This is tricky as there are many terminals.
		// Try x-terminal-emulator if available, or gnome-terminal, or konsole
		terminals := []string{"x-terminal-emulator", "gnome-terminal", "konsole", "xterm"}
		found := false
		for _, term := range terminals {
			if _, err := exec.LookPath(term); err == nil {
				cmd = exec.Command(term)
				cmd.Dir = dir
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no supported terminal emulator found")
		}
	}

	return cmd.Start()
}

// startup is called when the app starts. The context is saved
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.SetContext(ctx)

	// Initialize logging directory to be in the executable's directory
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		a.logDir = filepath.Join(exeDir, "logs")
		if err := os.MkdirAll(a.logDir, 0755); err != nil {
			fmt.Printf("Error creating log directory: %v\n", err)
		}
	} else {
		fmt.Printf("Error getting executable path: %v\n", err)
	}

	logger.InitLogger(a.logDir, false)
}

// shutdown is called when the app terminates
func (a *App) shutdown(ctx context.Context) {
	logger.Info("Application stopped", "System", "System")
}

// ReadLogs reads the current log file content
func (a *App) ReadLogs(filterDeviceID string) ([]string, error) {
	logPath := logger.GetLogPath(a.logDir)
	if logPath == "" {
		return []string{}, nil
	}

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Return last 500 lines to avoid overload
	if len(lines) > 500 {
		return lines[len(lines)-500:], nil
	}
	return lines, nil
}

// WriteLog allows the frontend to write to the log file
func (a *App) WriteLog(deviceID, level, message string) {
	switch strings.ToLower(level) {
	case "error":
		logger.Error(message, deviceID, level)
	case "warn":
		logger.Warn(message, deviceID, level)
	case "debug":
		logger.Debug(message, deviceID, level)
	default:
		logger.Info(message, deviceID, level)
	}
}

// ClearLogFile clears the main log file
func (a *App) ClearLogFile(deviceID string) error {
	logPath := logger.GetLogPath(a.logDir)
	if logPath == "" {
		return nil
	}

	// Truncate file to 0 bytes
	if err := os.Truncate(logPath, 0); err != nil {
		logger.Error(fmt.Sprintf("Error clearing log file: %v", err), "System", "Error")
		return err
	}

	return nil
}

// ExportLogs saves the logs to a user-selected file
func (a *App) ExportLogs(csvContent, defaultFileName string) error {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFileName,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "CSV Files (*.csv)",
				Pattern:     "*.csv",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return err
	}

	if filePath == "" {
		// User cancelled the dialog
		return nil
	}

	return os.WriteFile(filePath, []byte(csvContent), 0644)
}

// AutoSearchProbe scans all serial ports and probes them
func (a *App) AutoSearchProbe() ([]hubmanager.DeviceInfo, error) {
	return a.hubManager.AutoSearchProbe()
}

// OpenPort opens a serial connection
func (a *App) OpenPort(path string) error {
	return a.hubManager.OpenPort(path)
}

// SendCommand sends a string command and returns the response
func (a *App) SendCommand(deviceID string, cmd string) (string, error) {
	return a.hubManager.SendCommand(deviceID, cmd)
}

// GetPortStatus retrieves the current port states using the high-level API
func (a *App) GetPortStatus(deviceID string, totalPorts int) (map[int]bool, error) {
	return a.hubManager.GetPortStatus(deviceID, totalPorts)
}

// SetPortStatus sets the port states using the high-level API
func (a *App) SetPortStatus(deviceID string, password string, states map[int]bool, totalPorts int) (string, error) {
	return a.hubManager.SetPortStatus(deviceID, password, states, totalPorts)
}

// ResetHub sends the RH command
func (a *App) ResetHub(deviceID string, password string) (string, error) {
	return a.hubManager.ResetHub(deviceID, password)
}

// RestoreDefault sends the RD command
func (a *App) RestoreDefault(deviceID string, password string) (string, error) {
	return a.hubManager.RestoreDefault(deviceID, password)
}

// SavePortStates sends the WP command
func (a *App) SavePortStates(deviceID string, password string) (string, error) {
	return a.hubManager.SavePortStates(deviceID, password)
}

// SetVBUSPower sends the AP command
func (a *App) SetVBUSPower(deviceID string, password string, enabled bool) (string, error) {
	return a.hubManager.SetVBUSPower(deviceID, password, enabled)
}

// GetDeviceName sends the GN command
func (a *App) GetDeviceName(deviceID string, password string) (string, error) {
	return a.hubManager.GetDeviceName(deviceID, password)
}

// SetDeviceName sends the SN command
func (a *App) SetDeviceName(deviceID string, password string, name string) (string, error) {
	return a.hubManager.SetDeviceName(deviceID, password, name)
}

// ChangePassword sends the CP command
func (a *App) ChangePassword(deviceID string, oldPass string, newPass string) (string, error) {
	return a.hubManager.ChangePassword(deviceID, oldPass, newPass)
}

// ClosePort closes connection for specific device
func (a *App) ClosePort(deviceID string) error {
	return a.hubManager.ClosePort(deviceID)
}

// QuitApp quits the application
func (a *App) QuitApp() {
	logger.Info("QuitApp called", "System", "System")
	runtime.Quit(a.ctx)
}

// GetStoredPassword retrieves the password for a device from session memory
func (a *App) GetStoredPassword(portID string) string {
	a.passMu.Lock()
	defer a.passMu.Unlock()
	return a.sessionPasswords[portID]
}

// SetStoredPassword saves the password for a device to session memory
func (a *App) SetStoredPassword(portID, password string) error {
	a.passMu.Lock()
	defer a.passMu.Unlock()
	if password == "" {
		delete(a.sessionPasswords, portID)
	} else {
		a.sessionPasswords[portID] = password
	}
	return nil
}

// GetUSBTree is removed
func (a *App) GetUSBTree() (string, error) {
	return "{}", nil
}
