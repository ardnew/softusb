//go:build tinygo && atsamd51

// Package main provides an ATSAMD51-based USB MSC device HAL for TinyGo.
//
// This example implements a USB Mass Storage Class device that uses the
// onboard 8MB QSPI flash as storage. It targets the Adafruit Grand Central M4
// board and directly accesses the USB peripheral without using TinyGo's
// built-in USB driver.
//
// # Architecture
//
// The implementation consists of four main components:
//
//   - hal.go: Implements device/hal.DeviceHAL interface
//   - usb.go: USB peripheral register definitions and low-level operations
//   - qspi.go: QSPI flash storage backend implementing msc.Storage
//   - main.go: Entry point that wires everything together
//
// # Building
//
// Use TinyGo to build and flash:
//
//	tinygo flash -target=grandcentral-m4 ./examples/tinygo-device/msc-disk/atsamd51
//
// See README.md for detailed instructions.
package main
