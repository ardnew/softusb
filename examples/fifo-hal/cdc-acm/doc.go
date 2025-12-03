// Package main provides integration tests for the CDC-ACM FIFO HAL example.
//
// This package contains integration tests that verify the CDC-ACM host and
// device can communicate correctly using the FIFO-based HAL with hot-plugging
// support.
//
// # Running Tests
//
// Run the integration tests with:
//
//	go test -v ./examples/fifo-hal/cdc-acm/
//
// Override default timeouts with test flags:
//
//	go test -v ./examples/fifo-hal/cdc-acm/ -args \
//	    -enum-timeout=15s \
//	    -transfer-timeout=10s
//
// # Test Flags
//
// The following flags are forwarded to the host and device subprocesses:
//
//   - -enum-timeout: Timeout for enumeration (default 10s)
//   - -transfer-timeout: Timeout for data transfers (default 5s)
//   - -verbose: Enable verbose (debug) logging
//   - -json: Use JSON log format
//
// # Test Cases
//
// TestCDCACMIntegration verifies single device enumeration and communication.
// TestCDCACMMultipleDevices verifies hot-plugging with multiple sequential devices.
//
// # Structure
//
// The actual device and host implementations are in subdirectories:
//
//   - device/: CDC-ACM device that echoes received data
//   - host/: CDC-ACM host that sends test messages
//
// Both support command-line flags for timeout configuration:
//
//   - -v: Enable verbose (debug) logging
//   - -json: Use JSON log format
//   - -enum-timeout: Timeout for enumeration (default 10s)
//   - -transfer-timeout: Timeout for data transfers (default 5s)
//   - -hotplug-limit: Number of devices to service (host only, default 1)
package main

// main is a stub function to make this a valid main package.
// The actual functionality is in the device/ and host/ subdirectories.
// This package only contains integration tests.
func main() {}
