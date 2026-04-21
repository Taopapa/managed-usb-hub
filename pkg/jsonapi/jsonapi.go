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
	UID     string `json:"UID,omitempty"`
	FORMAT  string `json:"FORMAT,omitempty"`
}

func padPassword(p string) string {
	if len(p) >= 8 {
		return p[:8]
	}
	return fmt.Sprintf("%-8s", p)
}

func errorResponse(msg string) string {
	b, _ := json.Marshal(map[string]string{"RES": msg})
	return string(b)
}

func successResponse(data map[string]interface{}) string {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["RES"] = "OK"
	b, _ := json.Marshal(data)
	return string(b)
}

// ProcessCommand executes the JSON command and returns a JSON response string.
func ProcessCommand(hm *hubmanager.HubManager, jsonStr string) string {
	var req JSONCommand
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return errorResponse("Invalid JSON format")
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
	case "S", "SPST":
		return handleSetState(hm, req, false)
	case "F":
		return handleSetState(hm, req, true)
	case "P":
		return handleChangePassword(hm, req)
	case "G":
		return handleGetState(hm, req)
	case "W":
		return handleSaveState(hm, req)
	case "T":
		return handleGetDeviceName(hm, req)
	case "U":
		return handleGetDeviceUID(hm, req)
	case "X":
		return handleSetDeviceName(hm, req)
	case "Y":
		return handleSetDeviceUID(hm, req)
	case "B":
		return handleSetVBUS(hm, req, true)
	case "C":
		return handleSetVBUS(hm, req, false)
	case "D":
		return handleSimpleCommand(hm, req, "RS")
	case "R":
		return handleSimpleCommand(hm, req, "RT")
	default:
		return errorResponse("Invalid Command!")
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

type HubInfo struct {
	UID         string `json:"UID"`
	Description string `json:"Description"`
	PID         string `json:"PID"`
	NPorts      string `json:"nPorts"`
	PortStates  string `json:"Port_States"`
	FWVer       string `json:"FW_Ver"`
	COMNo       string `json:"COM_No"`
}

func handleQuery(hm *hubmanager.HubManager, req JSONCommand) string {
	var devices []hubmanager.DeviceInfo
	var err error

	if req.COM != "" {
		idResp, err := hm.SendCommand(req.COM, "?S")
		if err != nil {
			return errorResponse("Failed to open port")
		}
		idCheck := strings.ToUpper(strings.TrimSpace(idResp))
		if idCheck == "" || (!strings.Contains(idCheck, "CENTOS") && !strings.Contains(idCheck, "C2G")) {
			return errorResponse("Failed to open port")
		}

		gpResp, _ := hm.SendCommand(req.COM, "GW")
		goResp, _ := hm.SendCommand(req.COM, "CM")
		uidResp, _ := hm.GetDeviceUID(req.COM, req.PSW)
		nameResp, _ := hm.GetDeviceName(req.COM, req.PSW)

		if strings.HasPrefix(uidResp, "I+") {
			uidResp = strings.TrimRight(uidResp[2:], " \r\n")
		} else if strings.HasPrefix(uidResp, "I") {
			uidResp = strings.TrimRight(uidResp[1:], " \r\n")
		}
		if strings.HasPrefix(nameResp, "B+") {
			nameResp = strings.TrimRight(nameResp[2:], " \r\n")
		} else if strings.HasPrefix(nameResp, "B") {
			nameResp = strings.TrimRight(nameResp[1:], " \r\n")
		}

		devices = append(devices, hubmanager.DeviceInfo{
			Path:          req.COM,
			AsciiResponse: idResp,
			DeviceUID:     strings.TrimSpace(uidResp),
			DeviceName:    strings.TrimSpace(nameResp),
			LedStatus:     fmt.Sprintf("%X", strings.TrimSpace(gpResp)),
			GoData:        fmt.Sprintf("%X", strings.TrimSpace(goResp)),
			RawLedStatus:  strings.TrimSpace(gpResp),
			RawGoData:     strings.TrimSpace(goResp),
		})
	} else {
		devices, err = hm.AutoSearchProbe()
		if err != nil {
			return errorResponse(fmt.Sprintf("Error: %v", err))
		}
	}

	var infos []HubInfo
	numPorts := 7 // Hardcoded to 7 based on main.go

	for _, d := range devices {
		statusHex := cleanStatusHex(d.RawLedStatus)

		pid := "0002"
		if strings.Contains(d.AsciiResponse, "CENTOS0002") {
			pid = "0002"
		}

		uid := d.DeviceUID
		if uid == "" {
			uid = "Unknown"
		}

		desc := d.DeviceName
		if desc == "" {
			desc = "7-port Managed USB Hub"
		}

		fwVer := extractFirmwareVersion(d.AsciiResponse)

		infos = append(infos, HubInfo{
			UID:         uid,
			Description: desc,
			PID:         pid,
			NPorts:      strconv.Itoa(numPorts),
			PortStates:  statusHex,
			FWVer:       fwVer,
			COMNo:       d.Path,
		})
	}

	if infos == nil {
		infos = []HubInfo{}
	}

	return successResponse(map[string]interface{}{
		"HUBS": strconv.Itoa(len(infos)),
		"INFO": infos,
	})
}

func handleSetState(hm *hubmanager.HubManager, req JSONCommand, save bool) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}
	if req.STATES == "" {
		return errorResponse("No states specified")
	}

	currentStates, err := hm.GetPortStatus(req.COM, 7)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error getting status: %v", err))
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
				return errorResponse(fmt.Sprintf("Invalid binary format: %s", stateStr))
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
				return errorResponse(fmt.Sprintf("Invalid hex format: %s", stateStr))
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
			return errorResponse(fmt.Sprintf("Invalid format: %s", stateStr))
		}
	}

	resp, err := hm.SetPortStatus(req.COM, req.PSW, currentStates, 7)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	} else {
		cleanResp := strings.TrimSpace(resp)
		if idx := strings.IndexAny(cleanResp, "\r\n"); idx != -1 {
			cleanResp = cleanResp[:idx]
		}
		if strings.Contains(cleanResp, "E01") {
			return errorResponse("password error")
		}
		if strings.Contains(cleanResp, "E") && !strings.HasPrefix(cleanResp, "G") {
			return errorResponse(fmt.Sprintf("Result: %s", cleanResp))
		}
	}

	if save {
		resp, err := hm.SavePortStates(req.COM, req.PSW)
		if err != nil {
			return errorResponse(fmt.Sprintf("Error saving: %v", err))
		} else {
			cleanResp := strings.TrimSpace(resp)
			if idx := strings.IndexAny(cleanResp, "\r\n"); idx != -1 {
				cleanResp = cleanResp[:idx]
			}
			if strings.Contains(cleanResp, "E01") {
				return errorResponse("password error (on save)")
			} else if strings.Contains(cleanResp, "E") && !strings.HasPrefix(cleanResp, "G") && !strings.Contains(strings.ToUpper(cleanResp), "OK") && !strings.HasPrefix(cleanResp, "SS") {
				return errorResponse(fmt.Sprintf("Result(Save): %s", cleanResp))
			}
		}
	}

	return successResponse(nil)
}

