# softusb

[![Go Reference](https://pkg.go.dev/badge/github.com/ardnew/softusb.svg)](https://pkg.go.dev/github.com/ardnew/softusb)
[![Go Report Card](https://goreportcard.com/badge/github.com/ardnew/softusb)](https://goreportcard.com/report/github.com/ardnew/softusb)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Full-featured USB host and device stack written in pure Go.

## Features

- Zero-dependency: no external modules, cgo, assembly, or platform-specific code
- Zero-allocation: designed for embedded and bare-metal applications
- Full USB 1.1 and USB 2.0 support (Low/Full/High Speed)
- Comprehensive transfer support (control, bulk, interrupt, isochronous)
- Standard USB device class implementations (HID, CDC, MSC)
- Targets a [hardware abstraction layer (HAL)](#hardware-abstraction-layer-hal) for platform portability
- Asynchronous operation with [context](https://pkg.go.dev/context)-based cancellation (and no dynamic allocations)

### Hardware Abstraction Layer (HAL)

The HAL provides a platform-agnostic interface for the USB [host](host/hal/README.md) and [device](device/hal/README.md) stacks, enabling each to operate consistently across platforms.

The HAL is designed with the following principles in mind:

- Maximize portability by avoiding assumptions about target platform requirements
- Minimize complexity by implementing protocol/common logic within the stack(s)

These principles improve portability and adaptability but also sacrifice some ergonomics:

- Requires implementation-provided data structure storage and memory management
- Increased surface area of HAL interface
- Unable to automatically manage USB PHY or integrated drivers

## Documentation [![Go Reference](https://pkg.go.dev/badge/github.com/ardnew/softusb.svg)](https://pkg.go.dev/github.com/ardnew/softusb)

Always use [Go doc](https://pkg.go.dev/github.com/ardnew/softusb) for the full API reference. It includes additional package-level documentation, discussions, and usage examples.

### Device Stack

| README | Description |
|---------|-------------|
| [device/hal](device/hal/README.md) | Device HAL interface definition |
| [device/hal/fifo](device/hal/fifo/README.md) | FIFO-based device HAL implementation |
| [device/class/cdc](device/class/cdc/README.md) | CDC-ACM class driver |
| [device/class/hid](device/class/hid/README.md) | HID class driver |

### Host Stack

| README | Description |
|---------|-------------|
| [host/hal](host/hal/README.md) | Host HAL interface definition |
| [host/hal/fifo](host/hal/fifo/README.md) | FIFO-based host HAL implementation |
| [host/hal/linux](host/hal/linux/doc.go) | Linux usbfs host HAL implementation |

### Utilities

| README | Description |
|---------|-------------|
| [pkg/prof](pkg/prof/README.md) | Profiling utilities (build tag: `profile`) |

### Examples

| README | Description |
|---------|-------------|
| [examples/fifo-hal](examples/fifo-hal/README.md) | FIFO-based HAL examples overview |
| [examples/fifo-hal/cdc-acm](examples/fifo-hal/cdc-acm/README.md) | CDC-ACM serial device example |
| [examples/fifo-hal/hid-keyboard](examples/fifo-hal/hid-keyboard/README.md) | HID keyboard device example |
| [examples/linux-hal/hid-monitor](examples/linux-hal/hid-monitor/README.md) | Linux USB HID monitor example |

## Quick Start

```go
import (
    "github.com/ardnew/softusb/device"
    "github.com/ardnew/softusb/device/hal/fifo"
    "github.com/ardnew/softusb/device/class/cdc"
)

func main() {
    ctx := context.Background()

    // Create CDC-ACM driver
    acm := cdc.NewACM()

    // Build device
    builder := device.NewDeviceBuilder().
        WithVendorProduct(0xCAFE, 0xBABE).
        WithStrings("Vendor", "CDC Device", "12345").
        AddConfiguration(1)

    acm.ConfigureDevice(builder, 0x81, 0x82, 0x02)
    dev, _ := builder.Build(ctx)

    // Attach driver and start
    acm.AttachToInterfaces(dev, 1, 0, 1)
    hal := fifo.New("/tmp/usb-bus")
    stack := device.NewStack(dev, hal)
    acm.SetStack(stack)
    stack.Start(ctx)
}
```

## License

MIT License - see [LICENSE](LICENSE) for details.
