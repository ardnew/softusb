//go:build profile

package prof

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestStartCPU_Success(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.prof")

	err := StartCPU(path)
	if err != nil {
		t.Fatalf("StartCPU() error = %v, want nil", err)
	}
	defer StopCPU()

	if !IsCPUActive() {
		t.Error("IsCPUActive() = false, want true")
	}
}

func TestStartCPU_FailFastWhenActive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.prof")

	err := StartCPU(path)
	if err != nil {
		t.Fatalf("StartCPU() error = %v, want nil", err)
	}
	defer StopCPU()

	// Second call should fail fast
	err = StartCPU(filepath.Join(t.TempDir(), "cpu2.prof"))
	if !errors.Is(err, ErrCPUProfileActive) {
		t.Errorf("StartCPU() error = %v, want %v", err, ErrCPUProfileActive)
	}
}

func TestStartCPU_InvalidPath(t *testing.T) {
	err := StartCPU("/nonexistent/directory/cpu.prof")
	if err == nil {
		t.Error("StartCPU() error = nil, want error for invalid path")
		StopCPU()
	}
}

func TestStartCPUWriter_Success(t *testing.T) {
	var buf bytes.Buffer

	err := StartCPUWriter(&buf)
	if err != nil {
		t.Fatalf("StartCPUWriter() error = %v, want nil", err)
	}
	defer StopCPU()

	if !IsCPUActive() {
		t.Error("IsCPUActive() = false, want true")
	}
}

func TestStartCPUWriter_FailFastWhenActive(t *testing.T) {
	var buf bytes.Buffer

	err := StartCPUWriter(&buf)
	if err != nil {
		t.Fatalf("StartCPUWriter() error = %v, want nil", err)
	}
	defer StopCPU()

	// Second call should fail fast
	var buf2 bytes.Buffer
	err = StartCPUWriter(&buf2)
	if !errors.Is(err, ErrCPUProfileActive) {
		t.Errorf("StartCPUWriter() error = %v, want %v", err, ErrCPUProfileActive)
	}
}

func TestStopCPU_WhenNotActive(t *testing.T) {
	// Should not panic when called without active profiling
	StopCPU()
}

func TestStopCPU_ResetsState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.prof")

	err := StartCPU(path)
	if err != nil {
		t.Fatalf("StartCPU() error = %v, want nil", err)
	}

	StopCPU()

	if IsCPUActive() {
		t.Error("IsCPUActive() = true after StopCPU(), want false")
	}

	// Should be able to start again
	err = StartCPU(path)
	if err != nil {
		t.Errorf("StartCPU() after StopCPU() error = %v, want nil", err)
	}
	StopCPU()
}

func TestIsCPUActive_InitialState(t *testing.T) {
	// Ensure clean state
	StopCPU()

	if IsCPUActive() {
		t.Error("IsCPUActive() = true, want false initially")
	}
}

func TestWrite_SnapshotProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
	}{
		{"heap", ProfileHeap},
		{"allocs", ProfileAllocs},
		{"goroutine", ProfileGoroutine},
		{"threadcreate", ProfileThreadCreate},
		{"block", ProfileBlock},
		{"mutex", ProfileMutex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.name+".prof")

			err := Write(tt.profile, path)
			if err != nil {
				t.Errorf("Write(%v) error = %v, want nil", tt.profile, err)
			}

			// Verify file was created
			info, err := os.Stat(path)
			if err != nil {
				t.Errorf("os.Stat(%s) error = %v", path, err)
			} else if info.Size() == 0 {
				t.Errorf("Write(%v) created empty file", tt.profile)
			}
		})
	}
}

