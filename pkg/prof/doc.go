// Package prof provides profiling utilities for the softusb USB stack.
//
// This package wraps [runtime/pprof] with simplified APIs for on-demand
// profiling. It is conditionally compiled using the "profile" build tag:
//
//	go build -tags profile
//	go test -tags profile
//
// When built without the "profile" tag, all exported functions become no-ops,
// allowing profiling code to remain in place without overhead in production.
//
// # HTTP Profiling
//
// When built with the "profile" tag, the package automatically registers
// HTTP handlers at /debug/pprof/ via [net/http/pprof]. Start an HTTP server
// to access these endpoints:
//
//	import (
//	    "net/http"
//	    _ "github.com/ardnew/softusb/pkg/prof"
//	)
//
//	func main() {
//	    go http.ListenAndServe("localhost:6060", nil)
//	    // ... application code ...
//	}
//
// Then access profiles at http://localhost:6060/debug/pprof/
//
// # CPU Profiling
//
// CPU profiling streams samples to a writer and requires explicit start/stop:
//
//	prof.StartCPU("cpu.prof")
//	defer prof.StopCPU()
//	// ... code to profile ...
//
// Attempting to start CPU profiling while already active returns
// [ErrCPUProfileActive].
//
// # Snapshot Profiles
//
// Other profiles capture a point-in-time snapshot:
//
//	prof.Write(prof.ProfileHeap, "heap.prof")
//	prof.Write(prof.ProfileGoroutine, "goroutine.prof")
//
// Available snapshot profiles:
//
//   - [ProfileHeap]: Live object allocations
//   - [ProfileAllocs]: All past allocations (since program start)
//   - [ProfileGoroutine]: Stack traces of all goroutines
//   - [ProfileThreadCreate]: OS thread creation stacks
//   - [ProfileBlock]: Blocking on synchronization primitives
//   - [ProfileMutex]: Mutex contention
//
// Note: [ProfileCPU] cannot be used with [Write] or [WriteTo]; use
// [StartCPU]/[StopCPU] instead.
//
// # Block and Mutex Profiling
//
// Block and mutex profiles require enabling at runtime:
//
//	prof.SetBlockProfileRate(1)    // Enable block profiling
//	prof.SetMutexProfileFraction(1) // Enable mutex profiling
//
// # Debug Output
//
// Use [WriteToDebug] for human-readable output (debug=1) instead of
// binary protobuf (debug=0):
//
//	prof.WriteToDebug(prof.ProfileGoroutine, os.Stdout, 1)
package prof
