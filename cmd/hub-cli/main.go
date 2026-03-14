package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"managed-usb-hub-wails/pkg/hubcli"
)

var (
	hm *hubcli.HubManager
)

func main() {
	// Initialize Manager
	logger := func(deviceID, level, message string) {
		// Do not log anything in CLI to match strict output requirements
	}
	hm = hubcli.NewHubManager(logger)

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	arg := os.Args[1]

	// 1. Handle standard '-' prefix
	if strings.HasPrefix(arg, "-") {
		arg = "/" + arg[1:]
	}

	// 2. Handle Git Bash / Shell path conversion (e.g. /Q -> C:/.../Q or Q:/)
	if !strings.HasPrefix(arg, "/") {
		upper := strings.ToUpper(arg)
		cmds := []string{"Q", "S", "F", "P", "G", "W", "T", "X", "U", "D", "R", "J"}

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

	// Handle /J separately as it might not follow /Cmd:Port format strictly
	if strings.HasPrefix(strings.ToUpper(arg), "/J") {
		if len(os.Args) < 3 {
			fmt.Println("Invalid Command!") // Missing args for J
			return
		}
		handleJSON(os.Args[2])
		return
	}

	// Parse Command and COM port
	// Format: /CMD:COMn or /CMD
	// Note: The image shows /Q:COM or /Q.

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
		handleQuery(port)
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
		handleSimpleCommand(port, "GD", os.Args[2:]) // Guessing GD for Get Description
	case "/X":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSetDescription(port, os.Args[2:])
	case "/U":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSimpleCommand(port, "UID", nil) // Guessing UID
	case "/D":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSimpleCommand(port, "RD", os.Args[2:])
	case "/R":
		if port == "" {
			fmt.Println("Invalid Command!")
			return
		}
		handleSimpleCommand(port, "RH", os.Args[2:])
	default:
		fmt.Println("Invalid Command!")
	}
}

func printUsage() {
	exeName := "hub-cli" // Or "CUSBC" to match image exactly? Using hub-cli for reality.
	if len(os.Args) > 0 {
		// exeName = os.Args[0] // Full path is ugly
	}

	fmt.Printf("%s Managed USB Hub API v2.22 for Windows\n", exeName)
	fmt.Printf("Usage: %s command [password] [argument]\n", exeName)
	fmt.Println("command:")

	fmt.Println("    /Q        query (no password is required)")
	fmt.Printf("    Usage:    %s /Q [option]\n", exeName)
	fmt.Printf("              %s /Q:COM [option]\n", exeName)
	fmt.Println("    /Q        query all Managed USB Hubs")
	fmt.Println("    /Q:COM    query on COM port COMn (n = 1 to 256), or UID0123459789AB")
	// fmt.Println("    option    -F    output in formatted string")
	fmt.Println("")

	fmt.Println("    /S        set port states (password is required)")
	fmt.Printf("    Usage:    %s /S:COM [pass] [states]\n", exeName)
	fmt.Println("    COM       control port COMn (n = 1 to 255), or UID0123459789AB")
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

	fmt.Println("    /P        change password (8 characters maximum)")
	fmt.Printf("    Usage:    %s /P:COM [old_password] new_password\n", exeName)
	fmt.Println("    COM       control port COMn (n = 1 to 255), or UID0123459789AB")
	fmt.Println("    old_password  old password, default assumed if omitted")
	fmt.Println("    new_password  new password")
	fmt.Println("")

	fmt.Println("    /G        get current port states (no password is required)")
	fmt.Printf("    Usage:    %s /G:COM [option]\n", exeName)
	fmt.Println("    /G:COM    control port COMn (n = 1 to 255), or UID0123459789AB")
	fmt.Println("    option    -B    output in formatted binary string")
	fmt.Println("              -H    output in formatted little-endian hex string")
	fmt.Println("")

	fmt.Println("    /W        save states to flash as the initial states (password is required)")
	fmt.Printf("    Usage:    %s /W:COM [pass]\n", exeName)
	fmt.Println("")

	// fmt.Println("    /T        read Hub description string")
	// fmt.Printf("    Usage:    %s /T:COM\n", exeName)
	// fmt.Println("")

	// fmt.Println("    /X        write Hub description string (password is required)")
	// fmt.Printf("    Usage:    %s /X:COM [pass] 'description string'\n", exeName)
	// fmt.Println("")

	// fmt.Println("    /U        read Hub UID (Unique ID) string")
	// fmt.Printf("    Usage:    %s /U:COM\n", exeName)
	// fmt.Println("")

	fmt.Println("    /D        restore to factory default settings (password is required)")
	fmt.Printf("    Usage:    %s /D:COM [pass]\n", exeName)
	fmt.Println("")

	fmt.Println("    /R        reset the entire Managed USB Hub (password is required)")
	fmt.Printf("    Usage:    %s /R:COM [pass]\n", exeName)
	fmt.Println("")

	// fmt.Println("    /J        Commands in JSON format")
	// fmt.Printf("    Usage:    %s /J \"JSON_string\"\n", exeName)
	// fmt.Println("    \"JSON_string\" is input command and arguments in JSON format")
}

