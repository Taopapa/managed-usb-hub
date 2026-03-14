package usbtree

import (
	"fmt"
	"unsafe"
	"syscall"
	"golang.org/x/sys/windows"
)

// GUID_DEVINTERFACE_USB_HUB: {f18a0e88-c30c-11d0-8815-00a0c906bed8}
var GUID_DEVINTERFACE_USB_HUB = syscall.GUID{
	Data1: 0xf18a0e88,
	Data2: 0xc30c,
	Data3: 0x11d0,
	Data4: [8]byte{0x88, 0x15, 0x00, 0xa0, 0xc9, 0x06, 0xbe, 0xd8},
}

const (
	IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX = 0x220448
)

type USB_DEVICE_DESCRIPTOR struct {
	Length          uint8
	DescriptorType  uint8
	BcdUSB          uint16
	DeviceClass     uint8
	DeviceSubClass  uint8
	DeviceProtocol  uint8
	MaxPacketSize0  uint8
	IdVendor        uint16
	IdProduct       uint16
	BcdDevice       uint16
	IManufacturer   uint8
	IProduct        uint8
	ISerialNumber   uint8
	NumConfigurations uint8
}

type USB_NODE_CONNECTION_INFORMATION_EX struct {
	ConnectionIndex uint32
	DeviceDescriptor USB_DEVICE_DESCRIPTOR
	CurrentConfigurationValue uint8
	Speed uint8
	DeviceIsHub uint8
	DeviceAddress uint16
	NumberOfOpenPipes uint32
	ConnectionStatus uint32
	// PipeList is variable, but we don't need it for now.
	// We can allocate a larger buffer when calling.
}

// USB Device Speeds
const (
	UsbLowSpeed = 0
	UsbFullSpeed = 1
	UsbHighSpeed = 2
	UsbSuperSpeed = 3
)

func getHubPortDetails(hubPath string, portIndex uint32) (*USB_NODE_CONNECTION_INFORMATION_EX, error) {
	// Open Hub
	// Note: We need to remove the trailing NULL from the path if present? 
	// cmGetDeviceInterfaceList returns Go strings which are clean.
	// But CreateFileW needs UTF16 pointer.
	
	hHub, err := windows.CreateFile(
		syscall.StringToUTF16Ptr(hubPath),
		windows.GENERIC_WRITE, // Needs WRITE for IOCTL? usually yes or 0. GENERIC_WRITE is safe for IOCTL.
		windows.FILE_SHARE_WRITE|windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		// Try read only if write fails, sometimes IOCTLs work with read access depending on driver
		hHub, err = windows.CreateFile(
			syscall.StringToUTF16Ptr(hubPath),
			0,
			windows.FILE_SHARE_WRITE|windows.FILE_SHARE_READ,
			nil,
			windows.OPEN_EXISTING,
			0,
			0,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to open hub %s: %w", hubPath, err)
		}
	}
	defer windows.CloseHandle(hHub)

	// Prepare IOCTL
	info := USB_NODE_CONNECTION_INFORMATION_EX{
		ConnectionIndex: portIndex,
	}
	
	// We need a buffer large enough for the struct + potentially pipes, but we only care about the fixed part.
	// The kernel fills what fits.
	var bytesReturned uint32
	size := uint32(unsafe.Sizeof(info))
	
	err = windows.DeviceIoControl(
		hHub,
		IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX,
		(*byte)(unsafe.Pointer(&info)),
		size,
		(*byte)(unsafe.Pointer(&info)),
		size,
		&bytesReturned,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("DeviceIoControl failed: %w", err)
	}

	return &info, nil
}