func handleGetState(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}

	resp, err := hm.SendCommand(req.COM, "GW")
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
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
		return successResponse(map[string]interface{}{"Port_States": fmt.Sprintf("%08b", portStatusByte)})
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
		return successResponse(map[string]interface{}{"Port_States": sb.String()})
	}

	onStr, offStr := formatPortSummary(portStatusByte, 7)
	return successResponse(map[string]interface{}{
		"On":  onStr,
		"Off": offStr,
	})
}

func handleSaveState(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}

	resp, err := hm.SavePortStates(req.COM, req.PSW)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	if strings.Contains(resp, "E") && !strings.Contains(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") && !strings.HasPrefix(resp, "SV") {
		return errorResponse(fmt.Sprintf("Result: %s", resp))
	}

	return successResponse(nil)
}

func handleChangePassword(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}
	if req.NEW_PSW == "" {
		return errorResponse("Missing password arguments")
	}
	if len(req.NEW_PSW) < 3 || len(req.NEW_PSW) > 8 {
		return errorResponse("Password must be 3 to 8 characters")
	}

	resp, err := hm.ChangePassword(req.COM, req.PSW, req.NEW_PSW)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	if strings.Contains(resp, "E") && !strings.Contains(resp, "G") {
		if !strings.HasPrefix(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") {
			return errorResponse(fmt.Sprintf("Result: %s", resp))
		}
	}

	return successResponse(nil)
}

func handleGetDeviceUID(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}

	resp, err := hm.GetDeviceUID(req.COM, req.PSW)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") && !strings.HasPrefix(resp, "I") {
		return errorResponse("password error")
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

	if strings.HasPrefix(resp, "I+") {
		resp = strings.TrimRight(resp[2:], " ")
	} else if strings.HasPrefix(resp, "I") {
		resp = strings.TrimRight(resp[1:], " ")
	}

	return successResponse(map[string]interface{}{"UID": resp})
}

func handleGetDeviceName(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}

	resp, err := hm.GetDeviceName(req.COM, req.PSW)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") && !strings.HasPrefix(resp, "B") {
		return errorResponse("password error")
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

	return successResponse(map[string]interface{}{"NAME": resp})
}

func handleSetDeviceName(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}
	if req.NAME == "" {
		return errorResponse("Missing device name argument")
	}

	resp, err := hm.SetDeviceName(req.COM, req.PSW, req.NAME)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") {
		return errorResponse(fmt.Sprintf("Result: %s", resp))
	}

	return successResponse(nil)
}

func handleSetDeviceUID(hm *hubmanager.HubManager, req JSONCommand) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}
	if req.UID == "" {
		return errorResponse("Missing device uid argument")
	}

	resp, err := hm.SetDeviceUID(req.COM, req.PSW, req.UID)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") {
		return errorResponse(fmt.Sprintf("Result: %s", resp))
	}

	return successResponse(nil)
}

func handleSetVBUS(hm *hubmanager.HubManager, req JSONCommand, enabled bool) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
	}

	resp, err := hm.SetVBUSPower(req.COM, req.PSW, enabled)
	if err != nil {
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") {
		return errorResponse(fmt.Sprintf("Result: %s", resp))
	}

	return successResponse(nil)
}

func handleSimpleCommand(hm *hubmanager.HubManager, req JSONCommand, cmdPrefix string) string {
	if req.COM == "" {
		return errorResponse("Invalid Command!")
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
		return errorResponse(fmt.Sprintf("Error: %v", err))
	}

	if strings.Contains(resp, "E01") {
		return errorResponse("password error")
	}

	isActionCmd := (cmdPrefix == "RS" || cmdPrefix == "RT")
	if isActionCmd {
		looksLikeSuccess := strings.HasPrefix(resp, "G") || strings.Contains(strings.ToUpper(resp), "OK")
		if strings.HasPrefix(resp, cmdPrefix) {
			looksLikeSuccess = true
		}
		if !looksLikeSuccess && resp != "" {
			return errorResponse(fmt.Sprintf("Result: %s", resp))
		}
	}

	return successResponse(nil)
}
