# USB Host Hardware Abstraction Layer

> **Platform-agnostic interface for USB host controllers**

This package defines the `HostHAL` interface that abstracts hardware-specific USB host controller operations, enabling the USB host stack to run on any platform with an appropriate HAL implementation.

---

## Overview

The Host HAL provides a minimal, platform-agnostic interface between the USB host stack and the underlying hardware. By implementing this interface, hardware vendors can enable the pure Go USB host stack to run on their platforms without any modifications to the core stack code.

### Key Characteristics

- **Minimal Interface**: Only essential operations are exposed
- **Zero Allocation**: Interface methods designed for zero-heap-allocation usage
- **Platform Agnostic**: No platform-specific types or dependencies
- **Event-Driven**: Supports both polling and interrupt-driven designs

---

## Interface

```go
// HostHAL defines the Hardware Abstraction Layer interface for USB host stacks.
type HostHAL interface {
    // Initialization and Lifecycle

    // Init initializes the USB host controller hardware.
    Init(ctx context.Context) error

    // Start enables the host controller and applies power to ports.
    Start() error

    // Stop disables the host controller and removes power from ports.
    Stop() error

    // Close releases all resources associated with the HAL.
    Close() error

    // Port Operations

    // NumPorts returns the number of root hub ports.
    NumPorts() int

    // GetPortStatus returns the status of a port (1-indexed).
    GetPortStatus(port int) (PortStatus, error)

    // PortSpeed returns the connection speed of a device on the given port.
    PortSpeed(port int) Speed

    // ResetPort initiates a port reset (1-indexed).
    ResetPort(port int) error

    // EnablePort enables or disables a port.
    EnablePort(port int, enable bool) error

    // Control Transfers

    // ControlTransfer performs a control transfer to a device.
    ControlTransfer(ctx context.Context, addr DeviceAddress, setup *SetupPacket, data []byte) (int, error)

    // Data Transfers

    // BulkTransfer performs a bulk transfer to/from an endpoint.
    BulkTransfer(ctx context.Context, addr DeviceAddress, endpoint uint8, data []byte) (int, error)

    // InterruptTransfer performs an interrupt transfer to/from an endpoint.
    InterruptTransfer(ctx context.Context, addr DeviceAddress, endpoint uint8, data []byte) (int, error)

    // IsochronousTransfer performs an isochronous transfer.
    IsochronousTransfer(ctx context.Context, addr DeviceAddress, endpoint uint8, data []byte) (int, error)

    // Device Management

    // SetDeviceAddress assigns an address to a device at address 0.
    SetDeviceAddress(ctx context.Context, newAddr DeviceAddress) error

    // Interface Management

    // ClaimInterface claims exclusive access to an interface on a device.
    ClaimInterface(addr DeviceAddress, iface uint8) error

    // ReleaseInterface releases a previously claimed interface.
    ReleaseInterface(addr DeviceAddress, iface uint8) error

    // Connection Events

    // WaitForConnection blocks until a device connects or context is cancelled.
    WaitForConnection(ctx context.Context) (int, error)

    // WaitForDisconnection blocks until a device disconnects or context is cancelled.
    WaitForDisconnection(ctx context.Context) (int, error)
}
```

---

## Types

### DeviceAddress

```go
type DeviceAddress uint8
```

Represents a USB device address (1-127). Address 0 is reserved for unconfigured devices.

### TransferType

```go
type TransferType uint8

const (
    TransferControl     TransferType = 0x00
    TransferIsochronous TransferType = 0x01
    TransferBulk        TransferType = 0x02
    TransferInterrupt   TransferType = 0x03
)
```

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

---

## Implementing a HAL

To support a new platform, implement the `HostHAL` interface:

```go
package myplatform

import (
    "context"
    "github.com/ardnew/softusb/host/hal"
)

type MyPlatformHAL struct {
    // Platform-specific fields
    usbBase uintptr
    devices map[hal.DeviceAddress]*deviceState
}

func New(usbBase uintptr) *MyPlatformHAL {
    return &MyPlatformHAL{
        usbBase: usbBase,
        devices: make(map[hal.DeviceAddress]*deviceState),
    }
}

func (h *MyPlatformHAL) Init(ctx context.Context) error {
    // Initialize USB host controller:
    // 1. Enable clocks
    // 2. Reset controller
    // 3. Configure root hub
    // 4. Enable port power
    return nil
}

func (h *MyPlatformHAL) ControlTransfer(ctx context.Context, addr hal.DeviceAddress, setup, data []byte) (int, error) {
    // Execute control transfer:
    // 1. Setup stage (send 8-byte setup packet)
    // 2. Data stage (send/receive data if any)
    // 3. Status stage (handshake)
    return len(data), nil
}

func (h *MyPlatformHAL) Poll(ctx context.Context) (bool, error) {
    // Check for:
    // 1. Device connect/disconnect
    // 2. Transfer completion
    // 3. Errors
    return false, nil
}

// ... implement remaining methods
```

### Best Practices

1. **Device Tracking**: Maintain a map of connected devices and their states
2. **Error Recovery**: Handle NAK, STALL, and timeout conditions appropriately
3. **Hub Support**: For full USB support, implement hub enumeration
4. **Power Management**: Control VBUS power per-port if supported

---

## Available Implementations

### FIFO HAL (`hal/fifo`)

A software-based HAL implementation using named pipes (FIFOs) for inter-process communication. Useful for:

- Testing without hardware
- Simulating USB communication
- Development and debugging

```go
import "github.com/ardnew/softusb/host/hal/fifo"

hal := fifo.New("/tmp/usb-test")
```

See [`hal/fifo/README.md`](fifo/README.md) for details.

### Linux HAL (`hal/linux`)

A production HAL implementation for Linux systems using the kernel's usbfs interface. Features:

- Device discovery via sysfs (`/sys/bus/usb/devices/`)
- Device access via usbfs (`/dev/bus/usb/`)
- Hotplug detection via netlink
- Async I/O via epoll for efficient polling
- No cgo dependencies

```go
import "github.com/ardnew/softusb/host/hal/linux"

hal := linux.NewHostHAL()
```

See [`hal/linux/doc.go`](linux/doc.go) for requirements and architecture details.

---

## Usage with Host Stack

```go
import (
    "context"
    "github.com/ardnew/softusb/host"
    "github.com/ardnew/softusb/host/hal/fifo"
)

func main() {
    ctx := context.Background()

    // Create HAL instance
    hal := fifo.New("/tmp/usb-bus")

    // Create host stack
    h := host.New(hal)

    // Set device connect callback
    h.SetOnDeviceConnect(func(dev *host.Device) {
        fmt.Printf("Device connected: VID=%04X PID=%04X\n",
            dev.VendorID(), dev.ProductID())
    })

    // Start host
    if err := h.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer h.Stop()

    // Host is now running and will enumerate devices
    <-ctx.Done()
}
```
