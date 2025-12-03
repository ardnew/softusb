// Package main provides integration tests for the CDC-ACM FIFO HAL example.
//
// These tests verify that the CDC-ACM host and device can communicate
// correctly using the FIFO-based HAL with hot-plugging support.
//
// Run with: go test -v ./examples/fifo-hal/cdc-acm/
//
// The tests support overriding timeouts via flags:
//
//	go test -v ./examples/fifo-hal/cdc-acm/ -args \
//	    -enum-timeout=15s \
//	    -transfer-timeout=10s \
//	    -json \
//	    -m
//
// Note: Use -args to pass flags to the test binary (after the test flags).
package main

import (
	"bytes"
	"context"
	"flag"
	"io"
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
	jsonLog         = flag.Bool("json", false, "use JSON log format")
	mergeOutput     = flag.Bool("merge-output", false, "merge host and device output into a single stream")
)

// syncWriter wraps an io.Writer with mutex protection for concurrent writes.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// Write implements io.Writer with thread-safe writes.
func (sw *syncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// newSyncWriter creates a new thread-safe writer.
func newSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{w: w}
}

// TestCDCACMIntegration tests the CDC-ACM host-device communication.
func TestCDCACMIntegration(t *testing.T) {
	// Create temporary bus directory
	busDir, err := os.MkdirTemp("", "softusb-cdc-acm-test-*")
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

	// Set up output writers based on merge flag
	var hostOut io.Writer
	var hostBuf *bytes.Buffer
	var merged *syncWriter

	if *mergeOutput {
		merged = newSyncWriter(os.Stdout)
		hostOut = merged
		hostBuf = nil
	} else {
		hostBuf = &bytes.Buffer{}
		hostOut = hostBuf
	}

	// Start host expecting 1 device
	// Always pass -v to ensure info-level logs are captured for test assertions
	hostArgs := []string{
		"-v",
		"-hotplug-limit", "1",
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
	}
	if *jsonLog {
		hostArgs = append(hostArgs, "-json")
	}
	hostArgs = append(hostArgs, busDir)
	hostCmd := exec.CommandContext(ctx, hostBin, hostArgs...)
	hostCmd.Stdout = hostOut
	hostCmd.Stderr = hostOut // Merge stderr into stdout

	if err := hostCmd.Start(); err != nil {
		t.Fatalf("Failed to start host: %v", err)
	}

	// Give host time to start
	time.Sleep(500 * time.Millisecond)

	// Start device
	// Always pass -v to ensure info-level logs are captured for test assertions
	deviceArgs := []string{
		"-v",
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
	}
	if *jsonLog {
		deviceArgs = append(deviceArgs, "-json")
	}
	deviceArgs = append(deviceArgs, busDir)
	deviceCmd := exec.CommandContext(ctx, deviceBin, deviceArgs...)

	var deviceOut io.Writer
	var deviceBuf *bytes.Buffer
	if *mergeOutput {
		deviceOut = merged
		deviceBuf = nil
	} else {
		deviceBuf = &bytes.Buffer{}
		deviceOut = deviceBuf
	}
	deviceCmd.Stdout = deviceOut
	deviceCmd.Stderr = deviceOut // Merge stderr into stdout

	if err := deviceCmd.Start(); err != nil {
		t.Fatalf("Failed to start device: %v", err)
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
	case <-time.After(25 * time.Second):
		t.Logf("Test timeout waiting for host")
	}

	// Cleanup
	cancel()
	if deviceCmd.Process != nil {
		_ = deviceCmd.Process.Kill()
	}

	// Verify output (skip validation when merging, as output goes to stdout)
	if *mergeOutput {
		t.Log("Output merged to stdout")
		return
	}

	hostOutput := hostBuf.String()
	t.Logf("Host output:\n%s", hostOutput)
	if deviceBuf != nil {
		t.Logf("Device output:\n%s", deviceBuf.String())
	}

	// Check for expected output
	if !strings.Contains(hostOutput, "Device connected") {
		t.Error("Host did not detect device connection")
	}

	if !strings.Contains(hostOutput, "CDC-ACM device detected") {
		t.Error("Host did not identify device as CDC-ACM")
	}

	if !strings.Contains(hostOutput, "Hello from USB Host") {
		t.Error("Host did not send test message")
	}

	if deviceBuf != nil {
		deviceOutput := deviceBuf.String()
		if !strings.Contains(deviceOutput, "Host connected") {
			t.Error("Device did not detect host connection")
		}

		if !strings.Contains(deviceOutput, "received data") {
			t.Error("Device did not receive data from host")
		}
	}
}

