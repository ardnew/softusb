// Package main provides integration tests for the HID keyboard FIFO HAL example.
//
// These tests verify that the HID keyboard host and device can communicate
// correctly using the FIFO-based HAL with hot-plugging support.
//
// Run with: go test -v ./examples/fifo-hal/hid-keyboard/
//
// The tests support overriding timeouts via flags:
//
//	go test -v ./examples/fifo-hal/hid-keyboard/ -args \
//	    -enum-timeout=15s \
//	    -transfer-timeout=10s
//
// Note: Use -args to pass flags to the test binary (after the test flags).
package main

import (
	"bytes"
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test flags that are forwarded to host and device commands.
var (
	enumTimeout     = flag.Duration("enum-timeout", 10*time.Second, "timeout for enumeration")
	transferTimeout = flag.Duration("transfer-timeout", 5*time.Second, "timeout for data transfers")
)

// TestHIDKeyboardIntegration tests the HID keyboard host-device communication.
func TestHIDKeyboardIntegration(t *testing.T) {
	// Create temporary bus directory
	busDir, err := os.MkdirTemp("", "softusb-hid-keyboard-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(busDir)

	// Build host and device executables
	hostBin := filepath.Join(busDir, "host")
	deviceBin := filepath.Join(busDir, "device")

	if err := buildExecutable("./host", hostBin); err != nil {
		t.Fatalf("Failed to build host: %v", err)
	}

	if err := buildExecutable("./device", deviceBin); err != nil {
		t.Fatalf("Failed to build device: %v", err)
	}

	// Create context with overall test timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	hostOut := &bytes.Buffer{}
	deviceOut := &bytes.Buffer{}
	hostErr := &bytes.Buffer{}
	deviceErr := &bytes.Buffer{}

	// Start device first (it creates the subdirectory)
	deviceCmd := exec.CommandContext(ctx, deviceBin,
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
		busDir)
	deviceCmd.Stdout = deviceOut
	deviceCmd.Stderr = deviceErr

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := deviceCmd.Start(); err != nil {
			t.Errorf("Failed to start device: %v", err)
			return
		}
		// Device runs until killed or context cancelled
		_ = deviceCmd.Wait()
	}()

	// Give device time to create its subdirectory
	time.Sleep(500 * time.Millisecond)

	// Start host
	hostCmd := exec.CommandContext(ctx, hostBin,
		"-hotplug-limit", "1",
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
		busDir)
	hostCmd.Stdout = hostOut
	hostCmd.Stderr = hostErr

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := hostCmd.Run(); err != nil {
			// Context cancellation is expected
			if ctx.Err() == nil {
				t.Logf("Host exited with error: %v", err)
			}
		}
	}()

	// Wait for host to receive some reports or timeout
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			default:
				// Check if host has received reports
				output := hostOut.String()
				if strings.Contains(output, "Serviced 1 device") ||
					strings.Contains(output, "Received 20 reports") {
					close(done)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	select {
	case <-done:
		// Host completed successfully
	case <-time.After(25 * time.Second):
		t.Logf("Test timeout")
	}

	// Cancel context to clean up
	cancel()

	// Kill device if still running
	if deviceCmd.Process != nil {
		_ = deviceCmd.Process.Kill()
	}

	// Wait for all goroutines
	wg.Wait()

	// Verify output
	hostOutput := hostOut.String()
	deviceOutput := deviceOut.String()

	t.Logf("Host stdout:\n%s", hostOutput)
	t.Logf("Device stdout:\n%s", deviceOutput)

	if hostErr.Len() > 0 {
		t.Logf("Host stderr:\n%s", hostErr.String())
	}
	if deviceErr.Len() > 0 {
		t.Logf("Device stderr:\n%s", deviceErr.String())
	}

	// Check for expected output
	if !strings.Contains(hostOutput, "Device connected") {
		t.Error("Host did not detect device connection")
	}

	if !strings.Contains(hostOutput, "HID device detected") {
		t.Error("Host did not identify device as HID")
	}

	if !strings.Contains(deviceOutput, "Host connected") {
		t.Error("Device did not detect host connection")
	}

	// Check for keyboard report reception (if test ran long enough)
	if strings.Contains(hostOutput, "Report") {
		t.Log("Host received HID reports")
	}

	// Check for device typing
	if strings.Contains(deviceOutput, "Typed:") {
		t.Log("Device sent keyboard reports")
	}
}

// TestHIDKeyboardMultipleDevices tests hot-plugging with multiple HID devices.
func TestHIDKeyboardMultipleDevices(t *testing.T) {
	// Create temporary bus directory
	busDir, err := os.MkdirTemp("", "softusb-hid-keyboard-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(busDir)

	// Build host and device executables
	hostBin := filepath.Join(busDir, "host")
	deviceBin := filepath.Join(busDir, "device")

	if err := buildExecutable("./host", hostBin); err != nil {
		t.Fatalf("Failed to build host: %v", err)
	}

	if err := buildExecutable("./device", deviceBin); err != nil {
		t.Fatalf("Failed to build device: %v", err)
	}

	// Create context with overall test timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	hostOut := &bytes.Buffer{}
	hostErr := &bytes.Buffer{}

	// Start host expecting 2 devices
	hostCmd := exec.CommandContext(ctx, hostBin,
		"-hotplug-limit", "2",
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
		busDir)
	hostCmd.Stdout = hostOut
	hostCmd.Stderr = hostErr

	if err := hostCmd.Start(); err != nil {
		t.Fatalf("Failed to start host: %v", err)
	}

	// Give host time to start
	time.Sleep(500 * time.Millisecond)

	// Start first device
	device1Cmd := exec.CommandContext(ctx, deviceBin,
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
		busDir)
	device1Out := &bytes.Buffer{}
	device1Cmd.Stdout = device1Out
	device1Cmd.Stderr = device1Out

	if err := device1Cmd.Start(); err != nil {
		t.Fatalf("Failed to start device 1: %v", err)
	}

	// Wait for first device to send some reports
	time.Sleep(5 * time.Second)

	// Kill first device
	if device1Cmd.Process != nil {
		_ = device1Cmd.Process.Kill()
	}

	// Start second device
	device2Cmd := exec.CommandContext(ctx, deviceBin,
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
		busDir)
	device2Out := &bytes.Buffer{}
	device2Cmd.Stdout = device2Out
	device2Cmd.Stderr = device2Out

	if err := device2Cmd.Start(); err != nil {
		t.Fatalf("Failed to start device 2: %v", err)
	}

	// Wait for host to complete
	done := make(chan error, 1)
	go func() {
		done <- hostCmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil && ctx.Err() == nil {
			t.Logf("Host exited with error: %v", err)
		}
	case <-time.After(50 * time.Second):
		t.Logf("Test timeout waiting for host")
	}

	// Cleanup
	cancel()
	if device2Cmd.Process != nil {
		_ = device2Cmd.Process.Kill()
	}

	// Verify output
	hostOutput := hostOut.String()
	t.Logf("Host stdout:\n%s", hostOutput)
	t.Logf("Device 1 output:\n%s", device1Out.String())
	t.Logf("Device 2 output:\n%s", device2Out.String())

	if hostErr.Len() > 0 {
		t.Logf("Host stderr:\n%s", hostErr.String())
	}

	// Check for expected output - host should service 2 devices
	if !strings.Contains(hostOutput, "Serviced 2 device") {
		t.Error("Host did not service 2 devices")
	}
}

// buildExecutable builds a Go executable from source.
func buildExecutable(srcDir, output string) error {
	cmd := exec.Command("go", "build", "-o", output, srcDir)
	cmd.Dir = filepath.Dir(srcDir)
	if filepath.IsAbs(srcDir) {
		cmd.Dir = srcDir
	} else {
		// Get the current working directory of this test
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		cmd.Dir = wd
	}
	return cmd.Run()
}
