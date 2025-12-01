// Package fifo implements a FIFO-based HAL for USB device stacks using named pipes.
//
// This HAL is primarily intended for testing and simulation purposes. It allows
// host and device stacks to communicate via named pipes (FIFOs) in the filesystem,
// enabling integration testing of USB class drivers without actual hardware.
//
// # Architecture
//
// Each device instance creates a unique subdirectory under a shared bus directory:
//
//	/tmp/usb-bus/                    # Bus directory (shared with host)
//	└── device-{uuid}/               # Device subdirectory (unique per device)
//	    ├── connection               # Connection signaling (device → host)
//	    ├── host_to_device           # Control transfers from host (SETUP/DATA)
//	    ├── device_to_host           # Control transfer responses to host
//	    ├── ep1_in, ep1_out          # Endpoint 1 data FIFOs
//	    ├── ep2_in, ep2_out          # Endpoint 2 data FIFOs
//	    └── ...                      # (up to ep15_in/ep15_out)
//
// The UUID is generated using crypto/rand for cryptographic uniqueness,
// enabling safe parallel testing with multiple device instances.
//
// # Hot-Plugging Support
//
// The device signals connection and disconnection via the connection FIFO:
//   - 0x01: Device connected and ready
//   - 0x00: Device disconnecting
//
// This allows the host to poll for devices and handle them independently,
// supporting hot-plugging scenarios where devices connect/disconnect dynamically.
//
// # Zero-Allocation Design
//
// This implementation follows zero-allocation patterns:
//
//   - Fixed-size internal buffers for packet assembly
//   - Reuses caller-provided buffers for data transfer
//   - No dynamic memory allocation in hot paths
//
// # Usage
//
//	// Create device-side HAL with bus directory
//	hal := fifo.New("/tmp/usb-bus")
//
//	// Use with device stack
//	builder := device.NewDeviceBuilder().
//	    WithVendorProduct(0x1234, 0x5678).
//	    WithStrings("Vendor", "Product", "Serial").
//	    AddConfiguration(1)
//
//	// Configure class driver (e.g., CDC-ACM)
//	acm := cdc.NewACM()
//	acm.ConfigureDevice(builder, 0x81, 0x82, 0x02)
//
//	dev, _ := builder.Build(ctx)
//
//	// Attach class driver to interfaces (required!)
//	acm.AttachToInterfaces(dev, 1, 0, 1)
//
//	stack := device.NewStack(dev, hal)
//	acm.SetStack(stack)
//	stack.Start(ctx)
//
//	// Get the device's unique directory
//	fmt.Printf("Device directory: %s\n", hal.DeviceDir())
//
// The host-side process uses the corresponding host FIFO HAL with the same
// bus directory path to discover and communicate with devices.
package fifo
