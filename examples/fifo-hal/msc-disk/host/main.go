// Package main provides an MSC USB host example using the FIFO HAL.
//
// This example creates a USB host that communicates with an MSC device.
// It uses the FIFO-based HAL to communicate with a device process running
// in parallel.
//
// Usage:
//
//	go run . [options] /path/to/bus-dir
//
// The bus directory is shared with the device process. The host polls for
// device subdirectories (device-{uuid}/) for USB communication via named pipes.
//
// Options:
//
//	-hotplug-limit N           Number of devices to service before exiting (default: 1)
//	-enum-timeout duration     Timeout for enumeration (default: 10s)
//	-transfer-timeout duration Timeout for data transfers (default: 5s)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardnew/softusb/device/class/msc"
	"github.com/ardnew/softusb/host"
	"github.com/ardnew/softusb/host/hal/fifo"
	"github.com/ardnew/softusb/pkg"
)

func main() {
	hotplugLimit := flag.Int("hotplug-limit", 1, "number of devices to service")
	enumTimeout := flag.Duration("enum-timeout", 10*time.Second, "timeout for enumeration")
	transferTimeout := flag.Duration("transfer-timeout", 5*time.Second, "timeout for data transfers")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: host [options] <bus-dir>")
		os.Exit(1)
	}

	busDir := flag.Arg(0)

	// Set up logging
	pkg.SetLogLevel(slog.LevelDebug)

	// Create FIFO HAL
	hal := fifo.NewHostHAL(busDir)

	// Create host
	usbHost := host.New(hal)

	// Set up context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Start the host
	fmt.Println("Starting USB host...")
	fmt.Printf("FIFO directory: %s\n", busDir)

	if err := usbHost.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start host: %v\n", err)
		os.Exit(1)
	}
	defer usbHost.Stop()

	devicesServiced := 0

	for devicesServiced < *hotplugLimit {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for device with enumeration timeout
		fmt.Println("Waiting for device connection...")
		enumCtx, enumCancel := context.WithTimeout(ctx, *enumTimeout)
		dev, err := usbHost.WaitDevice(enumCtx)
		enumCancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error waiting for device: %v\n", err)
			continue
		}

		fmt.Printf("Device connected:\n")
		fmt.Printf("  Vendor ID:  0x%04X\n", dev.VendorID())
		fmt.Printf("  Product ID: 0x%04X\n", dev.ProductID())
		fmt.Printf("  Manufacturer: %s\n", dev.Manufacturer())
		fmt.Printf("  Product: %s\n", dev.Product())
		fmt.Printf("  Serial: %s\n", dev.SerialNumber())

		// Check if this is an MSC device
		if !isMSCDevice(dev) {
			fmt.Println("Not an MSC device, skipping...")
			continue
		}

		fmt.Println("MSC device detected!")

		// Test basic operations
		if err := testMSCDevice(ctx, dev, *transferTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "MSC test error: %v\n", err)
		}

		devicesServiced++
	}

	fmt.Printf("Serviced %d device(s)\n", devicesServiced)
}

// isMSCDevice checks if the device is an MSC device.
func isMSCDevice(dev *host.Device) bool {
	for _, iface := range dev.Interfaces() {
		if iface.InterfaceClass == msc.ClassMSC {
			return true
		}
	}
	return false
}

// testMSCDevice performs basic MSC operations.
func testMSCDevice(ctx context.Context, dev *host.Device, timeout time.Duration) error {
	fmt.Println("\nTesting MSC device...")

	// For now, just print that we detected it
	// Full SCSI command implementation would be in host/class/msc package
	fmt.Println("✓ Device enumerated successfully")
	fmt.Println("✓ MSC interface detected")

	fmt.Println("\nNote: Full SCSI command testing requires host MSC class driver")
	fmt.Println("This example demonstrates device enumeration and class detection")

	return nil
}
