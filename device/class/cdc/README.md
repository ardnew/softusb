# CDC-ACM Class Driver

> **USB Communications Device Class - Abstract Control Model**

This package implements the CDC-ACM (Abstract Control Model) class driver for USB serial port emulation. It provides a virtual COM port interface that appears as a standard serial port on the host system.

---

## Overview

The CDC-ACM class is commonly used for USB-to-serial adapters and devices that need a simple serial communication interface. When connected to a host, the device appears as:

- **Linux**: `/dev/ttyACM0`
- **macOS**: `/dev/cu.usbmodemXXXX`
- **Windows**: `COMx`

### Key Features

- **Virtual Serial Port**: Standard serial interface on host
- **Line Coding Support**: Baud rate, data bits, parity, stop bits
- **Flow Control**: DTR/RTS line state management
- **Zero Allocation**: Efficient data transfer without heap allocations

---

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                    CDC-ACM Device                           │
├─────────────────────────────────────────────────────────────┤
│  Interface 0: Communication Class (0x02)                    │
│  ├── Subclass: Abstract Control Model (0x02)                │
│  ├── Protocol: AT Commands (0x01)                           │
│  └── Endpoint: Interrupt IN (notifications)                 │
├─────────────────────────────────────────────────────────────┤
│  Interface 1: CDC Data Class (0x0A)                         │
│  ├── Subclass: None (0x00)                                  │
│  ├── Protocol: None (0x00)                                  │
│  ├── Endpoint: Bulk IN  (device → host)                     │
│  └── Endpoint: Bulk OUT (host → device)                     │
└─────────────────────────────────────────────────────────────┘
```

---

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/ardnew/softusb/device"
    "github.com/ardnew/softusb/device/class/cdc"
    "github.com/ardnew/softusb/device/hal/fifo"
)

func main() {
    ctx := context.Background()

    // Create HAL
    hal := fifo.New("/tmp/usb-cdc")

    // Build device
    builder := device.NewDeviceBuilder().
        WithVendorProduct(0x1234, 0x5678).
        WithStrings("Vendor", "CDC Device", "Serial").
        AddConfiguration(1)

    // Create and configure CDC-ACM
    acm := cdc.NewACM()
    acm.ConfigureDevice(builder,
        0x81,  // Notification endpoint (Interrupt IN)
        0x82,  // Data IN endpoint (Bulk IN)
        0x02,  // Data OUT endpoint (Bulk OUT)
    )

    // Build device
    dev, _ := builder.Build(ctx)

    // Attach ACM driver to interfaces in configuration 1
    // (interface 0 = control, interface 1 = data)
    acm.AttachToInterfaces(dev, 1, 0, 1)

    // Create stack and set reference
    stack := device.NewStack(dev, hal)
    acm.SetStack(stack)

    stack.Start(ctx)
    defer stack.Stop()

    // Wait for connection
    stack.WaitConnect(ctx)

    // Send data to host
    acm.Write(ctx, []byte("Hello from USB!\r\n"))

    // Read data from host
    buf := make([]byte, 64)
    n, _ := acm.Read(ctx, buf)
    fmt.Printf("Received: %s\n", buf[:n])
}
```

### Handling Line Coding Changes

```go
acm := cdc.NewACM()

// Set callback for line coding changes
acm.SetOnLineCodingChange(func(coding *cdc.LineCoding) {
    fmt.Printf("Baud: %d, Data: %d, Parity: %d, Stop: %d\n",
        coding.BaudRate,
        coding.DataBits,
        coding.Parity,
        coding.StopBits)
})

// Set callback for control line state changes
acm.SetOnControlStateChange(func(dtr, rts bool) {
    fmt.Printf("DTR: %v, RTS: %v\n", dtr, rts)
})
```

---

## API

### Types

#### ACM

The main CDC-ACM driver type.

