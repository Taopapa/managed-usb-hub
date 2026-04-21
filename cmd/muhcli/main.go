package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"managed-usb-hub-wails/pkg/hubmanager"
	"managed-usb-hub-wails/pkg/jsonapi"
	"managed-usb-hub-wails/pkg/logger"
)

var (
	hm         *hubmanager.HubManager
	appVersion = "dev"
)

func main() {
	// Initialize Manager
	logger.InitLogger("", true)
	hm = hubmanager.NewHubManager(false)

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	arg := os.Args[1]

	// Handle help command
	if arg == "/?" || arg == "-h" || arg == "--help" {
		printUsage()
		return
	}

	// 1. Handle standard '-' prefix
	if strings.HasPrefix(arg, "-") {
		arg = "/" + arg[1:]
	}

	// 2. Handle Git Bash / Shell path conversion (e.g. /Q -> C:/.../Q or Q:/)
	if !strings.HasPrefix(arg, "/") {
		upper := strings.ToUpper(arg)
		cmds := []string{"Q", "S", "F", "P", "G", "W", "T", "X", "B", "C", "U", "D", "R", "J"}

		found := false
		for _, c := range cmds {
			// Exact match "Q"
			if upper == c {
				arg = "/" + c
				found = true
				break
			}

			// Drive root match "Q:/" or "Q:\"
			if upper == c+":/" || upper == c+":\\" {
				arg = "/" + c
				found = true
				break
			}

			// Path suffix match ".../Q" or "...\Q"
			if strings.HasSuffix(upper, "/"+c) || strings.HasSuffix(upper, "\\"+c) {
				arg = "/" + c
				found = true
				break
			}

			// Path with args match ".../S:..." or "...\S:..."
			// We look for /C: or \C:
			if idx := strings.Index(upper, "/"+c+":"); idx != -1 {
				arg = "/" + arg[idx+1:] // Extract C:... -> /C:... and ensure prefix /
				found = true
				break
			}
			if idx := strings.Index(upper, "\\"+c+":"); idx != -1 {
				arg = "/" + arg[idx+1:]
				found = true
				break
			}

			// Start with "C:" (e.g. S:COM1)
			// But be careful with Q:/ handled above
			if strings.HasPrefix(upper, c+":") {
				// e.g. S:COM1
				arg = "/" + arg
				found = true
				break
			}
		}

		if !found {
			// If not a recognized command pattern and doesn't start with /
			fmt.Println("Invalid Command!")
			return
		}
	}

	if !strings.HasPrefix(arg, "/") {
		// Should have been caught or handled above
		fmt.Println("Invalid Command!")
		return
	}

	parts := strings.Split(arg, ":")
	cmd := strings.ToUpper(parts[0])
	port := ""
	if len(parts) > 1 {
		port = parts[1]
	}

	// Some commands require port
	// /Q can be without port.
	// Others mostly require port.

	switch cmd {
	case "/Q":
		handleQuery(port, os.Args[2:])
	case "/S":
		if port == "" {
			fmt.Println("Invalid Command!") // Port required
			return
		}
		handleSetState(port, os.Args[2:], false)
	case "/F":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSetState(port, os.Args[2:], true)
	case "/P":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleChangePassword(port, os.Args[2:])
	case "/G":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleGetState(port, os.Args[2:])
	case "/W":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSaveState(port, os.Args[2:])
	case "/T":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleGetDeviceName(port, os.Args[2:])
	case "/X":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSetDeviceName(port, os.Args[2:])
	case "/B":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSetVBUSPower(port, true, os.Args[2:])
	case "/C":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSetVBUSPower(port, false, os.Args[2:])
	case "/U":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleGetDeviceUID(port, os.Args[2:])
	case "/D":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSimpleCommand(port, "RS", os.Args[2:])
	case "/R":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSimpleCommand(port, "RT", os.Args[2:])
	case "/J":
		if len(os.Args) < 3 {
			fmt.Println(`{"RES":"Error: Missing JSON string argument"}`)
			return
		}
		jsonStr := os.Args[2]
		resp := jsonapi.ProcessCommand(hm, jsonStr)
		if resp != "" {
			fmt.Println(resp)
		}
	default:
		fmt.Println("Invalid Command!")
	}
}

