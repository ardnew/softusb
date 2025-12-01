# FIFO-Based Host HAL

> **Software USB host HAL using named pipes for testing and simulation**

This package provides a software-based implementation of the Host HAL interface using named pipes (FIFOs) for inter-process communication. It enables testing and development of USB host applications without physical hardware.

---

## Overview

The FIFO HAL simulates USB communication by using the filesystem to exchange data between a host process and a device process. It implements the same protocol as the device-side FIFO HAL for seamless interoperability.

### Key Features

- **No Hardware Required**: Test USB host code on any system
- **Cross-Platform**: Works on Linux, macOS, and other Unix-like systems
- **Full USB Simulation**: Supports control, bulk, and interrupt transfers
- **Device Enumeration**: Complete device enumeration simulation

---

## Architecture

```text
┌─────────────────┐                          ┌─────────────────┐
│   Host Process  │                          │  Device Process │
│                 │                          │                 │
│   Host Stack    │                          │   Device Stack  │
│        │        │                          │        │        │
│   FIFO Host     │                          │   FIFO Device   │
│      HAL        │                          │      HAL        │
└────────┬────────┘                          └────────┬────────┘
         │                                            │
         │    ┌──────────────────────────────────┐    │
         └────│         Bus Directory           │────┘
              │        /tmp/usb-bus/            │
              │                                  │
              │  device-{uuid}/                  │
              │  ├── connection  (device signal) │
              │  ├── host_to_device (control)   │
              │  ├── device_to_host (control)   │
              │  ├── ep1_in, ep1_out            │
              │  ├── ep2_in, ep2_out            │
              │  └── ... (up to ep15)           │
              └──────────────────────────────────┘
```

### Hot-Plugging Support

The host polls the bus directory (every 50ms) for device subdirectories matching
the pattern `device-*/`. When a new subdirectory is discovered:

1. Host reads the `connection` FIFO for connect signal (0x01)
2. Host opens the device's FIFOs for communication
3. Host performs USB enumeration
4. When device disconnects (0x00 signal), host cleans up

---

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/ardnew/softusb/host"
    "github.com/ardnew/softusb/host/hal/fifo"
)

func main() {
    ctx := context.Background()

    // Create FIFO HAL with bus directory path
    // Host will poll for device-*/ subdirectories
    hal := fifo.NewHostHAL("/tmp/usb-bus")

    // Create host stack
    h := host.New(hal)

    // Start host (begins polling for devices)
    if err := h.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer h.Stop()

    // Wait for a device to connect
    dev, err := h.WaitDevice(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Device connected: %s %s\n",
        dev.Manufacturer(), dev.Product())

    // Interact with device
    handleDevice(ctx, dev)
}

func handleDevice(ctx context.Context, dev *host.Device) {
    // Read from device (using Device's BulkTransfer method)
    buf := make([]byte, 64)
    n, err := dev.BulkTransfer(ctx, 0x81, buf) // IN endpoint
    if err != nil {
        log.Printf("Read error: %v", err)
        return
    }
    fmt.Printf("Received: %s\n", buf[:n])

    // Write to device
    _, err = dev.BulkTransfer(ctx, 0x01, []byte("Hello!")) // OUT endpoint
    if err != nil {
        log.Printf("Write error: %v", err)
    }
}
```

### Running Host and Device

Both the host and device processes use the same bus directory:

```bash
# Terminal 1: Start host (polls for devices)
./host /tmp/usb-bus

# Terminal 2: Start device (creates device-{uuid}/ subdirectory)
./device /tmp/usb-bus
```

The host and device can be started in any order.

---

## API

### Constructor

```go
func NewHostHAL(busDir string) *HostHAL
```

Creates a new FIFO-based host HAL. The `busDir` is the root bus directory
shared with devices. The host polls this directory for `device-*/` subdirectories.

### HostHAL Interface

The FIFO HAL implements the complete `hal.HostHAL` interface:

| Method | Description |
|--------|-------------|
| `Init(ctx)` | Opens connection to FIFO directory |
| `Deinit()` | Closes all file handles |
| `Start(ctx)` | Begins monitoring for devices |
| `Stop()` | Stops device monitoring |
| `Poll(ctx)` | Checks for USB events |
| `ControlTransfer(...)` | Executes control transfer |
| `BulkTransfer(...)` | Executes bulk transfer |
| `InterruptTransfer(...)` | Executes interrupt transfer |
| `GetDeviceDescriptor(...)` | Reads device descriptor |
| `GetConfigDescriptor(...)` | Reads configuration descriptor |
| `SetAddress(...)` | Assigns device address |
| `SetConfiguration(...)` | Sets active configuration |
| `GetConnectedDevices()` | Lists connected devices |
| `IsDeviceConnected(...)` | Checks device connection |
| `GetSpeed(...)` | Returns device speed |

---

## Device Enumeration

The FIFO HAL simulates the standard USB enumeration process:

1. **Device Detection**: Host discovers device subdirectory and reads connect signal
2. **Get Device Descriptor**: Read 18-byte device descriptor
3. **Set Address**: Assign unique address (1-127)
4. **Get Configuration Descriptor**: Read full configuration
5. **Set Configuration**: Activate a configuration

```go
// Wait for device connection with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

dev, err := h.WaitDevice(ctx)
if err != nil {
    log.Fatal(err)
}

// Device is fully enumerated at this point
fmt.Printf("VID: %04X\n", dev.VendorID())
fmt.Printf("PID: %04X\n", dev.ProductID())
fmt.Printf("Manufacturer: %s\n", dev.Manufacturer())
fmt.Printf("Product: %s\n", dev.Product())
fmt.Printf("Serial: %s\n", dev.SerialNumber())

for _, iface := range dev.Interfaces() {
    fmt.Printf("Interface %d: Class=%02X Subclass=%02X Protocol=%02X\n",
        iface.InterfaceNumber, iface.InterfaceClass, iface.InterfaceSubClass, iface.InterfaceProtocol)
}
```

---

## Examples

### CDC-ACM Host

See [`examples/fifo-hal/cdc-acm/host/`](../../../examples/fifo-hal/cdc-acm/host/) for a complete CDC serial port host example.

### HID Host

See [`examples/fifo-hal/hid-keyboard/host/`](../../../examples/fifo-hal/hid-keyboard/host/) for a HID keyboard host example.

---

## Limitations

- **Unix-Only**: Named pipes are not available on Windows (use WSL)
- **No Hub Support**: Hub enumeration not simulated
- **No Isochronous**: Real-time guarantees not supported in software
- **Sequential Processing**: Devices are processed one at a time

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

Monitor the FIFO directory:

```bash
# Watch for file changes
watch -n 0.5 'ls -la /tmp/usb-test/'

# Inspect control traffic
cat /tmp/usb-test/control | hexdump -C
```
