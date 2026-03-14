package usbtree

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
)

// Enumerate returns a list of USB Root Host Controllers and their connected device trees.
func Enumerate() ([]*USBDevice, error) {
	// USB Host Controller Class GUID: {36fc9e60-c465-11cf-8056-444553540000}
	guid := windows.GUID{
		Data1: 0x36fc9e60,
		Data2: 0xc465,
		Data3: 0x11cf,
		Data4: [8]byte{0x80, 0x56, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00},
	}

	// Get device information set for all present devices of this class
	// We use DIGCF_PRESENT to get currently present devices.
	// We do NOT use DIGCF_DEVICEINTERFACE because we are providing a Setup Class GUID, not an Interface GUID.
	devInfo, err := windows.SetupDiGetClassDevsEx(
		&guid,
		"",
		0,
		windows.DIGCF_PRESENT,
		0,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("SetupDiGetClassDevsEx failed: %w", err)
	}
	defer windows.SetupDiDestroyDeviceInfoList(devInfo)

	var roots []*USBDevice
    
    // Debug
    // fmt.Println("Enumerating SetupAPI...")

	// Enumerate devices in the set
	for i := 0; ; i++ {
		devInfoData, err := windows.SetupDiEnumDeviceInfo(devInfo, i)
		if err != nil {
			if err == windows.ERROR_NO_MORE_ITEMS {
				break
			}
			continue
		}

		// devInfoData.DevInst is the handle we need for CM functions
		devInst := DEVINST(devInfoData.DevInst)
		
		// Build the tree for this controller
		rootDevice, err := buildDeviceTree(devInst, "0")
		if err != nil {
			// Log error but continue?
			fmt.Printf("Error building tree for device index %d: %v\n", i, err)
			continue
		}
		roots = append(roots, rootDevice)
	}

	return roots, nil
}

func buildDeviceTree(devInst DEVINST, parentChain string) (*USBDevice, error) {
	// 1. Get Info for current node
	device := &USBDevice{}

	// Instance ID
	instanceID, err := cmGetDeviceID(devInst)
	if err == nil {
		device.InstanceID = instanceID
	}

	// Friendly Name (or Description if name missing)
	name, err := cmGetDevNodeRegistryProperty(devInst, CM_DRP_FRIENDLYNAME)
	if err != nil || name == "" {
		name, _ = cmGetDevNodeRegistryProperty(devInst, CM_DRP_DEVICEDESC)
	}
	device.Name = name

	// Hardware IDs
	hwIDs, _ := cmGetDevNodeRegistryProperty(devInst, CM_DRP_HARDWAREID)
	device.DeviceID = hwIDs
	
	// Parse VID/PID
	device.VendorID, device.ProductID = parseHardwareID(hwIDs)

	// Manufacturer
	mfg, _ := cmGetDevNodeRegistryProperty(devInst, CM_DRP_MFG)
	device.Manufacturer = mfg

	// Service
	service, _ := cmGetDevNodePropertyString(devInst, &DEVPKEY_Device_Service)
	if service == "" {
		service, _ = cmGetDevNodeRegistryProperty(devInst, CM_DRP_SERVICE)
	}
	device.Service = service

	// Address (Port Number) - Try Property Key first (more reliable for Port Num)
	address, err := cmGetDevNodePropertyDWORD(devInst, &DEVPKEY_Device_Address)
	if err != nil {
		// Fallback to Registry Property
		address, err = cmGetDevNodeRegistryPropertyDWORD(devInst, CM_DRP_ADDRESS)
	}
	if err == nil {
		device.Address = address
	}
	
	// UI Number
	uiNum, err := cmGetDevNodeRegistryPropertyDWORD(devInst, CM_DRP_UINUMBER)
	if err == nil {
		device.UINumber = uiNum
	}
	
	// Location
	loc, _ := cmGetDevNodeRegistryProperty(devInst, CM_DRP_LOCATION_INFORMATION)
	device.Location = loc
	
	// Port Chain
	if parentChain == "0" {
		if device.Address == 0 {
			device.PortChain = "0"
		} else {
			device.PortChain = fmt.Sprintf("%d", device.Address)
		}
	} else {
		if device.Address != 0 {
			device.PortChain = fmt.Sprintf("%s-%d", parentChain, device.Address)
		} else {
			device.PortChain = parentChain
		}
	}

	// Class
	class, _ := cmGetDevNodeRegistryProperty(devInst, CM_DRP_CLASS)
	device.Class = class

	// Status
	status, prob, err := cmGetDevNodeStatus(devInst)
	if err == nil {
		device.Status = status
		device.ProblemCode = prob
	}

	// Is it a Hub? (Simple check: Class=USB and Service=usbhub3 or usbhub)
	// Or check compatible IDs.
	// Update: Service might be empty or localized name check needed.
	if strings.EqualFold(class, "USB") {
		if strings.Contains(strings.ToLower(service), "hub") {
			device.IsHub = true
		} else if strings.Contains(strings.ToLower(name), "hub") || strings.Contains(name, "集线器") {
			device.IsHub = true
		}
	}

	// 2. Find Children
	childInst, err := cmGetChild(devInst)
	if err == nil && childInst != 0 {
		// Has at least one child
		childNode, err := buildDeviceTree(childInst, device.PortChain)
		if err == nil {
			device.Children = append(device.Children, childNode)
			
			// 3. Find Siblings of the Child
			curr := childInst
			for {
				siblingInst, err := cmGetSibling(curr)
				if err != nil || siblingInst == 0 {
					break
				}
				siblingNode, err := buildDeviceTree(siblingInst, device.PortChain)
				if err == nil {
					device.Children = append(device.Children, siblingNode)
				}
				curr = siblingInst
			}
		}
	}

	// 4. Enrich if Hub
	// Now that we have all children, if this is a hub, we can try to get detailed info for them.
	if device.IsHub {
		enrichHubPorts(device)
	}

	return device, nil
}