func printUsage() {
	exeName := "muhcli" // Or "CUSBC" to match image exactly? Using muhcli for reality.
	if len(os.Args) > 0 {
		// exeName = os.Args[0] // Full path is ugly
	}

	version := resolveAppVersion()

	// 根据操作系统显示不同的前缀和端口示例
	isWindows := true
	if os.Getenv("OS") != "Windows_NT" && os.PathSeparator == '/' {
		isWindows = false
	}

	if isWindows {
		fmt.Printf("%s C2G USB Hub Manager API %s for Windows\n", exeName, version)
		fmt.Printf("Usage: %s.exe command [password] [argument]\n", exeName)
	} else {
		fmt.Printf("%s C2G USB Hub Manager API %s for Linux/macOS\n", exeName, version)
		fmt.Printf("Usage: ./%s command [password] [argument]\n", exeName)
	}

	fmt.Println("command:")

	fmt.Println("    /Q        query (no password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /Q [option]\n", exeName)
		fmt.Printf("              %s.exe /Q:COM [option]\n", exeName)
		fmt.Println("    /Q        query all C2G USB Hub Managers")
		fmt.Println("    /Q:COM    query on COM port COMn (n = 1 to 256), or UID0123459789AB")
	} else {
		fmt.Printf("    Usage:    ./%s /Q [option]\n", exeName)
		fmt.Printf("              ./%s /Q:PORT [option]\n", exeName)
		fmt.Println("    /Q        query all C2G USB Hub Managers")
		fmt.Println("    /Q:PORT   query on port (e.g., ttyUSB0 or /dev/ttyUSB0), or UID0123459789AB")
	}
	fmt.Println("    option    -F    output in formatted string")
	fmt.Println("")

	fmt.Println("    /S        set port states (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /S:COM [pass] [states]\n", exeName)
		fmt.Println("    COM       control port COMn (n = 1 to 255), or UID0123459789AB")
	} else {
		fmt.Printf("    Usage:    ./%s /S:PORT [pass] [states]\n", exeName)
		fmt.Println("    PORT      control port (e.g., ttyUSB0 or /dev/ttyUSB0), or UID0123459789AB")
	}
	fmt.Println("    pass      password, default is used if this argument is not specified")
	fmt.Println("    states    port states to be set on, off, toggle or given binary/hex states")
	fmt.Println("              1:3,4     port 3 and 4 on")
	fmt.Println("              0:3       port 3 off")
	fmt.Println("              T:1,2     toggle port 1 and 2 states")
	fmt.Println("              1:ALL     all ports on")
	fmt.Println("              0:ALL     all ports off")
	fmt.Println("              B:0101    binary, port 1 and 3 on, port 2, 4 off")
	fmt.Println("              H:A601    byte hex (little-endian A6 01 ...):")
	fmt.Println("                        port 2, 3, 6, 8, 9 on, the others off")
	fmt.Println("")

	fmt.Println("    /F        set port states and save to flash as the initial states")
	fmt.Println("    Usage:    similar to set port states (/S)")
	fmt.Println("")

	fmt.Println("    /P        change password (3 to 8 characters)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /P:COM [old_password] new_password\n", exeName)
		fmt.Println("    COM       control port COMn (n = 1 to 255), or UID0123459789AB")
	} else {
		fmt.Printf("    Usage:    ./%s /P:PORT [old_password] new_password\n", exeName)
		fmt.Println("    PORT      control port (e.g., ttyUSB0 or /dev/ttyUSB0), or UID0123459789AB")
	}
	fmt.Println("    old_password  old password, default assumed if omitted")
	fmt.Println("    new_password  new password")
	fmt.Println("")

	fmt.Println("    /G        get current port states (no password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /G:COM [option]\n", exeName)
		fmt.Println("    /G:COM    control port COMn (n = 1 to 255), or UID0123459789AB")
	} else {
		fmt.Printf("    Usage:    ./%s /G:PORT [option]\n", exeName)
		fmt.Println("    /G:PORT   control port (e.g., ttyUSB0 or /dev/ttyUSB0), or UID0123459789AB")
	}
	fmt.Println("    option    -B    output in formatted binary string")
	fmt.Println("              -H    output in formatted little-endian hex string")
	fmt.Println("")

	fmt.Println("    /W        save states to flash as the initial states (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /W:COM [pass]\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /W:PORT [pass]\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /D        restore to factory default settings (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /D:COM [pass]\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /D:PORT [pass]\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /R        reset the entire C2G USB Hub Manager (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /R:COM [pass]\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /R:PORT [pass]\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /T        read Device Name (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /T:COM [pass]\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /T:PORT [pass]\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /U        read Device UID (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /U:COM [pass]\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /U:PORT [pass]\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /X        set Device Name (up to 30 chars) (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /X:COM [pass] 'name'\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /X:PORT [pass] 'name'\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /B        power on VBUS (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /B:COM [pass]\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /B:PORT [pass]\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /C        power off VBUS (password is required)")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /C:COM [pass]\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /C:PORT [pass]\n", exeName)
	}
	fmt.Println("")

	fmt.Println("    /J        execute command via JSON payload")
	if isWindows {
		fmt.Printf("    Usage:    %s.exe /J '{\"CMD\":\"Q\"}'\n", exeName)
	} else {
		fmt.Printf("    Usage:    ./%s /J '{\"CMD\":\"Q\"}'\n", exeName)
	}
	fmt.Println("")

	fmt.Println("Examples: To control port 3 of the hub, follow the steps below:")
	if isWindows {
		fmt.Printf("    %s.exe /Q  (auto search all the hubs connected)\n", exeName)
		fmt.Printf("    %s.exe /S:COM3 0:3  (turn OFF port 3, assumed COM3 is found by the /Q command)\n", exeName)
		fmt.Printf("    %s.exe /S:COM3 1:3  (turn ON port 3)\n", exeName)
	} else {
		fmt.Printf("    ./%s /Q  (auto search all the hubs connected)\n", exeName)
		fmt.Printf("    ./%s /S:PORT 0:3  (turn OFF port 3, assumed PORT is found by the /Q command)\n", exeName)
		fmt.Printf("    ./%s /S:PORT 1:3  (turn ON port 3)\n", exeName)
	}
	fmt.Println("")

	fmt.Println("Examples to control ports with input JSON string")
	if isWindows {
		fmt.Printf("    ./%s /J \"{\\\"CMD\\\":\\\"Q\\\"}\"  (Query all Hubs)\n", exeName)
		fmt.Printf("    ./%s /J \"{\\\"CMD\\\":\\\"SPST\\\",\\\"COM\\\":\\\"COM3\\\",\\\"PSW\\\":\\\"pass\\\",\\\"STATES\\\":\\\"F4,FF,FF,FF\\\"}\"  (turn Off ports 1,2,4)\n", exeName)
	} else {
		fmt.Printf("    ./%s /J '{\"CMD\":\"Q\"}'  (Query all Hubs)\n", exeName)
		fmt.Printf("    ./%s /J '{\"CMD\":\"SPST\",\"COM\":\"COM3\",\"PSW\":\"pass\",\"STATES\":\"F4,FF,FF,FF\"}'  (turn Off ports 1,2,4)\n", exeName)
	}
	fmt.Println("")
}

