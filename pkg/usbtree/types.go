package usbtree

// USBDevice represents a node in the USB device tree.
type USBDevice struct {
	Name          string       `json:"name"`           // Friendly Name or Description
	InstanceID    string       `json:"instance_id"`    // e.g. USB\VID_xxxx&PID_xxxx\1234
	DeviceID      string       `json:"device_id"`      // The hardware ID
	Class         string       `json:"class"`          // Device Class (e.g. USB, HID, Net)
	Service       string       `json:"service"`        // Driver service (e.g. usbstor)
	Address     uint32       `json:"address"`      // Port Number (usually)
	UINumber    uint32       `json:"ui_number"`    // CM_DRP_UINUMBER
	Location    string       `json:"location"`     // CM_DRP_LOCATION_INFORMATION
	Status      uint32       `json:"status"`       // CM_Get_DevNode_Status
	ProblemCode   uint32       `json:"problem_code"`   // CM_Get_DevNode_Status
	Children      []*USBDevice `json:"children"`       // Connected devices
	IsHub         bool         `json:"is_hub"`         // Helper flag

	// Detailed Info
	VendorID      string       `json:"vendor_id"`      // From HardwareID or Descriptor
	ProductID     string       `json:"product_id"`     // From HardwareID or Descriptor
	Manufacturer  string       `json:"manufacturer"`   // From Registry or Descriptor
	PortChain     string       `json:"port_chain"`     // e.g. 1-2-3
	Speed         string       `json:"speed"`          // Low, Full, High, Super, SuperPlus
	USBVersion    string       `json:"usb_version"`    // e.g. 2.0, 3.0 (bcdUSB)
	Power         string       `json:"power"`          // Self-powered / Bus-powered (if available)
	MaxPower      uint32       `json:"max_power"`      // In milliamps
}
