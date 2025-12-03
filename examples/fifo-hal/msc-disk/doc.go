// Package msc_disk provides a testable MSC (Mass Storage Class) example
// using the FIFO-based HAL implementation.
//
// This example demonstrates a complete USB mass storage device that can be
// tested on any POSIX-compliant system (Linux, macOS, WSL) without requiring
// actual USB hardware.
//
// # Structure
//
// The example consists of three components:
//
//   - device/ - MSC device implementation using FIFO HAL
//   - host/ - MSC host implementation using FIFO HAL
//   - Integration test - Automated testing of the complete system
//
// # Usage
//
// Run device in one terminal:
//
//	cd device
//	go run . -v /tmp/usb-msc-test
//
// Run host in another terminal:
//
//	cd host
//	go run . -v /tmp/usb-msc-test
//
// Or run the integration test:
//
//	go test -v
//
// # What This Example Demonstrates
//
//   - Creation of MSC device with in-memory storage
//   - BOT protocol (CBW/CSW) communication
//   - SCSI command handling (INQUIRY, READ CAPACITY, etc.)
//   - Block read/write operations
//   - Device enumeration from host perspective
//   - Error handling and sense data
//
// # Storage Backend
//
// The example uses an in-memory storage backend (MemoryStorage) which
// creates a RAM disk of configurable size. The default is 1MB with
// 512-byte blocks.
//
// Other storage backends can be used:
//
//   - FileStorage - File-backed disk image
//   - Custom implementations of the Storage interface
//
// # Limitations
//
// This FIFO-based example is intended for testing and development.
// For production use on real hardware:
//
//   - Implement platform-specific HAL (e.g., TinyGo for microcontrollers)
//   - Use appropriate storage backend (flash, SD card, etc.)
//   - Consider power management and USB suspend/resume
package msc_disk