// Helpers

func resolveAppVersion() string {
	if appVersion != "" && appVersion != "dev" {
		return appVersion
	}

	if version := readVersionFromWailsJSON(); version != "" {
		return version
	}

	return appVersion
}

func readVersionFromWailsJSON() string {
	type wailsInfo struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}

	candidates := []string{}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "wails.json"))
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "wails.json"),
			filepath.Join(exeDir, "..", "wails.json"),
			filepath.Join(exeDir, "..", "..", "wails.json"),
		)
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := seen[absPath]; ok {
			continue
		}
		seen[absPath] = struct{}{}

		data, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		var cfg wailsInfo
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}

		if cfg.Info.ProductVersion != "" {
			return cfg.Info.ProductVersion
		}
	}

	return ""
}

func hexStringToString(hexStr string) string {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return hexStr // Return original if not hex
	}
	return string(bytes)
}

func padPassword(p string) string {
	if len(p) >= 8 {
		return p[:8]
	}
	return fmt.Sprintf("%-8s", p)
}

func getPassword(args []string) (string, []string) {
	// Heuristic: Check if first arg looks like a state or option
	// If it contains ':', it's likely a state.
	// If args is empty, return default.
	if len(args) == 0 {
		return "pass    ", args
	}

	first := args[0]
	// Check for state patterns
	if strings.Contains(first, ":") || strings.HasPrefix(strings.ToUpper(first), "-") || strings.HasPrefix(strings.ToUpper(first), "/") {
		return "pass    ", args
	}

	// Assuming it's a password
	return padPassword(first), args[1:]
}

