package jsonapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"managed-usb-hub-wails/pkg/hubmanager"
)

// JSONCommand represents the incoming JSON request.
type JSONCommand struct {
	CMD     string `json:"CMD"`
	COM     string `json:"COM,omitempty"`
	PSW     string `json:"PSW,omitempty"`
	STATES  string `json:"STATES,omitempty"` // Examples: "1:1,2", "H:F4", "F4,FF,FF,FF"
	NEW_PSW string `json:"NEW_PSW,omitempty"`
	NAME    string `json:"NAME,omitempty"`
	FORMAT  string `json:"FORMAT,omitempty"`
}

func padPassword(p string) string {
	if len(p) >= 8 {
		return p[:8]
	}
	return fmt.Sprintf("%-8s", p)
}

// ProcessCommand executes the JSON command and returns a plain text response matching standard CLI.
func ProcessCommand(hm *hubmanager.HubManager, jsonStr string) string {
	var req JSONCommand
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return "Error: Invalid JSON format\n"
	}

	req.CMD = strings.ToUpper(strings.TrimSpace(req.CMD))
	req.COM = strings.TrimSpace(req.COM)
	if req.PSW == "" {
		req.PSW = "pass    "
	} else {
		req.PSW = padPassword(req.PSW)
	}

	switch req.CMD {
	case "Q":
		return handleQuery(hm, req)
	case "S":
		return handleSetState(hm, req, false)
	case "F":
		return handleSetState(hm, req, true)
	case "P":
		return handleChangePassword(hm, req)
	case "G":
		return handleGetState(hm, req)
	case "W":
		return handleSaveState(hm, req)
	case "T", "U":
		return handleGetDeviceName(hm, req)
	case "X":
		return handleSetDeviceName(hm, req)
	case "B":
		return handleSetVBUS(hm, req, true)
	case "C":
		return handleSetVBUS(hm, req, false)
	case "D":
		return handleSimpleCommand(hm, req, "RS")
	case "R":
		return handleSimpleCommand(hm, req, "RT")
	default:
		return "Invalid Command!\n"
	}
}

func cleanStatusHex(rawStatus string) string {
	cleanStatus := strings.TrimSpace(rawStatus)
	if strings.HasPrefix(strings.ToUpper(cleanStatus), "GW") {
		return cleanStatus[2:]
	}
	if strings.HasPrefix(strings.ToUpper(cleanStatus), "G") {
		return cleanStatus[1:]
	}
	return cleanStatus
}

func extractFirmwareVersion(asciiResponse string) string {
	fwVer := "Unknown"
	if idx := strings.LastIndex(asciiResponse, "v"); idx != -1 {
		return asciiResponse[idx:]
	}
	if len(asciiResponse) < 10 {
		return asciiResponse
	}
	return fwVer
}

func formatPortSummary(portStatusByte uint64, numPorts int) (string, string) {
	var onPorts []string
	var offPorts []string

	for i := 1; i <= numPorts; i++ {
		bitIndex := i - 1
		if (portStatusByte & (1 << bitIndex)) != 0 {
			onPorts = append(onPorts, fmt.Sprintf("%d", i))
		} else {
			offPorts = append(offPorts, fmt.Sprintf("%d", i))
		}
	}

	onStr := "None"
	if len(onPorts) > 0 {
		onStr = strings.Join(onPorts, ",")
	}

	offStr := "None"
	if len(offPorts) > 0 {
		offStr = strings.Join(offPorts, ",")
	}

	return onStr, offStr
}

