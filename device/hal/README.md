# USB Device Hardware Abstraction Layer

> **Platform-agnostic interface for USB device controllers**

This package defines the `DeviceHAL` interface that abstracts hardware-specific USB device controller operations, enabling the USB device stack to run on any platform with an appropriate HAL implementation.

---

## Overview

The Device HAL provides a minimal, platform-agnostic interface between the USB device stack and the underlying hardware. By implementing this interface, hardware vendors can enable the pure Go USB stack to run on their platforms without any modifications to the core stack code.

### Key Characteristics

- **Minimal Interface**: Only essential operations are exposed
- **Zero Allocation**: Interface methods designed for zero-heap-allocation usage
- **Platform Agnostic**: No platform-specific types or dependencies
- **Event-Driven**: Supports both polling and interrupt-driven designs

---

## Interface

```go
// DeviceHAL defines the Hardware Abstraction Layer interface for USB device stacks.
type DeviceHAL interface {
    // Init initializes the USB controller hardware.
    Init(ctx context.Context) error

    // Start enables the USB controller and attaches to the bus.
    Start() error

    // Stop detaches from the bus and disables the USB controller.
    Stop() error

    // SetAddress sets the device address in hardware.
    SetAddress(address uint8) error

    // ConfigureEndpoints configures hardware endpoints for the active configuration.
    ConfigureEndpoints(endpoints []EndpointConfig) error

    // Control Endpoint (EP0) Operations

    // ReadSetup reads a SETUP packet from EP0.
    ReadSetup(ctx context.Context, out *SetupPacket) error

    // WriteEP0 writes data to EP0 (control IN phase).
    WriteEP0(ctx context.Context, data []byte) error

    // ReadEP0 reads data from EP0 (control OUT phase).
    ReadEP0(ctx context.Context, buf []byte) (int, error)

    // StallEP0 stalls the control endpoint to indicate an error.
    StallEP0() error

    // AckEP0 sends a zero-length packet to acknowledge a successful control transfer.
    AckEP0() error

    // Data Endpoint Operations

    // Read reads data from an OUT endpoint into buf.
    Read(ctx context.Context, address uint8, buf []byte) (int, error)

    // Write writes data to an IN endpoint.
    Write(ctx context.Context, address uint8, data []byte) (int, error)

    // Connection Status

    // Speed returns the current connection speed.
    Speed() Speed
}
```

---

## Implementing a HAL

To support a new platform, implement the `DeviceHAL` interface:

```go
package myplatform

import (
    "context"
    "github.com/ardnew/softusb/device/hal"
)

type MyPlatformHAL struct {
    // Platform-specific fields
    usbBase   uintptr
    endpoints map[uint8]endpointState
}

func New(usbBase uintptr) *MyPlatformHAL {
    return &MyPlatformHAL{
        usbBase:   usbBase,
        endpoints: make(map[uint8]endpointState),
    }
}

func (h *MyPlatformHAL) Init(ctx context.Context) error {
    // Initialize USB peripheral:
    // 1. Enable clocks
    // 2. Reset USB controller
    // 3. Configure interrupts
    // 4. Set up endpoint 0
    return nil
}

func (h *MyPlatformHAL) Connect(ctx context.Context) error {
    // Enable D+ pull-up resistor to signal device presence
    return nil
}

func (h *MyPlatformHAL) Poll(ctx context.Context) (bool, error) {
    // Check interrupt flags and handle events
    // Return true if an event was handled
    return false, nil
}

// ... implement remaining methods
```

### Best Practices

1. **Minimize Allocations**: Use fixed-size buffers and avoid heap allocations in hot paths
2. **Handle Timeouts**: Respect context cancellation in long-running operations
3. **Report Errors**: Return descriptive errors for hardware failures
4. **Thread Safety**: Document any thread-safety requirements

---

## Available Implementations

### FIFO HAL (`hal/fifo`)

A software-based HAL implementation using named pipes (FIFOs) for inter-process communication. Useful for:

- Testing without hardware
- Simulating USB communication
- Development and debugging

```go
import "github.com/ardnew/softusb/device/hal/fifo"

hal := fifo.New("/tmp/usb-test")
```

See [`hal/fifo/README.md`](fifo/README.md) for details.

---

## Types

### Speed

```go
type Speed uint8

const (
    SpeedUnknown Speed = iota // Not connected or unknown
    SpeedLow                  // 1.5 Mbps
    SpeedFull                 // 12 Mbps
    SpeedHigh                 // 480 Mbps
)
```

### EndpointConfig

```go
type EndpointConfig struct {
    Address       uint8  // Endpoint address including direction bit
    Attributes    uint8  // Transfer type and sync/usage flags
    MaxPacketSize uint16 // Maximum packet size
    Interval      uint8  // Polling interval for interrupt/isochronous
}
```

### SetupPacket

```go
type SetupPacket struct {
    RequestType uint8  // Request characteristics
    Request     uint8  // Specific request
    Value       uint16 // Request-specific value
    Index       uint16 // Request-specific index
    Length      uint16 // Number of bytes to transfer
}
```

---

## Usage with Device Stack

```go
import (
    "context"
    "github.com/ardnew/softusb/device"
    "github.com/ardnew/softusb/device/class/cdc"
    "github.com/ardnew/softusb/device/hal/fifo"
)

func main() {
    ctx := context.Background()

    // Create HAL instance
    hal := fifo.New("/tmp/usb-bus")

    // Build device
    builder := device.NewDeviceBuilder().
        WithVendorProduct(0x1234, 0x5678).
        WithStrings("Vendor", "Product", "Serial").
        AddConfiguration(1)

    // Configure class driver
    acm := cdc.NewACM()
    acm.ConfigureDevice(builder, 0x81, 0x82, 0x02)

    dev, _ := builder.Build(ctx)

    // Attach class driver to interfaces (required!)
    acm.AttachToInterfaces(dev, 1, 0, 1)

    // Create and start stack
    stack := device.NewStack(dev, hal)
    acm.SetStack(stack)

    stack.Start(ctx)
    defer stack.Stop()

    // Wait for connection
    stack.WaitConnect(ctx)

    // Device is now enumerated and ready
}
```
