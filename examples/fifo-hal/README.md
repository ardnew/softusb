# FIFO HAL Examples

> **Testable USB examples using FIFO-based HAL**

This directory contains example USB device and host applications that use the FIFO-based HAL implementations for testing and demonstration without physical hardware.

---

## Overview

These examples demonstrate how to use the SoftUSB stack to create USB device and host applications. They communicate through named pipes (FIFOs), allowing you to test USB code on any Unix-like system.

### Available Examples

| Example | Description |
|---------|-------------|
| [cdc-acm](cdc-acm/) | USB CDC-ACM (virtual serial port) device and host |
| [hid-keyboard](hid-keyboard/) | USB HID keyboard device and host |
| [msc-disk](msc-disk/) | USB Mass Storage (virtual flash drive) device and host |

---

## Running the Examples

Each example consists of a device and host application that communicate via a shared FIFO directory.

### Quick Start

```bash
# Build all examples
go build ./examples/fifo-hal/...

# Create a bus directory
mkdir /tmp/usb-bus

# Terminal 1: Start the device
./examples/fifo-hal/cdc-acm/device /tmp/usb-bus

# Terminal 2: Start the host
./examples/fifo-hal/cdc-acm/host /tmp/usb-bus
```

### Command-Line Options

All examples support the following options:

| Option | Description | Default |
|--------|-------------|---------|
| `-enum-timeout` | Timeout for enumeration | 10s |
| `-transfer-timeout` | Timeout for data transfers | 5s |
| `-hotplug-limit` | Number of devices to service (host only) | 1 |

### CDC-ACM Example

The CDC-ACM example demonstrates a virtual serial port:

**Device side:**

```bash
cd examples/fifo-hal/cdc-acm/device
go run . -enum-timeout 15s /tmp/usb-bus
```

**Host side:**

```bash
cd examples/fifo-hal/cdc-acm/host
go run . -hotplug-limit 2 /tmp/usb-bus
```

The device echoes back any data received from the host. The host sends a test message and reads the response.

### HID Keyboard Example

The HID keyboard example demonstrates a virtual keyboard:

**Device side:**

```bash
cd examples/fifo-hal/hid-keyboard/device
go run . -transfer-timeout 3s /tmp/usb-bus
```

**Host side:**

```bash
cd examples/fifo-hal/hid-keyboard/host
go run . -enum-timeout 20s /tmp/usb-bus
```

The device types "Hello" repeatedly using boot keyboard reports, and the host receives and displays the key presses.

### MSC Disk Example

The MSC disk example demonstrates a virtual USB flash drive:

**Device side:**

```bash
cd examples/fifo-hal/msc-disk/device
go run . -size 1048576 /tmp/usb-bus
```

**Host side:**

```bash
cd examples/fifo-hal/msc-disk/host
go run . /tmp/usb-bus
```

The device creates a 1MB in-memory disk using the Mass Storage Class (Bulk-Only Transport) protocol. The host enumerates the device and detects it as an MSC device.

---

## Integration Tests

Each example includes integration tests that verify host-device communication:

```bash
# Run all integration tests
go test -v ./examples/fifo-hal/...

# Run CDC-ACM tests only
go test -v ./examples/fifo-hal/cdc-acm/

# Run HID keyboard tests only
go test -v ./examples/fifo-hal/hid-keyboard/

# Run MSC disk tests only
go test -v ./examples/fifo-hal/msc-disk/
```

### Test Flags

The integration tests support custom timeout configuration:

```bash
# Override timeouts
go test -v ./examples/fifo-hal/cdc-acm/ -args \
    -enum-timeout=15s \
    -transfer-timeout=10s
```

**Note:** Use `-args` to pass flags to the test binary (after the Go test flags).

Available test flags:

| Flag | Description | Default |
|------|-------------|---------|
| `-enum-timeout` | Timeout for enumeration | 10s |
| `-transfer-timeout` | Timeout for data transfers | 5s |

---

## Architecture

```text
┌─────────────────────┐        Bus Directory        ┌────────────────────┐
│   Device Process    │       /tmp/usb-bus/         │   Host Process     │
│                     │                             │                    │
│  ┌───────────────┐  │    device-{uuid}/           │  ┌──────────────┐  │
│  │ Device Stack  │  │    ├── connection           │  │  Host Stack  │  │
│  └───────┬───────┘  │    ├── host_to_device       │  └──────┬───────┘  │
│          │          │    ├── device_to_host       │         │          │
│  ┌───────┴───────┐  │    ├── ep1_in, ep1_out      │  ┌──────┴───────┐  │
│  │   FIFO HAL    │←─┼────┼── ep2_in, ep2_out ...──┼─→│   FIFO HAL   │  │
│  └───────────────┘  │                             │  └──────────────┘  │
└─────────────────────┘                             └────────────────────┘
```

### Hot-Plugging

The FIFO HAL supports hot-plugging:

- Each device creates a unique subdirectory (`device-{uuid}/`) using a cryptographically random UUID
- The host polls the bus directory (every 50ms) for new device subdirectories
- Devices signal connection/disconnection via the `connection` FIFO
- Multiple devices can connect and disconnect independently

---

## Creating Your Own Example

  1. **Create the device application:**

      ```go
      package main

      import (
          "context"
          "github.com/ardnew/softusb/device"
          "github.com/ardnew/softusb/device/class/cdc"  // or hid, msc
          "github.com/ardnew/softusb/device/hal/fifo"
      )

      func main() {
          ctx := context.Background()

          // Create FIFO HAL
          hal := fifo.New("/tmp/my-device")

          // Build device descriptor
          builder := device.NewDeviceBuilder().
              WithVendorProduct(0x1234, 0x5678).
              WithStrings("Vendor", "Product", "Serial").
              AddConfiguration(1)

          // Create and configure class driver
          acm := cdc.NewACM()
          acm.ConfigureDevice(builder, 0x81, 0x82, 0x02)

          // Build device
          dev, _ := builder.Build(ctx)

          // Attach class driver to interfaces
          // (must be done after Build and before Start)
          acm.AttachToInterfaces(dev, 1, 0, 1)

          // Create stack and set driver reference
          stack := device.NewStack(dev, hal)
          acm.SetStack(stack)

          stack.Start(ctx)
          defer stack.Stop()

          stack.WaitConnect(ctx)

          // Your device logic here...
      }
      ```

  2. **Create the host application:**

      ```go
      package main

      import (
          "context"
          "github.com/ardnew/softusb/host"
          "github.com/ardnew/softusb/host/hal/fifo"
      )

      func main() {
          ctx := context.Background()

          // Create FIFO HAL
          hal := fifo.New("/tmp/my-device")

          // Create host
          h := host.New(hal)

          h.SetOnDeviceConnect(func(dev *host.Device) {
              // Handle connected device...
          })

          h.Start(ctx)
          defer h.Stop()

          // Your host logic here...
      }
      ```

---

## Debugging

Enable debug logging to see detailed USB traffic:

```go
import (
  "log/slog"
  "github.com/ardnew/softusb/pkg"
)

pkg.SetLogLevel(slog.LevelDebug)
```

You can also watch the FIFO directory for activity:

```bash
watch -n 0.5 'ls -la /tmp/usb-test/'
```

---

## Limitations

- **Unix-Only**: Named pipes require a Unix-like system (Linux, macOS, etc.)
- **No Real-Time**: Timing-sensitive protocols may not work correctly
- **No Isochronous**: Isochronous transfer timing not guaranteed in software
