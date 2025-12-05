package linux

// =============================================================================
// Device and Endpoint Limits
// =============================================================================

// MaxDevices is the maximum number of devices that can be tracked simultaneously.
// USB 2.0 supports up to 127 devices per bus, but we use a practical limit that
// accommodates typical systems with multiple USB controllers and hubs.
const MaxDevices = 64

// MaxEndpointsPerDevice is the maximum number of endpoints per device.
// USB 2.0 allows up to 16 endpoints per direction (32 total including directions).
const MaxEndpointsPerDevice = 32

// MaxInterfacesPerDevice is the maximum number of interfaces per device.
const MaxInterfacesPerDevice = 16

// =============================================================================
// URB (USB Request Block) Configuration
// =============================================================================

// MaxURBsPerEndpoint is the maximum number of pending URBs per endpoint.
// This limits the async I/O queue depth per endpoint.
const MaxURBsPerEndpoint = 4

// URBBufferSize is the default buffer size for URB data transfers.
const URBBufferSize = 1024

// MaxControlTransferSize is the maximum size for control transfer data phase.
const MaxControlTransferSize = 4096

// =============================================================================
// Path Length Limits
// =============================================================================

// SysfsPathMaxLen is the maximum length of a sysfs path.
const SysfsPathMaxLen = 256

// DevfsPathMaxLen is the maximum length of a devfs path.
const DevfsPathMaxLen = 64

// =============================================================================
// System Paths
// =============================================================================

// SysfsUSBPath is the base path for USB devices in sysfs.
const SysfsUSBPath = "/sys/bus/usb/devices"

// DevfsUSBPath is the base path for USB device nodes.
const DevfsUSBPath = "/dev/bus/usb"

// =============================================================================
// Errno Constants
// =============================================================================

// Common errno values returned by usbfs operations.
const (
	EPERM   = 1  // Operation not permitted
	ENOENT  = 2  // No such file or directory
	EIO     = 5  // I/O error
	ENXIO   = 6  // No such device or address
	EBADF   = 9  // Bad file descriptor
	EAGAIN  = 11 // Resource temporarily unavailable
	ENOMEM  = 12 // Cannot allocate memory
	EACCES  = 13 // Permission denied
	EFAULT  = 14 // Bad address
	EBUSY   = 16 // Device or resource busy
	ENODEV  = 19 // No such device
	EINVAL  = 22 // Invalid argument
	ENOSPC  = 28 // No space left on device
	EPIPE   = 32 // Broken pipe
	ENODATA = 61 // No data available
	ETIME   = 62 // Timer expired
	ENOSR   = 63 // Out of streams resources
	EPROTO  = 71 // Protocol error
)

// =============================================================================
// USB Speed Constants (matching kernel values)
// =============================================================================

// USB device speeds as reported by sysfs.
const (
	SpeedUnknown = 0 // Unknown speed
	SpeedLow     = 1 // Low Speed (1.5 Mbit/s)
	SpeedFull    = 2 // Full Speed (12 Mbit/s)
	SpeedHigh    = 3 // High Speed (480 Mbit/s)
)

// =============================================================================
// URB Type Constants
// =============================================================================

// URB transfer types for USBDEVFS_SUBMITURB.
const (
	URBTypeISO       = 0 // Isochronous
	URBTypeInterrupt = 1 // Interrupt
	URBTypeControl   = 2 // Control
	URBTypeBulk      = 3 // Bulk
)

// URB flags.
const (
	URBShortNotOK    = 0x01 // Short read is an error
	URBISOAsap       = 0x02 // Schedule ISO transfer ASAP
	URBNoTransferDMA = 0x04 // Don't use DMA for transfer buffer
	URBNoFSBR        = 0x20 // Don't do front/back buffer rotation
	URBZeroPacket    = 0x40 // Send zero-length packet at end
	URBNoInterrupt   = 0x80 // Don't generate interrupt on completion
)

// URB status values.
const (
	URBStatusSuccess    = 0
	URBStatusInProgress = -115 // EINPROGRESS
)

// =============================================================================
// HID Class Constants
// =============================================================================

// USBClassHID is the USB HID class code.
const USBClassHID = 0x03

// HID subclass codes.
const (
	HIDSubclassNone = 0x00
	HIDSubclassBoot = 0x01
)

// HID protocol codes.
const (
	HIDProtocolNone     = 0x00
	HIDProtocolKeyboard = 0x01
	HIDProtocolMouse    = 0x02
)

// =============================================================================
// Netlink Constants
// =============================================================================

// NetlinkKObjectUEvent is the netlink protocol for udev events.
const NetlinkKObjectUEvent = 15 // NETLINK_KOBJECT_UEVENT

// UEventBufferSize is the buffer size for netlink messages.
const UEventBufferSize = 4096

// =============================================================================
// Polling Constants
// =============================================================================

// Epoll event flags.
const (
	EPOLLIN  = 0x001
	EPOLLOUT = 0x004
	EPOLLERR = 0x008
	EPOLLHUP = 0x010
)

// MaxEpollEvents is the maximum events to retrieve per epoll_wait call.
const MaxEpollEvents = 32