func TestWrite_CPUProfileRejected(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	path := filepath.Join(t.TempDir(), "cpu.prof")
	err := Write(ProfileCPU, path)

	w.Close()
	os.Stderr = oldStderr

	var stderr bytes.Buffer
	stderr.ReadFrom(r)

	if !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("Write(ProfileCPU) error = %v, want %v", err, ErrInvalidProfile)
	}

	// Verify stderr message contains instructions
	stderrStr := stderr.String()
	if !bytes.Contains([]byte(stderrStr), []byte("StartCPU")) {
		t.Errorf("Write(ProfileCPU) stderr = %q, want message containing 'StartCPU'", stderrStr)
	}
	if !bytes.Contains([]byte(stderrStr), []byte("StopCPU")) {
		t.Errorf("Write(ProfileCPU) stderr = %q, want message containing 'StopCPU'", stderrStr)
	}
}

func TestWrite_InvalidPath(t *testing.T) {
	err := Write(ProfileHeap, "/nonexistent/directory/heap.prof")
	if err == nil {
		t.Error("Write() error = nil, want error for invalid path")
	}
}

func TestWriteTo_SnapshotProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
	}{
		{"heap", ProfileHeap},
		{"allocs", ProfileAllocs},
		{"goroutine", ProfileGoroutine},
		{"threadcreate", ProfileThreadCreate},
		{"block", ProfileBlock},
		{"mutex", ProfileMutex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := WriteTo(tt.profile, &buf)
			if err != nil {
				t.Errorf("WriteTo(%v) error = %v, want nil", tt.profile, err)
			}

			if buf.Len() == 0 {
				t.Errorf("WriteTo(%v) wrote no data", tt.profile)
			}
		})
	}
}

func TestWriteTo_CPUProfileRejected(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	var buf bytes.Buffer
	err := WriteTo(ProfileCPU, &buf)

	w.Close()
	os.Stderr = oldStderr

	var stderr bytes.Buffer
	stderr.ReadFrom(r)

	if !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("WriteTo(ProfileCPU) error = %v, want %v", err, ErrInvalidProfile)
	}
}

func TestWriteToDebug_HumanReadable(t *testing.T) {
	var buf bytes.Buffer

	err := WriteToDebug(ProfileGoroutine, &buf, 1)
	if err != nil {
		t.Fatalf("WriteToDebug() error = %v, want nil", err)
	}

	// Debug level 1 should produce human-readable text
	output := buf.String()
	if len(output) == 0 {
		t.Error("WriteToDebug() wrote no data")
	}

	// Human-readable goroutine output should contain "goroutine"
	if !bytes.Contains(buf.Bytes(), []byte("goroutine")) {
		t.Errorf("WriteToDebug(ProfileGoroutine, _, 1) output does not contain 'goroutine'")
	}
}

func TestWriteToDebug_CPUProfileRejected(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	var buf bytes.Buffer
	err := WriteToDebug(ProfileCPU, &buf, 1)

	w.Close()
	os.Stderr = oldStderr

	var stderr bytes.Buffer
	stderr.ReadFrom(r)

	if !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("WriteToDebug(ProfileCPU) error = %v, want %v", err, ErrInvalidProfile)
	}
}

func TestWriteToDebug_InvalidProfile(t *testing.T) {
	var buf bytes.Buffer
	err := WriteToDebug(Profile("nonexistent"), &buf, 0)
	if !errors.Is(err, ErrInvalidProfile) {
		t.Errorf("WriteToDebug(invalid) error = %v, want %v", err, ErrInvalidProfile)
	}
}

func TestProfile_String(t *testing.T) {
	tests := []struct {
		profile Profile
		want    string
	}{
		{ProfileCPU, "cpu"},
		{ProfileHeap, "heap"},
		{ProfileAllocs, "allocs"},
		{ProfileGoroutine, "goroutine"},
		{ProfileThreadCreate, "threadcreate"},
		{ProfileBlock, "block"},
		{ProfileMutex, "mutex"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.profile.String(); got != tt.want {
				t.Errorf("Profile.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	const goroutines = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent IsCPUActive checks
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = IsCPUActive()
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentStartStop(t *testing.T) {
	const iterations = 50

	for i := 0; i < iterations; i++ {
		path := filepath.Join(t.TempDir(), "cpu.prof")

		err := StartCPU(path)
		if err != nil {
			t.Fatalf("iteration %d: StartCPU() error = %v", i, err)
		}

		StopCPU()
	}
}
