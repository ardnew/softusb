// Package main provides a CDC-ACM USB device example using the FIFO HAL.
//
// This example creates a USB device that acts as a virtual serial port.
// It uses the FIFO-based HAL to communicate with a host process running
// in parallel.
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
	"github.com/ardnew/softusb/device/class/cdc"
	"github.com/ardnew/softusb/device/hal/fifo"
	"github.com/ardnew/softusb/pkg"
)

func main() {
	enumTimeout := flag.Duration("enum-timeout", 10*time.Second, "timeout for enumeration")
	transferTimeout := flag.Duration("transfer-timeout", 5*time.Second, "timeout for data transfers")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: device [options] <bus-dir>")
		os.Exit(1)
	}

	busDir := flag.Arg(0)

	// Set up logging
	pkg.SetLogLevel(slog.LevelDebug)

	// Create FIFO HAL with bus directory
	hal := fifo.New(busDir)

	// Create device using builder
	builder := device.NewDeviceBuilder().
		WithVendorProduct(0x1234, 0x5678).
		WithStrings("SoftUSB Example", "CDC-ACM Serial Port", "12345678").
		AddConfiguration(1)

	// Create and register CDC-ACM class driver
	acm := cdc.NewACM()

	// Configure CDC-ACM descriptors (notify=0x81, dataIn=0x82, dataOut=0x02)
	acm.ConfigureDevice(builder, 0x81, 0x82, 0x02)

	// Build device
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dev, err := builder.Build(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build device: %v\n", err)
		os.Exit(1)
	}

	// Attach ACM driver to the CDC interfaces in configuration 1
	// (interface 0 = control, interface 1 = data)
	if err := acm.AttachToInterfaces(dev, 1, 0, 1); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to attach ACM driver: %v\n", err)
		os.Exit(1)
	}

	// Create device stack
	stack := device.NewStack(dev, hal)

	// Set the stack reference in ACM driver
	acm.SetStack(stack)

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Create a buffer for data
	var buf [64]byte

	// Start the device stack
	fmt.Println("Starting CDC-ACM device...")
	fmt.Printf("Bus directory: %s\n", busDir)
	fmt.Printf("Device directory: %s\n", hal.DeviceDir())

	if err := stack.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start device: %v\n", err)
		os.Exit(1)
	}
	defer stack.Stop()

	// Wait for connection with enumeration timeout
	fmt.Println("Waiting for host connection...")
	enumCtx, enumCancel := context.WithTimeout(ctx, *enumTimeout)
	defer enumCancel()

	if err := stack.WaitConnect(enumCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Host connected!")

	// Main loop - echo any data received
	fmt.Println("Echoing data (Ctrl+C to exit)...")
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read from host with transfer timeout
		transferCtx, transferCancel := context.WithTimeout(ctx, *transferTimeout)
		n, err := acm.Read(transferCtx, buf[:])
		transferCancel()
		if err != nil {
			// Short timeout to check context
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if n > 0 {
			fmt.Printf("Received %d bytes: %q\n", n, buf[:n])

			// Echo back with transfer timeout
			transferCtx, transferCancel = context.WithTimeout(ctx, *transferTimeout)
			_, err = acm.Write(transferCtx, buf[:n])
			transferCancel()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
			}
		}
	}
}