func handleQuery(hm *hubmanager.HubManager, req JSONCommand) string {
	var devices []hubmanager.DeviceInfo
	var err error

	if req.COM != "" {
		idResp, err := hm.SendCommand(req.COM, "?S")
		if err != nil {
			return "Failed to open port\n"
		}
		idCheck := strings.ToUpper(strings.TrimSpace(idResp))
		if idCheck == "" || (!strings.Contains(idCheck, "CENTOS") && !strings.Contains(idCheck, "C2G")) {
			return "Failed to open port\n"
		}

		gpResp, _ := hm.SendCommand(req.COM, "GW")
		goResp, _ := hm.SendCommand(req.COM, "CM")

		devices = append(devices, hubmanager.DeviceInfo{
			Path:          req.COM,
			AsciiResponse: idResp,
			LedStatus:     fmt.Sprintf("%X", strings.TrimSpace(gpResp)),
			GoData:        fmt.Sprintf("%X", strings.TrimSpace(goResp)),
			RawLedStatus:  strings.TrimSpace(gpResp),
			RawGoData:     strings.TrimSpace(goResp),
		})
	} else {
		devices, err = hm.AutoSearchProbe()
		if err != nil {
			return fmt.Sprintf("Error: %v\n", err)
		}
	}

	var sb strings.Builder
	foundCount := 0
	numPorts := 7 // Hardcoded to 7 based on main.go

	for _, d := range devices {
		foundCount++
		statusHex := cleanStatusHex(d.RawLedStatus)
		var portStatusByte uint64 = 0
		if len(statusHex) >= 2 {
			parsedByte, _ := strconv.ParseUint(statusHex[0:2], 16, 8)
			portStatusByte = parsedByte
		} else {
			portStatusByte, _ = strconv.ParseUint(statusHex, 16, 32)
		}

		onStr, offStr := formatPortSummary(portStatusByte, numPorts)
		fwVer := extractFirmwareVersion(d.AsciiResponse)

		// Match formatQueryDevice logic
		if req.FORMAT == "formatted" || req.FORMAT == "F" {
			sb.WriteString(fmt.Sprintf(
				"Path=%s, Description=%q, Ports=%d, On=%s, Off=%s, FW=%s, GW=%s, CM=%s\n",
				d.Path, d.AsciiResponse, numPorts, onStr, offStr, fwVer, statusHex, strings.TrimSpace(d.GoData),
			))
		} else {
			output := fmt.Sprintf("%s, %d ports, On=%s, Off=%s, FW=%s", d.Path, numPorts, onStr, offStr, fwVer)
			if req.COM == "" {
				sb.WriteString(fmt.Sprintf(" %s\n", output))
			} else {
				sb.WriteString(output + "\n")
			}
		}
	}

	if req.COM == "" {
		sb.WriteString(fmt.Sprintf("%d C2G USB Hub Manager(s) Found.\n", foundCount))
	}

	return sb.String()
}

