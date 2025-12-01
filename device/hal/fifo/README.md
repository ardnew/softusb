# FIFO-Based Device HAL

> **Software USB device HAL using named pipes for testing and simulation**

This package provides a software-based implementation of the Device HAL interface using named pipes (FIFOs) for inter-process communication. It enables testing and development of USB device applications without physical hardware.

---

## Overview

The FIFO HAL simulates USB communication by using the filesystem to exchange data between a device process and a host process. Each endpoint is represented by a pair of named pipes for bidirectional communication.

### Key Features

- **No Hardware Required**: Test USB device code on any system
- **Cross-Platform**: Works on Linux, macOS, and other Unix-like systems
- **Full USB Simulation**: Supports all transfer types and device states
- **Debugging**: Easy to inspect and log USB traffic

---

## Architecture

```text
┌─────────────────┐                          ┌─────────────────┐
│  Device Process │                          │   Host Process  │
│                 │                          │                 │
│   Device Stack  │                          │   Host Stack    │
│        │        │                          │        │        │
│   FIFO Device   │                          │   FIFO Host     │
│      HAL        │                          │      HAL        │
└────────┬────────┘                          └────────┬────────┘
         │                                            │
         │    ┌──────────────────────────────────┐    │
         └────│         Bus Directory           │────┘
              │        /tmp/usb-bus/            │
              │                                  │
              │  device-{uuid}/                  │
              │  ├── connection   (signal)       │
              │  ├── host_to_device             │
              │  ├── device_to_host             │
              │  └── interrupts                 │
              └──────────────────────────────────┘
```

### Hot-Plugging Support

Each device instance creates a unique subdirectory (`device-{uuid}/`) under the
bus directory using a cryptographically random UUID. This enables:

- **Independent Device Lifecycle**: Devices can start/stop in any order
- **Multiple Devices**: Multiple devices can connect to the same bus
- **Connection Signaling**: Device signals connect/disconnect via the `connection` FIFO

---

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/ardnew/softusb/device"
    "github.com/ardnew/softusb/device/hal/fifo"
)

func main() {
    ctx := context.Background()

    // Create FIFO HAL with bus directory path
    // Device will create: busDir/device-{uuid}/
    hal := fifo.New("/tmp/usb-bus")

    // Build your device
    builder := device.NewDeviceBuilder().
        WithVendorProduct(0x1234, 0x5678).
        WithStrings("Vendor", "Product", "Serial").
        AddConfiguration(1)

    // Add interfaces and endpoints...
    dev, _ := builder.Build(ctx)

    // Create and start the stack
    stack := device.NewStack(dev, hal)

    if err := stack.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer stack.Stop()

    // Get the device's unique directory
    fmt.Printf("Device directory: %s\n", hal.DeviceDir())

    // Wait for host connection
    stack.WaitConnect(ctx)

    // Device is ready for communication
}
```

### Running Device and Host

Both the device and host processes use the same bus directory:

```bash
# Terminal 1: Start device (creates device-{uuid}/ subdirectory)
./device /tmp/usb-bus

# Terminal 2: Start host (polls for device subdirectories)
./host /tmp/usb-bus
```

The device and host can be started in any order.

---

## API

### Constructor

```go
func New(busDir string) *HAL
```

Creates a new FIFO-based device HAL. The `busDir` is the root bus directory
shared with the host. The device will create its own subdirectory
(`device-{uuid}/`) inside busDir when initialized.

### Additional Methods

```go
func (h *HAL) DeviceDir() string
```

Returns the device's unique subdirectory path (`busDir/device-{uuid}/`).

### DeviceHAL Interface

The FIFO HAL implements the complete `hal.DeviceHAL` interface:

| Method | Description |
|--------|-------------|
| `Init(ctx)` | Creates device subdirectory and named pipes |
| `Deinit()` | Removes named pipes and cleans up |
| `Connect(ctx)` | Signals device presence to host via connection FIFO |
| `Disconnect()` | Signals device disconnection |
| `Poll(ctx)` | Checks for incoming USB events |
| `SetAddress(addr)` | Updates device address |
| `ConfigureEndpoint(...)` | Sets up endpoint pipes |
| `ReadSetup(buf)` | Reads setup packets from host |
| `ReadEP0(buf)` | Reads EP0 OUT data |
| `Read(addr, buf)` | Reads from endpoint |
| `Write(addr, data)` | Writes to endpoint |
| `Stall(addr, stall)` | Sets endpoint stall state |
| `IsConnected()` | Returns connection status |
| `Speed()` | Returns simulated USB speed |

---

## File Structure

When initialized, the FIFO HAL creates the following structure:

```text
/tmp/usb-bus/                    # Bus directory (shared with host)
└── device-a1b2c3d4e5f6/         # Device subdirectory (unique UUID)
    ├── connection               # Connection signaling (device → host)
    ├── host_to_device           # Control transfers from host (SETUP/DATA)
    ├── device_to_host           # Control transfer responses to host
    ├── interrupts               # (reserved for future use)
    ├── ep1_in                   # Endpoint 1 IN data (device → host)
    ├── ep1_out                  # Endpoint 1 OUT data (host → device)
    ├── ep2_in                   # Endpoint 2 IN data
    ├── ep2_out                  # Endpoint 2 OUT data
    └── ...                      # (up to ep15_in/ep15_out)
```

---

## Examples

### CDC-ACM Device

See [`examples/fifo-hal/cdc-acm/device/`](../../../examples/fifo-hal/cdc-acm/device/) for a complete CDC serial port device example.

### HID Keyboard

See [`examples/fifo-hal/hid-keyboard/device/`](../../../examples/fifo-hal/hid-keyboard/device/) for a HID keyboard device example.

---

## Limitations

- **Unix-Only**: Named pipes are not available on Windows (use WSL)
- **No Isochronous**: Real-time guarantees not supported in software
- **No Multi-Host**: Only one host can connect to a device at a time

---

## Debugging

Enable debug logging to see FIFO operations:

```go
import (
  "log/slog"
  "github.com/ardnew/softusb/pkg"
)

pkg.SetLogLevel(slog.LevelDebug)
```

You can also inspect the FIFOs directly:

```bash
# Watch FIFO activity
cat /tmp/usb-device/ep1_in | hexdump -C

# Send test data
echo -n "test" > /tmp/usb-device/ep1_out
```
