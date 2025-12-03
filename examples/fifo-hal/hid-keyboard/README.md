# HID Keyboard FIFO HAL Example

> **USB HID keyboard example using FIFO-based HAL**

This directory contains a complete HID keyboard example with both device and host implementations. It demonstrates USB Human Interface Device (HID) class communication using the FIFO-based HAL for testing without physical hardware.

---

## Overview

The HID keyboard example creates a virtual USB keyboard that:

- **Device**: Types "Hello\n" repeatedly using boot keyboard reports
- **Host**: Receives and displays HID keyboard reports

Both processes communicate via named pipes (FIFOs) in a shared bus directory.

---

## Architecture

```text
┌──────────────────────┐    Bus Directory     ┌──────────────────────┐
│    Device Process    │    /tmp/usb-bus/     │    Host Process      │
│                      │                      │                      │
│  ┌────────────────┐  │   device-{uuid}/     │  ┌────────────────┐  │
│  │  HID Keyboard  │  │  ├── connection      │  │   Host Stack   │  │
│  │    Driver      │  │  ├── host_to_device  │  │                │  │
│  └───────┬────────┘  │  ├── device_to_host  │  └───────┬────────┘  │
│          │           │  └── ep1_in/out ...  │          │           │
│  ┌───────┴────────┐  │                      │  ┌───────┴────────┐  │
│  │  Device Stack  │  │                      │  │   FIFO HAL     │  │
│  │  + FIFO HAL    │←─┼──────────────────────┼─→│  (discovery)   │  │
│  └────────────────┘  │                      │  └────────────────┘  │
└──────────────────────┘                      └──────────────────────┘
```

### Hot-Plugging Support

The FIFO HAL supports hot-plugging:

- Each device creates a unique subdirectory (`device-{uuid}/`)
- The host polls the bus directory for new device subdirectories
- Devices can connect/disconnect independently
- Multiple devices can be connected sequentially

---

## Usage

### Quick Start

```bash
# Create a shared bus directory
mkdir -p /tmp/usb-bus

# Terminal 1: Start the device
cd examples/fifo-hal/hid-keyboard/device
go run . /tmp/usb-bus

# Terminal 2: Start the host
cd examples/fifo-hal/hid-keyboard/host
go run . /tmp/usb-bus
```

### Device Options

```text
Usage: device [options] <bus-dir>

Options:
  -v
        Enable verbose (debug) logging
  -json
        Use JSON log format
  -enum-timeout duration
        Timeout for enumeration (default 10s)
  -transfer-timeout duration
        Timeout for data transfers (default 5s)
```

### Host Options

```text
Usage: host [options] <bus-dir>

Options:
  -v
        Enable verbose (debug) logging
  -json
        Use JSON log format
  -hotplug-limit int
        Number of devices to service before exiting (default 1)
  -enum-timeout duration
        Timeout for enumeration (default 10s)
  -transfer-timeout duration
        Timeout for data transfers (default 5s)
```

### Examples

```bash
# Device with verbose logging
go run ./device -v /tmp/usb-bus

# Host with JSON output for structured logging
go run ./host -json /tmp/usb-bus

# Device with longer timeouts
go run ./device -enum-timeout 30s -transfer-timeout 10s /tmp/usb-bus

# Host servicing multiple devices
go run ./host -hotplug-limit 3 /tmp/usb-bus

# Host with short timeouts for testing
go run ./host -enum-timeout 5s -transfer-timeout 2s /tmp/usb-bus
```

---

## Integration Tests

This example includes integration tests that verify host-device communication:

```bash
# Run integration tests
go test -v ./examples/fifo-hal/hid-keyboard/

# Run with custom timeouts
go test -v ./examples/fifo-hal/hid-keyboard/ -args \
    -enum-timeout=15s \
    -transfer-timeout=10s

# Run with verbose logging
go test -v ./examples/fifo-hal/hid-keyboard/ -args -verbose

# Run with JSON log output
go test -v ./examples/fifo-hal/hid-keyboard/ -args -json

# Run with merged output (host and device to single stream)
go test -v ./examples/fifo-hal/hid-keyboard/ -args -m

# Combined options
go test -v ./examples/fifo-hal/hid-keyboard/ -args \
    -verbose -json -m
```

**Note:** Use `-args` to pass flags to the test binary (after the Go test flags).

### Test Flags

| Flag | Description |
|------|-------------|
| `-enum-timeout` | Timeout for USB enumeration (default 10s) |
| `-transfer-timeout` | Timeout for data transfers (default 5s) |
| `-verbose` | Enable debug logging in host and device processes |
| `-json` | Use JSON log format for structured output |
| `-m` | Merge host and device output into a single stream |

### Test Cases

| Test | Description |
|------|-------------|
| `TestHIDKeyboardIntegration` | Single device communication test |
| `TestHIDKeyboardMultipleDevices` | Hot-plugging with sequential devices |

---

## HID Report Format

### Boot Keyboard Report (8 bytes)

| Byte | Description |
|------|-------------|
| 0 | Modifier keys (bit field) |
| 1 | Reserved (always 0) |
| 2-7 | Key codes (up to 6 simultaneous keys) |

### Modifier Keys

| Bit | Key |
|-----|-----|
| 0 | Left Ctrl |
| 1 | Left Shift |
| 2 | Left Alt |
| 3 | Left GUI (Windows/Command) |
| 4 | Right Ctrl |
| 5 | Right Shift |
| 6 | Right Alt |
| 7 | Right GUI |

---

## Expected Output

### Device

```text
Starting HID keyboard device...
FIFO directory: /tmp/usb-bus
Waiting for host connection...
Host connected!
Typing 'Hello' every 2 seconds (Ctrl+C to exit)...
Typed: 'H'
Typed: 'e'
Typed: 'l'
Typed: 'l'
Typed: 'o'
Typed: '
'
```

### Host

```text
Starting USB host...
FIFO directory: /tmp/usb-bus
Waiting for device connection...
Device connected:
  Vendor ID:  0x1234
  Product ID: 0x5679
  Manufacturer: SoftUSB Example
  Product: HID Keyboard
  Serial: 87654321
HID device detected!
Interrupt IN: 0x81
Reading HID reports (Ctrl+C to stop)...
Report 1: [2 0 11 0 0 0 0 0]
  Modifiers: 0x02 [LShift]
  Key: 0x0B = 'H'
Report 2: [0 0 0 0 0 0 0 0]
Report 3: [0 0 8 0 0 0 0 0]
  Key: 0x08 = 'e'
...
```

---

## Code Structure

```text
hid-keyboard/
├── device/
│   └── main.go          # HID keyboard device implementation
├── host/
│   └── main.go          # HID keyboard host implementation
├── example_test.go      # Integration tests
├── doc.go               # Package documentation
└── README.md            # This file
```

---

## See Also

- [HID Class Driver](../../../device/class/hid/) - HID class driver documentation
- [Device FIFO HAL](../../../device/hal/fifo/) - Device-side FIFO HAL documentation
- [Host FIFO HAL](../../../host/hal/fifo/) - Host-side FIFO HAL documentation
- [CDC-ACM Example](../cdc-acm/) - Another FIFO HAL example