// Handlers

func normalizeCommandName(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	if strings.HasPrefix(cmd, "-") {
		cmd = "/" + cmd[1:]
	}
	if !strings.HasPrefix(cmd, "/") {
		cmd = "/" + cmd
	}
	return strings.ToUpper(cmd)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeCLIOption(opt string) string {
	opt = strings.TrimSpace(opt)
	if opt == "" {
		return ""
	}
	if strings.HasPrefix(opt, "/") || strings.HasPrefix(opt, "-") {
		return strings.ToUpper(opt)
	}
	return "-" + strings.ToUpper(opt)
}

func parseQueryFormat(args []string) string {
	for _, arg := range args {
		switch normalizeCLIOption(arg) {
		case "-F":
			return "formatted"
		}
	}
	return "default"
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

func parseStatusByte(statusHex string) uint64 {
	if len(statusHex) >= 2 {
		firstByteHex := statusHex[0:2]
		parsedByte, errByte := strconv.ParseUint(firstByteHex, 16, 8)
		if errByte == nil {
			return parsedByte
		}
	}

	val, _ := strconv.ParseUint(statusHex, 16, 64)
	return val
}

func parsePortCountFromGoData(goData string) int {
	// Currently, this product only has 7 ports. Hardcoding to 7.
	return 7
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

func formatQueryDevice(d hubmanager.DeviceInfo, formatted bool) string {
	statusHex := cleanStatusHex(firstNonEmpty(d.RawLedStatus, d.LedStatus))
	portStatusByte := parseStatusByte(statusHex)
	numPorts := parsePortCountFromGoData(d.GoData)
	onStr, offStr := formatPortSummary(portStatusByte, numPorts)
	fwVer := extractFirmwareVersion(d.AsciiResponse)

	if formatted {
		return fmt.Sprintf(
			"Path=%s, Description=%q, Ports=%d, On=%s, Off=%s, FW=%s, GW=%s, CM=%s",
			d.Path,
			d.AsciiResponse,
			numPorts,
			onStr,
			offStr,
			fwVer,
			statusHex,
			strings.TrimSpace(d.GoData),
		)
	}

	return fmt.Sprintf("%s, %d ports, On=%s, Off=%s, FW=%s", d.Path, numPorts, onStr, offStr, fwVer)
}

func handleQuery(port string, args []string) {
	// Query all or specific
	var devices []hubmanager.DeviceInfo
	format := parseQueryFormat(args)

	if port != "" {
		// Single device
		// We can't use AutoSearchProbe because it scans all.
		// But AutoSearchProbe is what we have.
		// Let's use AutoSearchProbe anyway for now as it's safer,
		// or construct a single probe if we knew how.
		// Reusing AutoSearchProbe is easier but slower.
		// Let's manually probe the specific port.
		// hm.probeDevice is private. We should probably expose it or duplicate logic.
		// For simplicity, let's just OpenPort and run commands to simulate probe info.

		// If port is just "COM" or invalid format, we should probably catch it
		if port == "COM" {
			fmt.Println("Failed to open port")
			return
		}

		// 1. Get ID (?S)
		idResp, err := hm.SendCommand(port, "?S")
		if err != nil {
			fmt.Println("Failed to open port")
			return
		}

		// Check if it's a valid device
		idCheck := strings.ToUpper(strings.TrimSpace(idResp))
		if idCheck == "" || (!strings.Contains(idCheck, "CENTOS") && !strings.Contains(idCheck, "C2G")) {
			fmt.Println("Failed to open port")
			return
		}

		// 2. Get Status (GW)
		gpResp, err := hm.SendCommand(port, "GW")

		if err != nil {
			fmt.Println("Failed to open port")
			return
		}

		goResp, err := hm.SendCommand(port, "CM")
		if err != nil {
			fmt.Println("Failed to open port")
			return
		}

		devices = append(devices, hubmanager.DeviceInfo{
			Path:          port,
			AsciiResponse: idResp,
			ProbeResponse: idResp,
			LedStatus:     fmt.Sprintf("%X", strings.TrimSpace(gpResp)),
			GoData:        fmt.Sprintf("%X", strings.TrimSpace(goResp)),
			RawLedStatus:  strings.TrimSpace(gpResp),
			RawGoData:     strings.TrimSpace(goResp),
		})

	} else {
		// Query all - match GUI behavior
		// fmt.Println("Scanning for Managed USB Hubs...")
		var err error
		devices, err = hm.AutoSearchProbe()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	foundCount := 0
	for _, d := range devices {
		foundCount++
		output := formatQueryDevice(d, format == "formatted")
		if port == "" {
			fmt.Printf(" %s\n", output)
		} else {
			fmt.Println(output)
		}
	}

	if port == "" {
		fmt.Printf("%d C2G USB Hub Manager(s) Found.\n", foundCount)
	}
}

func handleSetState(port string, args []string, save bool) {
	pass, states := getPassword(args)

	if len(states) == 0 {
		fmt.Println("Error: No states specified")
		return
	}

	// 1. Get Current Status
	currentStates, err := hm.GetPortStatus(port, 7) // Using 7 as default max ports for CLI display if not specified, or we can use 16
	if err != nil {
		fmt.Printf("Error getting status: %v\n", err)
		return
	}

	for _, stateStr := range states {
		// Handle 1:ALL
		if strings.ToUpper(stateStr) == "1:ALL" {
			for i := 1; i <= 16; i++ {
				currentStates[i] = true
			}
			continue
		}

		// Handle 0:ALL
		if strings.ToUpper(stateStr) == "0:ALL" {
			for i := 1; i <= 16; i++ {
				currentStates[i] = false
			}
			continue
		}

		// Handle B:0101 (Binary)
		if strings.HasPrefix(strings.ToUpper(stateStr), "B:") {
			binStr := stateStr[2:]
			val, err := strconv.ParseUint(binStr, 2, 8)
			if err == nil {
				for i := 1; i <= 8; i++ {
					currentStates[i] = ((val >> (i - 1)) & 1) != 0
				}
			} else {
				fmt.Printf("Invalid binary format: %s\n", stateStr)
			}
			continue
		}

		// Handle H:A6 (Hex)
		if strings.HasPrefix(strings.ToUpper(stateStr), "H:") {
			hexStr := stateStr[2:]
			bytes, err := hex.DecodeString(hexStr)
			if err == nil && len(bytes) > 0 {
				val := bytes[0]
				for i := 1; i <= 8; i++ {
					currentStates[i] = ((val >> (i - 1)) & 1) != 0
				}
			} else {
				fmt.Printf("Invalid hex format: %s\n", stateStr)
			}
			continue
		}

		parts := strings.Split(stateStr, ":")
		if len(parts) != 2 {
			fmt.Printf("Invalid format: %s\n", stateStr)
			continue
		}

		action := parts[0]
		portsStr := parts[1]

		portNums := []int{}
		for _, p := range strings.Split(portsStr, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			val, err := strconv.Atoi(p)
			if err == nil {
				portNums = append(portNums, val)
			}
		}

		for _, p := range portNums {
			if p < 1 || p > 16 {
				continue
			}
			switch strings.ToUpper(action) {
			case "1":
				currentStates[p] = true
			case "0":
				currentStates[p] = false
			case "T":
				currentStates[p] = !currentStates[p]
			}
		}
	}

	resp, err := hm.SetPortStatus(port, pass, currentStates, 7) // Using 7 as default mask width
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Clean up response for parsing
		cleanResp := strings.TrimSpace(resp)
		if idx := strings.IndexAny(cleanResp, "\r\n"); idx != -1 {
			cleanResp = cleanResp[:idx]
		}

		// Handle E01 specifically
		if strings.Contains(cleanResp, "E01") {
			fmt.Println("password error")
			return
		}

		// Only print if it looks like an error
		if strings.Contains(cleanResp, "E") && !strings.HasPrefix(cleanResp, "G") {
			fmt.Printf("Result: %s\n", cleanResp)
		}
	}

	if save {
		// Save to flash
		resp, err := hm.SavePortStates(port, pass)
		if err != nil {
			fmt.Printf("Error saving: %v\n", err)
		} else {
			// Clean up response for parsing
			cleanResp := strings.TrimSpace(resp)
			if idx := strings.IndexAny(cleanResp, "\r\n"); idx != -1 {
				cleanResp = cleanResp[:idx]
			}

			// Handle E01 specifically
			if strings.Contains(cleanResp, "E01") {
				fmt.Println("password error (on save)")
			} else if strings.Contains(cleanResp, "E") && !strings.HasPrefix(cleanResp, "G") && !strings.Contains(strings.ToUpper(cleanResp), "OK") && !strings.HasPrefix(cleanResp, "SS") {
				// Check for SS error
				fmt.Printf("Result(Save): %s\n", cleanResp)
			}
		}
	}
}

func handleGetState(port string, args []string) {
	resp, err := hm.SendCommand(port, "GW")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	cleanStatus := strings.TrimSpace(resp)
	if strings.HasPrefix(strings.ToUpper(cleanStatus), "GW") {
		cleanStatus = cleanStatus[2:]
	} else if strings.HasPrefix(strings.ToUpper(cleanStatus), "G") {
		cleanStatus = cleanStatus[1:]
	}

	// Check for formatting flags
	format := "default"
	if len(args) > 0 {
		flag := strings.ToUpper(args[0])
		if flag == "-B" || flag == "/B" {
			format = "binary"
		} else if flag == "-H" || flag == "/H" {
			format = "hex"
		}
	}

	statusHex := cleanStatus

	// Parse the first byte (2 characters) as the status mask.
	var portStatusByte uint64 = 0
	if len(statusHex) >= 2 {
		firstByteHex := statusHex[0:2]
		parsedByte, errByte := strconv.ParseUint(firstByteHex, 16, 8)
		if errByte == nil {
			portStatusByte = parsedByte
		} else {
			val, _ := strconv.ParseUint(statusHex, 16, 32)
			portStatusByte = val
		}
	} else {
		val, _ := strconv.ParseUint(statusHex, 16, 32)
		portStatusByte = val
	}

	if format == "binary" {
		// Output binary string for 8 bits (Ports 1-8)
		// Or 7 bits for 7 ports?
		// Standard output usually shows full byte.
		// "B:0101" implies 4 bits or just bits.
		// Let's output 8 bits.
		// Note: Bit 0 is Port 1.
		// Standard binary print: 1100 -> Bit 3, 2.
		// So "00001100" means Ports 3, 4 ON.
		fmt.Printf("%08b\n", portStatusByte)
		return
	}

	if format == "hex" {
		// Output hex string.
		// User wants "formatted little-endian hex string".
		// Our cleanStatus is already the hex string.
		// If cleanStatus is "8CFFFFFF", byte 0 is 8C.
		// So just printing it is correct little-endian hex string representation (byte order).
		// Prefix with H:? Or just raw?
		// Usage for /G says: "output in formatted little-endian hex string".
		// Usage for /S says: "H:A601".
		// Let's assume just the hex string without H: prefix to match "formatted string".
		// Or maybe space separated? "A6 01 ..."?
		// Given the description "A6 01 ...", space separation is likely.

		// Let's format cleanStatus with spaces every 2 chars.
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
		fmt.Println(sb.String())
		return
	}

	var onPorts []string
	var offPorts []string

	numPorts := 7 // Assuming 7-port hub

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

	fmt.Printf("On=%s, Off=%s\n", onStr, offStr)
}

func handleSaveState(port string, args []string) {
	pass, _ := getPassword(args)

	resp, err := hm.SavePortStates(port, pass)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Handle E01 specifically
		if strings.Contains(resp, "E01") {
			fmt.Println("password error")
			return
		}

		if strings.Contains(resp, "E") && !strings.Contains(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") && !strings.HasPrefix(resp, "SV") {
			fmt.Printf("Result: %s\n", resp)
		}
	}
}

func handleChangePassword(port string, args []string) {
	oldPass := ""
	newPass := ""

	if len(args) == 1 {
		oldPass = "pass    "
		newPass = args[0]
	} else if len(args) >= 2 {
		oldPass = padPassword(args[0])
		newPass = args[1]
	} else {
		fmt.Println("Error: Missing password arguments")
		return
	}

	if len(newPass) < 3 || len(newPass) > 8 {
		fmt.Println("Error: Password must be 3 to 8 characters")
		return
	}

	resp, err := hm.ChangePassword(port, oldPass, newPass)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Handle E01 specifically
		if strings.Contains(resp, "E01") {
			fmt.Println("password error")
			return
		}

		// Only print if it looks like an error
		if strings.Contains(resp, "E") && !strings.Contains(resp, "G") {
			if !strings.HasPrefix(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") {
				fmt.Printf("Result: %s\n", resp)
			}
		}
	}
}

func handleGetDeviceUID(port string, args []string) {
	pass, _ := getPassword(args)

	// Pad password to 8 chars
	pass = padPassword(pass)

	resp, err := hm.GetDeviceUID(port, pass)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Handle E01 specifically
		if strings.Contains(resp, "E01") {
			fmt.Println("password error")
			return
		}

		// Check for E02 or other errors
		if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") && !strings.HasPrefix(resp, "I") {
			fmt.Println("password error")
			return
		}

		// Clean up the response like frontend does
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

		fmt.Printf("%s\n", resp)
	}
}