```go
type ACM struct {
    // contains filtered or unexported fields
}

func NewACM() *ACM
func (a *ACM) ConfigureDevice(builder *device.DeviceBuilder, notifyEP, dataInEP, dataOutEP uint8) *device.DeviceBuilder
func (a *ACM) AttachToInterfaces(dev *device.Device, configValue, controlIfaceNum, dataIfaceNum uint8) error
func (a *ACM) SetStack(stack *device.Stack)
func (a *ACM) Write(ctx context.Context, data []byte) (int, error)
func (a *ACM) Read(ctx context.Context, buf []byte) (int, error)
func (a *ACM) SetOnLineCodingChange(fn func(*LineCoding))
func (a *ACM) SetOnControlStateChange(fn func(dtr, rts bool))
func (a *ACM) SetOnBreak(fn func(millis uint16))
func (a *ACM) LineCoding() LineCoding
func (a *ACM) DTR() bool
func (a *ACM) RTS() bool
func (a *ACM) SendSerialState(state uint16) error
```

#### AttachToInterfaces

Attaches the ACM driver to the CDC interfaces. Must be called after `builder.Build()` and before using `Read()` or `Write()`.

```go
func (a *ACM) AttachToInterfaces(dev *device.Device, configValue, controlIfaceNum, dataIfaceNum uint8) error
```

Parameters:
- `dev`: The built device
- `configValue`: Configuration value (typically 1)
- `controlIfaceNum`: Interface number for the CDC control interface
- `dataIfaceNum`: Interface number for the CDC data interface

#### LineCoding

Line coding parameters as set by the host.

```go
type LineCoding struct {
    BaudRate uint32 // Baud rate (bits per second)
    StopBits uint8  // 0=1, 1=1.5, 2=2 stop bits
    Parity   uint8  // 0=None, 1=Odd, 2=Even, 3=Mark, 4=Space
    DataBits uint8  // 5, 6, 7, 8, or 16 data bits
}
```

### Constants

```go
// USB Class Codes
const (
    ClassCDC     = 0x02  // Communications Device Class
    ClassCDCData = 0x0A  // CDC Data Class
)

// CDC Subclass Codes
const (
    SubclassACM = 0x02  // Abstract Control Model
)

// CDC Protocol Codes
const (
    ProtocolNone = 0x00  // No protocol
    ProtocolAT   = 0x01  // AT Commands (V.250)
)

// CDC Requests
const (
    RequestSetLineCoding        = 0x20
    RequestGetLineCoding        = 0x21
    RequestSetControlLineState  = 0x22
    RequestSendBreak            = 0x23
)
```

---

## Examples

See [`examples/fifo-hal/cdc-acm/`](../../examples/fifo-hal/cdc-acm/) for complete device and host examples.

---

## Protocol Details

### Enumeration

1. Host reads device descriptor (class 0x00 - defined at interface level)
2. Host reads configuration descriptor with:
   - Communication interface (class 0x02)
   - CDC functional descriptors
   - Data interface (class 0x0A)
3. Host loads CDC-ACM driver

### Line Coding

The host sends `SET_LINE_CODING` (0x20) with 7 bytes:

| Offset | Size | Description |
|--------|------|-------------|
| 0 | 4 | Baud rate (little-endian) |
| 4 | 1 | Stop bits (0=1, 1=1.5, 2=2) |
| 5 | 1 | Parity (0=None, 1=Odd, 2=Even) |
| 6 | 1 | Data bits (5, 6, 7, 8, or 16) |

### Control Line State

The host sends `SET_CONTROL_LINE_STATE` (0x22) with:

| Bit | Signal |
|-----|--------|
| 0 | DTR (Data Terminal Ready) |
| 1 | RTS (Request To Send) |

---

## Functional Descriptors

CDC-ACM requires specific functional descriptors in the configuration:

1. **Header Functional Descriptor**: CDC version
2. **Call Management Functional Descriptor**: Call management capabilities
3. **ACM Functional Descriptor**: ACM capabilities
4. **Union Functional Descriptor**: Links communication and data interfaces

These are automatically added by `ConfigureDevice()`.
