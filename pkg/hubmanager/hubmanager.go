package hubmanager

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"managed-usb-hub-wails/pkg/config"
	"managed-usb-hub-wails/pkg/logger"

	"go.bug.st/serial"
)

func normalizePortPath(path string) string {
	if runtime.GOOS != "windows" {
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "COM") {
			return "/dev/" + path
		}
	}
	return path
}

func stripDevPrefix(path string) string {
	if runtime.GOOS != "windows" {
		return strings.TrimPrefix(path, "/dev/")
	}
	return path
}

// DeviceInfo represents the information of a discovered hub device
type DeviceInfo struct {
	Path          string `json:"path"`
	ProbeResponse string `json:"probeResponse"`
	AsciiResponse string `json:"asciiResponse"`
	DeviceName    string `json:"deviceName"`
	LedStatus     string `json:"ledStatus"` // Hex encoded
	GoData        string `json:"goData"`    // Hex encoded
	RawLedStatus  string `json:"-"`         // Raw ASCII, not sent to frontend
	RawGoData     string `json:"-"`         // Raw ASCII, not sent to frontend
	Success       bool   `json:"success"`
}

// HubManager handles the communication with USB Hub devices
type HubManager struct {
	mu          sync.Mutex
	persistent  bool
	currentPort serial.Port
	currentPath string
}

const (
	openRetryAttempts    = 3
	commandRetryAttempts = 2
	retryDelay           = 120 * time.Millisecond
)

// NewHubManager creates a new HubManager instance
func NewHubManager(persistent bool) *HubManager {
	return &HubManager{persistent: persistent}
}

func (hm *HubManager) closeCurrentPortLocked() error {
	if hm.currentPort == nil {
		hm.currentPath = ""
		return nil
	}

	err := hm.currentPort.Close()
	hm.currentPort = nil
	hm.currentPath = ""
	return err
}

func isTransientSerialError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "access is denied") ||
		strings.Contains(msg, "incorrect function")
}

func (hm *HubManager) openPortWithRetry(path string) (serial.Port, error) {
	var p serial.Port
	var err error
	retryDelay := 10 * time.Millisecond
	openRetryAttempts := 5

	normalizedPath := normalizePortPath(path)

	for attempt := 1; attempt <= openRetryAttempts; attempt++ {
		p, err = serial.Open(normalizedPath, &serial.Mode{
			BaudRate: config.DefaultBaudRate,
		})

		if err == nil {
			return p, nil
		}

		if !isTransientSerialError(err) {
			return nil, err
		}

		time.Sleep(time.Duration(attempt) * retryDelay)
	}

	return nil, err
}

func (hm *HubManager) getOrOpenPortLocked(path string) (serial.Port, error) {
	if hm.persistent && hm.currentPort != nil && hm.currentPath == path {
		return hm.currentPort, nil
	}

	if hm.persistent && hm.currentPort != nil && hm.currentPath != path {
		_ = hm.closeCurrentPortLocked()
	}

	port, err := hm.openPortWithRetry(path)
	if err != nil {
		return nil, err
	}

	if hm.persistent {
		hm.currentPort = port
		hm.currentPath = path
	}

	return port, nil
}

