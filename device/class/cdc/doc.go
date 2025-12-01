// Package cdc implements the USB Communications Device Class (CDC) for the
// softusb device stack.
//
// This package provides CDC-ACM (Abstract Control Model) functionality for
// implementing USB serial devices. CDC-ACM is the standard class for USB
// to serial adapters and virtual COM ports.
//
// # Architecture
//
// A CDC-ACM device consists of two interfaces:
//
//   - Control Interface (Communications Class): Handles CDC-specific requests
//     like SET_LINE_CODING and SET_CONTROL_LINE_STATE
//   - Data Interface (Data Class): Handles bulk data transfer via IN and OUT
//     endpoints
//
// # Zero-Allocation Design
//
// This implementation follows zero-allocation patterns:
//
//   - Fixed-size buffers for line coding and other CDC structures
//   - Caller-provided buffers for data transfer
//   - No dynamic allocation in hot paths
//
// # Usage
//
// To create a CDC-ACM device:
//
//	// Create the CDC-ACM class driver
//	acm := cdc.NewACM()
//
//	// Configure callbacks
//	acm.SetOnLineCodingChange(func(lc *cdc.LineCoding) {
//	    // Handle baud rate, data bits, etc. changes
//	})
//
//	// Add to a device configuration
//	builder := device.NewDeviceBuilder().
//	    WithVendorProduct(0xCAFE, 0xBABE).
//	    WithStrings("Manufacturer", "CDC Device", "12345").
//	    AddConfiguration(1)
//
//	// Add CDC-ACM interfaces (notify EP, data IN EP, data OUT EP)
//	acm.ConfigureDevice(builder, 0x81, 0x82, 0x02)
//
//	// Build device
//	dev, _ := builder.Build(ctx)
//
//	// Attach ACM driver to interfaces in configuration 1
//	// (interface 0 = control, interface 1 = data)
//	acm.AttachToInterfaces(dev, 1, 0, 1)
//
//	// Create stack and start
//	stack := device.NewStack(dev, hal)
//	acm.SetStack(stack)
//	stack.Start(ctx)
//
//	// Read and write data
//	n, _ := acm.Read(ctx, buf)
//	acm.Write(ctx, data)
//
// # CDC Descriptors
//
// The package includes functional descriptors required by CDC-ACM:
//
//   - Header Functional Descriptor
//   - Call Management Functional Descriptor
//   - ACM Functional Descriptor
//   - Union Functional Descriptor
package cdc
