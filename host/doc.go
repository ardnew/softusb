// Package host implements a pure-Go USB 1.1/2.0 host stack.
//
// It is platform-agnostic and interacts with hardware via the [hal.HostHAL]
// interface defined in the github.com/ardnew/softusb/host/hal package.
// The HAL exposes generic operations for initialization, port management,
// transfer execution, and device detection, allowing platform vendors to provide
// concrete implementations without changing the host stack.
//
// # Architecture
//
// The host stack is organized into several layers:
//
//   - Host manages the USB host controller and connected devices
//   - Device represents a connected USB device with its descriptors
//   - Transfer handles USB transfer execution
//   - Enumeration performs device discovery and configuration
//
// # Transfer Types
//
// All four USB transfer types are supported:
//
//   - Control: Setup/data/status phases for device configuration
//   - Bulk: Large data transfers with error recovery
//   - Interrupt: Periodic transfers with guaranteed latency
//   - Isochronous: Real-time streaming without retries
//
// # Device Management
//
// The host stack handles:
//
//   - Device detection on port connect/disconnect
//   - Bus enumeration and address assignment
//   - Descriptor retrieval and parsing
//   - Configuration selection
//
// # Zero-Allocation Design
//
// The stack is designed for bare-metal and TinyGo compatibility with minimal heap
// allocations. Key patterns include:
//
//   - Fixed-size arrays for device tracking
//   - Parse functions with output parameters
//   - Caller-provided buffers for transfers
//
// # Example
//
//	host := host.New(hal)
//	host.Start(ctx)
//
//	// Wait for a device to connect
//	dev, err := host.WaitDevice(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Read device descriptor
//	desc := dev.Descriptor()
//
//	// Perform transfers
//	buf := make([]byte, 64)
//	n, err := dev.BulkTransfer(ctx, 0x81, buf)
//
// A FIFO-based HAL for testing is available in
// [github.com/ardnew/softusb/host/hal/fifo].
package host
