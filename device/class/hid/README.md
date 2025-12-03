# HID Class Driver

> **USB Human Interface Device Class**

This package implements the HID (Human Interface Device) class driver for creating USB input devices such as keyboards, mice, gamepads, and custom HID devices.

---

## Overview

The HID class provides a standardized way to create USB input devices that work with the built-in HID drivers on all major operating systems. No custom host-side drivers are required.

### Key Features

- **Boot Protocol Support**: Works in BIOS/UEFI environments
- **Custom Reports**: Define any HID report structure
- **Zero Allocation**: Efficient report sending without heap allocations
- **Standard Key Codes**: Complete USB HID usage tables

---

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                       HID Device                            │
├─────────────────────────────────────────────────────────────┤
│  Interface: HID Class (0x03)                                │
│  ├── Subclass: Boot Interface (0x01) or None (0x00)         │
│  ├── Protocol: Keyboard (0x01), Mouse (0x02), or None       │
│  ├── HID Descriptor                                         │
│  ├── Report Descriptor                                      │
│  └── Endpoint: Interrupt IN (reports to host)               │
└─────────────────────────────────────────────────────────────┘
```

---

## Usage

### Keyboard Example

```go
import (
    "context"
    "github.com/ardnew/softusb/device"
    "github.com/ardnew/softusb/device/class/hid"
    "github.com/ardnew/softusb/device/hal/fifo"
)

// Boot keyboard report descriptor
var keyboardDescriptor = []byte{
    0x05, 0x01, // Usage Page (Generic Desktop)
    0x09, 0x06, // Usage (Keyboard)
    0xA1, 0x01, // Collection (Application)
    0x05, 0x07, //   Usage Page (Key Codes)
    0x19, 0xE0, //   Usage Minimum (224)
    0x29, 0xE7, //   Usage Maximum (231)
    0x15, 0x00, //   Logical Minimum (0)
    0x25, 0x01, //   Logical Maximum (1)
    0x75, 0x01, //   Report Size (1)
    0x95, 0x08, //   Report Count (8)
    0x81, 0x02, //   Input (Data, Variable, Absolute) - Modifier byte
    0x95, 0x01, //   Report Count (1)
    0x75, 0x08, //   Report Size (8)
    0x81, 0x01, //   Input (Constant) - Reserved byte
    0x95, 0x06, //   Report Count (6)
    0x75, 0x08, //   Report Size (8)
    0x15, 0x00, //   Logical Minimum (0)
    0x25, 0x65, //   Logical Maximum (101)
    0x05, 0x07, //   Usage Page (Key Codes)
    0x19, 0x00, //   Usage Minimum (0)
    0x29, 0x65, //   Usage Maximum (101)
    0x81, 0x00, //   Input (Data, Array) - Key array
    0xC0,       // End Collection
}

