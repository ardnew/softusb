// Package main provides a HID keyboard USB host example using the FIFO HAL.
//
// This example creates a USB host that communicates with a HID keyboard device.
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

	"github.com/ardnew/softusb/device/class/hid"
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

		// Check if this is a HID device
		if !isHIDDevice(dev) {
			fmt.Println("Not a HID device, skipping...")
			continue
		}

		fmt.Println("HID device detected!")

		// Find interrupt IN endpoint
		intIn := findInterruptInEndpoint(dev)
		if intIn == 0 {
			fmt.Println("Could not find interrupt IN endpoint")
			continue
		}

		fmt.Printf("Interrupt IN: 0x%02X\n", intIn)

		// Read HID reports from device
		if err := readHIDReports(ctx, dev, intIn, *transferTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
		}

		devicesServiced++
	}

	fmt.Printf("Serviced %d device(s)\n", devicesServiced)
}

// isHIDDevice checks if the device is a HID device.
func isHIDDevice(dev *host.Device) bool {
	// Check interface classes
	for _, iface := range dev.Interfaces() {
		if iface.InterfaceClass == hid.ClassHID {
			return true
		}
	}
	return false
}

// findInterruptInEndpoint finds the interrupt IN endpoint.
func findInterruptInEndpoint(dev *host.Device) uint8 {
	for _, ep := range dev.Endpoints() {
		if ep.IsInterrupt() && ep.IsIn() {
			return ep.EndpointAddress
		}
	}
	return 0
}

// readHIDReports reads and displays HID reports from the device.
func readHIDReports(ctx context.Context, dev *host.Device, endpoint uint8, timeout time.Duration) error {
	fmt.Println("Reading HID reports (Ctrl+C to stop)...")

	var buf [8]byte // Boot keyboard report is 8 bytes
	reportCount := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Create timeout context for this transfer
		transferCtx, cancel := context.WithTimeout(ctx, timeout)

		// Read interrupt transfer
		n, err := dev.InterruptTransfer(transferCtx, endpoint, buf[:])
		cancel()
		if err != nil {
			// Timeout or NAK is normal for interrupt endpoints
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if n > 0 {
			reportCount++
			fmt.Printf("Report %d: %v\n", reportCount, buf[:n])

			// Parse boot keyboard report
			if n >= 8 {
				modifiers := buf[0]
				keycodes := buf[2:8]

				fmt.Printf("  Modifiers: 0x%02X", modifiers)
				if modifiers&0x01 != 0 {
					fmt.Print(" [LCtrl]")
				}
				if modifiers&0x02 != 0 {
					fmt.Print(" [LShift]")
				}
				if modifiers&0x04 != 0 {
					fmt.Print(" [LAlt]")
				}
				if modifiers&0x08 != 0 {
					fmt.Print(" [LWin]")
				}
				if modifiers&0x10 != 0 {
					fmt.Print(" [RCtrl]")
				}
				if modifiers&0x20 != 0 {
					fmt.Print(" [RShift]")
				}
				if modifiers&0x40 != 0 {
					fmt.Print(" [RAlt]")
				}
				if modifiers&0x80 != 0 {
					fmt.Print(" [RWin]")
				}
				fmt.Println()

				// Print pressed keys
				for _, kc := range keycodes {
					if kc != 0 {
						ch := keycodeToChar(kc, modifiers&0x22 != 0) // Check shift
						if ch != 0 {
							fmt.Printf("  Key: 0x%02X = '%c'\n", kc, ch)
						} else {
							fmt.Printf("  Key: 0x%02X\n", kc)
						}
					}
				}
			}

			// Stop after 20 reports for demo purposes
			if reportCount >= 20 {
				fmt.Println("Received 20 reports, stopping...")
				return nil
			}
		}
	}
}

// keycodeToChar converts a HID keycode to a character.
func keycodeToChar(kc uint8, shift bool) byte {
	if kc >= hid.KeyA && kc <= hid.KeyZ {
		ch := 'a' + (kc - hid.KeyA)
		if shift {
			ch = ch - 32 // Convert to uppercase
		}
		return ch
	}

	if kc >= hid.Key1 && kc <= hid.Key9 {
		return '1' + (kc - hid.Key1)
	}

	switch kc {
	case hid.Key0:
		return '0'
	case hid.KeySpace:
		return ' '
	case hid.KeyEnter:
		return '\n'
	}

	return 0
}
