package usbtree

import (
	"syscall"
	"unsafe"
)

var (
	cfgmgr32 = syscall.NewLazyDLL("cfgmgr32.dll")

	procCM_Get_Child              = cfgmgr32.NewProc("CM_Get_Child")
	procCM_Get_Sibling            = cfgmgr32.NewProc("CM_Get_Sibling")
	procCM_Get_Device_IDW         = cfgmgr32.NewProc("CM_Get_Device_IDW")
	procCM_Get_DevNode_Status     = cfgmgr32.NewProc("CM_Get_DevNode_Status")
	procCM_Get_DevNode_Registry_PropertyW = cfgmgr32.NewProc("CM_Get_DevNode_Registry_PropertyW")
	procCM_Get_DevNode_PropertyW  = cfgmgr32.NewProc("CM_Get_DevNode_PropertyW")
)

type DEVINST uint32
type CONFIGRET uint32
type DEVPROPTYPE uint32

const (
	CR_SUCCESS CONFIGRET = 0
)

const (
	CM_DRP_DEVICEDESC = 0x00000001
	CM_DRP_HARDWAREID = 0x00000002
	CM_DRP_SERVICE    = 0x00000004
	CM_DRP_CLASS      = 0x00000008
	CM_DRP_MFG          = 0x0000000B
	CM_DRP_FRIENDLYNAME = 0x0000000C
	CM_DRP_LOCATION_INFORMATION = 0x0000000D
	CM_DRP_UINUMBER     = 0x00000010
	CM_DRP_ADDRESS      = 0x0000001C
)

type DEVPROPKEY struct {
	FmtID syscall.GUID
	PID   uint32
}

var (
	// DEVPKEY_Device_Address: {a45c254e-df1c-4efd-8020-67d146a850e0}, 30
	DEVPKEY_Device_Address = DEVPROPKEY{
		FmtID: syscall.GUID{Data1: 0xa45c254e, Data2: 0xdf1c, Data3: 0x4efd, Data4: [8]byte{0x80, 0x20, 0x67, 0xd1, 0x46, 0xa8, 0x50, 0xe0}},
		PID:   30,
	}
	// DEVPKEY_Device_Service: {a45c254e-df1c-4efd-8020-67d146a850e0}, 6
	DEVPKEY_Device_Service = DEVPROPKEY{
		FmtID: syscall.GUID{Data1: 0xa45c254e, Data2: 0xdf1c, Data3: 0x4efd, Data4: [8]byte{0x80, 0x20, 0x67, 0xd1, 0x46, 0xa8, 0x50, 0xe0}},
		PID:   6,
	}
)

func cmGetChild(dnDevInst DEVINST) (DEVINST, error) {
	var child DEVINST
	ret, _, _ := procCM_Get_Child.Call(
		uintptr(unsafe.Pointer(&child)),
		uintptr(dnDevInst),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return 0, syscall.Errno(ret)
	}
	return child, nil
}

func cmGetSibling(dnDevInst DEVINST) (DEVINST, error) {
	var sibling DEVINST
	ret, _, _ := procCM_Get_Sibling.Call(
		uintptr(unsafe.Pointer(&sibling)),
		uintptr(dnDevInst),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return 0, syscall.Errno(ret)
	}
	return sibling, nil
}

func cmGetDeviceID(dnDevInst DEVINST) (string, error) {
	var buf [200]uint16 // MAX_DEVICE_ID_LEN is usually 200
	ret, _, _ := procCM_Get_Device_IDW.Call(
		uintptr(dnDevInst),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return "", syscall.Errno(ret)
	}
	return syscall.UTF16ToString(buf[:]), nil
}

func cmGetDevNodeStatus(dnDevInst DEVINST) (uint32, uint32, error) {
	var status uint32
	var problem uint32
	ret, _, _ := procCM_Get_DevNode_Status.Call(
		uintptr(unsafe.Pointer(&status)),
		uintptr(unsafe.Pointer(&problem)),
		uintptr(dnDevInst),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return 0, 0, syscall.Errno(ret)
	}
	return status, problem, nil
}

func cmGetDevNodeRegistryProperty(dnDevInst DEVINST, property uint32) (string, error) {
	var buf [1024]uint16 // Reasonable buffer size
	var size uint32 = uint32(len(buf) * 2)

	ret, _, _ := procCM_Get_DevNode_Registry_PropertyW.Call(
		uintptr(dnDevInst),
		uintptr(property),
		0, // pulRegDataType (ignored)
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return "", syscall.Errno(ret)
	}
	return syscall.UTF16ToString(buf[:]), nil
}

func cmGetDevNodeRegistryPropertyDWORD(dnDevInst DEVINST, property uint32) (uint32, error) {
	var value uint32
	var size uint32 = 4

	ret, _, _ := procCM_Get_DevNode_Registry_PropertyW.Call(
		uintptr(dnDevInst),
		uintptr(property),
		0,
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&size)),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return 0, syscall.Errno(ret)
	}
	return value, nil
}

func cmGetDevNodePropertyDWORD(dnDevInst DEVINST, key *DEVPROPKEY) (uint32, error) {
	var value uint32
	var propType DEVPROPTYPE
	var size uint32 = 4

	ret, _, _ := procCM_Get_DevNode_PropertyW.Call(
		uintptr(dnDevInst),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(&propType)),
		uintptr(unsafe.Pointer(&value)),
		uintptr(unsafe.Pointer(&size)),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return 0, syscall.Errno(ret)
	}
	return value, nil
}

func cmGetDevNodePropertyString(dnDevInst DEVINST, key *DEVPROPKEY) (string, error) {
	var buf [1024]uint16
	var propType DEVPROPTYPE
	var size uint32 = uint32(len(buf) * 2)

	ret, _, _ := procCM_Get_DevNode_PropertyW.Call(
		uintptr(dnDevInst),
		uintptr(unsafe.Pointer(key)),
		uintptr(unsafe.Pointer(&propType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
	)
	if CONFIGRET(ret) != CR_SUCCESS {
		return "", syscall.Errno(ret)
	}
	return syscall.UTF16ToString(buf[:]), nil
}
