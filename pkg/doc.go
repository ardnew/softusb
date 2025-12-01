// Package pkg provides shared utilities for the softusb USB stack.
//
// This package contains common functionality used across both the device
// and host stacks, including:
//
//   - Structured logging via Go's standard [log/slog] package
//   - Sentinel error types for USB protocol errors
//   - Component identifiers for log filtering
//
// The package is designed to have zero external dependencies, relying
// only on the Go standard library.
//
// # Logging
//
// The logging subsystem wraps [log/slog] with USB-specific context:
//
//	pkg.SetLogLevel(slog.LevelDebug)
//	pkg.LogInfo(pkg.ComponentDevice, "device configured", "config", 1)
//
// # Errors
//
// Common USB errors are defined as sentinel values:
//
//	if errors.Is(err, pkg.ErrStall) {
//	    // Handle endpoint stall
//	}
package pkg
