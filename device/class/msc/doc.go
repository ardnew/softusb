// Package msc implements the USB Mass Storage Class (MSC) device driver
// using Bulk-Only Transport (BOT) protocol with SCSI transparent command set.
//
// The MSC class allows a USB device to appear as a standard disk drive,
// USB flash drive, or other mass storage device to the host system.
//
// # Architecture
//
// The MSC driver consists of three main components:
//
//  1. BOT Protocol Handler - Processes CBW/CSW packets
//  2. SCSI Command Processor - Handles SCSI commands
//  3. Storage Backend - Provides block-level storage
//
// # Bulk-Only Transport (BOT) Protocol
//
// The BOT protocol uses three phases for each command:
//
//  1. Command Phase - Host sends Command Block Wrapper (CBW)
//  2. Data Phase - Optional bidirectional data transfer
//  3. Status Phase - Device sends Command Status Wrapper (CSW)
//
// # SCSI Command Support
//
// The driver implements a subset of SCSI commands sufficient for
// disk operation:
//
//   - INQUIRY - Device identification
//   - READ CAPACITY - Get disk size
//   - READ (10/16) - Read blocks
//   - WRITE (10/16) - Write blocks
//   - TEST UNIT READY - Check if ready
//   - REQUEST SENSE - Get error information
//   - MODE SENSE - Get device parameters
//   - PREVENT/ALLOW MEDIUM REMOVAL - Media lock control
//
// # Storage Backend
//
// Storage is abstracted through the Storage interface, allowing
// different backend implementations:
//
//   - MemoryStorage - In-memory RAM disk
//   - FileStorage - File-backed disk image
//   - Custom implementations - Any block device
//
// # Usage Example
//
//	// Create 1MB in-memory storage
//	storage := msc.NewMemoryStorage(1024*1024, 512)
//
//	// Create MSC driver
//	disk := msc.New(storage, "softusb", "Virtual Disk")
//
//	// Configure device with builder
//	builder := device.NewDeviceBuilder().
//	    WithVendorProduct(0x1234, 0x5680).
//	    WithStrings("softusb", "Mass Storage", "12345678").
//	    AddConfiguration(1)
//
//	// Add MSC interface (bulkIn=0x81, bulkOut=0x01)
//	disk.ConfigureDevice(builder, 0x81, 0x01)
//
//	// Build device and attach driver
//	dev, _ := builder.Build(ctx)
//	disk.AttachToInterface(dev, 1, 0)
//
//	// Create stack and start
//	stack := device.NewStack(dev, hal)
//	disk.SetStack(stack)
//	stack.Start(ctx)
//
//	// Run MSC processing loop
//	disk.Run(ctx)
//
// # References
//
//   - USB Mass Storage Class Specification 1.0
//   - USB Mass Storage Bulk-Only Transport 1.0
//   - SCSI Primary Commands (SPC-4)
//   - SCSI Block Commands (SBC-3)
package msc
