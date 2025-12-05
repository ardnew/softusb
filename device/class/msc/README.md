# MSC Class Driver

> **USB Mass Storage Class - Bulk-Only Transport**

This package implements the USB Mass Storage Class (MSC) device driver using Bulk-Only Transport (BOT) protocol with SCSI transparent command set. It enables creation of USB flash drives, virtual disks, and other mass storage devices.

---

## Overview

The MSC class driver allows a USB device to appear as a standard block device (disk drive) to the host system. When connected, the device appears as:

- **Linux**: `/dev/sdX` (e.g., `/dev/sdb`, `/dev/sdc`)
- **macOS**: `/dev/diskX`
- **Windows**: Drive letter (e.g., `E:`, `F:`)

### Key Features

- **Bulk-Only Transport (BOT)**: Industry-standard protocol
- **SCSI Transparent Command Set**: Full SCSI command support
- **Storage Abstraction**: Pluggable storage backends
- **Zero Allocation**: Efficient block transfers
- **Read/Write Support**: Full bidirectional data transfer

---

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                    MSC Device                               │
├─────────────────────────────────────────────────────────────┤
│  Interface: Mass Storage Class (0x08)                       │
│  ├── Subclass: SCSI Transparent (0x06)                      │
│  ├── Protocol: Bulk-Only Transport (0x50)                   │
│  ├── Endpoint: Bulk IN  (device → host)                     │
│  └── Endpoint: Bulk OUT (host → device)                     │
├─────────────────────────────────────────────────────────────┤
│  BOT Protocol                                               │
│  ├── Command Phase (CBW)                                    │
│  ├── Data Phase (optional)                                  │
│  └── Status Phase (CSW)                                     │
├─────────────────────────────────────────────────────────────┤
│  SCSI Commands                                              │
│  ├── INQUIRY                                                │
│  ├── READ CAPACITY                                          │
│  ├── READ (10/16)                                           │
│  ├── WRITE (10/16)                                          │
│  └── ... and more                                           │
├─────────────────────────────────────────────────────────────┤
│  Storage Backend                                            │
│  ├── MemoryStorage (RAM disk)                               │
│  ├── FileStorage (disk image)                               │
│  └── Custom implementations                                 │
└─────────────────────────────────────────────────────────────┘
```

---

## Usage

### In-Memory Storage Example

```go
import (
    "context"
    "github.com/ardnew/softusb/device"
    "github.com/ardnew/softusb/device/class/msc"
    "github.com/ardnew/softusb/device/hal/fifo"
)

func main() {
    ctx := context.Background()

    // Create 1MB in-memory storage
    storage := msc.NewMemoryStorage(1024*1024, 512)

    // Create MSC driver
    disk := msc.New(storage, "softusb", "Virtual Disk")

    // Build device
    builder := device.NewDeviceBuilder().
        WithVendorProduct(0x1234, 0x5680).
        WithStrings("softusb example", "Mass Storage Device", "12345678").
        AddConfiguration(1)

    // Configure MSC interface (bulkIn=0x81, bulkOut=0x01)
    disk.ConfigureDevice(builder, 0x81, 0x01)

    // Build device
    dev, _ := builder.Build(ctx)

    // Attach MSC driver to interface in configuration 1 (interface 0)
    disk.AttachToInterface(dev, 1, 0)

    // Create stack and set reference
    hal := fifo.New("/tmp/usb-msc")
    stack := device.NewStack(dev, hal)
    disk.SetStack(stack)

    stack.Start(ctx)
    defer stack.Stop()

    // Run MSC processing loop
    disk.Run(ctx)
}
```

### File-Backed Storage Example

```go
// Create 10MB disk image file
f, _ := os.Create("disk.img")
f.Truncate(10 * 1024 * 1024)
f.Close()

// Open as storage backend
storage, _ := msc.NewFileStorage("disk.img", 512, false)
defer storage.Close()

// Create MSC driver with file storage
disk := msc.New(storage, "softusb", "File Disk")

// ... rest of setup same as above
```

### Read-Only Storage

```go
storage := msc.NewMemoryStorage(1024*1024, 512)
storage.SetReadOnly(true)

disk := msc.New(storage, "softusb", "Read-Only Disk")
// ... continue with device setup
```

### Removable Media

```go
storage := msc.NewMemoryStorage(1024*1024, 512)
storage.SetRemovable(true)

disk := msc.New(storage, "softusb", "Removable Disk")

// Later: eject media
storage.Eject()

// Reinsert media
storage.SetPresent(true)
```

---

## API

### Types

#### MSC

The main MSC driver type.

```go
type MSC struct {
    // contains filtered or unexported fields
}