func main() {
    ctx := context.Background()

    // Create HAL
    hal := fifo.New("/tmp/usb-keyboard")

    // Build device
    builder := device.NewDeviceBuilder().
        WithVendorProduct(0x1234, 0x5679).
        WithStrings("Vendor", "USB Keyboard", "Serial").
        AddConfiguration(1)

    // Create HID keyboard
    keyboard := hid.New(keyboardDescriptor)
    keyboard.ConfigureDevice(builder, 0x81, hid.SubclassBoot, hid.ProtocolKeyboard)

    // Build device
    dev, _ := builder.Build(ctx)

    // Attach HID driver to interface in configuration 1 (interface 0)
    keyboard.AttachToInterface(dev, 1, 0)

    // Create stack and set reference
    stack := device.NewStack(dev, hal)
    keyboard.SetStack(stack)

    stack.Start(ctx)
    defer stack.Stop()

    // Wait for connection
    stack.WaitConnect(ctx)

    // Send key press (boot keyboard report format)
    // [modifiers, reserved, key1, key2, key3, key4, key5, key6]
    report := []byte{0x00, 0x00, hid.KeyA, 0, 0, 0, 0, 0}
    keyboard.SendReport(ctx, report)

    // Release key
    keyboard.SendReport(ctx, []byte{0, 0, 0, 0, 0, 0, 0, 0})
}
```

### Mouse Example

```go
// Boot mouse report descriptor
var mouseDescriptor = []byte{
    0x05, 0x01, // Usage Page (Generic Desktop)
    0x09, 0x02, // Usage (Mouse)
    0xA1, 0x01, // Collection (Application)
    0x09, 0x01, //   Usage (Pointer)
    0xA1, 0x00, //   Collection (Physical)
    0x05, 0x09, //     Usage Page (Buttons)
    0x19, 0x01, //     Usage Minimum (1)
    0x29, 0x03, //     Usage Maximum (3)
    0x15, 0x00, //     Logical Minimum (0)
    0x25, 0x01, //     Logical Maximum (1)
    0x95, 0x03, //     Report Count (3)
    0x75, 0x01, //     Report Size (1)
    0x81, 0x02, //     Input (Data, Variable, Absolute) - Buttons
    0x95, 0x01, //     Report Count (1)
    0x75, 0x05, //     Report Size (5)
    0x81, 0x01, //     Input (Constant) - Padding
    0x05, 0x01, //     Usage Page (Generic Desktop)
    0x09, 0x30, //     Usage (X)
    0x09, 0x31, //     Usage (Y)
    0x15, 0x81, //     Logical Minimum (-127)
    0x25, 0x7F, //     Logical Maximum (127)
    0x75, 0x08, //     Report Size (8)
    0x95, 0x02, //     Report Count (2)
    0x81, 0x06, //     Input (Data, Variable, Relative) - X, Y
    0xC0,       //   End Collection
    0xC0,       // End Collection
}

mouse := hid.New(mouseDescriptor)
mouse.ConfigureDevice(builder, 0x81, hid.SubclassBoot, hid.ProtocolMouse)

// Build device and attach driver
dev, _ := builder.Build(ctx)
mouse.AttachToInterface(dev, 1, 0) // config 1, interface 0

stack := device.NewStack(dev, hal)
mouse.SetStack(stack)

// Send mouse movement: [buttons, x, y]
mouse.SendReport(ctx, []byte{0x00, 10, -5}) // Move right 10, up 5
mouse.SendReport(ctx, []byte{0x01, 0, 0})   // Left button press
mouse.SendReport(ctx, []byte{0x00, 0, 0})   // Release
```

---

## API

### Types

#### HID

The main HID driver type.

```go
type HID struct {
    // contains filtered or unexported fields
}

