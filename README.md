# softusb

[![Go Reference](https://pkg.go.dev/badge/github.com/ardnew/softusb.svg)](https://pkg.go.dev/github.com/ardnew/softusb)
[![Go Report Card](https://goreportcard.com/badge/github.com/ardnew/softusb)](https://goreportcard.com/report/github.com/ardnew/softusb)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Pure Go USB host and device stack with zero dependencies outside the standard library.

## Features

- Written in pure Go with no cgo, assembly, or platform-specific code
- Works on all OS/architecture combinations supported by Go
- Zero-allocation design for embedded and real-time applications
- Hardware abstraction layer (HAL) for platform portability
- Full USB 1.1 and USB 2.0 support (Low/Full/High Speed)
- Standard USB device class implementations (HID, CDC, MSC)
- Hot-plugging and dynamic device management
- Comprehensive transfer support (control, bulk, interrupt, isochronous)

## Documentation

### Device Stack

| Package | Description |
|---------|-------------|
| [device/hal](device/hal/README.md) | Device HAL interface definition |
| [device/hal/fifo](device/hal/fifo/README.md) | FIFO-based device HAL implementation |
| [device/class/cdc](device/class/cdc/README.md) | CDC-ACM (serial) class driver |
| [device/class/hid](device/class/hid/README.md) | HID class driver |

### Host Stack

| Package | Description |
|---------|-------------|
| [host/hal](host/hal/README.md) | Host HAL interface definition |
| [host/hal/fifo](host/hal/fifo/README.md) | FIFO-based host HAL implementation |

### Examples

| Example | Description |
|---------|-------------|
| [examples/fifo-hal](examples/fifo-hal/README.md) | FIFO-based HAL examples overview |
| [examples/fifo-hal/cdc-acm](examples/fifo-hal/cdc-acm/README.md) | CDC-ACM serial device example |
| [examples/fifo-hal/hid-keyboard](examples/fifo-hal/hid-keyboard/README.md) | HID keyboard device example |

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
