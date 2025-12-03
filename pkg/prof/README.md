# prof

Package `prof` provides profiling utilities for the softusb USB stack, wrapping Go's [`runtime/pprof`](https://pkg.go.dev/runtime/pprof) with ergonomic APIs for on-demand profiling.

## Build Tag

This package uses conditional compilation. Build with the `profile` tag to enable profiling:

```bash
go build -tags profile
go test -tags profile ./...
```

Without the tag, all functions become no-ops with zero overhead, allowing profiling code to remain in place in production builds.

## HTTP Profiling

When built with the `profile` tag, HTTP handlers are automatically registered at `/debug/pprof/` via [`net/http/pprof`](https://pkg.go.dev/net/http/pprof). Start an HTTP server to access these endpoints:

```go
import (
    "net/http"
    _ "github.com/ardnew/softusb/pkg/prof"
)

func main() {
    go http.ListenAndServe("localhost:6060", nil)
    // ... application code ...
}
```

Access profiles at:

- [http://localhost:6060/debug/pprof/](http://localhost:6060/debug/pprof/)
- [http://localhost:6060/debug/pprof/heap](http://localhost:6060/debug/pprof/heap)
- [http://localhost:6060/debug/pprof/goroutine](http://localhost:6060/debug/pprof/goroutine)
- [http://localhost:6060/debug/pprof/profile?seconds=30](http://localhost:6060/debug/pprof/profile?seconds=30) (CPU)

## API Overview

### CPU Profiling

CPU profiling streams samples to a writer and requires explicit start/stop:

```go
import "github.com/ardnew/softusb/pkg/prof"

func main() {
    if err := prof.StartCPU("cpu.prof"); err != nil {
        log.Fatal(err)
    }
    defer prof.StopCPU()

    // ... code to profile ...
}
```

Or write to any `io.Writer`:

```go
var buf bytes.Buffer
prof.StartCPUWriter(&buf)
defer prof.StopCPU()
```

Attempting to start CPU profiling while already active returns `ErrCPUProfileActive`.

### Snapshot Profiles

Capture point-in-time snapshots of other profile types:

```go
// Write to file
prof.Write(prof.ProfileHeap, "heap.prof")
prof.Write(prof.ProfileGoroutine, "goroutine.prof")

// Write to io.Writer
prof.WriteTo(prof.ProfileAllocs, &buf)

// Human-readable output (debug=1)
prof.WriteToDebug(prof.ProfileGoroutine, os.Stdout, 1)
```

Available profiles:

| Profile | Description |
|---------|-------------|
| `ProfileHeap` | Live object allocations |
| `ProfileAllocs` | All past allocations (since program start) |
| `ProfileGoroutine` | Stack traces of all goroutines |
| `ProfileThreadCreate` | OS thread creation stacks |
| `ProfileBlock` | Blocking on synchronization primitives |
| `ProfileMutex` | Mutex contention |

**Note:** `ProfileCPU` cannot be used with `Write`/`WriteTo`; use `StartCPU`/`StopCPU` instead.

### Block and Mutex Profiling

Block and mutex profiles require runtime configuration:

```go
// Enable block profiling (rate=1 records every event)
prof.SetBlockProfileRate(1)

// Enable mutex profiling (rate=1 records every event)
prof.SetMutexProfileFraction(1)

// ... code to profile ...

prof.Write(prof.ProfileBlock, "block.prof")
prof.Write(prof.ProfileMutex, "mutex.prof")
```

## Analyzing Profiles

Use `go tool pprof` to analyze profiles:

```bash
# Interactive mode
go tool pprof cpu.prof

# Web UI
go tool pprof -http=:8080 cpu.prof

# Text report
go tool pprof -text heap.prof
```

For HTTP endpoints:

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

## Errors

| Error | Description |
|-------|-------------|
| `ErrCPUProfileActive` | CPU profiling is already active |
| `ErrCPUProfileNotActive` | CPU profiling is not active |
| `ErrInvalidProfile` | Invalid or unsupported profile type |