func New(storage Storage, vendorID, productID string) *MSC
func (m *MSC) ConfigureDevice(builder *device.DeviceBuilder, bulkInEP, bulkOutEP uint8) *device.DeviceBuilder
func (m *MSC) AttachToInterface(dev *device.Device, configValue, ifaceNum uint8) error
func (m *MSC) SetStack(stack *device.Stack)
func (m *MSC) SetMaxLUN(lun uint8)
func (m *MSC) Run(ctx context.Context) error
```

#### Storage

Interface for storage backends.

```go
type Storage interface {
    BlockSize() uint32
    BlockCount() uint64
    Read(lba uint64, blocks uint32, buf []byte) (uint32, error)
    Write(lba uint64, blocks uint32, buf []byte) (uint32, error)
    Sync() error
    IsReadOnly() bool
    IsRemovable() bool
    IsPresent() bool
    Eject() error
}
```

#### MemoryStorage

In-memory storage implementation.

```go
type MemoryStorage struct { ... }

func NewMemoryStorage(size uint64, blockSize uint32) *MemoryStorage
func (m *MemoryStorage) SetReadOnly(readOnly bool)
func (m *MemoryStorage) SetRemovable(removable bool)
func (m *MemoryStorage) SetPresent(present bool)
```

#### FileStorage

File-backed storage implementation.

```go
type FileStorage struct { ... }

func NewFileStorage(path string, blockSize uint32, readOnly bool) (*FileStorage, error)
func (f *FileStorage) Close() error
```

### Constants

```go
// USB Class Codes
const (
    ClassMSC         = 0x08
    SubclassSCSI     = 0x06
    ProtocolBulkOnly = 0x50
)

// SCSI Commands
const (
    SCSITestUnitReady        = 0x00
    SCSIRequestSense         = 0x03
    SCSIInquiry              = 0x12
    SCSIReadCapacity10       = 0x25
    SCSIRead10               = 0x28
    SCSIWrite10              = 0x2A
    // ... and more
)

// Default Values
const (
    DefaultBlockSize  = 512
    MaxTransferSize   = 65536
)
```

---

## Supported SCSI Commands

The driver implements the following SCSI commands:

| Command | Opcode | Description |
|---------|--------|-------------|
| TEST UNIT READY | 0x00 | Check if device is ready |
| REQUEST SENSE | 0x03 | Get error information |
| INQUIRY | 0x12 | Get device identification |
| MODE SENSE (6) | 0x1A | Get device parameters |
| START/STOP UNIT | 0x1B | Start/stop or eject media |
| PREVENT/ALLOW MEDIUM REMOVAL | 0x1E | Lock/unlock media |
| READ FORMAT CAPACITIES | 0x23 | Get format information |
| READ CAPACITY (10) | 0x25 | Get disk capacity (32-bit) |
| READ (10) | 0x28 | Read blocks (32-bit LBA) |
| WRITE (10) | 0x2A | Write blocks (32-bit LBA) |
| VERIFY (10) | 0x2F | Verify blocks |
| SYNCHRONIZE CACHE (10) | 0x35 | Flush write cache |
| SERVICE ACTION IN (16) | 0x9E | Extended commands |
| └─ READ CAPACITY (16) | 0x10 | Get disk capacity (64-bit) |

---

## BOT Protocol

### 1. Command Phase

- Host sends **Command Block Wrapper (CBW)** (31 bytes)

  - Signature: "USBC" (0x43425355)
  - Tag: Command identifier
  - Data transfer length
  - Direction flag (IN/OUT)
  - SCSI Command Descriptor Block (CDB)

### 2. Data Phase (optional)

- Bidirectional data transfer via bulk endpoints

  - IN: Device sends data to host
  - OUT: Host sends data to device

### 3. Status Phase

- Device sends **Command Status Wrapper (CSW)** (13 bytes)

  - Signature: "USBS" (0x53425355)
  - Tag: Matches CBW tag
  - Data residue
  - Status: Good (0x00), Failed (0x01), Phase Error (0x02)

---

## Storage Backend Guidelines

When implementing custom storage backends:

1. **Thread Safety**: Implement proper locking
2. **Block Alignment**: Ensure reads/writes are block-aligned
3. **Error Handling**: Return appropriate errors
4. **Sync Support**: Implement Sync() for write-back caching
5. **Removable Media**: Handle IsPresent() correctly

---

## Examples

See [`examples/fifo-hal/msc-disk/`](../../../examples/fifo-hal/msc-disk/) for complete device and host examples.

---

## Testing

```bash
# Run unit tests
go test ./device/class/msc/

# Run integration tests
go test ./examples/fifo-hal/msc-disk/
```

---

## Troubleshooting

### Device not recognized

- Check USB descriptors are correct
- Verify endpoints are bulk type
- Ensure device responds to INQUIRY

### Read/write errors

- Check LBA range validation
- Verify block size matches storage
- Check storage backend error handling

### Performance issues

- Use larger MaxTransferSize
- Implement proper caching in storage backend
- Consider async I/O for file storage

---

## References

- [USB Mass Storage Class Specification 1.0](https://www.usb.org/document-library/mass-storage-class-specification-overview-10)
- [USB Mass Storage Bulk-Only Transport 1.0](https://www.usb.org/document-library/mass-storage-bulk-only-10)
- [SCSI Primary Commands (SPC-4)](https://www.t10.org/drafts.htm)
- [SCSI Block Commands (SBC-3)](https://www.t10.org/drafts.htm)
