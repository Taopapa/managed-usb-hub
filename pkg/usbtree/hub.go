package usbtree

import (
	"fmt"
	"strings"
)

// enrichHubPorts iterates over the children of a Hub device, queries the Hub via IOCTL,
// and fills in details for each child.
func enrichHubPorts(hubDevice *USBDevice) {
	// 1. Get Hub Device Interface Path
	// Note: The Hub Device Instance ID is hubDevice.InstanceID.
	// We need to find the interface path for GUID_DEVINTERFACE_USB_HUB.
	paths, err := cmGetDeviceInterfaceList(hubDevice.InstanceID, &GUID_DEVINTERFACE_USB_HUB)
	if err != nil || len(paths) == 0 {
		// fmt.Printf("DEBUG: No interface path for hub %s (IsHub=%v): %v\n", hubDevice.Name, hubDevice.IsHub, err)
		return
	}
	hubPath := paths[0]
	// fmt.Printf("DEBUG: Hub %s path: %s\n", hubDevice.Name, hubPath)

	// 2. Iterate children
	for _, child := range hubDevice.Children {
		// Address is the Port Number
		if child.Address == 0 {
			// fmt.Printf("DEBUG: Skipping child %s with Address 0\n", child.Name)
			continue
		}

		info, err := getHubPortDetails(hubPath, child.Address)
		if err != nil {
			// fmt.Printf("DEBUG: Failed to get port details for child %s (Port %d): %v\n", child.Name, child.Address, err)
			continue
		}

		// 3. Fill details
		child.Speed = mapSpeed(info.Speed)
		child.USBVersion = fmt.Sprintf("%x.%02x", info.DeviceDescriptor.BcdUSB>>8, info.DeviceDescriptor.BcdUSB&0xFF)
		
		// Map Vendor/Product from descriptor if not already set (though HardwareID usually has it)
		// child.VendorID = fmt.Sprintf("0x%04X", info.DeviceDescriptor.IdVendor)
		// child.ProductID = fmt.Sprintf("0x%04X", info.DeviceDescriptor.IdProduct)
		
		// If Device is self powered? 
		// Note: The Connection Info struct doesn't strictly have "SelfPowered" bit in the fixed part easily visible 
		// without checking Config Descriptor (which requires more IOCTLs).
		// But ConnectionStatus tells us if it's connected.
	}
}

func mapSpeed(speed uint8) string {
	switch speed {
	case UsbLowSpeed:
		return "Low-Speed (1.5 Mbps)"
	case UsbFullSpeed:
		return "Full-Speed (12 Mbps)"
	case UsbHighSpeed:
		return "High-Speed (480 Mbps)"
	case UsbSuperSpeed:
		return "SuperSpeed (5 Gbps)"
	default:
		return fmt.Sprintf("Unknown (%d)", speed)
	}
}

func parseHardwareID(hwID string) (string, string) {
	// Format usually: USB\VID_xxxx&PID_xxxx...
	// Simple parser
	vid := ""
	pid := ""
	
	upper := strings.ToUpper(hwID)
	if idx := strings.Index(upper, "VID_"); idx != -1 {
		if len(upper) >= idx+8 {
			vid = "0x" + upper[idx+4 : idx+8]
		}
	}
	if idx := strings.Index(upper, "PID_"); idx != -1 {
		if len(upper) >= idx+8 {
			pid = "0x" + upper[idx+4 : idx+8]
		}
	}
	return vid, pid
}
