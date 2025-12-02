//go:build !profile

package prof

import "io"

// Profiling errors (defined for API compatibility but never returned by stubs).
var (
	// ErrCPUProfileActive indicates CPU profiling is already active.
	ErrCPUProfileActive error

	// ErrCPUProfileNotActive indicates CPU profiling is not active.
	ErrCPUProfileNotActive error

	// ErrInvalidProfile indicates an invalid or unsupported profile type.
	ErrInvalidProfile error
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

// StartCPU is a no-op when built without the "profile" tag.
func StartCPU(_ string) error {
	return nil
}

// StartCPUWriter is a no-op when built without the "profile" tag.
func StartCPUWriter(_ io.Writer) error {
	return nil
}

// StopCPU is a no-op when built without the "profile" tag.
func StopCPU() {}

// IsCPUActive always returns false when built without the "profile" tag.
func IsCPUActive() bool {
	return false
}

// Write is a no-op when built without the "profile" tag.
func Write(_ Profile, _ string) error {
	return nil
}

// WriteTo is a no-op when built without the "profile" tag.
func WriteTo(_ Profile, _ io.Writer) error {
	return nil
}

// WriteToDebug is a no-op when built without the "profile" tag.
func WriteToDebug(_ Profile, _ io.Writer, _ int) error {
	return nil
}

// SetBlockProfileRate is a no-op when built without the "profile" tag.
func SetBlockProfileRate(_ int) {}

// SetMutexProfileFraction is a no-op when built without the "profile" tag.
func SetMutexProfileFraction(_ int) {}