func New(reportDescriptor []byte) *HID
func (h *HID) ConfigureDevice(builder *device.DeviceBuilder, inEP uint8, subclass, protocol uint8) *device.DeviceBuilder
func (h *HID) ConfigureDeviceWithOutEP(builder *device.DeviceBuilder, inEP, outEP uint8, subclass, protocol uint8) *device.DeviceBuilder
func (h *HID) AttachToInterface(dev *device.Device, configValue, ifaceNum uint8) error
func (h *HID) SetStack(stack *device.Stack)
func (h *HID) SendReport(ctx context.Context, report []byte) error
func (h *HID) SendKeyboardReport(ctx context.Context, report *KeyboardReport) error
func (h *HID) SendMouseReport(ctx context.Context, report *MouseReport) error
func (h *HID) ReceiveReport(ctx context.Context, buf []byte) (int, error)
func (h *HID) SetOnOutputReport(fn func(data []byte))
func (h *HID) SetOnFeatureReport(fn func(reportID uint8, data []byte))
func (h *HID) SetOnSetProtocol(fn func(protocol uint8))
func (h *HID) SetOnSetIdle(fn func(rate, reportID uint8))
func (h *HID) ReportDescriptor() []byte
func (h *HID) Protocol() uint8
func (h *HID) IdleRate() uint8
```

#### AttachToInterface

Attaches the HID driver to a HID interface. Must be called after `builder.Build()` and before using `SendReport()`.

```go
func (h *HID) AttachToInterface(dev *device.Device, configValue, ifaceNum uint8) error
```

Parameters:

- `dev`: The built device
- `configValue`: Configuration value (typically 1)
- `ifaceNum`: Interface number for the HID interface

### Constants

#### Class and Subclass

```go
const (
    ClassHID       = 0x03  // HID class code
    SubclassNone   = 0x00  // No subclass
    SubclassBoot   = 0x01  // Boot interface subclass
)
```

#### Protocols

```go
const (
    ProtocolNone     = 0x00  // No protocol
    ProtocolKeyboard = 0x01  // Keyboard boot protocol
    ProtocolMouse    = 0x02  // Mouse boot protocol
)
```

#### Key Codes

```go
const (
    KeyA         = 0x04
    KeyB         = 0x05
    // ... through KeyZ = 0x1D
    Key1         = 0x1E
    Key2         = 0x1F
    // ... through Key0 = 0x27
    KeyEnter     = 0x28
    KeyEscape    = 0x29
    KeyBackspace = 0x2A
    KeyTab       = 0x2B
    KeySpace     = 0x2C
    // ... and more
)
```

#### Modifier Keys

```go
const (
    ModLeftCtrl   = 0x01
    ModLeftShift  = 0x02
    ModLeftAlt    = 0x04
    ModLeftGUI    = 0x08
    ModRightCtrl  = 0x10
    ModRightShift = 0x20
    ModRightAlt   = 0x40
    ModRightGUI   = 0x80
)
```

---

## Report Formats

### Boot Keyboard Report (8 bytes)

| Byte | Description |
|------|-------------|
| 0 | Modifier keys (bit field) |
| 1 | Reserved (always 0) |
| 2-7 | Key codes (up to 6 simultaneous keys) |

### Boot Mouse Report (3 bytes)

| Byte | Description |
|------|-------------|
| 0 | Buttons (bit 0=left, 1=right, 2=middle) |
| 1 | X movement (-127 to 127) |
| 2 | Y movement (-127 to 127) |

---

## Examples

See [`examples/fifo-hal/hid-keyboard/`](../../examples/fifo-hal/hid-keyboard/) for complete device and host examples.

---

## HID Descriptors

### HID Descriptor

The HID descriptor is automatically generated and includes:

- HID specification version (1.11)
- Country code (0 = not localized)
- Number of class descriptors
- Report descriptor type and length

### Report Descriptor

The report descriptor defines the format and meaning of reports. It uses a compact binary format with:

- **Usage Pages**: Categories of usages (e.g., Generic Desktop, Keyboard)
- **Usages**: Specific controls (e.g., X axis, Button 1)
- **Collections**: Groupings of related items
- **Input/Output/Feature**: Data direction and type

### Common Report Descriptor Items

| Tag | Description |
|-----|-------------|
| 0x05 | Usage Page (global) |
| 0x09 | Usage (local) |
| 0xA1 | Collection start |
| 0xC0 | Collection end |
| 0x15 | Logical Minimum |
| 0x25 | Logical Maximum |
| 0x75 | Report Size (bits) |
| 0x95 | Report Count |
| 0x81 | Input |
| 0x91 | Output |

---

## Boot Protocol vs Report Protocol

### Boot Protocol

- Fixed report formats (keyboard: 8 bytes, mouse: 3 bytes)
- Works in BIOS/UEFI before OS loads
- Subclass must be 0x01 (Boot Interface)
- Limited functionality

### Report Protocol

- Custom report formats defined by descriptor
- Full feature support
- OS switches from boot to report protocol after loading

The device should support both protocols. Use `SubclassBoot` and the appropriate `Protocol*` constant for boot-capable devices.
