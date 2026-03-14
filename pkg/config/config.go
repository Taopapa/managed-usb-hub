package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type ScheduledTask struct {
	ID         string `json:"id"`
	DeviceID   string `json:"device_id"`   // Port Name e.g., COM3
	DaysOfWeek []int  `json:"days_of_week"` // 0-6 (Sun-Sat)
	StartTime  string `json:"start_time"`   // HH:mm
	StopTime   string `json:"stop_time"`    // HH:mm
	StartMask  string `json:"start_mask"`   // Hex mask for StartTime e.g. "FFFFFFFF"
	StopMask   string `json:"stop_mask"`    // Hex mask for StopTime e.g. "00000000"
	Enabled    bool   `json:"enabled"`
}

type AppConfig struct {
	DevicePasswords map[string]string `json:"device_passwords"`
	ScheduledTasks  []ScheduledTask   `json:"scheduled_tasks"`
}

var (
	configDir  string
	configPath string
	current    *AppConfig
	mu         sync.Mutex
)

func init() {
	// Initialize paths
	userDir, err := os.UserConfigDir()
	if err != nil {
		userDir = "." // Fallback to local
	}
	configDir = filepath.Join(userDir, "ManagedUSBHub")
	configPath = filepath.Join(configDir, "config.json")

	current = &AppConfig{
		DevicePasswords: make(map[string]string),
		ScheduledTasks:  []ScheduledTask{},
	}
}

// Load reads the config from disk
func Load() error {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config yet, use default
		}
		return err
	}

	return json.Unmarshal(data, current)
}

// Save writes the config to disk
func Save() error {
	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// GetScheduledTasks returns the list of scheduled tasks
func GetScheduledTasks() []ScheduledTask {
	mu.Lock()
	defer mu.Unlock()
	// Return a copy to avoid race conditions
	tasks := make([]ScheduledTask, len(current.ScheduledTasks))
	copy(tasks, current.ScheduledTasks)
	return tasks
}

// SaveScheduledTasks saves the list of scheduled tasks
func SaveScheduledTasks(tasks []ScheduledTask) error {
	mu.Lock()
	current.ScheduledTasks = tasks
	mu.Unlock()
	return Save()
}

// GetPassword returns the password for a specific port ID
func GetPassword(portID string) string {
	mu.Lock()
	defer mu.Unlock()
	return current.DevicePasswords[portID]
}

// SetPassword sets the password for a specific port ID and saves
func SetPassword(portID, password string) error {
	mu.Lock()
	current.DevicePasswords[portID] = password
	mu.Unlock()
	return Save()
}
