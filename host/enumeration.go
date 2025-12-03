package host

import (
	"errors"

	"github.com/ardnew/softusb/host/hal"
	"github.com/ardnew/softusb/pkg"
)

// Enumeration errors.
var (
	ErrEnumerationFailed = errors.New("enumeration failed")
	ErrNoAddress         = errors.New("no address available")
)

// enumerateDevice performs the USB enumeration sequence for a new device.
func (h *Host) enumerateDevice(port int) (*Device, error) {
	pkg.LogDebug(pkg.ComponentHost, "starting enumeration", "port", port)

	// Get port speed
	speed := h.hal.PortSpeed(port)

	// Reset the port
	if err := h.hal.ResetPort(port); err != nil {
		return nil, err
	}

	// Allow device to stabilize after reset
	// (HAL should handle any necessary delays)

	// Create device at address 0
	dev := newDevice(h, port, 0, speed)

	// Read initial device descriptor (just the first 8 bytes to get bMaxPacketSize0)
	var buf [MaxDescriptorSize]byte
	setup := hal.SetupPacket{
		RequestType: RequestTypeIn | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestGetDescriptor,
		Value:       uint16(DescriptorTypeDevice) << 8,
		Index:       0,
		Length:      8, // Just get the first 8 bytes
	}

	n, err := h.hal.ControlTransfer(h.ctx, hal.DeviceAddress(0), &setup, buf[:8])
	if err != nil {
		return nil, err
	}
	if n < 8 {
		return nil, ErrEnumerationFailed
	}

	// Get max packet size from partial descriptor
	maxPacketSize0 := buf[7]
	if maxPacketSize0 == 0 {
		maxPacketSize0 = 8 // Default for low-speed
	}

	pkg.LogDebug(pkg.ComponentHost, "got max packet size", "size", maxPacketSize0)

	// Allocate address
	address := h.allocateAddress()
	if address == 0 {
		return nil, ErrNoAddress
	}

	// Set address
	setup = hal.SetupPacket{
		RequestType: RequestTypeOut | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestSetAddress,
		Value:       uint16(address),
		Index:       0,
		Length:      0,
	}

	_, err = h.hal.ControlTransfer(h.ctx, hal.DeviceAddress(0), &setup, nil)
	if err != nil {
		return nil, err
	}

	pkg.LogDebug(pkg.ComponentHost, "assigned address", "address", address)

	// Update device address
	dev.address = address
	dev.state = DeviceStateAddress

	// Now read full device descriptor using the new address
	setup = hal.SetupPacket{
		RequestType: RequestTypeIn | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestGetDescriptor,
		Value:       uint16(DescriptorTypeDevice) << 8,
		Index:       0,
		Length:      DeviceDescriptorSize,
	}

	n, err = h.hal.ControlTransfer(h.ctx, hal.DeviceAddress(address), &setup, buf[:DeviceDescriptorSize])
	if err != nil {
		return nil, err
	}
	if n < DeviceDescriptorSize {
		return nil, ErrEnumerationFailed
	}

	// Parse device descriptor
	dev.parseDeviceDescriptor(buf[:n])

	pkg.LogDebug(pkg.ComponentHost, "device descriptor",
		"vendorID", dev.descriptor.VendorID,
		"productID", dev.descriptor.ProductID,
		"class", dev.descriptor.DeviceClass)

	// Read configuration descriptor (just header first to get total length)
	setup = hal.SetupPacket{
		RequestType: RequestTypeIn | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestGetDescriptor,
		Value:       uint16(DescriptorTypeConfiguration) << 8,
		Index:       0,
		Length:      ConfigurationDescriptorSize,
	}

	n, err = h.hal.ControlTransfer(h.ctx, hal.DeviceAddress(address), &setup, buf[:ConfigurationDescriptorSize])
	if err != nil {
		return nil, err
	}
	if n < ConfigurationDescriptorSize {
		return nil, ErrEnumerationFailed
	}

	// Get total length
	totalLength := uint16(buf[2]) | uint16(buf[3])<<8
	if totalLength > uint16(len(buf)) {
		totalLength = uint16(len(buf))
	}

	// Read full configuration descriptor
	setup.Length = totalLength
	n, err = h.hal.ControlTransfer(h.ctx, hal.DeviceAddress(address), &setup, buf[:totalLength])
	if err != nil {
		return nil, err
	}

	// Parse configuration tree
	dev.parseConfigurationTree(buf[:n])

	pkg.LogDebug(pkg.ComponentHost, "configuration descriptor",
		"numInterfaces", dev.config.NumInterfaces,
		"configValue", dev.config.ConfigurationValue)

	// Read string descriptors if available
	if err := h.readStringDescriptors(dev, buf[:]); err != nil {
		// Non-fatal, continue without strings
		pkg.LogDebug(pkg.ComponentHost, "string descriptor read failed", "error", err)
	}

	// Set configuration (use the first configuration)
	if dev.config.ConfigurationValue > 0 {
		if err := dev.SetConfiguration(h.ctx, dev.config.ConfigurationValue); err != nil {
			return nil, err
		}
	}

	return dev, nil
}

// readStringDescriptors reads and caches string descriptors for a device.
func (h *Host) readStringDescriptors(dev *Device, buf []byte) error {
	// Helper to read a single string descriptor
	readString := func(index uint8) (string, error) {
		if index == 0 {
			return "", nil
		}

		// Request string descriptor
		setup := hal.SetupPacket{
			RequestType: RequestTypeIn | RequestTypeStandard | RequestTypeDevice,
			Request:     RequestGetDescriptor,
			Value:       uint16(DescriptorTypeString)<<8 | uint16(index),
			Index:       LangIDUSEnglish,
			Length:      uint16(len(buf)),
		}

		n, err := h.hal.ControlTransfer(h.ctx, hal.DeviceAddress(dev.address), &setup, buf)
		if err != nil {
			return "", err
		}

		if n < 2 {
			return "", nil
		}

		// Parse Unicode string (skip header, convert UTF-16LE to string)
		length := int(buf[0])
		if length > n {
			length = n
		}
		if length < 2 {
			return "", nil
		}

		// Simple UTF-16LE to ASCII conversion (ignoring non-ASCII characters)
		result := make([]byte, 0, (length-2)/2)
		for i := 2; i < length-1; i += 2 {
			if buf[i+1] == 0 && buf[i] >= 0x20 && buf[i] < 0x7F {
				result = append(result, buf[i])
			}
		}

		return string(result), nil
	}

	// Read manufacturer string
	if s, err := readString(dev.descriptor.ManufacturerIndex); err == nil && len(s) > 0 {
		if int(dev.descriptor.ManufacturerIndex) < len(dev.strings) {
			dev.strings[dev.descriptor.ManufacturerIndex] = s
			pkg.LogDebug(pkg.ComponentHost, "manufacturer", "value", s)
		}
	}

	// Read product string
	if s, err := readString(dev.descriptor.ProductIndex); err == nil && len(s) > 0 {
		if int(dev.descriptor.ProductIndex) < len(dev.strings) {
			dev.strings[dev.descriptor.ProductIndex] = s
			pkg.LogDebug(pkg.ComponentHost, "product", "value", s)
		}
	}

	// Read serial number string
	if s, err := readString(dev.descriptor.SerialNumberIndex); err == nil && len(s) > 0 {
		if int(dev.descriptor.SerialNumberIndex) < len(dev.strings) {
			dev.strings[dev.descriptor.SerialNumberIndex] = s
			pkg.LogDebug(pkg.ComponentHost, "serial", "value", s)
		}
	}

	return nil
}
