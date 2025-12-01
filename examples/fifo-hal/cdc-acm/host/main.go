// Package main provides a CDC-ACM USB host example using the FIFO HAL.
//
// This example creates a USB host that communicates with a CDC-ACM device
// (virtual serial port). It uses the FIFO-based HAL to communicate with
// a device process running in parallel.
//
// Usage:
//
//	go run . [options] /path/to/bus-dir
//
// The bus directory is shared with the device process. The host monitors
// this directory for device subdirectories (device-{uuid}/) and connects
// to them via named pipes.
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

	"github.com/ardnew/softusb/device/class/cdc"
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

	// Create FIFO HAL with bus directory
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
	fmt.Printf("Bus directory: %s\n", busDir)

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

		// Check if this is a CDC-ACM device
		if !isCDCDevice(dev) {
			fmt.Println("Not a CDC device, skipping...")
			continue
		}

		fmt.Println("CDC-ACM device detected!")

		// Find bulk endpoints
		bulkIn, bulkOut := findBulkEndpoints(dev)
		if bulkIn == 0 || bulkOut == 0 {
			fmt.Println("Could not find bulk endpoints")
			continue
		}

		fmt.Printf("Bulk IN: 0x%02X, Bulk OUT: 0x%02X\n", bulkIn, bulkOut)

		// Communicate with device using transfer timeout
		if err := communicateWithDevice(ctx, dev, bulkIn, bulkOut, *transferTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "Communication error: %v\n", err)
		}

		devicesServiced++
	}

	fmt.Printf("Serviced %d device(s)\n", devicesServiced)
}

// isCDCDevice checks if the device is a CDC device.
func isCDCDevice(dev *host.Device) bool {
	// Check device class
	if dev.DeviceClass() == cdc.ClassCDC {
		return true
	}

	// Check interface classes
	for _, iface := range dev.Interfaces() {
		if iface.InterfaceClass == cdc.ClassCDC ||
			iface.InterfaceClass == cdc.ClassCDCData {
			return true
		}
	}

	return false
}

// findBulkEndpoints finds the bulk IN and OUT endpoints.
func findBulkEndpoints(dev *host.Device) (in, out uint8) {
	for _, ep := range dev.Endpoints() {
		if ep.IsBulk() {
			if ep.IsIn() {
				in = ep.EndpointAddress
			} else {
				out = ep.EndpointAddress
			}
		}
	}
	return
}

// communicateWithDevice sends test data to the device and reads the response.
func communicateWithDevice(ctx context.Context, dev *host.Device, bulkIn, bulkOut uint8, timeout time.Duration) error {
	// Send test message with timeout
	testMessage := []byte("Hello from USB Host!")
	fmt.Printf("Sending: %q\n", testMessage)

	transferCtx, cancel := context.WithTimeout(ctx, timeout)
	n, err := dev.BulkTransfer(transferCtx, bulkOut, testMessage)
	cancel()
	if err != nil {
		return fmt.Errorf("bulk OUT failed: %w", err)
	}
	fmt.Printf("Sent %d bytes\n", n)

	// Wait a bit for device to process
	time.Sleep(100 * time.Millisecond)

	// Read response with timeout
	var buf [64]byte
	transferCtx, cancel = context.WithTimeout(ctx, timeout)
	n, err = dev.BulkTransfer(transferCtx, bulkIn, buf[:])
	cancel()
	if err != nil {
		return fmt.Errorf("bulk IN failed: %w", err)
	}

	fmt.Printf("Received %d bytes: %q\n", n, buf[:n])

	return nil
}
