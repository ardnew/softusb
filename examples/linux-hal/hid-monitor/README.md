# HID Monitor Example

This example demonstrates how to use the Linux host HAL to monitor USB HID devices.

## Features

- Detects USB HID devices (keyboards, mice, joysticks)
- Hotplug support for device connect/disconnect
- Claims HID interfaces and reads interrupt IN endpoints
- Structured logging with optional JSON output
- Device summary on demand (Ctrl+T)

## Requirements

- Linux operating system
- Read/write access to USB device nodes (`/dev/bus/usb/`)

### Granting USB Access

Option 1: Run as root (not recommended for production)

```bash
sudo go run .
```

Option 2: Use udev rules (recommended)

Use the `softusb-udev-rules` command to generate appropriate udev rules:

```bash
go run github.com/ardnew/softusb/cmd/softusb-udev-rules
```

Or create a file `/etc/udev/rules.d/99-usb-hid.rules` manually:

```bash
# Allow user access to all USB HID devices
SUBSYSTEM=="usb", ATTR{bInterfaceClass}=="03", MODE="0666"
```

Then reload udev rules:

```bash
sudo udevadm control --reload-rules
sudo udevadm trigger
```

## Usage

```bash
# Run with default settings
go run .

# Enable verbose logging
go run . -v

# Output as JSON (useful for log processing)
go run . -json

# Filter by vendor ID (hex)
go run . -vid 046d

# Filter by vendor and product ID (hex)
go run . -vid 046d -pid c52b
```

### Interactive Commands

While running:

- **Ctrl+T**: Print summary of all connected devices
- **Ctrl+C**: Exit the monitor

## Output

The monitor uses structured logging via `slog`. Default text output:

```bash
time=2024-01-15T10:30:00.000Z level=INFO msg=started component=monitor message="Waiting for HID devices... (Ctrl+T for device summary, Ctrl+C to exit)"
time=2024-01-15T10:30:01.000Z level=INFO msg="device connected" component=monitor port=1 speed=Full
time=2024-01-15T10:30:01.100Z level=INFO msg="device enumerated" component=monitor port=1 vid=1133 pid=50475 speed=Full manufacturer=Logitech product="USB Receiver"
time=2024-01-15T10:30:01.200Z level=INFO msg="hid report" component=monitor port=1 interface=0 length=8 data=0000000400000000
```

With `-json` flag:

```json
{"time":"2024-01-15T10:30:01.000Z","level":"INFO","msg":"device connected","component":"monitor","port":1,"speed":"Full"}
```

## Architecture

The example uses the Linux host HAL which provides:

- **sysfs**: Device discovery via `/sys/bus/usb/devices/`
- **usbfs**: Device access via `/dev/bus/usb/BBB/DDD`
- **netlink**: Hotplug event monitoring
- **epoll**: Async I/O for efficient polling
