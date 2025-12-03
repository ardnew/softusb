# CDC-ACM FIFO HAL Example

> **USB virtual serial port example using FIFO-based HAL**

This directory contains a complete CDC-ACM (Communications Device Class - Abstract Control Model) example with both device and host implementations. It demonstrates USB serial port emulation using the FIFO-based HAL for testing without physical hardware.

---

## Overview

The CDC-ACM example creates a virtual USB serial port that:

- **Device**: Echoes any data received from the host
- **Host**: Sends test messages and reads responses

Both processes communicate via named pipes (FIFOs) in a shared bus directory.

---

## Architecture

```text
┌──────────────────────┐    Bus Directory     ┌──────────────────────┐
│    Device Process    │    /tmp/usb-bus/     │    Host Process      │
│                      │                      │                      │
│  ┌────────────────┐  │   device-{uuid}/     │  ┌────────────────┐  │
│  │  CDC-ACM ACM   │  │  ├── connection      │  │   Host Stack   │  │
│  │    Driver      │  │  ├── host_to_device  │  │                │  │
│  └───────┬────────┘  │  ├── device_to_host  │  └───────┬────────┘  │
│          │           │  ├── ep1_in/out      │          │           │
│  ┌───────┴────────┐  │  └── ep2_in/out      │  ┌───────┴────────┐  │
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
cd examples/fifo-hal/cdc-acm/device
go run . /tmp/usb-bus

# Terminal 2: Start the host
cd examples/fifo-hal/cdc-acm/host
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
go test -v ./examples/fifo-hal/cdc-acm/

# Run with custom timeouts
go test -v ./examples/fifo-hal/cdc-acm/ -args \
    -enum-timeout=15s \
    -transfer-timeout=10s

# Run with verbose logging
go test -v ./examples/fifo-hal/cdc-acm/ -args -verbose

# Run with JSON log output
go test -v ./examples/fifo-hal/cdc-acm/ -args -json

# Run with merged output (host and device to single stream)
go test -v ./examples/fifo-hal/cdc-acm/ -args -m

# Combined options
go test -v ./examples/fifo-hal/cdc-acm/ -args \
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
| `TestCDCACMIntegration` | Single device communication test |
| `TestCDCACMMultipleDevices` | Hot-plugging with sequential devices |

---

## Expected Output

### Device

```text
Starting CDC-ACM device...
Bus directory: /tmp/usb-bus
Device directory: /tmp/usb-bus/device-a1b2c3d4/
Waiting for host connection...
Host connected!
Echoing data (Ctrl+C to exit)...
Received 20 bytes: "Hello from USB Host!"
```

### Host

```text
Starting USB host...
Bus directory: /tmp/usb-bus
Waiting for device connection...
Device connected:
  Vendor ID:  0x1234
  Product ID: 0x5678
  Manufacturer: SoftUSB Example
  Product: CDC-ACM Serial Port
  Serial: 12345678
CDC-ACM device detected!
Bulk IN: 0x82, Bulk OUT: 0x02
Sending: "Hello from USB Host!"
Sent 20 bytes
Received 20 bytes: "Hello from USB Host!"
Serviced 1 device(s)
```

---

## Code Structure

```text
cdc-acm/
├── device/
│   └── main.go          # CDC-ACM device implementation
├── host/
│   └── main.go          # CDC-ACM host implementation
├── example_test.go      # Integration tests
├── doc.go               # Package documentation
└── README.md            # This file
```

---

## See Also

- [CDC Class Driver](../../../device/class/cdc/) - CDC-ACM class driver documentation
- [Device FIFO HAL](../../../device/hal/fifo/) - Device-side FIFO HAL documentation
- [Host FIFO HAL](../../../host/hal/fifo/) - Host-side FIFO HAL documentation
- [HID Keyboard Example](../hid-keyboard/) - Another FIFO HAL example