// TestCDCACMMultipleDevices tests hot-plugging with multiple devices.
func TestCDCACMMultipleDevices(t *testing.T) {
	// Create temporary bus directory
	busDir, err := os.MkdirTemp("", "softusb-cdc-acm-multi-*")
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

	// Set up output writers based on merge flag
	var hostOut io.Writer
	var hostBuf *bytes.Buffer
	var merged *syncWriter

	if *mergeOutput {
		merged = newSyncWriter(os.Stdout)
		hostOut = merged
		hostBuf = nil
	} else {
		hostBuf = &bytes.Buffer{}
		hostOut = hostBuf
	}

	// Start host expecting 2 devices
	// Always pass -v to ensure info-level logs are captured for test assertions
	hostArgs := []string{
		"-v",
		"-hotplug-limit", "2",
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
	}
	if *jsonLog {
		hostArgs = append(hostArgs, "-json")
	}
	hostArgs = append(hostArgs, busDir)
	hostCmd := exec.CommandContext(ctx, hostBin, hostArgs...)
	hostCmd.Stdout = hostOut
	hostCmd.Stderr = hostOut // Merge stderr into stdout

	if err := hostCmd.Start(); err != nil {
		t.Fatalf("Failed to start host: %v", err)
	}

	// Give host time to start
	time.Sleep(500 * time.Millisecond)

	// Start first device
	// Always pass -v to ensure info-level logs are captured for test assertions
	device1Args := []string{
		"-v",
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
	}
	if *jsonLog {
		device1Args = append(device1Args, "-json")
	}
	device1Args = append(device1Args, busDir)
	device1Cmd := exec.CommandContext(ctx, deviceBin, device1Args...)

	var device1Out io.Writer
	var device1Buf *bytes.Buffer
	if *mergeOutput {
		device1Out = merged
		device1Buf = nil
	} else {
		device1Buf = &bytes.Buffer{}
		device1Out = device1Buf
	}
	device1Cmd.Stdout = device1Out
	device1Cmd.Stderr = device1Out // Merge stderr into stdout

	if err := device1Cmd.Start(); err != nil {
		t.Fatalf("Failed to start device 1: %v", err)
	}

	// Wait for first device to be serviced
	time.Sleep(1 * time.Second)

	// Kill first device
	if device1Cmd.Process != nil {
		_ = device1Cmd.Process.Kill()
	}

	// Start second device
	// Always pass -v to ensure info-level logs are captured for test assertions
	device2Args := []string{
		"-v",
		"-enum-timeout", enumTimeout.String(),
		"-transfer-timeout", transferTimeout.String(),
	}
	if *jsonLog {
		device2Args = append(device2Args, "-json")
	}
	device2Args = append(device2Args, busDir)
	device2Cmd := exec.CommandContext(ctx, deviceBin, device2Args...)

	var device2Out io.Writer
	var device2Buf *bytes.Buffer
	if *mergeOutput {
		device2Out = merged
		device2Buf = nil
	} else {
		device2Buf = &bytes.Buffer{}
		device2Out = device2Buf
	}
	device2Cmd.Stdout = device2Out
	device2Cmd.Stderr = device2Out // Merge stderr into stdout

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

	// Verify output (skip validation when merging, as output goes to stdout)
	if *mergeOutput {
		t.Log("Output merged to stdout")
		return
	}

	hostOutput := hostBuf.String()
	t.Logf("Host output:\n%s", hostOutput)
	if device1Buf != nil {
		t.Logf("Device 1 output:\n%s", device1Buf.String())
	}
	if device2Buf != nil {
		t.Logf("Device 2 output:\n%s", device2Buf.String())
	}

	// Check for expected output - host should service 2 devices
	if !strings.Contains(hostOutput, "Serviced devices") || !strings.Contains(hostOutput, "count=2") {
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
