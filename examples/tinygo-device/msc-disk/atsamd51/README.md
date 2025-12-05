# ATSAMD51 USB MSC Device HAL

> **USB Mass Storage Device for Adafruit Grand Central M4**

This example implements a USB Mass Storage Class (MSC) device HAL for the
ATSAMD51 microcontroller, specifically targeting the Adafruit Grand Central M4
board. It uses the onboard 8MB QSPI flash as the storage backend.

---

## Overview

This HAL provides direct access to the ATSAMD51's USB peripheral without using
TinyGo's built-in USB driver. It implements the `device/hal.DeviceHAL` interface
to enable the softusb device stack to run on embedded hardware.

### Features

- **Direct USB peripheral access**: Bypasses TinyGo's USB driver for full control
- **Fixed static arrays**: Zero heap allocations for embedded-friendly operation
- **Polling-based transfers**: Simple, predictable execution model
- **8MB QSPI flash storage**: Uses the onboard GD25Q64 flash chip
- **Graceful shutdown**: `SetRunning(false)` to stop all polling loops

---

## Hardware Requirements

- **Board**: Adafruit Grand Central M4 (ATSAMD51P20A)
- **Flash**: GD25Q64 8MB QSPI flash (onboard)
- **USB**: Full Speed (12 Mbps)

---

## Building and Flashing

### Prerequisites

1. Install TinyGo (0.30.0 or later recommended):
   ```bash
   # macOS
   brew install tinygo

   # Linux
   wget https://github.com/tinygo-org/tinygo/releases/download/v0.30.0/tinygo_0.30.0_amd64.deb
   sudo dpkg -i tinygo_0.30.0_amd64.deb
   ```

2. Install the softusb module in your Go workspace

### Build and Flash

```bash
# Navigate to the softusb source directory
cd /path/to/softusb

# Flash to the Grand Central M4
tinygo flash -target=grandcentral-m4 ./examples/tinygo-device/msc-disk/atsamd51
```

### Build Only (without flashing)

```bash
tinygo build -target=grandcentral-m4 -o msc-device.uf2 ./examples/tinygo-device/msc-disk/atsamd51
```

Then copy `msc-device.uf2` to the GCENTRALBOOT drive that appears when the
board is in bootloader mode (double-tap reset button).

---

## Usage

After flashing:

1. The board will reset and enumerate as a USB mass storage device
2. A new drive will appear on your host computer
3. The drive is backed by the 8MB QSPI flash

### First-Time Use

The QSPI flash may contain random data on first use. Format the drive:

- **Windows**: Right-click the drive → Format → FAT32
- **macOS**: Disk Utility → Erase → FAT32/MS-DOS
- **Linux**: `sudo mkfs.vfat /dev/sdX` (replace X with your device)

### Storage Capacity

- **Total size**: 8 MB (8,388,608 bytes)
- **Block size**: 512 bytes
- **Block count**: 16,384 blocks

---

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                    main.go                                  │
│  - Creates QSPIStorage                                      │
│  - Wires MSC driver with HAL                                │
│  - Runs disk.Run() loop                                     │
├─────────────────────────────────────────────────────────────┤
│                    hal.go                                   │
│  - Implements device/hal.DeviceHAL                          │
│  - Fixed static buffers for endpoints                       │
│  - Polling-based USB transfers                              │
├─────────────────────────────────────────────────────────────┤
│                    usb.go                                   │
│  - USB peripheral register definitions                      │
│  - Low-level USB operations                                 │
│  - Clock and pad calibration                                │
├─────────────────────────────────────────────────────────────┤
│                    qspi.go                                  │
│  - Implements msc.Storage for QSPI flash                    │
│  - 4KB sector caching for read-modify-write                 │
│  - Memory-mapped read access                                │
└─────────────────────────────────────────────────────────────┘
```

---

## Files

| File | Description |
|------|-------------|
| `main.go` | Entry point, device configuration |
| `hal.go` | USB device HAL implementation |
| `usb.go` | USB peripheral register access |
| `qspi.go` | QSPI flash storage backend |

---

## Implementation Notes

### USB Peripheral

- Base address: `0x41000000`
- Supports 8 endpoints (EP0-EP7)
- Full Speed only (12 Mbps)
- Uses endpoint descriptor table in SRAM

### QSPI Flash

- Base address: `0x42003400` (peripheral), `0x04000000` (memory-mapped)
- GD25Q64 compatible commands
- 4KB sector erase granularity
- 256-byte page program size
- Sector caching to minimize erases

### Polling vs Interrupts

This implementation uses polling for simplicity:

- `ReadSetup()` polls for RXSTP flag
- `Read()`/`Write()` poll for TRCPT flags
- `WaitConnect()` polls for EORST flag

A future enhancement could add interrupt support for better power efficiency.

---

## Limitations

1. **Full Speed only**: ATSAMD51 USB is limited to 12 Mbps
2. **No wear leveling**: Direct flash access may cause premature wear
3. **Single LUN**: Only one logical unit supported
4. **Polling overhead**: CPU-intensive compared to interrupt-driven

---

## Future Improvements

- [ ] Add interrupt-driven transfer support
- [ ] Implement wear-leveling layer
- [ ] Add LED status indicators
- [ ] Support multiple configurations
- [ ] Add power management (suspend/resume)

---

## Troubleshooting

### Device not recognized

1. Check USB cable (some cables are charge-only)
2. Try a different USB port
3. Double-tap reset to enter bootloader and reflash

### Slow performance

- This is expected for Full Speed USB (12 Mbps max)
- QSPI flash operations add latency
- Sector caching helps for sequential writes

### Data corruption

- Always "safely eject" before unplugging
- Call `Sync()` to flush pending writes
- Consider formatting the drive fresh

---

## References

- [ATSAMD51 Datasheet](https://ww1.microchip.com/downloads/en/DeviceDoc/SAM_D5x_E5x_Family_Data_Sheet_DS60001507G.pdf)
- [USB 2.0 Specification](https://www.usb.org/document-library/usb-20-specification)
- [USB Mass Storage Class Specification](https://www.usb.org/document-library/mass-storage-class-specification-overview-10)
- [GD25Q64 Datasheet](https://www.gigadevice.com/flash-memory/gd25q64c/)
- [Adafruit Grand Central M4](https://www.adafruit.com/product/4064)
