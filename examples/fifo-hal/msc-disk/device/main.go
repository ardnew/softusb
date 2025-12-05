// Package main provides an MSC USB device example using the FIFO HAL.
//
// This example creates a USB device that acts as a mass storage device
// (virtual USB flash drive). It uses the FIFO-based HAL to communicate
// with a host process running in parallel.
//
// Usage:
//
//	go run . [options] /path/to/bus-dir
//
// The bus directory is shared with the host process. The device creates
// its own subdirectory (device-{uuid}/) for USB communication via named pipes.
//
// Options:
//
//	-size N                    Disk size in bytes (default: 1MB)
//	-v                         Enable verbose (debug) logging
//	-json                      Use JSON log format
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

	"github.com/ardnew/softusb/device"
	"github.com/ardnew/softusb/device/class/msc"
	"github.com/ardnew/softusb/device/hal/fifo"
	"github.com/ardnew/softusb/pkg"
)

const component = pkg.ComponentDevice

func main() {
	diskSize := flag.Uint64("size", 1024*1024, "disk size in bytes")
	verbose := flag.Bool("v", false, "enable verbose (debug) logging")
	jsonLog := flag.Bool("json", false, "use JSON log format")
	enumTimeout := flag.Duration("enum-timeout", 10*time.Second, "timeout for enumeration")
	transferTimeout := flag.Duration("transfer-timeout", 5*time.Second, "timeout for data transfers")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: device [options] <bus-dir>")
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

	// Create in-memory storage
	storage := msc.NewMemoryStorage(*diskSize, 512)

	pkg.LogInfo(component, "creating MSC device",
		"size", *diskSize,
		"blockSize", storage.BlockSize(),
		"blocks", storage.BlockCount())

	// Create MSC driver
	disk := msc.New(storage, "softusb", "Virtual Disk")

	// Create FIFO HAL
	hal := fifo.New(busDir)

	// Build device
	builder := device.NewDeviceBuilder().
		WithVendorProduct(0x1234, 0x5680).
		WithStrings("softusb example", "Mass Storage Device", "12345678").
		AddConfiguration(1)

	// Configure MSC interface (bulkIn=0x81, bulkOut=0x01)
	disk.ConfigureDevice(builder, 0x81, 0x01)

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		pkg.LogInfo(component, "shutting down...")
		cancel()
	}()

	// Build device
	dev, err := builder.Build(ctx)
	if err != nil {
		pkg.LogError(component, "failed to build device", "error", err)
		os.Exit(1)
	}

	// Attach MSC driver to interface
	if err := disk.AttachToInterface(dev, 1, 0); err != nil {
		pkg.LogError(component, "failed to attach driver", "error", err)
		os.Exit(1)
	}

	// Create stack
	stack := device.NewStack(dev, hal)
	disk.SetStack(stack)

	// Configure timeouts
	_ = enumTimeout
	_ = transferTimeout

	pkg.LogInfo(component, "starting device stack",
		"busDir", busDir)

	// Start stack
	if err := stack.Start(ctx); err != nil {
		pkg.LogError(component, "failed to start stack", "error", err)
		os.Exit(1)
	}
	defer stack.Stop()

	// Wait for connection
	pkg.LogInfo(component, "waiting for host connection...")
	if err := stack.WaitConnect(ctx); err != nil {
		pkg.LogError(component, "connection wait failed", "error", err)
		os.Exit(1)
	}

	pkg.LogInfo(component, "host connected, running MSC protocol")

	// Run MSC processing loop
	if err := disk.Run(ctx); err != nil && err != context.Canceled {
		pkg.LogError(component, "MSC processing error", "error", err)
		os.Exit(1)
	}

	pkg.LogInfo(component, "device stopped")
}
