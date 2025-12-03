//go:build linux

package linux

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ardnew/softusb/host/hal"
)

// =============================================================================
// USB Device Information
// =============================================================================

// usbDeviceInfo holds information about a USB device discovered via sysfs.
type usbDeviceInfo struct {
	sysfsPath   string    // Path in /sys/bus/usb/devices
	devfsPath   string    // Path in /dev/bus/usb
	busNum      uint8     // Bus number
	devNum      uint8     // Device number
	vendorID    uint16    // USB Vendor ID
	productID   uint16    // USB Product ID
	deviceClass uint8     // bDeviceClass
	speed       hal.Speed // Device speed

	// Interface information (for filtering by class)
	interfaces []usbInterfaceInfo
}

// usbInterfaceInfo holds information about a USB interface.
type usbInterfaceInfo struct {
	number   uint8 // bInterfaceNumber
	class    uint8 // bInterfaceClass
	subclass uint8 // bInterfaceSubClass
	protocol uint8 // bInterfaceProtocol
}

// =============================================================================
// Sysfs Parsing
// =============================================================================

// scanUSBDevices scans sysfs for USB devices.
func scanUSBDevices() ([]usbDeviceInfo, error) {
	entries, err := os.ReadDir(SysfsUSBPath)
	if err != nil {
		return nil, err
	}

	var devices []usbDeviceInfo

	for _, entry := range entries {
		name := entry.Name()

		// USB devices have names like "1-1", "1-1.2", etc.
		// Skip entries that are:
		// - Hub port entries (usb1, usb2, etc.)
		// - Interface entries (1-1:1.0)
		if strings.HasPrefix(name, "usb") {
			continue
		}
		if strings.Contains(name, ":") {
			continue
		}

		// Parse device information
		devPath := filepath.Join(SysfsUSBPath, name)
		info, err := parseUSBDevice(devPath)
		if err != nil {
			continue // Skip devices we can't parse
		}

		devices = append(devices, info)
	}

	return devices, nil
}

// parseUSBDevice parses USB device information from sysfs.
func parseUSBDevice(sysfsPath string) (usbDeviceInfo, error) {
	info := usbDeviceInfo{
		sysfsPath: sysfsPath,
	}

	// Read bus number
	busNum, err := readSysfsUint(filepath.Join(sysfsPath, "busnum"))
	if err != nil {
		return info, err
	}
	info.busNum = uint8(busNum)

	// Read device number
	devNum, err := readSysfsUint(filepath.Join(sysfsPath, "devnum"))
	if err != nil {
		return info, err
	}
	info.devNum = uint8(devNum)

	// Construct devfs path
	info.devfsPath = formatDevfsPath(info.busNum, info.devNum)

	// Read vendor ID
	vendorID, err := readSysfsHex(filepath.Join(sysfsPath, "idVendor"))
	if err == nil {
		info.vendorID = uint16(vendorID)
	}

	// Read product ID
	productID, err := readSysfsHex(filepath.Join(sysfsPath, "idProduct"))
	if err == nil {
		info.productID = uint16(productID)
	}

	// Read device class
	deviceClass, err := readSysfsHex(filepath.Join(sysfsPath, "bDeviceClass"))
	if err == nil {
		info.deviceClass = uint8(deviceClass)
	}

	// Read speed
	speedStr, err := readSysfsString(filepath.Join(sysfsPath, "speed"))
	if err == nil {
		info.speed = parseSpeed(speedStr)
	}

	// Scan interfaces
	info.interfaces = scanInterfaces(sysfsPath)

	return info, nil
}

