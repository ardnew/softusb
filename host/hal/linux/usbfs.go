//go:build linux

package linux

import (
	"syscall"
	"unsafe"
)

// =============================================================================
// URB (USB Request Block) Structures
// =============================================================================

// urb represents a USB Request Block for async I/O.
// This must match the kernel's struct usbdevfs_urb layout.
type urb struct {
	typ          uint8            // URB type (control, bulk, interrupt, iso)
	endpoint     uint8            // Endpoint address
	status       int32            // URB status after completion
	flags        uint32           // URB flags
	buffer       uintptr          // Pointer to data buffer
	bufferLength int32            // Length of data buffer
	actualLength int32            // Actual bytes transferred
	startFrame   int32            // Start frame for ISO transfers
	streamID     uint32           // Stream ID for USB 3.0 bulk streams
	errorCount   int32            // Error count for ISO transfers
	signr        uint32           // Signal number for async notification
	userContext  uintptr          // User context pointer
	isoFrameDesc [0]isoPacketDesc // ISO frame descriptors (variable length)
}

// isoPacketDesc describes an isochronous packet.
type isoPacketDesc struct {
	length       uint32 // Expected length
	actualLength uint32 // Actual length
	status       uint32 // Status
}

// ctrlTransfer represents a control transfer request.
// This must match the kernel's struct usbdevfs_ctrltransfer layout.
type ctrlTransfer struct {
	requestType uint8   // bmRequestType
	request     uint8   // bRequest
	value       uint16  // wValue
	index       uint16  // wIndex
	length      uint16  // wLength
	timeout     uint32  // Timeout in milliseconds
	data        uintptr // Data buffer pointer
}

// bulkTransfer represents a bulk transfer request.
// This must match the kernel's struct usbdevfs_bulktransfer layout.
type bulkTransfer struct {
	endpoint uint32  // Endpoint address
	length   uint32  // Data length
	timeout  uint32  // Timeout in milliseconds
	data     uintptr // Data buffer pointer
}

// connectInfo holds device connection information.
type connectInfo struct {
	devnum uint32 // Device number
	slow   uint8  // Low speed flag
}

// =============================================================================
// Raw Syscall Wrappers
// =============================================================================

