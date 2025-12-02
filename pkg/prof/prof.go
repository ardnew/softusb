//go:build profile

package prof

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	_ "net/http/pprof" // Register HTTP handlers at /debug/pprof/
)

func init() {
	go func() {
		println(http.ListenAndServe("localhost:6060", nil))
	}()
}

// Profiling errors.
var (
	// ErrCPUProfileActive indicates CPU profiling is already active.
	ErrCPUProfileActive = errors.New("cpu profile already active")

	// ErrCPUProfileNotActive indicates CPU profiling is not active.
	ErrCPUProfileNotActive = errors.New("cpu profile not active")

	// ErrInvalidProfile indicates an invalid or unsupported profile type.
	ErrInvalidProfile = errors.New("invalid profile")
)

// Profile represents a pprof profile type.
type Profile string

// Profile type constants.
const (
	ProfileCPU          Profile = "cpu"
	ProfileHeap         Profile = "heap"
	ProfileAllocs       Profile = "allocs"
	ProfileGoroutine    Profile = "goroutine"
	ProfileThreadCreate Profile = "threadcreate"
	ProfileBlock        Profile = "block"
	ProfileMutex        Profile = "mutex"
)

// String returns the string representation of the profile type.
func (p Profile) String() string {
	return string(p)
}

var (
	// cpuMutex protects CPU profiling state.
	cpuMutex sync.Mutex

	// cpuFile holds the file handle when profiling to a file path.
	cpuFile *os.File

	// cpuActive indicates whether CPU profiling is currently active.
	cpuActive bool
)

// StartCPU starts CPU profiling and writes the profile to the specified path.
// Returns [ErrCPUProfileActive] if CPU profiling is already active.
func StartCPU(path string) error {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	if cpuActive {
		return ErrCPUProfileActive
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return err
	}

	cpuFile = f
	cpuActive = true
	return nil
}

// StartCPUWriter starts CPU profiling and writes the profile to the given writer.
// Returns [ErrCPUProfileActive] if CPU profiling is already active.
func StartCPUWriter(w io.Writer) error {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	if cpuActive {
		return ErrCPUProfileActive
	}

	if err := pprof.StartCPUProfile(w); err != nil {
		return err
	}

	cpuActive = true
	return nil
}

// StopCPU stops CPU profiling. It is safe to call even if profiling is not active.
func StopCPU() {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()

	if !cpuActive {
		return
	}

	pprof.StopCPUProfile()

	if cpuFile != nil {
		cpuFile.Close()
		cpuFile = nil
	}

	cpuActive = false
}

// IsCPUActive reports whether CPU profiling is currently active.
func IsCPUActive() bool {
	cpuMutex.Lock()
	defer cpuMutex.Unlock()
	return cpuActive
}

// Write writes the specified profile to a file at the given path.
// Returns [ErrInvalidProfile] if [ProfileCPU] is specified; use
// [StartCPU]/[StopCPU] for CPU profiling.
func Write(profile Profile, path string) error {
	if profile == ProfileCPU {
		printCPUProfileError()
		return ErrInvalidProfile
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return writeProfile(profile, f, 0)
}

// WriteTo writes the specified profile to the given writer in binary protobuf
// format. Returns [ErrInvalidProfile] if [ProfileCPU] is specified; use
// [StartCPU]/[StopCPU] for CPU profiling.
func WriteTo(profile Profile, w io.Writer) error {
	return WriteToDebug(profile, w, 0)
}

// WriteToDebug writes the specified profile to the given writer with the
// specified debug level. Debug level 0 produces binary protobuf output
// suitable for go tool pprof; debug level 1 produces human-readable text.
// Returns [ErrInvalidProfile] if [ProfileCPU] is specified; use
// [StartCPU]/[StopCPU] for CPU profiling.
func WriteToDebug(profile Profile, w io.Writer, debug int) error {
	if profile == ProfileCPU {
		printCPUProfileError()
		return ErrInvalidProfile
	}

	return writeProfile(profile, w, debug)
}

// writeProfile writes the named profile to w with the given debug level.
func writeProfile(profile Profile, w io.Writer, debug int) error {
	p := pprof.Lookup(string(profile))
	if p == nil {
		return ErrInvalidProfile
	}
	return p.WriteTo(w, debug)
}

// printCPUProfileError writes an instructional message to stderr explaining
// that CPU profiling requires StartCPU/StopCPU.
func printCPUProfileError() {
	fmt.Fprint(os.Stderr, `prof: CPU profiling requires StartCPU/StopCPU:

	prof.StartCPU("cpu.prof")
	defer prof.StopCPU()
`)
}

// SetBlockProfileRate controls the fraction of goroutine blocking events
// that are reported in the blocking profile. The rate is the average number
// of nanoseconds to block before a blocking event is recorded. Set rate to 0
// to disable block profiling, or 1 to record every blocking event.
func SetBlockProfileRate(rate int) {
	runtime.SetBlockProfileRate(rate)
}

// SetMutexProfileFraction controls the fraction of mutex contention events
// that are reported in the mutex profile. Set rate to 0 to disable mutex
// profiling, or 1 to record every mutex contention event. On average, 1/rate
// events are recorded.
func SetMutexProfileFraction(rate int) {
	runtime.SetMutexProfileFraction(rate)
}
