package hubcli

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

// DeviceInfo represents the information of a discovered hub device
type DeviceInfo struct {
	Path          string `json:"path"`
	ProbeResponse string `json:"probeResponse"`
	AsciiResponse string `json:"asciiResponse"`
	LedStatus     string `json:"ledStatus"`
	GoData        string `json:"goData"`
	Success       bool   `json:"success"`
}

// HubManager handles the communication with USB Hub devices
type HubManager struct {
	port     serial.Port
	portPath string
	mu       sync.Mutex
	logger   func(deviceID, level, message string)
}

// NewHubManager creates a new HubManager instance
func NewHubManager(logger func(deviceID, level, message string)) *HubManager {
	if logger == nil {
		logger = func(deviceID, level, message string) {
			// Default no-op logger or print to stdout for CLI
			fmt.Printf("[%s] [%s] %s\n", level, deviceID, message)
		}
	}
	return &HubManager{
		logger: logger,
	}
}

// AutoSearchProbe scans all serial ports and probes them
func (hm *HubManager) AutoSearchProbe() ([]DeviceInfo, error) {
	hm.logger("System", "System", "Starting auto search for USB hubs")
	ports, err := serial.GetPortsList()
	if err != nil {
		hm.logger("System", "Error", fmt.Sprintf("Failed to get ports list: %v", err))
		return nil, err
	}

	var results []DeviceInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	hm.logger("System", "System", fmt.Sprintf("Found %d serial ports", len(ports)))

	for _, p := range ports {
		wg.Add(1)
		go func(portName string) {
			defer wg.Done()
			// Try to probe each port
			res := hm.probeDevice(portName)
			if res.Success {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
				hm.logger("System", "System", fmt.Sprintf("Found device at %s", portName))
			}
		}(p)
	}

	wg.Wait()
	hm.logger("System", "System", fmt.Sprintf("Auto search completed. Found %d devices", len(results)))
	return results, nil
}

// OpenPort opens a serial connection
func (hm *HubManager) OpenPort(path string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.logger(path, "System", fmt.Sprintf("Opening port %s", path))

	if hm.port != nil {
		hm.port.Close()
	}
	mode := &serial.Mode{BaudRate: 9600}
	p, err := serial.Open(path, mode)
	if err != nil {
		hm.logger(path, "Error", fmt.Sprintf("Failed to open port %s: %v", path, err))
		return err
	}
	hm.port = p
	hm.portPath = path
	hm.logger(path, "System", fmt.Sprintf("Port %s opened successfully", path))
	return nil
}

// CloseCurrentPort closes the current connection and resets state
func (hm *HubManager) CloseCurrentPort() error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.port != nil {
		hm.logger(hm.portPath, "System", fmt.Sprintf("Closing port %s", hm.portPath))
		hm.port.Close()
		hm.port = nil
		hm.portPath = ""
	}
	return nil
}

// SendCommand sends a string command and returns the response
func (hm *HubManager) SendCommand(cmd string) (string, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.port == nil {
		hm.logger("Unknown", "Error", "Attempted to send command but port is not open")
		return "", fmt.Errorf("Port not open")
	}

	// Drain before sending
	hm.drain(hm.port, hm.portPath)

	// Mask password in logs if present
	logCmd := cmd
	if strings.HasPrefix(cmd, "SP") || strings.HasPrefix(cmd, "WP") || strings.HasPrefix(cmd, "RD") || strings.HasPrefix(cmd, "RH") || strings.HasPrefix(cmd, "CP") {
		logCmd = cmd[:2] + "********"
	}
	hm.logger(hm.portPath, "Command", fmt.Sprintf("Sending: %s", logCmd))

	resp, err := hm.sendAndRead(hm.port, hm.portPath, cmd, 500*time.Millisecond)
	if err != nil {
		hm.logger(hm.portPath, "Error", fmt.Sprintf("Command failed: %v", err))
		return "", err
	}

	hm.logger(hm.portPath, "Response", fmt.Sprintf("Received: %s", resp))
	return resp, nil
}

// Internal helper methods