// Helpers

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
	if strings.Contains(first, ":") {
		return "pass    ", args
	}

	// Assuming it's a password
	return padPassword(first), args[1:]
}

// Handlers

func handleQuery(port string) {
	// Query all or specific
	var devices []hubcli.DeviceInfo

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

		if err := hm.OpenPort(port); err != nil {
			fmt.Println("Failed to open port")
			return
		}

		// 1. Get ID (?Q)
		idResp, err := hm.SendCommand("?Q")
		if err != nil {
			hm.CloseCurrentPort()
			fmt.Println("Failed to open port")
			return
		}

		// Check if it's a valid device (e.g., contains CENTOS or is not empty)
		if idResp == "" {
			hm.CloseCurrentPort()
			fmt.Println("Failed to open port")
			return
		}

		// 2. Get Status (GP)
		gpResp, err := hm.SendCommand("GP")
		hm.CloseCurrentPort()

		if err != nil {
			fmt.Println("Failed to open port")
			return
		}

		// Clean GP response
		gpHex := strings.TrimSpace(gpResp)
		if strings.HasPrefix(gpHex, "GP") {
			gpHex = gpHex[2:]
		} else if strings.HasPrefix(gpHex, "G") {
			gpHex = gpHex[1:] // Assuming G+Hex
		}

		devices = append(devices, hubcli.DeviceInfo{
			Path:          port,
			AsciiResponse: idResp,
			LedStatus:     gpHex,
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
		// Format: COM6, 7 ports, On=None, Off=1,2,3,4,5,6,7, FW=v02

		// 1. Parse Hex Status
		// Decode LedStatus if it is hex encoded ASCII
		// NOTE: In hubcli/hubmanager.go, we are now returning raw ASCII string (gpDataRaw),
		// NOT hex encoded string. So we DO NOT need to hexStringToString.
		// However, we should check if it needs decoding just in case.
		// But based on my change in hubcli/hubmanager.go, it returns string(gpDataRaw).

		cleanStatus := strings.TrimSpace(d.LedStatus)

		// Remove possible prefixes "GP" or "G"
		if strings.HasPrefix(cleanStatus, "GP") {
			cleanStatus = cleanStatus[2:]
		} else if strings.HasPrefix(cleanStatus, "G") {
			cleanStatus = cleanStatus[1:]
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
				// Fallback
				val, _ := strconv.ParseUint(statusHex, 16, 32)
				portStatusByte = val
			}
		} else {
			val, _ := strconv.ParseUint(statusHex, 16, 32)
			portStatusByte = val
		}

		foundCount++

		// 2. Determine On/Off
		var onPorts []string
		var offPorts []string

		numPorts := 7

		// We use portStatusByte instead of val
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

		// 3. Extract FW version
		// Try to find version in string
		fwVer := "Unknown"
		if idx := strings.LastIndex(d.AsciiResponse, "v"); idx != -1 {
			fwVer = d.AsciiResponse[idx:]
		} else {
			// If not found, use the whole string or keep Unknown?
			// Let's use the AsciiResponse as fallback if it's short
			if len(d.AsciiResponse) < 10 {
				fwVer = d.AsciiResponse
			}
		}

		if port != "" {
			fmt.Printf("%s, %d ports, On=%s, Off=%s, FW=%s",
				d.Path, numPorts, onStr, offStr, fwVer)
		} else {
			fmt.Printf(" %s, %d ports, On=%s, Off=%s, FW=%s\n",
				d.Path, numPorts, onStr, offStr, fwVer)
		}
	}

	if port == "" {
		fmt.Printf("%d Managed USB Hub(s) Found.\n", foundCount)
	}
}

