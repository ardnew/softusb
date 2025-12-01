// Package fifo provides a FIFO-based HAL implementation for USB host stacks.
//
// This package implements the [hal.HostHAL] interface using named pipes (FIFOs)
// for inter-process communication. It's designed for testing and simulation
// purposes, allowing the host stack to communicate with device stacks running
// in separate processes.
//
// # Architecture
//
// The host polls a bus directory for device subdirectories matching the pattern
// `device-*/`. Each device creates its own subdirectory with named pipes:
//
//	/tmp/usb-bus/                    # Bus directory
//	├── device-a1b2c3d4/             # Device 1 subdirectory
//	│   ├── connection               # Connection signaling
//	│   ├── host_to_device           # Host → device commands
//	│   ├── device_to_host           # Device → host responses
//	│   ├── ep1_in, ep1_out          # Endpoint 1 data FIFOs
//	│   ├── ep2_in, ep2_out          # Endpoint 2 data FIFOs
//	│   └── ...                      # (up to ep15)
//	└── device-e5f6g7h8/             # Device 2 subdirectory
//	    └── ...                      # Same structure
//
// # Hot-Plugging Support
//
// The host polls the bus directory every 50ms for new device subdirectories.
// When a device connects, it writes 0x01 to its connection FIFO; when it
// disconnects, it writes 0x00. This enables:
//   - Independent device lifecycle (devices can start/stop in any order)
//   - Multiple devices on the same bus
//   - Dynamic device discovery
//
// # Usage
//
//	hal := fifo.NewHostHAL("/tmp/usb-bus")
//	host := host.New(hal)
//
//	ctx := context.Background()
//	if err := host.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer host.Stop()
//
//	// Wait for device connection with timeout
//	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
//	defer cancel()
//
//	dev, err := host.WaitDevice(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Communicate with device
//	data := make([]byte, 64)
//	n, err := dev.BulkTransfer(ctx, 0x81, data)
//
// # Zero-Allocation
//
// The implementation uses fixed-size internal buffers and avoids allocations
// in the hot path where possible. Callers provide buffers for data transfers.
//
// # Protocol
//
// Each FIFO message uses a simple framing protocol:
//
//	[1 byte: message type][2 bytes: length][N bytes: payload]
//
// Message types:
//   - 0x01: SETUP packet
//   - 0x02: DATA packet
//   - 0x03: ACK
//   - 0x04: NAK
//   - 0x05: STALL
//   - 0x10: Connect notification
//   - 0x11: Disconnect notification
//   - 0x12: Reset notification
package fifo
