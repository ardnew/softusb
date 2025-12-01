// Package hal defines the Hardware Abstraction Layer interface for USB host stacks.
//
// The HAL provides a platform-agnostic interface between the host stack and
// underlying USB controller hardware. Platform vendors implement this interface
// to enable the softusb host stack on their specific hardware.
//
// # Design Principles
//
// The HAL is designed to be:
//   - Minimal: Only expose operations essential for USB host functionality
//   - Generic: No platform-specific assumptions or details
//   - Flexible: Adaptable to a wide range of hardware configurations
//
// The host stack implements all USB protocol logic, leaving the HAL to handle
// only low-level hardware interactions.
//
// # Interface Overview
//
// The [HostHAL] interface defines the contract for host-side USB operations:
//   - Initialization and port management
//   - Control transfers for device enumeration
//   - Data transfers for bulk, interrupt, and isochronous endpoints
//   - Port status and device connection detection
//
// # Implementing a HAL
//
// To implement a HAL for a new platform:
//  1. Create a type that implements all [HostHAL] methods
//  2. Handle hardware-specific initialization in Init()
//  3. Implement port operations for device detection and reset
//  4. Implement control and data transfers
//  5. Track device connections and disconnections
//
// # Zero-Allocation Design
//
// HAL implementations should follow zero-allocation patterns where feasible:
//   - Reuse buffers provided by the stack
//   - Avoid allocations in the hot path (Read/Write operations)
//   - Use fixed-size internal buffers where dynamic allocation would occur
//
// # Example
//
//	type MyHostHAL struct {
//	    // Platform-specific fields
//	}
//
//	func (h *MyHostHAL) Init(ctx context.Context) error {
//	    // Initialize USB host controller hardware
//	    return nil
//	}
//
//	func (h *MyHostHAL) Start() error {
//	    // Enable host controller and port power
//	    return nil
//	}
//
//	// ... implement remaining HostHAL methods
//
// A FIFO-based HAL for testing is available in [github.com/ardnew/softusb/host/hal/fifo].
package hal
