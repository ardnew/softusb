// Package main provides a HID keyboard USB device example using the FIFO HAL.
//
// This example creates a USB device that acts as a HID keyboard.
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
//	-v                         Enable verbose (debug) logging
//	-json                      Use JSON log format
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

	"github.com/ardnew/softusb/device"
	"github.com/ardnew/softusb/device/class/hid"
	"github.com/ardnew/softusb/device/hal/fifo"
	"github.com/ardnew/softusb/pkg"
)

// component identifies this executable for structured logging.
const component = pkg.ComponentDevice

// Boot keyboard report descriptor (standard 8-byte report)
var keyboardReportDescriptor = []byte{
	0x05, 0x01, // Usage Page (Generic Desktop)
	0x09, 0x06, // Usage (Keyboard)
	0xA1, 0x01, // Collection (Application)
	0x05, 0x07, //   Usage Page (Key Codes)
	0x19, 0xE0, //   Usage Minimum (224)
	0x29, 0xE7, //   Usage Maximum (231)
	0x15, 0x00, //   Logical Minimum (0)
	0x25, 0x01, //   Logical Maximum (1)
	0x75, 0x01, //   Report Size (1)
	0x95, 0x08, //   Report Count (8)
	0x81, 0x02, //   Input (Data, Variable, Absolute) - Modifier byte
	0x95, 0x01, //   Report Count (1)
	0x75, 0x08, //   Report Size (8)
	0x81, 0x01, //   Input (Constant) - Reserved byte
	0x95, 0x06, //   Report Count (6)
	0x75, 0x08, //   Report Size (8)
	0x15, 0x00, //   Logical Minimum (0)
	0x25, 0x65, //   Logical Maximum (101)
	0x05, 0x07, //   Usage Page (Key Codes)
	0x19, 0x00, //   Usage Minimum (0)
	0x29, 0x65, //   Usage Maximum (101)
	0x81, 0x00, //   Input (Data, Array) - Key array (6 keys)
	0xC0, // End Collection
}

func main() {
	verbose := flag.Bool("v", false, "enable verbose (debug) logging")
	jsonLog := flag.Bool("json", false, "use JSON log format")
	enumTimeout := flag.Duration("enum-timeout", 10*time.Second, "timeout for enumeration")
	transferTimeout := flag.Duration("transfer-timeout", 5*time.Second, "timeout for data transfers")
	flag.Parse()

	if flag.NArg() < 1 {
		pkg.LogError(component, "missing bus directory argument",
			"usage", "device [options] <bus-dir>")
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

	hal := fifo.New(busDir)

	builder := device.NewDeviceBuilder().
		WithVendorProduct(0x1234, 0x5679).
		WithStrings("softusb example", "HID Keyboard", "87654321").
		AddConfiguration(1)

	keyboard := hid.New(keyboardReportDescriptor)
	keyboard.ConfigureDevice(builder, 0x81, hid.SubclassBoot, hid.ProtocolKeyboard)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dev, err := builder.Build(ctx)
	if err != nil {
		pkg.LogError(component, "failed to build device", "error", err)
		os.Exit(1)
	}

	// Attach HID driver to the interface in configuration 1 (interface 0)
	if err := keyboard.AttachToInterface(dev, 1, 0); err != nil {
		pkg.LogError(component, "failed to attach HID driver", "error", err)
		os.Exit(1)
	}

	stack := device.NewStack(dev, hal)
	keyboard.SetStack(stack)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		pkg.LogInfo(component, "shutting down")
		cancel()
	}()

	pkg.LogInfo(component, "starting HID keyboard device", "busDir", busDir)

	if err := stack.Start(ctx); err != nil {
		pkg.LogError(component, "failed to start device", "error", err)
		os.Exit(1)
	}
	defer stack.Stop()

	pkg.LogInfo(component, "waiting for host connection")
	connectCtx, connectCancel := context.WithTimeout(ctx, *enumTimeout)
	if err := stack.WaitConnect(connectCtx); err != nil {
		connectCancel()
		pkg.LogError(component, "connection failed", "error", err)
		os.Exit(1)
	}
	connectCancel()
	pkg.LogInfo(component, "Host connected!")

	pkg.LogInfo(component, "typing 'Hello' every 2 seconds")
	typeString := []byte("Hello\n")
	idx := 0

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Boot keyboard report format: [modifiers, reserved, key1, key2, key3, key4, key5, key6]
	var report [8]byte

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if idx < len(typeString) {
				ch := typeString[idx]
				keycode := charToKeycode(ch)
				shift := needsShift(ch)

				// Key press report
				if shift {
					report[0] = 0x02 // Left Shift
				} else {
					report[0] = 0x00
				}
				report[1] = 0x00    // Reserved
				report[2] = keycode // First key
				report[3] = 0x00    // Remaining keys
				report[4] = 0x00
				report[5] = 0x00
				report[6] = 0x00
				report[7] = 0x00

				sendCtx, sendCancel := context.WithTimeout(ctx, *transferTimeout)
				if err := keyboard.SendReport(sendCtx, report[:]); err != nil {
					sendCancel()
					pkg.LogError(component, "SendReport error", "error", err)
				}
				sendCancel()

				time.Sleep(50 * time.Millisecond)

				// Key release report (all zeros)
				for i := range report {
					report[i] = 0
				}
				releaseCtx, releaseCancel := context.WithTimeout(ctx, *transferTimeout)
				if err := keyboard.SendReport(releaseCtx, report[:]); err != nil {
					releaseCancel()
					pkg.LogError(component, "SendReport error", "error", err)
				}
				releaseCancel()

				pkg.LogInfo(component, "Typed:", "char", string(ch))
				idx++
			} else {
				idx = 0
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func charToKeycode(ch byte) uint8 {
	switch {
	case ch >= 'a' && ch <= 'z':
		return hid.KeyA + (ch - 'a')
	case ch >= 'A' && ch <= 'Z':
		return hid.KeyA + (ch - 'A')
	case ch >= '1' && ch <= '9':
		return hid.Key1 + (ch - '1')
	case ch == '0':
		return hid.Key0
	case ch == '\n' || ch == '\r':
		return hid.KeyEnter
	case ch == ' ':
		return hid.KeySpace
	default:
		return 0
	}
}

func needsShift(ch byte) bool {
	return ch >= 'A' && ch <= 'Z'
}
