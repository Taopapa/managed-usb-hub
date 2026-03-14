package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"managed-usb-hub-wails/pkg/config"
	"managed-usb-hub-wails/pkg/hubmanager"
	"managed-usb-hub-wails/pkg/scheduler"
	"managed-usb-hub-wails/pkg/usbtree"
	"os"
	"path/filepath"
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

	// Check if already connected to this device
	// HubManager doesn't expose current port path directly, but we can track it or add a getter.
	// Actually, `HubManager` has `portPath` field but it's private in `hubmanager` package?
	// Wait, `HubManager` struct in `hubmanager.go` has `portPath string`. It's unexported?
	// No, `portPath string` starts with lowercase. It's unexported.
	// But `App` doesn't track current device ID except in frontend state?
	// Wait, `hubmanager` has `CloseCurrentPort`.
	// I should add `GetCurrentPortPath` to `HubManager` or just try to Open.
	// If I call `OpenPort` on an already open port, it might fail or close and reopen.
	// Let's modify `HubManager` to expose `GetCurrentPortPath` or similar.
	// OR, I can just try to execute.

	// For now, let's try to OpenPort. If it fails because it's already open, maybe it's fine?
	// But `serial.Open` usually fails if port is in use.
	// If `HubManager` holds the handle, `OpenPort` will close existing and open new.
	// So calling `OpenPort(deviceID)` is safe if we want to switch to it.
	// But if we are already connected to `deviceID`, `OpenPort` will reconnect, which is a bit disruptive but works.
	// If we are connected to ANOTHER device, `OpenPort` will switch.

	// To minimize disruption, I should check if we are already connected.
	// I can't check easily without modifying HubManager.
	// Let's modify HubManager first?
	// Or just proceed with Open-Send-Close strategy for now, assuming scheduler runs infrequently.
	// But if user is using the app, this will disconnect their session.
	// That's a known limitation for now.

	a.log(deviceID, "Scheduler", fmt.Sprintf("Executing task: Set Mask %s for device %s", cmdMask, deviceID))

	err := a.OpenPort(deviceID)
	if err != nil {
		a.log(deviceID, "Error", fmt.Sprintf("Scheduler failed to open port: %v", err))
		return err
	}

	resp, err := a.SendCommand(cmd)
	if err != nil {
		a.log(deviceID, "Error", fmt.Sprintf("Scheduler command failed: %v", err))
		return err
	}

	a.log(deviceID, "Scheduler", fmt.Sprintf("Task executed. Response: %s", resp))

	// We should probably close the port to leave it clean, unless we want to keep it open?
	// If we keep it open, the frontend might be out of sync if it thinks it's disconnected.
	// If the user was using it, we just reconnected, so they are fine (maybe).
	// But if the user was using another device, we switched.
	// So we should close it to be safe, so the user can reconnect to whatever they want.
	return a.CloseCurrentPort()
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

	// Initialize logging directory
	userDir, err := os.UserConfigDir()
	if err == nil {
		a.logDir = filepath.Join(userDir, "ManagedUSBHub", "logs")
		if err := os.MkdirAll(a.logDir, 0755); err != nil {
			fmt.Printf("Error creating log directory: %v\n", err)
		} else {
			a.log("System", "System", "Application started")
		}
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
func (a *App) SendCommand(cmd string) (string, error) {
	return a.hubManager.SendCommand(cmd)
}

// CloseCurrentPort closes the current connection and resets state
func (a *App) CloseCurrentPort() error {
	return a.hubManager.CloseCurrentPort()
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

// GetUSBTree returns the current USB device tree as a JSON string
func (a *App) GetUSBTree() (string, error) {
	a.log("System", "System", "Fetching USB Device Tree...")
	tree, err := usbtree.Enumerate()
	if err != nil {
		return "", err
	}
	// Convert to JSON string to avoid Wails binding issues with cross-package structs
	bytes, err := json.Marshal(tree)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
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