func handleSetState(port string, args []string, save bool) {
	pass, states := getPassword(args)

	if len(states) == 0 {
		fmt.Println("Error: No states specified")
		return
	}

	if err := hm.OpenPort(port); err != nil {
		fmt.Printf("Error opening %s: %v\n", port, err)
		return
	}
	defer hm.CloseCurrentPort()

	// 1. Get Current Status
	gpResp, err := hm.SendCommand("GP")
	if err != nil {
		fmt.Printf("Error getting status: %v\n", err)
		return
	}
	// fmt.Printf("Debug: GP Response='%s'\n", gpResp)

	currentHex := strings.TrimSpace(gpResp)
	// Remove G prefix if present (e.g. "GP00...")
	if strings.HasPrefix(currentHex, "GP") {
		currentHex = currentHex[2:]
	} else if strings.HasPrefix(currentHex, "G") {
		// Maybe G0?
		// Just try to parse what we have if it looks like hex
		currentHex = currentHex[1:] // Assume G+Hex
	}

	currentVal, err := strconv.ParseUint(currentHex, 16, 32)
	if err != nil {
		fmt.Printf("Error parsing status '%s': %v\n", currentHex, err)
		return
	}
	// fmt.Printf("Debug: Current Status (Hex)=%08X\n", currentVal)

	// Logic for state modification:
	// We only modify the FIRST byte (first 2 chars of hex string) which is the port mask.
	// The rest of the bytes (FFFFFF) should be preserved or just appended?
	// Based on "F3FFFFFF", F3 is the mask.
	// F3 = 1111 0011 (Binary).
	// Bit 0 = 1 (Port 1 ON)
	// Bit 1 = 1 (Port 2 ON)
	// Bit 2 = 0 (Port 3 OFF)
	// Bit 3 = 0 (Port 4 OFF)
	// ...
	// Wait, F3 = 1111 0011.
	// Bit 0 (1) -> 1
	// Bit 1 (2) -> 1
	// Bit 2 (4) -> 0
	// Bit 3 (8) -> 0
	// Bit 4 (16) -> 1
	// ...

	// The user wants "0:1" -> Turn OFF Port 1.
	// Current is F3. New should be F3 & ^1 = F2.
	// So new hex should be F2FFFFFF.

	// Let's use currentVal as base.
	// currentVal parses "F3FFFFFF" correctly as a large number.
	// But our bit manipulation logic relies on port numbers mapping to bits.
	// Port 1 -> Bit 0.

	newState := currentVal

	for _, stateStr := range states {
		// Handle 1:ALL
		if strings.ToUpper(stateStr) == "1:ALL" {
			shift := 0
			if len(currentHex) == 8 {
				shift = 24
			}
			mask := uint64(0xFF) << shift
			newState &^= mask // Clear
			newState |= mask  // Set to FF
			continue
		}

		// Handle 0:ALL
		if strings.ToUpper(stateStr) == "0:ALL" {
			shift := 0
			if len(currentHex) == 8 {
				shift = 24
			}
			mask := uint64(0xFF) << shift
			newState &^= mask
			continue
		}

		// Handle B:0101
		if strings.HasPrefix(strings.ToUpper(stateStr), "B:") {
			binStr := stateStr[2:]
			val, err := strconv.ParseUint(binStr, 2, 8)
			if err == nil {
				shift := 0
				if len(currentHex) == 8 {
					shift = 24
				}
				mask := uint64(0xFF) << shift
				newState &^= mask          // Clear old mask
				newState |= (val << shift) // Set new mask
			} else {
				fmt.Printf("Invalid binary format: %s\n", stateStr)
			}
			continue
		}

		// Handle H:A601
		if strings.HasPrefix(strings.ToUpper(stateStr), "H:") {
			hexStr := stateStr[2:]
			bytes, err := hex.DecodeString(hexStr)
			if err == nil {
				shift := 0
				if len(currentHex) == 8 {
					shift = 24
				}

				// Apply Byte 0
				if len(bytes) > 0 {
					mask := uint64(0xFF) << shift
					newState &^= mask
					newState |= (uint64(bytes[0]) << shift)
				}

				// Apply subsequent bytes
				for i := 1; i < len(bytes) && i < 4; i++ {
					if shift >= 8 {
						shift -= 8
						mask := uint64(0xFF) << shift
						newState &^= mask
						newState |= (uint64(bytes[i]) << shift)
					}
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
			if p < 1 {
				continue
			}
			bitIndex := p - 1
			// Note: Port 1 is Bit 0.
			// But wait, "F3FFFFFF" suggests the lowest byte is at the TOP (big endian string)?
			// "F3" is the first byte.
			// If we parse "F3FFFFFF" as uint32, F3 is the most significant byte (bits 24-31).
			// BUT, usually protocols are little endian.
			// If "F3" controls the ports, and it's at the start of the string...
			// If we treat the whole thing as a number, we need to know which bits control the ports.

			// Previous logic assumed standard parsing.
			// Let's assume the FIRST BYTE (highest byte in parsed uint32 if string is "AABBCCDD") controls the ports.
			// If string is "F3FFFFFF", value is 0xF3FFFFFF.
			// F3 is bits 24-31.
			// If Port 1 corresponds to Bit 0 of that byte... that would be Bit 24 of the uint32?

			// Let's change the logic to operate on the byte level or adjust the mask.
			// Or simpler: We parsed "F3FFFFFF".
			// We want to modify the "F3" part.
			// That corresponds to the bits 24-31 if we parsed it as big-endian string (which strconv does).
			// Actually, let's look at the mask construction in frontend:
			// byte0.toString(16) + "FFFFFF".
			// So "F3" is indeed the byte we care about.
			// And Port 1 corresponds to Bit 0 of that byte.

			// So if we have uint32 val = 0xF3FFFFFF.
			// We want to modify the byte at (val >> 24).
			// So Port 1 -> Bit 24. Port 2 -> Bit 25.
			// Wait, strconv.ParseUint("F3FFFFFF", 16, 32) returns a value where F3 is at the top.
			// So yes, Port 1 is Bit 24 (0 + 24).

			// Let's adjust the shift.
			// But wait, "FFFFFF" part (24 bits) is constant?
			// The device seems to return 8 chars.

			// Let's try shifting by 24 bits?
			// Or better: Let's parse just the first 2 chars, modify them, and reconstruct the string.
			// That's safer than bitwise math on the whole 32-bit word if we aren't sure about the rest.

			// NO, we need to accumulate changes.
			// Let's stick to bitwise but shift the mask.
			// Since "F3" is the first byte in the string "F3FFFFFF", it is the MSB of the parsed value.
			// Port 1 (index 1) -> Bit 0 of MSB -> Bit 24 of whole value.
			// Port 2 (index 2) -> Bit 1 of MSB -> Bit 25.
			// ...
			// Port 8 -> Bit 7 -> Bit 31.

			// However, previously I used `1 << (p-1)`. This targets Bit 0, 1, 2... (LSB).
			// If the device expects "F3......", then we were modifying the WRONG END (the "FF" at the end).
			// This explains why "G00FFFFFF" had no reaction or wrong reaction if we sent "00......" but meant "......00".
			// Actually, if we send "SP...F2FFFFFF", it works.

			// So, we need to shift our mask by 24 bits?
			// `mask := uint64(1 << (bitIndex + 24))`?

			// Let's verify.
			// Frontend logic: `byte0.toString(16) + "FFFFFF"`.
			// So yes, the first byte (MSB in string) holds the port states.
			// And inside that byte, `byte0 |= (1 << (i - 1))`.
			// So Port 1 is Bit 0 of that byte.

			// So yes, we need to target the MSB of the uint32.

			// FIX:
			shift := 0
			// If the string length is 8, we assume MSB is the port mask.
			// But wait, what if it's 4 chars?
			if len(currentHex) == 8 {
				shift = 24
			}

			mask := uint64(1 << (bitIndex + shift))

			switch action {
			case "1":
				newState |= mask
			case "0":
				newState &^= mask
			case "T":
				newState ^= mask
			}
		}
	}

	maskStr := fmt.Sprintf("%08X", newState)
	cmd := fmt.Sprintf("SP%s%s", pass, maskStr)

	// fmt.Printf("Debug: Sending Command='%s'\n", cmd)
	resp, err := hm.SendCommand(cmd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Only print if it's NOT a success response?
		if strings.Contains(resp, "E") && !strings.Contains(resp, "G") {
			if !strings.HasPrefix(resp, "G") {
				fmt.Printf("Result: %s\n", resp)
			}
		}
	}

	if save {
		// Save to flash
		// Assuming SS command
		cmdSS := fmt.Sprintf("SS%s", pass)
		resp, err := hm.SendCommand(cmdSS)
		if err != nil {
			fmt.Printf("Error saving: %v\n", err)
		} else {
			// Check for SS error
			if strings.Contains(resp, "E") && !strings.Contains(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") && !strings.HasPrefix(resp, "SS") {
				fmt.Printf("Result(Save): %s\n", resp)
			}
		}
	}
}

func handleGetState(port string, args []string) {
	if err := hm.OpenPort(port); err != nil {
		fmt.Printf("Error opening %s: %v\n", port, err)
		return
	}
	defer hm.CloseCurrentPort()

	resp, err := hm.SendCommand("GP")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	cleanStatus := strings.TrimSpace(resp)
	if strings.HasPrefix(cleanStatus, "GP") {
		cleanStatus = cleanStatus[2:]
	} else if strings.HasPrefix(cleanStatus, "G") {
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
	handleSimpleCommand(port, "SS", []string{pass})
}

func handleChangePassword(port string, args []string) {
	if len(args) == 0 {
		fmt.Println("New password required")
		return
	}

	oldPass := "pass    "
	newPass := ""

	if len(args) == 1 {
		newPass = args[0]
	} else {
		oldPass = padPassword(args[0])
		newPass = args[1]
	}

	newPass = padPassword(newPass)

	if err := hm.OpenPort(port); err != nil {
		fmt.Printf("Error opening %s: %v\n", port, err)
		return
	}
	defer hm.CloseCurrentPort()

	cmd := fmt.Sprintf("CP%s%s", oldPass, newPass)
	resp, err := hm.SendCommand(cmd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Only print if it looks like an error
		if strings.Contains(resp, "E") && !strings.Contains(resp, "G") {
			if !strings.HasPrefix(resp, "G") && !strings.Contains(strings.ToUpper(resp), "OK") {
				fmt.Printf("Result: %s\n", resp)
			}
		}
	}
}

func handleSetDescription(port string, args []string) {
	pass, rest := getPassword(args)
	if len(rest) == 0 {
		fmt.Println("Description string required")
		return
	}
	desc := strings.Join(rest, " ") // Handle spaces in desc if split by shell

	// Assuming SD command: SD[pass][desc]
	// desc might need quotes handling? The image says 'description string'
	// Removing single quotes if present
	desc = strings.Trim(desc, "'")

	if err := hm.OpenPort(port); err != nil {
		fmt.Printf("Error opening %s: %v\n", port, err)
		return
	}
	defer hm.CloseCurrentPort()

	cmd := fmt.Sprintf("SD%s%s", pass, desc)
	resp, err := hm.SendCommand(cmd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %s\n", resp)
	}
}

func handleSimpleCommand(port string, cmdPrefix string, args []string) {
	pass := ""
	if len(args) > 0 {
		pass = padPassword(args[0])
	} else {
		pass = "pass    " // Default
	}

	// For commands that don't need password (like GD, UID), pass is ignored or not sent?
	// UID -> ?Q or GU?
	// GD -> GD?
	// RD, RH, SS need password.

	cmd := cmdPrefix
	if cmdPrefix == "RD" || cmdPrefix == "RH" || cmdPrefix == "SS" {
		cmd += pass
	}

	if err := hm.OpenPort(port); err != nil {
		fmt.Printf("Error opening %s: %v\n", port, err)
		return
	}
	defer hm.CloseCurrentPort()

	resp, err := hm.SendCommand(cmd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		// Silent success for commands that perform actions (SS, RD, RH)
		// Assuming SS returns something like "OK" or echoes.

		isActionCmd := (cmdPrefix == "SS" || cmdPrefix == "RD" || cmdPrefix == "RH")

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
			// GD, UID -> Print result
			fmt.Printf("%s\n", resp)
		}
	}
}

func handleJSON(jsonStr string) {
	// Simple map to handle JSON commands
	var req map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &req)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	// Basic implementation for Q and SPST
	cmd, ok := req["CMD"].(string)
	if !ok {
		fmt.Println("Missing CMD in JSON")
		return
	}

	switch cmd {
	case "Q":
		handleQuery("")
	case "SPST":
		// {"CMD":"SPST","COM":"COM3","PSW":"pass","STATES":"F4,FF,FF,FF"}
		com, _ := req["COM"].(string)
		psw, _ := req["PSW"].(string)
		// states, _ := req["STATES"].(string)
		// This requires mapping hex states to our logic or sending raw SP
		// For now, just a placeholder
		fmt.Printf("JSON SPST for %s with pass %s (Logic not fully implemented)\n", com, psw)
	default:
		fmt.Printf("Unknown JSON CMD: %s\n", cmd)
	}
}
