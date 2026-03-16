package main

import (
	"bufio"
	"context"
	"fmt"
	"managed-usb-hub-wails/pkg/config"
	"managed-usb-hub-wails/pkg/hubmanager"
	"managed-usb-hub-wails/pkg/scheduler"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	mu         sync.Mutex
	logDir     string
	logMu      sync.Mutex
	hubManager *hubmanager.HubManager
	scheduler  *scheduler.Scheduler
}

// NewApp creates a new App application struct
func NewApp() *App {
	a := &App{}
	// Initialize HubManager with a logger that points to App's log method
	a.hubManager = hubmanager.NewHubManager(func(deviceID, level, message string) {
		a.log(deviceID, level, message)
	})

	// Initialize Scheduler
	a.scheduler = scheduler.NewScheduler(func(deviceID string, mask string) error {
		return a.executeScheduledTask(deviceID, mask)
	})

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

// executeScheduledTask executes the scheduled task
func (a *App) executeScheduledTask(deviceID string, mask string) error {
	password := a.GetStoredPassword(deviceID)
	if password == "" {
		password = "pass    " // Default password padded to 8 chars? Or just "pass"
	}
	// Ensure password is padded to 8 chars if needed.
	// Based on frontend code: `password.padEnd(8, ' ')`
	if len(password) < 8 {
		password = password + strings.Repeat(" ", 8-len(password))
	} else if len(password) > 8 {
		password = password[:8]
	}

	cmdMask := mask
	// Basic validation of mask
	if len(cmdMask) != 8 {
		// Fallback defaults if invalid
		if cmdMask == "" || cmdMask == "0" {
			cmdMask = "00000000"
		} else {
			cmdMask = "FFFFFFFF"
		}
	}

	cmd := fmt.Sprintf("SP%s%s", password, cmdMask)

	a.log(deviceID, "Scheduler", fmt.Sprintf("Executing task: Set Mask %s for device %s", cmdMask, deviceID))

	err := a.OpenPort(deviceID)
	if err != nil {
		a.log(deviceID, "Error", fmt.Sprintf("Scheduler failed to open port: %v", err))
		return err
	}

	resp, err := a.SendCommand(deviceID, cmd)
	if err != nil {
		a.log(deviceID, "Error", fmt.Sprintf("Scheduler command failed: %v", err))
		return err
	}

	a.log(deviceID, "Scheduler", fmt.Sprintf("Task executed. Mask: %s (%s)", cmdMask, a.formatMaskLog(cmdMask)))

	// Notify frontend
	runtime.EventsEmit(a.ctx, "task-executed", map[string]string{
		"deviceID":  deviceID,
		"mask":      cmdMask,
		"response":  resp,
		"timestamp": time.Now().Format("15:04:05"),
	})

	// Do not close port to keep connection alive
	// return a.CloseCurrentPort()
	return nil
}

// Helper to format mask log
func (a *App) formatMaskLog(maskHex string) string {
	// Parse hex mask
	if len(maskHex) < 8 {
		return maskHex
	}

	// Special cases
	if strings.ToUpper(maskHex) == "FFFFFFFF" {
		return "On=1,2,3,4,5,6,7"
	}
	if maskHex == "00000000" {
		return "Off=1,2,3,4,5,6,7"
	}

	// The mask sent to the device is 8 hex chars.
	// Based on frontend (controlPanel.vue) logic:
	// byte0 = 0x80 | (states...)
	// maskHex = byte0 + "FFFFFF"
	// So only the first 2 hex chars (byte0) matter for port status 1-7.

	firstByteHex := maskHex[:2]
	var b int
	_, err := fmt.Sscanf(firstByteHex, "%X", &b)
	if err != nil {
		return maskHex
	}

	var onPorts []string
	var offPorts []string

	// Bits 0-6 correspond to Ports 1-7
	for i := 1; i <= 7; i++ {
		if (b>>(i-1))&1 == 1 {
			onPorts = append(onPorts, fmt.Sprintf("%d", i))
		} else {
			offPorts = append(offPorts, fmt.Sprintf("%d", i))
		}
	}

	parts := []string{}
	if len(onPorts) > 0 {
		parts = append(parts, fmt.Sprintf("On=%s", strings.Join(onPorts, ",")))
	}
	if len(offPorts) > 0 {
		parts = append(parts, fmt.Sprintf("Off=%s", strings.Join(offPorts, ",")))
	}

	return strings.Join(parts, ", ")
}

// startup is called when the app starts. The context is saved
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load configuration
	if err := config.Load(); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
	}

	// Reload tasks into scheduler after config is loaded
	a.scheduler.ReloadTasks()

	// Start Scheduler
	a.scheduler.Start()

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
}