// scanInterfaces scans sysfs for interfaces of a device.
func scanInterfaces(devicePath string) []usbInterfaceInfo {
	entries, err := os.ReadDir(devicePath)
	if err != nil {
		return nil
	}

	var interfaces []usbInterfaceInfo
	deviceName := filepath.Base(devicePath)

	for _, entry := range entries {
		name := entry.Name()

		// Interface entries have names like "1-1:1.0"
		// Format: <device>:<config>.<interface>
		if !strings.HasPrefix(name, deviceName+":") {
			continue
		}

		ifacePath := filepath.Join(devicePath, name)
		iface, err := parseInterface(ifacePath)
		if err != nil {
			continue
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces
}

// parseInterface parses USB interface information from sysfs.
func parseInterface(sysfsPath string) (usbInterfaceInfo, error) {
	info := usbInterfaceInfo{}

	// Read interface number
	ifaceNum, err := readSysfsHex(filepath.Join(sysfsPath, "bInterfaceNumber"))
	if err != nil {
		return info, err
	}
	info.number = uint8(ifaceNum)

	// Read interface class
	ifaceClass, err := readSysfsHex(filepath.Join(sysfsPath, "bInterfaceClass"))
	if err == nil {
		info.class = uint8(ifaceClass)
	}

	// Read interface subclass
	ifaceSubclass, err := readSysfsHex(filepath.Join(sysfsPath, "bInterfaceSubClass"))
	if err == nil {
		info.subclass = uint8(ifaceSubclass)
	}

	// Read interface protocol
	ifaceProtocol, err := readSysfsHex(filepath.Join(sysfsPath, "bInterfaceProtocol"))
	if err == nil {
		info.protocol = uint8(ifaceProtocol)
	}

	return info, nil
}

// =============================================================================
// Sysfs Read Helpers
// =============================================================================

// readSysfsString reads a string from a sysfs attribute file.
func readSysfsString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readSysfsUint reads an unsigned integer from a sysfs attribute file.
func readSysfsUint(path string) (uint64, error) {
	s, err := readSysfsString(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(s, 10, 64)
}

// readSysfsHex reads a hexadecimal value from a sysfs attribute file.
func readSysfsHex(path string) (uint64, error) {
	s, err := readSysfsString(path)
	if err != nil {
		return 0, err
	}
	// Remove any "0x" prefix
	s = strings.TrimPrefix(s, "0x")
	return strconv.ParseUint(s, 16, 64)
}

// =============================================================================
// Path Helpers
// =============================================================================

// formatDevfsPath constructs a /dev/bus/usb path from bus and device numbers.
func formatDevfsPath(busNum, devNum uint8) string {
	// Path format: /dev/bus/usb/BBB/DDD where BBB and DDD are zero-padded
	var buf [DevfsPathMaxLen]byte
	n := copy(buf[:], DevfsUSBPath)
	buf[n] = '/'
	n++
	n += formatPadded(buf[n:], busNum, 3)
	buf[n] = '/'
	n++
	n += formatPadded(buf[n:], devNum, 3)
	return string(buf[:n])
}

// formatPadded formats a number with zero-padding to a fixed width.
func formatPadded(buf []byte, val uint8, width int) int {
	// Convert to string
	s := strconv.FormatUint(uint64(val), 10)

	// Pad with zeros
	padding := width - len(s)
	for i := 0; i < padding && i < len(buf); i++ {
		buf[i] = '0'
	}

	// Copy digits
	copy(buf[padding:], s)
	return width
}

// parseSysfsDevicePath extracts bus and device numbers from a sysfs device path.
func parseSysfsDevicePath(path string) (busNum, devNum uint8, ok bool) {
	// Read from busnum and devnum files
	busNumVal, err := readSysfsUint(filepath.Join(path, "busnum"))
	if err != nil {
		return 0, 0, false
	}

	devNumVal, err := readSysfsUint(filepath.Join(path, "devnum"))
	if err != nil {
		return 0, 0, false
	}

	return uint8(busNumVal), uint8(devNumVal), true
}

// =============================================================================
// Speed Parsing
// =============================================================================

// parseSpeed converts a sysfs speed string to a hal.Speed value.
func parseSpeed(s string) hal.Speed {
	switch s {
	case "1.5":
		return hal.SpeedLow
	case "12":
		return hal.SpeedFull
	case "480":
		return hal.SpeedHigh
	default:
		return hal.SpeedUnknown
	}
}

// =============================================================================
// Filtering
// =============================================================================

// hasHIDInterface returns true if the device has an HID interface.
func (d *usbDeviceInfo) hasHIDInterface() bool {
	for _, iface := range d.interfaces {
		if iface.class == USBClassHID {
			return true
		}
	}
	return false
}

// getHIDInterfaces returns all HID interfaces for the device.
func (d *usbDeviceInfo) getHIDInterfaces() []usbInterfaceInfo {
	var result []usbInterfaceInfo
	for _, iface := range d.interfaces {
		if iface.class == USBClassHID {
			result = append(result, iface)
		}
	}
	return result
}

// findHIDDevices scans for USB devices with HID interfaces.
func findHIDDevices() ([]usbDeviceInfo, error) {
	devices, err := scanUSBDevices()
	if err != nil {
		return nil, err
	}

	var hidDevices []usbDeviceInfo
	for _, dev := range devices {
		if dev.hasHIDInterface() {
			hidDevices = append(hidDevices, dev)
		}
	}

	return hidDevices, nil
}
