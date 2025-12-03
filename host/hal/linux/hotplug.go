//go:build linux

package linux

import (
	"bytes"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// =============================================================================
// UEvent Types
// =============================================================================

// ueventAction represents a udev action.
type ueventAction uint8

const (
	ueventUnknown ueventAction = iota
	ueventAdd
	ueventRemove
	ueventChange
	ueventBind
	ueventUnbind
)

// uevent represents a parsed netlink uevent.
type uevent struct {
	action    ueventAction
	devpath   string // DEVPATH value
	subsystem string // SUBSYSTEM value
	devtype   string // DEVTYPE value
	busnum    string // BUSNUM value
	devnum    string // DEVNUM value

	// USB-specific
	vendorID  string // ID_VENDOR_ID
	productID string // ID_MODEL_ID

	// Interface-specific
	interfaceClass string // INTERFACE value (for bind/unbind)
}

// =============================================================================
// Hotplug Monitor
// =============================================================================

// hotplugMonitor monitors for USB device hotplug events.
type hotplugMonitor struct {
	fd       int                    // Netlink socket file descriptor
	buf      [UEventBufferSize]byte // Buffer for receiving events
	addCh    chan usbDeviceInfo     // Channel for add events
	removeCh chan usbDeviceInfo     // Channel for remove events
	done     chan struct{}          // Signal to stop monitoring
}

// newHotplugMonitor creates a new hotplug monitor.
func newHotplugMonitor() (*hotplugMonitor, error) {
	// Create netlink socket
	fd, err := syscall.Socket(
		syscall.AF_NETLINK,
		syscall.SOCK_DGRAM|syscall.SOCK_CLOEXEC|syscall.SOCK_NONBLOCK,
		NetlinkKObjectUEvent,
	)
	if err != nil {
		return nil, err
	}

	// Bind to kernel broadcast group
	addr := syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Groups: 1, // Kernel broadcast group
	}
	if err := syscall.Bind(fd, &addr); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	return &hotplugMonitor{
		fd:       fd,
		addCh:    make(chan usbDeviceInfo, 16),
		removeCh: make(chan usbDeviceInfo, 16),
		done:     make(chan struct{}),
	}, nil
}

// close shuts down the hotplug monitor.
func (h *hotplugMonitor) close() error {
	close(h.done)
	return syscall.Close(h.fd)
}

// fd returns the netlink socket file descriptor for polling.
func (h *hotplugMonitor) socketFD() int {
	return h.fd
}

// addChannel returns the channel for add events.
func (h *hotplugMonitor) addChannel() <-chan usbDeviceInfo {
	return h.addCh
}

// removeChannel returns the channel for remove events.
func (h *hotplugMonitor) removeChannel() <-chan usbDeviceInfo {
	return h.removeCh
}

// processEvent reads and processes a uevent from the socket.
// Returns true if an event was processed, false if no data available.
func (h *hotplugMonitor) processEvent() (bool, error) {
	n, err := syscall.Read(h.fd, h.buf[:])
	if err != nil {
		if err == syscall.EAGAIN {
			return false, nil
		}
		return false, err
	}

	if n <= 0 {
		return false, nil
	}

	// Parse the uevent
	evt := parseUEvent(h.buf[:n])

	// Filter for USB device events
	if evt.subsystem != "usb" {
		return true, nil
	}
	if evt.devtype != "usb_device" {
		return true, nil
	}

	// Get device info from sysfs
	sysfsPath := filepath.Join(SysfsUSBPath, filepath.Base(evt.devpath))

	switch evt.action {
	case ueventAdd:
		// Parse device info
		info, err := parseUSBDevice(sysfsPath)
		if err == nil {
			select {
			case h.addCh <- info:
			default:
				// Channel full, drop event
			}
		}

	case ueventRemove:
		// For remove events, we may not be able to parse from sysfs
		// Create minimal info from the uevent
		info := usbDeviceInfo{
			sysfsPath: sysfsPath,
		}

		// Try to parse bus/dev numbers from devpath
		if busNum, devNum, ok := parseDevpathNumbers(evt.devpath); ok {
			info.busNum = busNum
			info.devNum = devNum
			info.devfsPath = formatDevfsPath(busNum, devNum)
		}

		select {
		case h.removeCh <- info:
		default:
			// Channel full, drop event
		}
	}

	return true, nil
}

// =============================================================================
// UEvent Parsing
// =============================================================================

// parseUEvent parses a netlink uevent message.
func parseUEvent(data []byte) uevent {
	evt := uevent{}

	// Split into null-terminated strings
	lines := bytes.Split(data, []byte{0})

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		s := string(line)

		// Parse key=value pairs
		idx := strings.IndexByte(s, '=')
		if idx < 0 {
			// First line is often just the action@devpath
			if strings.HasPrefix(s, "add@") {
				evt.action = ueventAdd
				evt.devpath = s[4:]
			} else if strings.HasPrefix(s, "remove@") {
				evt.action = ueventRemove
				evt.devpath = s[7:]
			} else if strings.HasPrefix(s, "change@") {
				evt.action = ueventChange
				evt.devpath = s[7:]
			} else if strings.HasPrefix(s, "bind@") {
				evt.action = ueventBind
				evt.devpath = s[5:]
			} else if strings.HasPrefix(s, "unbind@") {
				evt.action = ueventUnbind
				evt.devpath = s[7:]
			}
			continue
		}

		key := s[:idx]
		value := s[idx+1:]

		switch key {
		case "ACTION":
			switch value {
			case "add":
				evt.action = ueventAdd
			case "remove":
				evt.action = ueventRemove
			case "change":
				evt.action = ueventChange
			case "bind":
				evt.action = ueventBind
			case "unbind":
				evt.action = ueventUnbind
			}
		case "DEVPATH":
			evt.devpath = value
		case "SUBSYSTEM":
			evt.subsystem = value
		case "DEVTYPE":
			evt.devtype = value
		case "BUSNUM":
			evt.busnum = value
		case "DEVNUM":
			evt.devnum = value
		case "ID_VENDOR_ID":
			evt.vendorID = value
		case "ID_MODEL_ID":
			evt.productID = value
		case "INTERFACE":
			evt.interfaceClass = value
		}
	}

	return evt
}

// parseDevpathNumbers extracts bus and device numbers from a devpath.
// Devpath format: /devices/pci.../usb1/1-1 where the last component is the port path.
func parseDevpathNumbers(devpath string) (busNum, devNum uint8, ok bool) {
	// Try to find the sysfs device and read from there
	sysfsPath := filepath.Join(SysfsUSBPath, filepath.Base(devpath))
	busNum, devNum, ok = parseSysfsDevicePath(sysfsPath)
	return
}

// =============================================================================
// Netlink Helpers
// =============================================================================

// sockaddrNetlink is the netlink socket address structure.
type sockaddrNetlink struct {
	family uint16
	pad    uint16
	pid    uint32
	groups uint32
}

// bindNetlink binds a netlink socket to the kernel broadcast group.
func bindNetlink(fd int) error {
	addr := sockaddrNetlink{
		family: syscall.AF_NETLINK,
		groups: 1, // UDEV_MONITOR_KERNEL
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_BIND,
		uintptr(fd),
		uintptr(unsafe.Pointer(&addr)),
		unsafe.Sizeof(addr),
	)
	if errno != 0 {
		return errno
	}
	return nil
}
