package hubmanager

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
	ports  map[string]serial.Port
	mu     sync.Mutex
	logger func(deviceID, level, message string)
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
		ports:  make(map[string]serial.Port),
		logger: logger,
	}
}

// AutoSearchProbe scans all serial ports and probes them
func (hm *HubManager) AutoSearchProbe() ([]DeviceInfo, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		hm.logger("System", "Error", fmt.Sprintf("Failed to get ports list: %v", err))
		return nil, err
	}

	var results []DeviceInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, p := range ports {
		wg.Add(1)
		go func(portName string) {
			defer wg.Done()
			// Try to probe each port
			fmt.Printf("[AutoSearch] Probing port %s\n", portName)
			res := hm.probeDevice(portName)
			if res.Success {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
				fmt.Printf("[AutoSearch] Found device at %s\n", portName)
			}
		}(p)
	}

	wg.Wait()
	hm.logger("System", "System", fmt.Sprintf("Auto search completed. Found %d devices", len(results)))

	// Open all found devices to keep them connected
	for _, dev := range results {
		hm.OpenPort(dev.Path)
	}

	return results, nil
}

// OpenPort opens a serial connection
func (hm *HubManager) OpenPort(path string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Check if already open
	if _, ok := hm.ports[path]; ok {
		return nil
	}

	mode := &serial.Mode{BaudRate: 9600}
	p, err := serial.Open(path, mode)
	if err != nil {
		hm.logger(path, "Error", fmt.Sprintf("Failed to open port %s: %v", path, err))
		return err
	}
	hm.ports[path] = p
	return nil
}

// CloseCurrentPort closes the connection for specific port
func (hm *HubManager) ClosePort(path string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if p, ok := hm.ports[path]; ok {
		p.Close()
		delete(hm.ports, path)
	}
	return nil
}

// SendCommand sends a string command and returns the response
func (hm *HubManager) SendCommand(path string, cmd string) (string, error) {
	fmt.Printf("[Command] Sending to %s: %s\n", path, cmd)
	hm.mu.Lock()

	port, ok := hm.ports[path]
	hm.mu.Unlock() // Unlock immediately after map access

	if !ok {
		// Try to open it if not open
		err := hm.OpenPort(path)
		if err != nil {
			hm.logger(path, "Error", "Attempted to send command but port is not open and failed to open")
			return "", fmt.Errorf("Port not open")
		}
		// Re-acquire lock and get port
		hm.mu.Lock()
		port, ok = hm.ports[path]
		hm.mu.Unlock()

		if !ok {
			return "", fmt.Errorf("Port failed to open")
		}
	}

	// Drain before sending
	hm.drain(port, path)

	resp, err := hm.sendAndRead(port, path, cmd, 500*time.Millisecond)
	if err != nil {
		hm.logger(path, "Error", fmt.Sprintf("Command failed: %v", err))
		// If error, maybe close port?
		hm.ClosePort(path)
		return "", err
	}

	fmt.Printf("[Command] Response: %s\n", resp)
	return resp, nil
}

// Internal helper methods

func (hm *HubManager) probeDevice(path string) DeviceInfo {
	// Check if already open in our map
	hm.mu.Lock()
	_, isOpen := hm.ports[path]
	hm.mu.Unlock()

	if isOpen {
		// If already open, use it? Or close and reopen?
		// Probing usually requires clean state.
		// Let's close it to be safe, or try to use it.
		// For simplicity in probe, let's close and reopen.
		hm.ClosePort(path)
	}

	mode := &serial.Mode{BaudRate: 9600}
	p, err := serial.Open(path, mode)
	if err != nil {
		fmt.Printf("[Probe] Failed to open %s: %v\n", path, err)
		return DeviceInfo{Path: path, Success: false}
	}
	defer p.Close()

	// Step 1: ID
	time.Sleep(100 * time.Millisecond)

	// Fast Fail: Check ID with short timeout
	idAscii, err := hm.sendAndRead(p, path, "?Q", 500*time.Millisecond)
	if err != nil {
		fmt.Printf("[Probe] Failed to read ID from %s: %v\n", path, err)
		return DeviceInfo{Path: path, Success: false}
	}

	if idx := strings.IndexAny(idAscii, "\r\n"); idx != -1 {
		idAscii = idAscii[:idx]
	}

	fmt.Printf("[Probe] %s ID Response: %s\n", path, idAscii)

	if !strings.Contains(idAscii, "CENTOS") {
		fmt.Printf("[Probe] %s is not a target device (ID mismatch)\n", path)
		return DeviceInfo{Path: path, Success: false, AsciiResponse: idAscii}
	}

	hm.drain(p, path)

	// Step 2: GP
	gpDataRaw, _ := hm.sendAndRead(p, path, "GP", 500*time.Millisecond)
	fmt.Printf("[Probe] %s GP Response: %s\n", path, gpDataRaw)
	if idx := strings.IndexAny(gpDataRaw, "\r\n"); idx != -1 {
		gpDataRaw = gpDataRaw[:idx]
	}
	hm.drain(p, path)
	gpData := fmt.Sprintf("%X", gpDataRaw)

	// Step 3: GO
	goDataRaw, _ := hm.sendAndRead(p, path, "GO", 500*time.Millisecond)
	fmt.Printf("[Probe] %s GO Response: %s\n", path, goDataRaw)
	if idx := strings.IndexAny(goDataRaw, "\r\n"); idx != -1 {
		goDataRaw = goDataRaw[:idx]
	}
	hm.drain(p, path)
	goData := fmt.Sprintf("%X", goDataRaw)

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

	return cleanResp, nil
}
