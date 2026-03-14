package usbtree

import (
	"syscall"
	"unsafe"
)

var (
	procCM_Get_Device_Interface_ListW = cfgmgr32.NewProc("CM_Get_Device_Interface_ListW")
	procCM_Get_Device_Interface_List_SizeW = cfgmgr32.NewProc("CM_Get_Device_Interface_List_SizeW")
)

const (
	CM_GET_DEVICE_INTERFACE_LIST_PRESENT = 0
)

func cmGetDeviceInterfaceList(deviceID string, interfaceGUID *syscall.GUID) ([]string, error) {
	var size uint32
	pDeviceID, err := syscall.UTF16PtrFromString(deviceID)
	if err != nil {
		return nil, err
	}

	// 1. Get Size
	ret, _, _ := procCM_Get_Device_Interface_List_SizeW.Call(
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Pointer(interfaceGUID)),
		uintptr(unsafe.Pointer(pDeviceID)),
		CM_GET_DEVICE_INTERFACE_LIST_PRESENT,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return nil, syscall.Errno(ret)
	}

	if size <= 1 {
		return nil, nil
	}

	// 2. Get List
	buf := make([]uint16, size)
	ret, _, _ = procCM_Get_Device_Interface_ListW.Call(
		uintptr(unsafe.Pointer(interfaceGUID)),
		uintptr(unsafe.Pointer(pDeviceID)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(size),
		CM_GET_DEVICE_INTERFACE_LIST_PRESENT,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return nil, syscall.Errno(ret)
	}

	// Parse multi-sz string
	var result []string
	start := 0
	for i, c := range buf {
		if c == 0 {
			if i > start {
				result = append(result, syscall.UTF16ToString(buf[start:i]))
			}
			start = i + 1
		}
	}
	return result, nil
}