func handleSetState(hm *hubmanager.HubManager, req JSONCommand, save bool) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}
	if req.STATES == "" {
		return "Error: No states specified\n"
	}

	currentStates, err := hm.GetPortStatus(req.COM, 7)
	if err != nil {
		return fmt.Sprintf("Error getting status: %v\n", err)
	}

	statesInput := strings.TrimSpace(req.STATES)

	// Handle custom format "F4,FF,FF,FF"
	if strings.Contains(statesInput, ",") && len(strings.Split(statesInput, ",")) > 1 && !strings.Contains(statesInput, ":") {
		parts := strings.Split(statesInput, ",")
		firstByteHex := strings.TrimSpace(parts[0])
		if len(firstByteHex) == 2 {
			statesInput = "H:" + firstByteHex
		}
	}

	// If it's just a raw hex byte like "F4"
	if len(statesInput) == 2 && !strings.Contains(statesInput, ":") {
		statesInput = "H:" + statesInput
	}

	var sb strings.Builder

	for _, stateStr := range strings.Split(statesInput, ";") {
		stateStr = strings.TrimSpace(stateStr)
		if stateStr == "" {
			continue
		}

		if strings.ToUpper(stateStr) == "1:ALL" {
			for i := 1; i <= 7; i++ {
				currentStates[i] = true
			}
			continue
		}

		if strings.ToUpper(stateStr) == "0:ALL" {
			for i := 1; i <= 7; i++ {
				currentStates[i] = false
			}
			continue
		}

		if strings.HasPrefix(strings.ToUpper(stateStr), "B:") {
			binStr := stateStr[2:]
			val, err := strconv.ParseUint(binStr, 2, 8)
			if err == nil {
				for i := 1; i <= 7; i++ {
					currentStates[i] = ((val >> (i - 1)) & 1) != 0
				}
			} else {
				sb.WriteString(fmt.Sprintf("Invalid binary format: %s\n", stateStr))
			}
			continue
		}

		if strings.HasPrefix(strings.ToUpper(stateStr), "H:") {
			hexStr := stateStr[2:]
			bytes, err := hex.DecodeString(hexStr)
			if err == nil && len(bytes) > 0 {
				val := bytes[0]
				for i := 1; i <= 7; i++ {
					currentStates[i] = ((val >> (i - 1)) & 1) != 0
				}
			} else {
				sb.WriteString(fmt.Sprintf("Invalid hex format: %s\n", stateStr))
			}
			continue
		}

		parts := strings.Split(stateStr, ":")
		if len(parts) == 2 {
			action := parts[0]
			for _, p := range strings.Split(parts[1], ",") {
				p = strings.TrimSpace(p)
				val, err := strconv.Atoi(p)
				if err == nil && val >= 1 && val <= 7 {
					switch strings.ToUpper(action) {
					case "1":
						currentStates[val] = true
					case "0":
						currentStates[val] = false
					case "T":
						currentStates[val] = !currentStates[val]
					}
				}
			}
		} else {
			sb.WriteString(fmt.Sprintf("Invalid format: %s\n", stateStr))
		}
	}

	resp, err := hm.SetPortStatus(req.COM, req.PSW, currentStates, 7)
	if err != nil {
		sb.WriteString(fmt.Sprintf("Error: %v\n", err))
	} else {
		cleanResp := strings.TrimSpace(resp)
		if idx := strings.IndexAny(cleanResp, "\r\n"); idx != -1 {
			cleanResp = cleanResp[:idx]
		}
		if strings.Contains(cleanResp, "E01") {
			return "password error\n"
		}
		if strings.Contains(cleanResp, "E") && !strings.HasPrefix(cleanResp, "G") {
			sb.WriteString(fmt.Sprintf("Result: %s\n", cleanResp))
		}
	}

	if save {
		resp, err := hm.SavePortStates(req.COM, req.PSW)
		if err != nil {
			sb.WriteString(fmt.Sprintf("Error saving: %v\n", err))
		} else {
			cleanResp := strings.TrimSpace(resp)
			if idx := strings.IndexAny(cleanResp, "\r\n"); idx != -1 {
				cleanResp = cleanResp[:idx]
			}
			if strings.Contains(cleanResp, "E01") {
				sb.WriteString("password error (on save)\n")
			} else if strings.Contains(cleanResp, "E") && !strings.HasPrefix(cleanResp, "G") && !strings.Contains(strings.ToUpper(cleanResp), "OK") && !strings.HasPrefix(cleanResp, "SS") {
				sb.WriteString(fmt.Sprintf("Result(Save): %s\n", cleanResp))
			}
		}
	}

	return sb.String()
}

func handleGetState(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}

	resp, err := hm.SendCommand(req.COM, "GW")
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	cleanStatus := strings.TrimSpace(resp)
	if strings.HasPrefix(strings.ToUpper(cleanStatus), "GW") {
		cleanStatus = cleanStatus[2:]
	} else if strings.HasPrefix(strings.ToUpper(cleanStatus), "G") {
		cleanStatus = cleanStatus[1:]
	}

	statusHex := cleanStatus
	var portStatusByte uint64 = 0
	if len(statusHex) >= 2 {
		firstByteHex := statusHex[0:2]
		parsedByte, errByte := strconv.ParseUint(firstByteHex, 16, 8)
		if errByte == nil {
			portStatusByte = parsedByte
		} else {
			portStatusByte, _ = strconv.ParseUint(statusHex, 16, 32)
		}
	} else {
		portStatusByte, _ = strconv.ParseUint(statusHex, 16, 32)
	}

	format := req.FORMAT
	if format == "binary" || format == "B" || format == "-B" {
		return fmt.Sprintf("%08b\n", portStatusByte)
	}

	if format == "hex" || format == "H" || format == "-H" {
		var sb strings.Builder
		for i := 0; i < len(cleanStatus); i += 2 {
			if i+2 <= len(cleanStatus) {
				sb.WriteString(cleanStatus[i : i+2])
				if i+2 < len(cleanStatus) {
					sb.WriteString(" ")
				}
			} else {
				sb.WriteString(cleanStatus[i:])
			}
		}
		return sb.String() + "\n"
	}

	onStr, offStr := formatPortSummary(portStatusByte, 7)
	return fmt.Sprintf("On=%s, Off=%s\n", onStr, offStr)
}