func (hm *HubManager) probeDevice(path string) DeviceInfo {
	mode := &serial.Mode{BaudRate: 9600}
	p, err := serial.Open(path, mode)
	if err != nil {
		return DeviceInfo{Path: path, Success: false}
	}
	defer p.Close()

	// Step 1: ID
	time.Sleep(100 * time.Millisecond)

	// Fast Fail: Check ID with short timeout
	idAscii, err := hm.sendAndRead(p, path, "?Q", 300*time.Millisecond)
	if err != nil {
		return DeviceInfo{Path: path, Success: false}
	}

	if idx := strings.IndexAny(idAscii, "\r\n"); idx != -1 {
		idAscii = idAscii[:idx]
	}

	if !strings.Contains(idAscii, "CENTOS") {
		return DeviceInfo{Path: path, Success: false, AsciiResponse: idAscii}
	}

	hm.drain(p, path)

	// Step 2: GP
	gpDataRaw, _ := hm.sendAndRead(p, path, "GP", 500*time.Millisecond)
	if idx := strings.IndexAny(gpDataRaw, "\r\n"); idx != -1 {
		gpDataRaw = gpDataRaw[:idx]
	}
	hm.drain(p, path)
	// For CLI, we return raw ASCII string, not hex encoded
	// This separates logic from GUI which might expect hex encoded
	// But to match current behavior, we'll keep it as is, or modify as requested.
	// User wants separation. So here we can define CLI specific behavior.
	// Let's keep it raw for CLI or simple hex string.
	// Current CLI code expects hex encoded string.
	// But let's use raw string for flexibility if CLI wants to parse differently.
	// Wait, CLI main.go currently parses Hex Encoded string.
	// If we want to separate, we can change CLI to parse raw string, and here we return raw string.
	
	// Let's return raw string here.
	gpData := gpDataRaw

	// Step 3: GO
	goDataRaw, _ := hm.sendAndRead(p, path, "GO", 500*time.Millisecond)
	if idx := strings.IndexAny(goDataRaw, "\r\n"); idx != -1 {
		goDataRaw = goDataRaw[:idx]
	}
	hm.drain(p, path)
	goData := goDataRaw

	return DeviceInfo{
		Path:          path,
		AsciiResponse: idAscii,
		ProbeResponse: idAscii,
		LedStatus:     gpData,
		GoData:        goData,
		Success:       true,
	}
}

func (hm *HubManager) drain(port serial.Port, portPath string) {
	port.SetReadTimeout(200 * time.Millisecond)
	buf := make([]byte, 128)
	for {
		n, err := port.Read(buf)
		if err != nil || n == 0 {
			break
		}
	}
}

func (hm *HubManager) readResponse(port serial.Port, portPath string, timeout time.Duration) (string, error) {
	port.SetReadTimeout(timeout)
	buf := make([]byte, 128)
	var response string
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		n, err := port.Read(buf)
		if err != nil {
			return response, err
		}
		if n > 0 {
			chunk := string(buf[:n])
			chunk = strings.ReplaceAll(chunk, "\x00", "")
			if chunk == "" {
				continue
			}
			response += chunk
			if strings.Contains(chunk, "\n") || strings.Contains(chunk, "\r") {
				break
			}
		}
	}
	return strings.TrimSpace(response), nil
}

func (hm *HubManager) sendAndRead(port serial.Port, portPath string, cmd string, timeout time.Duration) (string, error) {
	fullCmd := cmd + "\r"
	_, err := port.Write([]byte(fullCmd))
	if err != nil {
		return "", err
	}
	time.Sleep(50 * time.Millisecond)

	resp, err := hm.readResponse(port, portPath, timeout)
	if err != nil {
		return resp, err
	}

	cleanResp := strings.TrimSpace(resp)
	cleanCmd := strings.TrimSpace(cmd)

	if cleanResp == cleanCmd {
		return hm.readResponse(port, portPath, timeout)
	}

	if strings.HasPrefix(cleanResp, cleanCmd) {
		cleanResp = strings.TrimPrefix(cleanResp, cleanCmd)
		cleanResp = strings.TrimSpace(cleanResp)
		if cleanResp == "" {
			return hm.readResponse(port, portPath, timeout)
		}
		return cleanResp, nil
	}

	return resp, nil
}