// shutdown is called when the app terminates
func (a *App) shutdown(ctx context.Context) {
	a.log("System", "System", "Application stopped")
}

// getLogPath returns the path to the current day's log file for a specific device
func (a *App) getLogPath(deviceID string) string {
	if a.logDir == "" {
		return ""
	}

	// Sanitize deviceID to be a valid folder name
	safeDeviceID := strings.ReplaceAll(deviceID, ":", "")
	safeDeviceID = strings.ReplaceAll(safeDeviceID, "\\", "")
	safeDeviceID = strings.ReplaceAll(safeDeviceID, "/", "")

	if safeDeviceID == "" {
		safeDeviceID = "System"
	}

	// Windows reserves COM1-COM9, LPT1-LPT9, NUL, CON, etc.
	// We cannot name a folder "COM6". Prepend a prefix.
	deviceLogDir := filepath.Join(a.logDir, "Device_"+safeDeviceID)
	if err := os.MkdirAll(deviceLogDir, 0755); err != nil {
		fmt.Printf("Error creating device log directory: %v\n", err)
		return ""
	}

	dateStr := time.Now().Format("2006-01-02")
	return filepath.Join(deviceLogDir, fmt.Sprintf("hub_manager_%s.log", dateStr))
}

// log writes a log entry to the file
func (a *App) log(deviceID, level, message string) {
	if deviceID == "" || deviceID == "System" {
		// deviceID = "System"
		// User requested to disable System logs. Only log device-specific events.
		return
	}

	logPath := a.getLogPath(deviceID)
	if logPath == "" {
		return
	}

	a.logMu.Lock()
	defer a.logMu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, deviceID, message)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		fmt.Printf("Error writing to log file: %v\n", err)
	}
}

// ReadLogs reads the current log file content
// If filterDeviceID is provided, it reads only that device's log file.
// If empty, it might read System logs? Or we should change logic to read specific file.
// Given the requirement "select device -> view logs", we should read that device's file.
func (a *App) ReadLogs(filterDeviceID string) ([]string, error) {
	targetDevice := filterDeviceID
	if targetDevice == "" {
		targetDevice = "System"
	}

	logPath := a.getLogPath(targetDevice)
	if logPath == "" {
		return []string{}, nil
	}

	a.logMu.Lock()
	defer a.logMu.Unlock()

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

	// Return last 200 lines to avoid overload
	if len(lines) > 200 {
		return lines[len(lines)-200:], nil
	}
	return lines, nil
}

// WriteLog allows the frontend to write to the log file
func (a *App) WriteLog(deviceID, level, message string) {
	a.log(deviceID, level, message)
}

// ClearLogFile clears the log file for the specified device
func (a *App) ClearLogFile(deviceID string) error {
	if deviceID == "" {
		deviceID = "System"
	}

	logPath := a.getLogPath(deviceID)
	if logPath == "" {
		return nil
	}

	a.logMu.Lock()
	defer a.logMu.Unlock()

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil
	}

	// Truncate file to 0 bytes
	if err := os.Truncate(logPath, 0); err != nil {
		fmt.Printf("Error clearing log file: %v\n", err)
		return err
	}

	return nil
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

// ClosePort closes connection for specific device
func (a *App) ClosePort(deviceID string) error {
	return a.hubManager.ClosePort(deviceID)
}

// QuitApp quits the application
func (a *App) QuitApp() {
	a.log("System", "System", "QuitApp called")
	runtime.Quit(a.ctx)
}

// GetStoredPassword retrieves the password for a device from persistent storage
func (a *App) GetStoredPassword(portID string) string {
	return config.GetPassword(portID)
}

// SetStoredPassword saves the password for a device to persistent storage
func (a *App) SetStoredPassword(portID, password string) error {
	return config.SetPassword(portID, password)
}

// GetUSBTree is removed
func (a *App) GetUSBTree() (string, error) {
	return "{}", nil
}

// GetScheduledTasks returns all scheduled tasks
func (a *App) GetScheduledTasks() []config.ScheduledTask {
	return a.scheduler.GetTasks()
}

// AddScheduledTask adds a new scheduled task
func (a *App) AddScheduledTask(task config.ScheduledTask) error {
	return a.scheduler.AddTask(task)
}

// RemoveScheduledTask removes a scheduled task
func (a *App) RemoveScheduledTask(id string) error {
	return a.scheduler.RemoveTask(id)
}

// UpdateScheduledTask updates an existing scheduled task
func (a *App) UpdateScheduledTask(task config.ScheduledTask) error {
	return a.scheduler.UpdateTask(task)
}