func handleGetDeviceName(port string, args []string) {
	pass, _ := getPassword(args)

	// Pad password to 8 chars
	pass = padPassword(pass)

	resp, err := hm.GetDeviceName(port, pass)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Handle E01 specifically
		if strings.Contains(resp, "E01") {
			fmt.Println("password error")
			return
		}

		// Check for E02 or other errors
		if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") && !strings.HasPrefix(resp, "B") {
			// Some commands might just return "E02" without anything else
			fmt.Println("password error")
			return
		}

		// Clean up the response like frontend does
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

		fmt.Printf("%s\n", resp)
	}
}

func handleSetDeviceName(port string, args []string) {
	pass, nameArgs := getPassword(args)
	if len(nameArgs) == 0 {
		fmt.Println("Error: Missing device name argument")
		return
	}

	name := strings.Join(nameArgs, " ")

	// Remove surrounding quotes if user provided them
	if (strings.HasPrefix(name, "'") && strings.HasSuffix(name, "'")) ||
		(strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"")) {
		name = name[1 : len(name)-1]
	}

	// Pad password to 8 chars
	pass = padPassword(pass)

	resp, err := hm.SetDeviceName(port, pass, name)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Handle E01 specifically
		if strings.Contains(resp, "E01") {
			fmt.Println("password error")
			return
		}

		if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") {
			fmt.Printf("Result: %s\n", resp)
		}
	}
}

