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
//	-v                         Enable verbose (debug) logging
//	-json                      Use JSON log format
//	-hotplug-limit N           Number of devices to service before exiting (default: 1)
//	-enum-timeout duration     Timeout for enumeration (default: 10s)
//	-transfer-timeout duration Timeout for data transfers (default: 5s)
package main

import (
	"context"
	"errors"
	"flag"
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

// component identifies this executable for structured logging.
const component = pkg.ComponentHost

// Error types for this executable.
var (
	errBulkOutFailed = errors.New("bulk OUT failed")
	errBulkInFailed  = errors.New("bulk IN failed")
)

func main() {
	verbose := flag.Bool("v", false, "enable verbose (debug) logging")
	jsonLog := flag.Bool("json", false, "use JSON log format")
	hotplugLimit := flag.Int("hotplug-limit", 1, "number of devices to service")
	enumTimeout := flag.Duration("enum-timeout", 10*time.Second, "timeout for enumeration")
	transferTimeout := flag.Duration("transfer-timeout", 5*time.Second, "timeout for data transfers")
	flag.Parse()

	if flag.NArg() < 1 {
		pkg.LogError(component, "missing bus directory argument",
			"usage", "host [options] <bus-dir>")
		os.Exit(1)
	}

	busDir := flag.Arg(0)

	// Set up logging
	if *verbose {
		pkg.SetLogLevel(slog.LevelDebug)
	}
	if *jsonLog {
		pkg.SetLogFormat(pkg.LogFormatJSON)
	}

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
		pkg.LogInfo(component, "shutting down")
		cancel()
	}()

	// Start the host
	pkg.LogInfo(component, "starting USB host", "busDir", busDir)

	if err := usbHost.Start(ctx); err != nil {
		pkg.LogError(component, "failed to start host", "error", err)
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
		pkg.LogInfo(component, "waiting for device connection")
		enumCtx, enumCancel := context.WithTimeout(ctx, *enumTimeout)
		dev, err := usbHost.WaitDevice(enumCtx)
		enumCancel()
		if err != nil {
			pkg.LogError(component, "error waiting for device", "error", err)
			continue
		}

		pkg.LogInfo(component, "Device connected",
			"vendorID", dev.VendorID(),
			"productID", dev.ProductID(),
			"manufacturer", dev.Manufacturer(),
			"product", dev.Product(),
			"serial", dev.SerialNumber())

		// Check if this is a CDC-ACM device
		if !isCDCDevice(dev) {
			pkg.LogInfo(component, "not a CDC device, skipping")
			continue
		}

		pkg.LogInfo(component, "CDC-ACM device detected!")

		// Find bulk endpoints
		bulkIn, bulkOut := findBulkEndpoints(dev)
		if bulkIn == 0 || bulkOut == 0 {
			pkg.LogWarn(component, "could not find bulk endpoints")
			continue
		}

		pkg.LogInfo(component, "found bulk endpoints",
			"bulkIn", bulkIn,
			"bulkOut", bulkOut)

		// Communicate with device using transfer timeout
		if err := communicateWithDevice(ctx, dev, bulkIn, bulkOut, *transferTimeout); err != nil {
			pkg.LogError(component, "communication error", "error", err)
		}

		devicesServiced++
	}

	pkg.LogInfo(component, "Serviced devices", "count", devicesServiced)
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
	pkg.LogInfo(component, "sending data", "data", string(testMessage))

	transferCtx, cancel := context.WithTimeout(ctx, timeout)
	n, err := dev.BulkTransfer(transferCtx, bulkOut, testMessage)
	cancel()
	if err != nil {
		return errors.Join(errBulkOutFailed, err)
	}
	pkg.LogInfo(component, "sent data", "bytes", n)

	// Wait a bit for device to process
	time.Sleep(100 * time.Millisecond)

	// Read response with timeout
	var buf [64]byte
	transferCtx, cancel = context.WithTimeout(ctx, timeout)
	n, err = dev.BulkTransfer(transferCtx, bulkIn, buf[:])
	cancel()
	if err != nil {
		return errors.Join(errBulkInFailed, err)
	}

	pkg.LogInfo(component, "Received data",
		"bytes", n,
		"data", string(buf[:n]))

	return nil
}
