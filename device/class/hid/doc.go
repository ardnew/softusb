// Package hid implements the USB Human Interface Device (HID) class for the
// softusb device stack.
//
// This package provides HID functionality for implementing USB input devices
// such as keyboards, mice, gamepads, and other human interface devices.
//
// # Architecture
//
// A HID device consists of a single HID interface with:
//
//   - An Interrupt IN endpoint for sending input reports to the host
//   - An optional Interrupt OUT endpoint for receiving output reports
//   - HID class descriptors (HID descriptor, Report descriptor)
//
// # Zero-Allocation Design
//
// This implementation follows zero-allocation patterns:
//
//   - Fixed-size buffers for HID reports
//   - Caller-provided buffers for data transfer
//   - Report descriptors are stored by reference, not copied
//
// # Usage
//
// To create a HID keyboard device:
//
//	// Create the HID class driver
//	keyboard := hid.New(hid.KeyboardReportDescriptor)
//
//	// Configure callbacks
//	keyboard.SetOnOutputReport(func(data []byte) {
//	    // Handle LED state from host
//	})
//
//	// Add to a device configuration
//	builder := device.NewDeviceBuilder().
//	    WithVendorProduct(0xCAFE, 0xBABE).
//	    WithStrings("Manufacturer", "HID Keyboard", "12345").
//	    AddConfiguration(1)
//
//	// Add HID interface (interrupt IN EP, boot subclass, keyboard protocol)
//	keyboard.ConfigureDevice(builder, 0x81, hid.SubclassBoot, hid.ProtocolKeyboard)
//
//	// Build device
//	dev, _ := builder.Build(ctx)
//
//	// Attach HID driver to interface in configuration 1 (interface 0)
//	keyboard.AttachToInterface(dev, 1, 0)
//
//	// Create stack and start
//	stack := device.NewStack(dev, hal)
//	keyboard.SetStack(stack)
//	stack.Start(ctx)
//
//	// Send keyboard reports
//	keyboard.SendReport(ctx, keyboardReport)
//
// # Report Descriptors
//
// The package includes common report descriptors:
//
//   - KeyboardReportDescriptor: Standard 8-byte keyboard report
//   - MouseReportDescriptor: Standard 4-byte mouse report (3 buttons, X/Y/wheel)
//
// Custom report descriptors can be created using the HID report descriptor
// specification and passed to [New].
package hid