func handleSetVBUSPower(port string, enabled bool, args []string) {
	pass, _ := getPassword(args)

	// Pad password to 8 chars
	pass = padPassword(pass)

	resp, err := hm.SetVBUSPower(port, pass, enabled)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Handle E01 specifically
		if strings.Contains(resp, "E01") {
			fmt.Println("password error")
			return
		}

		if strings.Contains(resp, "E") && !strings.HasPrefix(resp, "G") {
			fmt.Printf("Result: %s\n", resp)
		}
	}
}

func handleSimpleCommand(port string, cmdPrefix string, args []string) {
	pass, _ := getPassword(args)

	var resp string
	var err error

	switch cmdPrefix {
	case "RT":
		resp, err = hm.ResetHub(port, pass)
	case "RS":
		resp, err = hm.RestoreDefault(port, pass)
	case "SV":
		resp, err = hm.SavePortStates(port, pass)
	default:
		cmd := cmdPrefix
		if cmdPrefix == "SS" || cmdPrefix == "GD" {
			cmd += pass
		}
		resp, err = hm.SendCommand(port, cmd)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Handle E01 specifically
		if strings.Contains(resp, "E01") {
			fmt.Println("password error")
			return
		}

		// Silent success for commands that perform actions (SS, RS, RT)
		// Assuming SS returns something like "OK" or echoes.

		isActionCmd := (cmdPrefix == "SS" || cmdPrefix == "RS" || cmdPrefix == "RT")

		if isActionCmd {
			// Print ONLY if it looks like an error
			// If it contains "E" and NOT "G" and NOT "OK", print it.
			// Or simpler: if it starts with E?
			// Let's stick to the heuristic:
			// Not starting with G (status) AND Not containing OK AND Containing E (Error?)
			// Or just: If it contains "Error" or starts with "E" followed by digits?
			// Let's use the same logic as /P

			looksLikeSuccess := strings.HasPrefix(resp, "G") || strings.Contains(strings.ToUpper(resp), "OK")
			// SS usually returns nothing or OK? Or maybe echoes command?
			// If it echoes "SS...", that's success?
			if strings.HasPrefix(resp, cmdPrefix) {
				looksLikeSuccess = true
			}

			if !looksLikeSuccess {
				// If it's empty, consider success?
				if resp != "" {
					fmt.Printf("Result: %s\n", resp)
				}
			}
		} else {
			// GD -> Print result (cleaning up any raw prefix if present)
			cleanResp := strings.TrimSpace(resp)
			if idx := strings.IndexAny(cleanResp, "\r\n"); idx != -1 {
				cleanResp = cleanResp[:idx]
			}

			if strings.HasPrefix(cleanResp, "GD") {
				cleanResp = strings.TrimSpace(cleanResp[2:])
			} else if strings.HasPrefix(cleanResp, "E01") {
				// Some firmware versions might enforce password on GD or it leaked
				fmt.Println("password error")
				return
			}
			fmt.Printf("%s\n", cleanResp)
		}
	}
}
