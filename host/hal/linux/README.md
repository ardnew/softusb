# Linux USB Host HAL

> **Production HAL implementation for Linux using usbfs**

This package provides a USB host Hardware Abstraction Layer (HAL) implementation for Linux systems. It enables pure Go USB host functionality without cgo by leveraging the Linux kernel's usbfs and sysfs interfaces.

---

## Overview

The Linux HAL implements the `hal.HostHAL` interface using:

- **usbfs** (`/dev/bus/usb/`) — Direct USB device access and transfer execution
- **sysfs** (`/sys/bus/usb/devices/`) — Device discovery and enumeration
- **netlink** — Real-time hotplug event monitoring

### Key Features

- **Pure Go**: No cgo, assembly, or external libraries required
- **Zero Allocation**: Hot-path operations use pre-allocated buffers and free-lists
- **Async I/O**: Efficient epoll-based polling for USB transfer completion
- **Hotplug Support**: Automatic device detection via netlink sockets
- **USB 1.1/2.0**: Supports Low Speed (1.5 Mbps), Full Speed (12 Mbps), and High Speed (480 Mbps)

---

## Requirements

### Permissions

USB device access requires read/write permissions on `/dev/bus/usb/` device nodes. Options include:

1. **Running as root** (not recommended for production)

2. **udev rules** (recommended):

   ```bash
   # Generate rules for all USB devices accessible by the plugdev group
   go run github.com/ardnew/softusb/cmd/softusb-udev-rules -all -group plugdev
   ```

   See [`cmd/softusb-udev-rules`](../../../cmd/softusb-udev-rules/README.md) for detailed usage.

3. **Manual permissions**:

   ```bash
   sudo chmod 666 /dev/bus/usb/001/002  # Specific device
   ```

### Kernel Support

The following kernel features must be enabled (standard on most distributions):

- `CONFIG_USB` — USB support
- `CONFIG_USB_DEVICEFS` — USB device filesystem (usbfs)
- `CONFIG_CONNECTOR` — Kernel connector (for netlink hotplug)

---

## Architecture

### Device Tracking

```text
┌─────────────────────────────────────────────────────────────┐
│                        HostHAL                              │
├─────────────────────────────────────────────────────────────┤
│  devicePool                                                 │
│  ┌─────────┬─────────┬─────────┬─────────┐                  │
│  │ slot[0] │ slot[1] │ slot[2] │   ...   │  (MaxDevices=16) │
│  └────┬────┴────┬────┴────┬────┴─────────┘                  │
│       │         │         │                                 │
│       ▼         ▼         ▼                                 │
│  deviceConn  deviceConn  (nil)                              │
│  ┌────────┐ ┌────────┐                                      │
│  │ fd     │ │ fd     │   ← /dev/bus/usb/BBB/DDD             │
│  │ info   │ │ info   │   ← Device metadata from sysfs       │
│  │ addr   │ │ addr   │   ← Assigned USB address             │
│  └────────┘ └────────┘                                      │
└─────────────────────────────────────────────────────────────┘
```

Devices are tracked in a fixed-size pool with free-list management for O(1) allocation and deallocation.

### Async I/O Flow

```text
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Submit    │      │    Poll     │      │    Reap     │
│     URB     │ ───► │   (epoll)   │ ───► │     URB     │
└─────────────┘      └─────────────┘      └─────────────┘
       │                    │                    │
       ▼                    ▼                    ▼
 USBDEVFS_SUBMITURB   epoll_wait()      USBDEVFS_REAPURBNDELAY
```

USB Request Blocks (URBs) are submitted asynchronously. The poller uses `epoll` to efficiently wait for completion events, then reaps completed URBs without blocking.

### Hotplug Monitoring

```text
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Kernel    │      │  Netlink    │      │   HostHAL   │
│   uevents   │ ───► │   Socket    │ ───► │  Callbacks  │
└─────────────┘      └─────────────┘      └─────────────┘
                                                 │
                           ┌─────────────────────┴─────────────────────┐
                           ▼                                           ▼
                    connectCh <- port                         disconnectCh <- port
```

The hotplug monitor receives kernel uevents via a netlink socket and translates them into connect/disconnect notifications.

---

## Usage

### Basic Example