func handleSaveState(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}

	resp, err := hm.SavePortStates(req.COM, req.PSW)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	if strings.Contains(resp, "E01") {
		return "password error\n"
	}

	if strings.Contains(resp, "E") && !strings.Contains(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") && !strings.HasPrefix(resp, "SV") {
		return fmt.Sprintf("Result: %s\n", resp)
	}

	return ""
}

func handleChangePassword(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}
	if req.NEW_PSW == "" {
		return "Error: Missing password arguments\n"
	}
	if len(req.NEW_PSW) < 3 || len(req.NEW_PSW) > 8 {
		return "Error: Password must be 3 to 8 characters\n"
	}

	resp, err := hm.ChangePassword(req.COM, req.PSW, req.NEW_PSW)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	if strings.Contains(resp, "E01") {
		return "password error\n"
	}

	if strings.Contains(resp, "E") && !strings.Contains(resp, "G") {
		if !strings.HasPrefix(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") {
			return fmt.Sprintf("Result: %s\n", resp)
		}
	}

	return ""
}

func handleGetDeviceName(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}

	resp, err := hm.GetDeviceName(req.COM, req.PSW)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	if strings.Contains(resp, "E01") {
		return "password error\n"
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") && !strings.HasPrefix(resp, "B") {
		return "password error\n"
	}

	if idx := strings.IndexAny(resp, "\r\n"); idx != -1 {
		resp = resp[:idx]
	}

	var sb strings.Builder
	for _, ch := range resp {
		if ch >= 32 && ch <= 126 {
			sb.WriteRune(ch)
		}
	}
	resp = sb.String()

	if strings.HasPrefix(resp, "B+") {
		resp = strings.TrimRight(resp[2:], " ")
	} else if strings.HasPrefix(resp, "B") {
		resp = strings.TrimRight(resp[1:], " ")
	}

	if resp == "" {
		resp = "C2G 7-port Managed USB HUB"
	}

	return fmt.Sprintf("%s\n", resp)
}

func handleSetDeviceName(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}
	if req.NAME == "" {
		return "Error: Missing device name argument\n"
	}

	resp, err := hm.SetDeviceName(req.COM, req.PSW, req.NAME)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	if strings.Contains(resp, "E01") {
		return "password error\n"
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") {
		return fmt.Sprintf("Result: %s\n", resp)
	}

	return ""
}

func handleSetVBUS(hm *hubmanager.HubManager, req JSONCommand, enabled bool) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}

	resp, err := hm.SetVBUSPower(req.COM, req.PSW, enabled)
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	if strings.Contains(resp, "E01") {
		return "password error\n"
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") {
		return fmt.Sprintf("Result: %s\n", resp)
	}

	return ""
}

func handleSimpleCommand(hm *hubmanager.HubManager, req JSONCommand, cmdPrefix string) string {
	if req.COM == "" {
		return "Invalid Command!\n"
	}

	var resp string
	var err error

	switch cmdPrefix {
	case "RT":
		resp, err = hm.ResetHub(req.COM, req.PSW)
	case "RS":
		resp, err = hm.RestoreDefault(req.COM, req.PSW)
	}

	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	if strings.Contains(resp, "E01") {
		return "password error\n"
	}

	isActionCmd := (cmdPrefix == "RS" || cmdPrefix == "RT")
	if isActionCmd {
		looksLikeSuccess := strings.HasPrefix(resp, "G") || strings.Contains(strings.ToUpper(resp), "OK")
		if strings.HasPrefix(resp, cmdPrefix) {
			looksLikeSuccess = true
		}
		if !looksLikeSuccess && resp != "" {
			return fmt.Sprintf("Result: %s\n", resp)
		}
	}

	return ""
}
