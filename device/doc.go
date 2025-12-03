// Package device implements a pure-Go USB 1.1/2.0 device stack.
//
// It is platform-agnostic and interacts with hardware via the
// [hal.DeviceHAL] interface defined in the [github.com/ardnew/softusb/device/hal]
// package. The HAL exposes generic operations for initialization, connection,
// endpoint configuration, data I/O, and event polling, allowing platform
// vendors to provide concrete implementations without changing the device stack.
//
// # Architecture
//
// The device stack is organized into several layers:
//
//   - [Device] manages device state, descriptors, and endpoint registry
//   - [Stack] orchestrates USB protocol handling and event dispatch
//   - [Endpoint] handles individual endpoint configuration and data toggle
//   - [Interface] groups endpoints and manages class drivers
//   - [Transfer] represents in-flight data transfers
//
// # Transfer Types
//
// All four USB transfer types are supported:
//
//   - Control: Setup/data/status phases for device configuration
//   - Bulk: Large data transfers with error recovery
//   - Interrupt: Periodic transfers with guaranteed latency
//   - Isochronous: Real-time streaming without retries (USB Audio, etc.)
//
// # Device States
//
// The stack implements the USB 2.0 device state machine:
//
//	Attached → Powered → Default → Address → Configured → Suspended
//
// # Zero-Allocation Design
//
// The stack is designed for bare-metal and TinyGo compatibility with minimal
// heap allocations. Key patterns include:
//
//   - Serialization via MarshalTo(buf) instead of allocating Bytes()
//   - Parse functions with output parameters instead of returning pointers
//   - Fixed-size arrays instead of maps for endpoints, interfaces, etc.
//   - Caller-provided buffers for descriptor and string generation
//
// # Class Drivers
//
// The [ClassDriver] interface enables USB class implementations:
//
//	type ClassDriver interface {
//	    Init(iface *Interface) error
//	    HandleSetup(iface *Interface, setup *SetupPacket, data []byte) (bool, error)
//	    SetAlternate(iface *Interface, alt uint8) error
//	    Close() error
//	}
//
// Built-in support includes:
//
//   - [github.com/ardnew/softusb/device/class/hid] - Human Interface Device
//   - [github.com/ardnew/softusb/device/class/cdc] - Communications Device Class (CDC-ACM)
//   - [github.com/ardnew/softusb/device/class/msc] - Mass Storage Class (Bulk-Only Transport)
//
// Additional classes (USB Audio, CDC-ETM) can be implemented via this interface.
//
// # Example
//
//	dev := device.NewDevice(&device.DeviceDescriptor{
//	    USBVersion:    0x0200,
//	    VendorID:      0xCAFE,
//	    ProductID:     0xBABE,
//	    MaxPacketSize0: 64,
//	})
//	stack := device.NewStack(dev, hal)
//	stack.Start(ctx)
//
// A FIFO-based HAL for testing is available in
// [github.com/ardnew/softusb/device/hal/fifo].
package device
