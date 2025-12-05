//go:build tinygo && atsamd51

// Package main provides an MSC USB device example using the ATSAMD51 HAL.
//
// This example creates a USB device that acts as a mass storage device
// backed by the onboard 8MB QSPI flash. It targets the Adafruit Grand
// Central M4 board.
//
// Build and flash:
//
//	tinygo flash -target=grandcentral-m4 ./examples/tinygo-device/msc-disk/atsamd51
//
// After flashing, the board will appear as a USB mass storage device.
// The QSPI flash (8MB) will be accessible as a disk drive.
//
// Note: The first time you use this, the flash may contain random data.
// You may need to format the drive from your host OS.
package main

import (
	"context"

	"github.com/ardnew/softusb/device"
	"github.com/ardnew/softusb/device/class/msc"
)

func main() {
	// Create QSPI storage backend (8MB flash)
	storage := NewQSPIStorage()

	// Create MSC driver
	disk := msc.New(storage, "softusb", "QSPI Flash")

	// Create HAL
	hal := New()

	// Build USB device descriptor
	builder := device.NewDeviceBuilder().
		WithVendorProduct(0x239A, 0x8031). // Adafruit VID, Grand Central M4 PID
		WithStrings("Adafruit", "Grand Central M4 MSC", "GCMSC001").
		AddConfiguration(1)

	// Configure MSC interface
	// Bulk IN = 0x81 (EP1 IN), Bulk OUT = 0x01 (EP1 OUT)
	disk.ConfigureDevice(builder, 0x81, 0x01)

	// Build device
	ctx := context.Background()
	dev, err := builder.Build(ctx)
	if err != nil {
		// On embedded, we can't really handle this gracefully
		// In production, you might blink an LED or similar
		for {
			delayMicroseconds(1000000)
		}
	}

	// Attach MSC driver to interface
	if err := disk.AttachToInterface(dev, 1, 0); err != nil {
		for {
			delayMicroseconds(1000000)
		}
	}

	// Create device stack
	stack := device.NewStack(dev, hal)
	disk.SetStack(stack)

	// Start the device stack
	if err := stack.Start(ctx); err != nil {
		for {
			delayMicroseconds(1000000)
		}
	}

	// Wait for host connection
	if err := stack.WaitConnect(ctx); err != nil {
		for {
			delayMicroseconds(1000000)
		}
	}

	// Run MSC processing loop
	// This blocks forever, processing USB mass storage requests
	disk.Run(ctx)
}
