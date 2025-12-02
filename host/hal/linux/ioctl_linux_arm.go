//go:build linux && arm

package linux

// ioctl encoding for arm (32-bit).
// The ioctl number encoding uses the following bit layout:
//
//	bits 0-7:   command number (nr)
//	bits 8-15:  ioctl type (type)
//	bits 16-29: argument size (size)
//	bits 30-31: direction (dir)

const (
	iocNone  = 0
	iocWrite = 1
	iocRead  = 2
)

const (
	iocNRBits   = 8
	iocTypeBits = 8
	iocSizeBits = 14
	iocDirBits  = 2

	iocNRShift   = 0
	iocTypeShift = iocNRShift + iocNRBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits
)

// ioc constructs an ioctl number from direction, type, number, and size.
func ioc(dir, typ, nr, size uintptr) uintptr {
	return (dir << iocDirShift) | (typ << iocTypeShift) | (nr << iocNRShift) | (size << iocSizeShift)
}

// ior constructs a read ioctl number.
func ior(typ, nr, size uintptr) uintptr {
	return ioc(iocRead, typ, nr, size)
}

// iow constructs a write ioctl number.
func iow(typ, nr, size uintptr) uintptr {
	return ioc(iocWrite, typ, nr, size)
}

// iowr constructs a read/write ioctl number.
func iowr(typ, nr, size uintptr) uintptr {
	return ioc(iocRead|iocWrite, typ, nr, size)
}

// ioctl constructs an ioctl number with no data transfer.
func ioctl(typ, nr uintptr) uintptr {
	return ioc(iocNone, typ, nr, 0)
}

// usbdevfs ioctl type character.
const usbdevfsType = 'U'

// usbdevfs ioctl command numbers.
const (
	ioctlControl          = 0
	ioctlBulk             = 2
	ioctlResetEP          = 3
	ioctlSetInterface     = 4
	ioctlSetConfiguration = 5
	ioctlGetDriver        = 8
	ioctlSubmitURB        = 10
	ioctlDiscardURB       = 11
	ioctlReapURB          = 12
	ioctlReapURBNDelay    = 13
	ioctlClaimInterface   = 15
	ioctlReleaseInterface = 16
	ioctlConnectInfo      = 17
	ioctlReset            = 20
	ioctlDisconnect       = 22
	ioctlConnect          = 23
	ioctlGetCapabilities  = 26
	ioctlDropPrivileges   = 29
)

// Size constants for ioctl argument structures.
// arm (32-bit) uses smaller pointer sizes.
const (
	sizeofCtrlTransfer = 16 // struct usbdevfs_ctrltransfer (32-bit pointers)
	sizeofBulkTransfer = 16 // struct usbdevfs_bulktransfer (32-bit)
	sizeofInt          = 4
	sizeofPointer      = 4
)

// Usbdevfs ioctl numbers for arm (32-bit).
// These are computed at init time using the _IOC macros.
var (
	ioctlUsbdevfsControl          = iowr(usbdevfsType, ioctlControl, sizeofCtrlTransfer)
	ioctlUsbdevfsBulk             = iowr(usbdevfsType, ioctlBulk, sizeofBulkTransfer)
	ioctlUsbdevfsResetEP          = ior(usbdevfsType, ioctlResetEP, sizeofInt)
	ioctlUsbdevfsSetInterface     = ior(usbdevfsType, ioctlSetInterface, 8)
	ioctlUsbdevfsSetConfiguration = ior(usbdevfsType, ioctlSetConfiguration, sizeofInt)
	ioctlUsbdevfsGetDriver        = iow(usbdevfsType, ioctlGetDriver, 264)
	ioctlUsbdevfsSubmitURB        = ior(usbdevfsType, ioctlSubmitURB, sizeofPointer)
	ioctlUsbdevfsDiscardURB       = ioctl(usbdevfsType, ioctlDiscardURB)
	ioctlUsbdevfsReapURB          = iow(usbdevfsType, ioctlReapURB, sizeofPointer)
	ioctlUsbdevfsReapURBNDelay    = iow(usbdevfsType, ioctlReapURBNDelay, sizeofPointer)
	ioctlUsbdevfsClaimInterface   = ior(usbdevfsType, ioctlClaimInterface, sizeofInt)
	ioctlUsbdevfsReleaseInterface = ior(usbdevfsType, ioctlReleaseInterface, sizeofInt)
	ioctlUsbdevfsConnectInfo      = iow(usbdevfsType, ioctlConnectInfo, 8)
	ioctlUsbdevfsReset            = ioctl(usbdevfsType, ioctlReset)
	ioctlUsbdevfsDisconnect       = ioctl(usbdevfsType, ioctlDisconnect)
	ioctlUsbdevfsConnect          = ioctl(usbdevfsType, ioctlConnect)
	ioctlUsbdevfsGetCapabilities  = ior(usbdevfsType, ioctlGetCapabilities, sizeofInt)
	ioctlUsbdevfsDropPrivileges   = iow(usbdevfsType, ioctlDropPrivileges, sizeofInt)
)