```go
package main

import (
    "context"
    "log"

    "github.com/ardnew/softusb/host"
    "github.com/ardnew/softusb/host/hal/linux"
)

func main() {
    ctx := context.Background()

    // Create Linux HAL
    hal := linux.NewHostHAL()

    // Optional: set transfer timeout (default 5000ms)
    hal.SetTransferTimeout(3000)

    // Create host stack with HAL
    h := host.New(hal)

    // Set up device callbacks
    h.SetOnDeviceConnect(func(dev *host.Device) {
        log.Printf("Connected: VID=%04X PID=%04X",
            dev.VendorID(), dev.ProductID())
    })

    h.SetOnDeviceDisconnect(func(dev *host.Device) {
        log.Printf("Disconnected: VID=%04X PID=%04X",
            dev.VendorID(), dev.ProductID())
    })

    // Start host
    if err := h.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer h.Stop()

    // Wait for context cancellation
    <-ctx.Done()
}
```

### HID Device Example

See [`examples/linux-hal/hid-monitor`](../../../examples/linux-hal/hid-monitor/README.md) for a complete example that monitors HID devices and displays input reports.

---

## API Reference

### Constructor

```go
func NewHostHAL() *HostHAL
```

Creates a new Linux host HAL instance.

### Configuration

```go
func (h *HostHAL) SetTransferTimeout(ms uint32)
```

Sets the timeout for USB transfers in milliseconds. Default is 5000ms (5 seconds).

### Lifecycle

| Method | Description |
|--------|-------------|
| `Init(ctx)` | Initialize the HAL, create poller and hotplug monitor |
| `Start()` | Begin device scanning and event processing |
| `Stop()` | Stop all background goroutines |
| `Close()` | Release all resources and close device connections |

### Transfer Methods

| Method | Description |
|--------|-------------|
| `ControlTransfer(ctx, addr, setup, data)` | Execute a control transfer |
| `BulkTransfer(ctx, addr, endpoint, data)` | Execute a bulk transfer |
| `InterruptTransfer(ctx, addr, endpoint, data)` | Execute an interrupt transfer |
| `IsochronousTransfer(ctx, addr, endpoint, data)` | Execute an isochronous transfer |

### Interface Management

| Method | Description |
|--------|-------------|
| `ClaimInterface(addr, iface)` | Claim exclusive access to an interface |
| `ReleaseInterface(addr, iface)` | Release a previously claimed interface |

### Connection Events

| Method | Description |
|--------|-------------|
| `WaitForConnection(ctx)` | Block until a device connects |
| `WaitForDisconnection(ctx)` | Block until a device disconnects |

---

## Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `MaxDevices` | 16 | Maximum tracked devices |
| `MaxEndpointsPerDevice` | 32 | Maximum endpoints per device |
| `MaxInterfacesPerDevice` | 16 | Maximum interfaces per device |
| `MaxURBsPerEndpoint` | 4 | Async I/O queue depth |
| `URBBufferSize` | 1024 | Default URB buffer size |
| `MaxControlTransferSize` | 4096 | Maximum control transfer data |

---

## Troubleshooting

### Permission Denied

```text
open /dev/bus/usb/001/002: permission denied
```

**Solution**: Configure udev rules or run as root. See [Requirements](#requirements).

### Device Busy

```text
claim interface: device or resource busy
```

**Solution**: Another driver has claimed the interface. The HAL will attempt to detach kernel drivers automatically, but some drivers (e.g., `usbhid`) may need to be unloaded:

```bash
# Check which driver is bound
cat /sys/bus/usb/devices/1-1:1.0/driver/module/name

# Unbind the driver
echo "1-1:1.0" | sudo tee /sys/bus/usb/drivers/usbhid/unbind
```

### No Devices Found

Verify USB devices are visible:

```bash
# List USB devices via sysfs
ls /sys/bus/usb/devices/

# List USB device nodes
ls -la /dev/bus/usb/
```

### Transfer Timeout

```text
transfer timed out
```

**Solution**: Increase the transfer timeout:

```go
hal.SetTransferTimeout(10000)  // 10 seconds
```

---

## See Also

- [`host/hal`](../README.md) — Host HAL interface definition
- [`host/hal/fifo`](../fifo/README.md) — FIFO-based HAL for testing
- [`cmd/softusb-udev-rules`](../../../cmd/softusb-udev-rules/README.md) — udev rules generator
- [`examples/linux-hal/hid-monitor`](../../../examples/linux-hal/hid-monitor/README.md) — HID monitoring example
