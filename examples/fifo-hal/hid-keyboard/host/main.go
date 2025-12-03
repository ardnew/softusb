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
//	-v                         Enable verbose (debug) logging
//	-json                      Use JSON log format
//	-hotplug-limit N           Number of devices to service before exiting (default: 1)
//	-enum-timeout duration     Timeout for enumeration (default: 10s)
//	-transfer-timeout duration Timeout for data transfers (default: 5s)
package main

import (
	"context"
	"flag"
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

// component identifies this executable for structured logging.
const component = pkg.ComponentHost

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

		// Check if this is a HID device
		if !isHIDDevice(dev) {
			pkg.LogInfo(component, "not a HID device, skipping")
			continue
		}

		pkg.LogInfo(component, "HID device detected!")

		// Find interrupt IN endpoint
		intIn := findInterruptInEndpoint(dev)
		if intIn == 0 {
			pkg.LogWarn(component, "could not find interrupt IN endpoint")
			continue
		}

		pkg.LogInfo(component, "found interrupt endpoint", "interruptIn", intIn)

		// Read HID reports from device
		if err := readHIDReports(ctx, dev, intIn, *transferTimeout); err != nil {
			pkg.LogError(component, "read error", "error", err)
		}

		devicesServiced++
	}

	pkg.LogInfo(component, "Serviced devices", "count", devicesServiced)
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
	pkg.LogInfo(component, "reading HID reports")

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

			// Parse boot keyboard report
			if n >= 8 {
				modifiers := buf[0]
				keycodes := buf[2:8]

				// Build list of modifier names
				var modNames []string
				if modifiers&0x01 != 0 {
					modNames = append(modNames, "LCtrl")
				}
				if modifiers&0x02 != 0 {
					modNames = append(modNames, "LShift")
				}
				if modifiers&0x04 != 0 {
					modNames = append(modNames, "LAlt")
				}
				if modifiers&0x08 != 0 {
					modNames = append(modNames, "LWin")
				}
				if modifiers&0x10 != 0 {
					modNames = append(modNames, "RCtrl")
				}
				if modifiers&0x20 != 0 {
					modNames = append(modNames, "RShift")
				}
				if modifiers&0x40 != 0 {
					modNames = append(modNames, "RAlt")
				}
				if modifiers&0x80 != 0 {
					modNames = append(modNames, "RWin")
				}

				// Build list of pressed keys
				var keys []any
				for _, kc := range keycodes {
					if kc != 0 {
						ch := keycodeToChar(kc, modifiers&0x22 != 0) // Check shift
						if ch != 0 {
							keys = append(keys, "keycode", kc, "char", string(ch))
						} else {
							keys = append(keys, "keycode", kc)
						}
					}
				}

				logArgs := []any{
					"reportNum", reportCount,
					"rawData", buf[:n],
					"modifiers", modifiers,
				}
				if len(modNames) > 0 {
					logArgs = append(logArgs, "modifierNames", modNames)
				}
				if len(keys) > 0 {
					logArgs = append(logArgs, keys...)
				}
				pkg.LogInfo(component, "Report", logArgs...)
			} else {
				pkg.LogInfo(component, "Report",
					"reportNum", reportCount,
					"rawData", buf[:n])
			}

			// Stop after 20 reports for demo purposes
			if reportCount >= 20 {
				pkg.LogInfo(component, "Received 20 reports, stopping")
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
