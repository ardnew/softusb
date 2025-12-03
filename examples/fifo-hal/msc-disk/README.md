# MSC Disk Example

> **USB Mass Storage Class example using FIFO HAL**

This example demonstrates a complete USB Mass Storage Class (MSC) device implementation using the FIFO-based HAL for testing on any POSIX-compliant system.

---

## Overview

This example creates a virtual USB flash drive that can be tested without actual USB hardware. It consists of:

- **Device**: MSC device with in-memory storage (RAM disk)
- **Host**: MSC host that enumerates and detects the device
- **Integration test**: Automated testing of device-host communication

---

## Quick Start

### Prerequisites

- Go 1.19 or later
- POSIX-compliant system (Linux, macOS, WSL)

### Run Integration Test

```bash
go test -v
```

### Manual Testing

Terminal 1 (device):
```bash
cd device
go run . -v /tmp/usb-msc-test
```

Terminal 2 (host):
```bash
cd host
go run . -v /tmp/usb-msc-test
```

---

## Device Options

```bash
go run ./device [options] <bus-dir>

Options:
  -size N                    Disk size in bytes (default: 1048576 = 1MB)
  -v                         Enable verbose (debug) logging
  -json                      Use JSON log format
  -enum-timeout duration     Timeout for enumeration (default: 10s)
  -transfer-timeout duration Timeout for data transfers (default: 5s)
```

### Examples

Create a 10MB disk:
```bash
go run ./device -size 10485760 /tmp/usb-msc-test
```

Create a 512KB disk with verbose logging:
```bash
go run ./device -size 524288 -v /tmp/usb-msc-test
```

---

## Host Options

```bash
go run ./host [options] <bus-dir>

Options:
  -hotplug-limit N           Number of devices to service (default: 1)
  -enum-timeout duration     Timeout for enumeration (default: 10s)
  -transfer-timeout duration Timeout for data transfers (default: 5s)
```

---

## What Gets Tested

The example demonstrates:

1. **Device Initialization**
   - Creating in-memory storage backend
   - Configuring MSC interface descriptors
   - Starting device stack

2. **BOT Protocol**
   - Command Block Wrapper (CBW) reception
   - Data phase transfers
   - Command Status Wrapper (CSW) response

3. **SCSI Commands**
   - INQUIRY - Device identification
   - READ CAPACITY - Disk size query
   - TEST UNIT READY - Readiness check
   - And more...

4. **Host Enumeration**
   - Device detection
   - Descriptor reading
   - Class identification

---

## Architecture

```text
┌─────────────────┐         FIFO Bus          ┌─────────────────┐
│  Device Process │  ←────────────────────→  │   Host Process  │
│                 │  Named Pipes (FIFOs)      │                 │
│  ┌───────────┐  │                           │  ┌───────────┐  │
│  │ MSC Driver│  │                           │  │Host Stack │  │
│  ├───────────┤  │                           │  └───────────┘  │
│  │  Storage  │  │                           │                 │
│  │(MemStorage)│  │                           │  Enumerates &  │
│  ├───────────┤  │                           │  Detects MSC    │
│  │FIFO Device│  │                           │  Device         │
│  │    HAL    │  │                           │                 │
│  └───────────┘  │                           │                 │
└─────────────────┘                           └─────────────────┘
```

---

## Storage Backend

The device example uses **MemoryStorage**, an in-memory RAM disk:

- Configurable size (default: 1MB)
- Fixed block size (512 bytes)
- Non-persistent (data lost on exit)
- Fast performance

For persistent storage, you can modify the device code to use **FileStorage**:

```go
// Instead of:
storage := msc.NewMemoryStorage(diskSize, 512)

// Use:
storage, err := msc.NewFileStorage("disk.img", 512, false)
if err != nil {
    log.Fatal(err)
}
defer storage.Close()
```

---

## Troubleshooting

### "Failed to start device"

- Check that the bus directory is writable
- Ensure no other process is using the same directory
- Try a different directory path

### "Error waiting for device"

- Ensure device process is running
- Check device logs for errors
- Increase `-enum-timeout` if enumeration is slow

### Test timeout

- Increase test timeout in `example_test.go`
- Check system load (may slow down FIFO communication)
- Enable verbose logging to see where it's stuck

---

## Extending the Example

### Add SCSI Command Testing

Implement actual SCSI command testing in the host to:
- Send INQUIRY and verify response
- Read disk capacity
- Perform block read/write operations
- Test error conditions

### Custom Storage Backend

Create a custom storage backend that:
- Compresses data on-the-fly
- Implements wear leveling
- Adds encryption
- Logs all accesses

### File System Integration

Mount the device as a real file system:
- Format storage with FAT32
- Create files and directories  
- Test with actual OS file operations

---

## Performance

Approximate performance on a modern system:

- **Enumeration**: < 1 second
- **INQUIRY command**: < 10ms
- **Block read (4KB)**: < 5ms
- **Block write (4KB)**: < 5ms

FIFO-based HAL is for testing only. Real hardware HAL will have different performance characteristics.

---

## References

- [USB Mass Storage Class Specification](https://www.usb.org/document-library/mass-storage-class-specification-overview-10)
- [SCSI Primary Commands](https://www.t10.org/drafts.htm)
- [Parent MSC Class Driver](../../../device/class/msc/)