// AutoSearchProbe scans all serial ports and probes them concurrently
func (hm *HubManager) AutoSearchProbe() ([]DeviceInfo, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, err
	}

	var results []DeviceInfo
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, p := range ports {
		wg.Add(1)
		go func(portPath string) {
			defer wg.Done()
			res := hm.probeDevice(portPath)
			if res.Success {
				mu.Lock()
				results = append(results, res)
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait()

	return results, nil
}

// OpenPort tests opening a serial connection and closes it immediately
func (hm *HubManager) OpenPort(path string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	p, err := hm.getOrOpenPortLocked(path)
	if err != nil {
		return err
	}

	if !hm.persistent {
		p.Close()
	}

	return nil
}

// ClosePort closes the persistent connection when enabled
func (hm *HubManager) ClosePort(path string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.persistent {
		return nil
	}

	if path != "" && hm.currentPath != "" && hm.currentPath != path {
		return nil
	}

	return hm.closeCurrentPortLocked()
}

func (hm *HubManager) handleCommandErrorLocked(path string) {
	if !hm.persistent {
		return
	}
	if hm.currentPath == path {
		_ = hm.closeCurrentPortLocked()
	}
}

// SendCommand sends a string command and returns the response
func (hm *HubManager) SendCommand(path string, cmd string) (string, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	attempts := 1
	if !hm.persistent {
		attempts = commandRetryAttempts
	}

	var lastResp string
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		port, err := hm.getOrOpenPortLocked(path)
		if err != nil {
			hm.handleCommandErrorLocked(path)
			lastErr = fmt.Errorf("Port not open: %v", err)
			if attempt < attempts && isTransientSerialError(err) {
				time.Sleep(time.Duration(attempt) * retryDelay)
				continue
			}
			return "", lastErr
		}

		// Drain before sending
		hm.drain(port, path)

		resp, cmdErr := hm.sendAndRead(port, path, cmd, time.Duration(config.CommandTimeoutMs)*time.Millisecond)
		lastResp = resp

		if !hm.persistent {
			_ = port.Close()
		}

		if cmdErr == nil {
			return resp, nil
		}

		lastErr = cmdErr
		hm.handleCommandErrorLocked(path)

		if attempt < attempts && isTransientSerialError(cmdErr) {
			time.Sleep(time.Duration(attempt) * retryDelay)
			continue
		}

		return "", cmdErr
	}

	return lastResp, lastErr
}

func (hm *HubManager) sendCommandAllowDisconnect(path string, cmd string) (string, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	port, err := hm.getOrOpenPortLocked(path)
	if err != nil {
		return "", fmt.Errorf("Port not open: %v", err)
	}

	if !hm.persistent {
		defer port.Close()
	}

	hm.drain(port, path)

	resp, err := hm.sendAndReadAllowDisconnect(port, path, cmd, time.Duration(config.CommandTimeoutMs)*time.Millisecond)
	if err != nil {
		hm.handleCommandErrorLocked(path)
		return "", err
	}

	return resp, nil
}

// --- High-Level Protocol API ---

// GetPortStatus retrieves the current port states
func (hm *HubManager) GetPortStatus(portPath string, totalPorts int) (map[int]bool, error) {
	resp, err := hm.SendCommand(portPath, "GW")
	if err != nil {
		return nil, err
	}

	return ParseGWResponse(resp, totalPorts)
}

// ParseGWResponse parses the GW command response into a boolean map.
// Only the first byte is used as the port-state byte:
// bit0 -> port1, bit1 -> port2, ..., 1 means on and 0 means off.
func ParseGWResponse(resp string, totalPorts int) (map[int]bool, error) {
	cleanHex := strings.TrimSpace(resp)
	if strings.HasPrefix(strings.ToUpper(cleanHex), "GW") {
		cleanHex = cleanHex[2:]
	} else if strings.HasPrefix(strings.ToUpper(cleanHex), "G") {
		cleanHex = cleanHex[1:]
	}

	states := make(map[int]bool)
	if len(cleanHex) == 0 {
		return states, fmt.Errorf("empty response")
	}

	// First byte (2 hex chars) contains the port states
	firstByteHex := cleanHex
	if len(cleanHex) >= 2 {
		firstByteHex = cleanHex[:2]
	}

	var maskByte int64
	_, err := fmt.Sscanf(firstByteHex, "%X", &maskByte)
	if err != nil {
		return states, err
	}

	for i := 1; i <= totalPorts; i++ {
		states[i] = ((maskByte >> (i - 1)) & 1) != 0
	}

	return states, nil
}

func (hm *HubManager) verifyPortStatus(portPath string, states map[int]bool, totalPorts int) (string, bool) {
	time.Sleep(time.Duration(config.StateVerifyDelayMs) * time.Millisecond)
	verifyResp, verifyErr := hm.SendCommand(portPath, "GW")
	if verifyErr != nil {
		return "", false
	}

	verifyStates, parseErr := ParseGWResponse(verifyResp, totalPorts)
	if parseErr != nil {
		return verifyResp, false
	}

	for i := 1; i <= totalPorts; i++ {
		if verifyStates[i] != states[i] {
			return verifyResp, false
		}
	}

	logger.Info("GW verification succeeded", portPath, "Verify")
	return verifyResp, true
}

// SetPortStatus sets the port states using the ST command
func (hm *HubManager) SetPortStatus(portPath string, password string, states map[int]bool, totalPorts int) (string, error) {
	if password == "" {
		password = config.DefaultPassword
	}

	var byte0 byte = 0x00
	limit := totalPorts
	if limit > 8 {
		limit = 8
	}
	for i := 1; i <= limit; i++ {
		if states[i] {
			byte0 |= (1 << (i - 1))
		}
	}

	maskHex := fmt.Sprintf("%02XFFFFFF", byte0)
	if byte0 == 0x00 {
		maskHex = "80FFFFFF"
	}
	cmd := fmt.Sprintf("ST%s%s", password, maskHex)
	resp, err := hm.SendCommand(portPath, cmd)
	if err != nil {
		return resp, err
	}

	if strings.TrimSpace(resp) != "" {
		return resp, nil
	}

	verifyResp, matched := hm.verifyPortStatus(portPath, states, totalPorts)
	if matched {
		logger.Info("ST had no direct response, but GW verification succeeded", portPath, "Verify")
		return verifyResp, nil
	}

	logger.Warn("ST had no direct response and GW verification did not match requested state", portPath, "Verify")
	return resp, nil
}

// ResetHub sends the RT command
func (hm *HubManager) ResetHub(portPath string, password string) (string, error) {
	if password == "" {
		password = config.DefaultPassword
	}
	cmd := fmt.Sprintf("RT%s", password)
	return hm.sendCommandAllowDisconnect(portPath, cmd)
}

// RestoreDefault sends the RS command
func (hm *HubManager) RestoreDefault(portPath string, password string) (string, error) {
	if password == "" {
		password = config.DefaultPassword
	}
	cmd := fmt.Sprintf("RS%s", password)
	return hm.SendCommand(portPath, cmd)
}

// SavePortStates sends the SV command
func (hm *HubManager) SavePortStates(portPath string, password string) (string, error) {
	if password == "" {
		password = config.DefaultPassword
	}
	cmd := fmt.Sprintf("SV%s", password)
	return hm.SendCommand(portPath, cmd)
}

// SetVBUSPower sends the AP command with 1/0 payload.
func (hm *HubManager) SetVBUSPower(portPath string, password string, enabled bool) (string, error) {
	if password == "" {
		password = config.DefaultPassword
	}

	state := "0"
	if enabled {
		state = "1"
	}

	cmd := fmt.Sprintf("AP%s%s", password, state)
	return hm.SendCommand(portPath, cmd)
}

// GetDeviceName sends the GN command.
func (hm *HubManager) GetDeviceName(portPath string, password string) (string, error) {
	if password == "" {
		password = config.DefaultPassword
	}

	cmd := fmt.Sprintf("GN%s", password)
	return hm.SendCommand(portPath, cmd)
}

// SetDeviceName sends the SN command.
func (hm *HubManager) SetDeviceName(portPath string, password string, name string) (string, error) {
	if password == "" {
		password = config.DefaultPassword
	}

	cmd := fmt.Sprintf("SN%s%s", password, name)
	return hm.SendCommand(portPath, cmd)
}

// ChangePassword sends the MD command
func (hm *HubManager) ChangePassword(portPath string, oldPass string, newPass string) (string, error) {
	// Need to pad passwords to 8 characters
	padPassword := func(p string) string {
		if len(p) > 8 {
			return p[:8]
		}
		return p + strings.Repeat(" ", 8-len(p))
	}
	cmd := fmt.Sprintf("MD%s%s", padPassword(oldPass), padPassword(newPass))
	return hm.SendCommand(portPath, cmd)
}

// Internal helper methods

func (hm *HubManager) probeDevice(path string) DeviceInfo {
	for attempt := 1; attempt <= config.ProbeRetries; attempt++ {
		p, err := hm.openPortWithRetry(path)
		if err != nil {
			time.Sleep(time.Duration(config.ProbeRetryDelayMs) * time.Millisecond)
			continue
		}

		// Give Windows serial drivers a brief moment after opening.
		time.Sleep(80 * time.Millisecond)

		deviceInfo, ok, retry := func() (DeviceInfo, bool, bool) {
			defer p.Close()

			hm.drain(p, path)
			idRaw, idErr := hm.sendAndRead(p, path, "?S", time.Duration(config.ProbeTimeoutMs)*time.Millisecond)
			if idErr != nil || idRaw == "" {
				return DeviceInfo{Success: false}, false, isTransientSerialError(idErr) || idRaw == ""
			}

			idCheck := strings.ToUpper(strings.TrimSpace(idRaw))
			if !strings.Contains(idCheck, "CENTOS") && !strings.Contains(idCheck, "C2G") {
				return DeviceInfo{Success: false}, false, true
			}

			idAscii := idRaw
			if idx := strings.IndexAny(idAscii, "\r\n"); idx != -1 {
				idAscii = idAscii[:idx]
			}

			hm.drain(p, path)
			gpDataRaw, gpErr := hm.sendAndRead(p, path, "GW", time.Duration(config.ProbeTimeoutMs)*time.Millisecond)
			if gpErr != nil {
				return DeviceInfo{Success: false}, false, isTransientSerialError(gpErr)
			}
			if idx := strings.IndexAny(gpDataRaw, "\r\n"); idx != -1 {
				gpDataRaw = gpDataRaw[:idx]
			}
			gpDataHex := fmt.Sprintf("%X", gpDataRaw)

			hm.drain(p, path)
			goDataRaw, goErr := hm.sendAndRead(p, path, "CM", time.Duration(config.ProbeTimeoutMs)*time.Millisecond)
			if goErr != nil {
				return DeviceInfo{Success: false}, false, isTransientSerialError(goErr)
			}
			if idx := strings.IndexAny(goDataRaw, "\r\n"); idx != -1 {
				goDataRaw = goDataRaw[:idx]
			}
			goDataHex := fmt.Sprintf("%X", goDataRaw)

			deviceName := ""
			hm.drain(p, path)
			nameResp, nameErr := hm.sendAndRead(p, path, fmt.Sprintf("GN%s", config.DefaultPassword), time.Duration(config.ProbeTimeoutMs)*time.Millisecond)
			if nameErr == nil {
				if idx := strings.IndexAny(nameResp, "\r\n"); idx != -1 {
					nameResp = nameResp[:idx]
				}

				var sb strings.Builder
				for _, ch := range nameResp {
					if ch >= 32 && ch <= 126 {
						sb.WriteRune(ch)
					}
				}
				// We don't trim spaces right away because spaces are valid within names.
				normalizedNameResp := sb.String()

				if strings.HasPrefix(normalizedNameResp, "B+") {
					deviceName = strings.TrimRight(normalizedNameResp[2:], " ")
				} else if strings.HasPrefix(normalizedNameResp, "B") {
					deviceName = strings.TrimRight(normalizedNameResp[1:], " ")
				}
			}

			return DeviceInfo{
				Path:          stripDevPrefix(path),
				AsciiResponse: idAscii,
				ProbeResponse: idAscii,
				DeviceName:    deviceName,
				LedStatus:     gpDataHex,
				GoData:        goDataHex,
				RawLedStatus:  gpDataRaw,
				RawGoData:     goDataRaw,
				Success:       true,
			}, true, false
		}()

		if ok {
			return deviceInfo
		}

		if attempt < config.ProbeRetries && retry {
			time.Sleep(time.Duration(config.ProbeRetryDelayMs) * time.Millisecond)
			continue
		}
	}

	return DeviceInfo{Success: false}
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
	cleanResp := strings.TrimSpace(response)
	return cleanResp, nil
}

func (hm *HubManager) sendAndRead(port serial.Port, portPath string, cmd string, timeout time.Duration) (string, error) {
	fullCmd := cmd + "\r"
	txBytes := []byte(fullCmd)
	_, err := port.Write(txBytes)
	if err != nil {
		logger.Error(fmt.Sprintf("TX failed for %q: %v", cmd, err), portPath, "TX")
		return "", err
	}
	time.Sleep(50 * time.Millisecond)

	resp, err := hm.readResponse(port, portPath, timeout)
	if err != nil {
		logger.Error(fmt.Sprintf("RX failed for %q: %v", cmd, err), portPath, "RX")
		return resp, err
	}

	cleanResp := strings.TrimSpace(resp)
	cleanCmd := strings.TrimSpace(cmd)

	if cleanResp == cleanCmd {
		resp2, err := hm.readResponse(port, portPath, timeout)
		if err != nil {
			return resp2, err
		}
		return resp2, nil
	}

	if strings.HasPrefix(cleanResp, cleanCmd) {
		cleanResp = strings.TrimPrefix(cleanResp, cleanCmd)
		cleanResp = strings.TrimSpace(cleanResp)
		if cleanResp == "" {
			resp2, err := hm.readResponse(port, portPath, timeout)
			if err != nil {
				return resp2, err
			}
			return resp2, nil
		}
		return cleanResp, nil
	}

	return cleanResp, nil
}

func (hm *HubManager) sendAndReadAllowDisconnect(port serial.Port, portPath string, cmd string, timeout time.Duration) (string, error) {
	fullCmd := cmd + "\r"
	txBytes := []byte(fullCmd)
	_, err := port.Write(txBytes)
	if err != nil {
		return "", err
	}
	time.Sleep(50 * time.Millisecond)

	resp, err := hm.readResponse(port, portPath, timeout)
	if err != nil {
		if isTransientSerialError(err) {
			return strings.TrimSpace(resp), nil
		}
		return resp, err
	}

	return strings.TrimSpace(resp), nil
}
