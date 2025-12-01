// Package hal defines the Hardware Abstraction Layer interface for USB device stacks.
//
// The HAL provides a platform-agnostic interface between the device stack and
// underlying USB controller hardware. Platform vendors implement this interface
// to enable the softusb device stack on their specific hardware.
//
// # Design Principles
//
// The HAL is designed to be:
//
//   - Minimal: Only expose operations essential for USB device functionality
//   - Generic: No platform-specific assumptions or details
//   - Flexible: Adaptable to a wide range of hardware configurations
//
// The device stack implements all USB protocol logic, leaving the HAL to handle
// only low-level hardware interactions.
//
// # Interface Overview
//
// The [DeviceHAL] interface defines the contract for device-side USB operations:
//
//   - Initialization and lifecycle management
//   - Control endpoint (EP0) operations for enumeration
//   - Data endpoint operations for bulk, interrupt, and isochronous transfers
//   - Connection state and speed negotiation
//
// # Implementing a HAL
//
// To implement a HAL for a new platform:
//
//  1. Create a type that implements all [DeviceHAL] methods
//  2. Handle hardware-specific initialization in Init()
//  3. Implement EP0 operations for control transfers
//  4. Implement Read/Write for data endpoints
//  5. Track connection state and negotiated speed
//
// # Zero-Allocation Design
//
// HAL implementations should follow zero-allocation patterns where feasible:
//
//   - Reuse buffers provided by the stack
//   - Avoid allocations in the hot path (Read/Write operations)
//   - Use fixed-size internal buffers where dynamic allocation would occur
//
// # Example
//
//	type MyHAL struct {
//	    // Platform-specific fields
//	}
//
//	func (h *MyHAL) Init(ctx context.Context) error {
//	    // Initialize USB controller hardware
//	    return nil
//	}
//
//	func (h *MyHAL) Start() error {
//	    // Enable USB controller and attach to bus
//	    return nil
//	}
//
//	// ... implement remaining DeviceHAL methods
//
// A FIFO-based HAL for testing is available in
// [github.com/ardnew/softusb/device/hal/fifo].
package hal