// openDevice opens a USB device file for read/write access.
func openDevice(path string) (int, error) {
	pathBytes := make([]byte, len(path)+1)
	copy(pathBytes, path)

	fd, _, errno := syscall.Syscall(
		syscall.SYS_OPEN,
		uintptr(unsafe.Pointer(&pathBytes[0])),
		uintptr(syscall.O_RDWR|syscall.O_CLOEXEC),
		0,
	)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

// closeDevice closes a device file descriptor.
func closeDevice(fd int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_CLOSE, uintptr(fd), 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

// ioctlRaw performs a raw ioctl syscall.
func ioctlRaw(fd int, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

// ioctlRetval performs an ioctl syscall and returns the result value.
func ioctlRetval(fd int, req uintptr, arg uintptr) (int, error) {
	r, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
	if errno != 0 {
		return int(r), errno
	}
	return int(r), nil
}

// =============================================================================
// USBDEVFS Operations
// =============================================================================

// doControlTransfer performs a synchronous control transfer.
func doControlTransfer(fd int, reqType, req uint8, value, index uint16, data []byte, timeout uint32) (int, error) {
	ctrl := ctrlTransfer{
		requestType: reqType,
		request:     req,
		value:       value,
		index:       index,
		length:      uint16(len(data)),
		timeout:     timeout,
	}
	if len(data) > 0 {
		ctrl.data = uintptr(unsafe.Pointer(&data[0]))
	}

	n, err := ioctlRetval(fd, ioctlUsbdevfsControl, uintptr(unsafe.Pointer(&ctrl)))
	if err != nil {
		return 0, err
	}
	return n, nil
}

// doBulkTransfer performs a synchronous bulk transfer.
func doBulkTransfer(fd int, endpoint uint8, data []byte, timeout uint32) (int, error) {
	bulk := bulkTransfer{
		endpoint: uint32(endpoint),
		length:   uint32(len(data)),
		timeout:  timeout,
	}
	if len(data) > 0 {
		bulk.data = uintptr(unsafe.Pointer(&data[0]))
	}

	n, err := ioctlRetval(fd, ioctlUsbdevfsBulk, uintptr(unsafe.Pointer(&bulk)))
	if err != nil {
		return 0, err
	}
	return n, nil
}

// claimInterface claims exclusive access to an interface.
func claimInterface(fd int, iface uint8) error {
	ifaceNum := uint32(iface)
	return ioctlRaw(fd, ioctlUsbdevfsClaimInterface, uintptr(unsafe.Pointer(&ifaceNum)))
}

// releaseInterface releases a previously claimed interface.
func releaseInterface(fd int, iface uint8) error {
	ifaceNum := uint32(iface)
	return ioctlRaw(fd, ioctlUsbdevfsReleaseInterface, uintptr(unsafe.Pointer(&ifaceNum)))
}

// disconnectDriver disconnects the kernel driver from an interface.
func disconnectDriver(fd int, iface uint8) error {
	ifaceNum := uint32(iface)
	return ioctlRaw(fd, ioctlUsbdevfsDisconnect, uintptr(unsafe.Pointer(&ifaceNum)))
}

// connectDriver reconnects the kernel driver to an interface.
func connectDriver(fd int, iface uint8) error {
	ifaceNum := uint32(iface)
	return ioctlRaw(fd, ioctlUsbdevfsConnect, uintptr(unsafe.Pointer(&ifaceNum)))
}

// resetDevice resets the USB device.
func resetDevice(fd int) error {
	return ioctlRaw(fd, ioctlUsbdevfsReset, 0)
}

// resetEndpoint resets an endpoint.
func resetEndpoint(fd int, endpoint uint8) error {
	ep := uint32(endpoint)
	return ioctlRaw(fd, ioctlUsbdevfsResetEP, uintptr(unsafe.Pointer(&ep)))
}

// getCapabilities retrieves device capabilities.
func getCapabilities(fd int) (uint32, error) {
	var caps uint32
	err := ioctlRaw(fd, ioctlUsbdevfsGetCapabilities, uintptr(unsafe.Pointer(&caps)))
	if err != nil {
		return 0, err
	}
	return caps, nil
}

// getConnectInfo retrieves device connection information.
func getConnectInfo(fd int) (connectInfo, error) {
	var info connectInfo
	err := ioctlRaw(fd, ioctlUsbdevfsConnectInfo, uintptr(unsafe.Pointer(&info)))
	return info, err
}

// =============================================================================
// Async URB Operations
// =============================================================================

// submitURB submits a URB for asynchronous processing.
func submitURB(fd int, u *urb) error {
	return ioctlRaw(fd, ioctlUsbdevfsSubmitURB, uintptr(unsafe.Pointer(u)))
}

// reapURB waits for and retrieves a completed URB (blocking).
func reapURB(fd int) (*urb, error) {
	var urbPtr *urb
	err := ioctlRaw(fd, ioctlUsbdevfsReapURB, uintptr(unsafe.Pointer(&urbPtr)))
	if err != nil {
		return nil, err
	}
	return urbPtr, nil
}

// reapURBNDelay retrieves a completed URB without blocking.
// Returns EAGAIN if no URB is available.
func reapURBNDelay(fd int) (*urb, error) {
	var urbPtr *urb
	err := ioctlRaw(fd, ioctlUsbdevfsReapURBNDelay, uintptr(unsafe.Pointer(&urbPtr)))
	if err != nil {
		return nil, err
	}
	return urbPtr, nil
}

// discardURB cancels a pending URB.
func discardURB(fd int, u *urb) error {
	return ioctlRaw(fd, ioctlUsbdevfsDiscardURB, uintptr(unsafe.Pointer(u)))
}

// =============================================================================
// URB Helpers
// =============================================================================

// initControlURB initializes a URB for a control transfer.
func initControlURB(u *urb, endpoint uint8, setup []byte, data []byte) {
	u.typ = URBTypeControl
	u.endpoint = endpoint
	u.flags = 0
	u.status = 0

	// For control transfers, the setup packet is prepended to the data
	// We need to allocate a buffer that holds both
	totalLen := 8 + len(data) // 8-byte setup packet + data
	u.bufferLength = int32(totalLen)

	if len(setup) >= 8 {
		// Caller provides combined setup+data buffer
		u.buffer = uintptr(unsafe.Pointer(&setup[0]))
	}
}

// initBulkURB initializes a URB for a bulk transfer.
func initBulkURB(u *urb, endpoint uint8, data []byte) {
	u.typ = URBTypeBulk
	u.endpoint = endpoint
	u.flags = 0
	u.status = 0
	u.bufferLength = int32(len(data))
	if len(data) > 0 {
		u.buffer = uintptr(unsafe.Pointer(&data[0]))
	}
}

// initInterruptURB initializes a URB for an interrupt transfer.
func initInterruptURB(u *urb, endpoint uint8, data []byte) {
	u.typ = URBTypeInterrupt
	u.endpoint = endpoint
	u.flags = 0
	u.status = 0
	u.bufferLength = int32(len(data))
	if len(data) > 0 {
		u.buffer = uintptr(unsafe.Pointer(&data[0]))
	}
}

// =============================================================================
// Error Helpers
// =============================================================================

// isNoDevice returns true if the error indicates the device was disconnected.
func isNoDevice(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.ENODEV
	}
	return false
}

// isAgain returns true if the error indicates try again (EAGAIN/EWOULDBLOCK).
func isAgain(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.EAGAIN
	}
	return false
}

// isPipe returns true if the error indicates a stall (EPIPE).
func isPipe(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.EPIPE
	}
	return false
}
