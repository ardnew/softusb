//go:build linux

package linux

import (
	"syscall"
	"testing"
)

// =============================================================================
// epollEvent Tests
// =============================================================================

func TestEpollEvent_Size(t *testing.T) {
	// Verify the structure has the expected layout
	var event epollEvent

	// events is 4 bytes, data is 8 bytes = 12 bytes total
	// But Go struct may have padding
	if event.events != 0 {
		t.Errorf("events should be zero-initialized")
	}
}

// =============================================================================
// pollDesc Tests
// =============================================================================

func TestPollDesc_Fields(t *testing.T) {
	desc := pollDesc{
		fd:       42,
		events:   EPOLLIN | EPOLLOUT,
		callback: func(events uint32) {},
	}

	if desc.fd != 42 {
		t.Errorf("fd = %d, want 42", desc.fd)
	}
	if desc.events != EPOLLIN|EPOLLOUT {
		t.Errorf("events = 0x%X, want 0x%X", desc.events, EPOLLIN|EPOLLOUT)
	}
	if desc.callback == nil {
		t.Error("callback should not be nil")
	}
}

// =============================================================================
// poller Tests (requires Linux)
// =============================================================================

func TestNewPoller(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	if p.epfd < 0 {
		t.Error("epfd should be >= 0")
	}
	if p.wakefd < 0 {
		t.Error("wakefd should be >= 0")
	}
	if p.fds == nil {
		t.Error("fds map should not be nil")
	}
	if p.done == nil {
		t.Error("done channel should not be nil")
	}
}

func TestPoller_Close(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}

	err = p.close()
	if err != nil {
		t.Errorf("close failed: %v", err)
	}

	// Second close should not panic
	err = p.close()
	if err != nil {
		t.Logf("second close returned: %v (expected)", err)
	}
}

func TestPoller_Wake(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	// Wake should not block
	err = p.wake()
	if err != nil {
		t.Errorf("wake failed: %v", err)
	}

	// Multiple wakes should work
	for i := 0; i < 3; i++ {
		if err := p.wake(); err != nil {
			t.Errorf("wake %d failed: %v", i, err)
		}
	}
}

func TestPoller_AddDelFD(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	// Create a pipe for testing
	var pipeFds [2]int
	if err := createPipe(&pipeFds); err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer closeFd(pipeFds[0])
	defer closeFd(pipeFds[1])

	// Add the read end to the poller
	err = p.addFD(pipeFds[0], EPOLLIN, func(events uint32) {
		// callback invoked
	})
	if err != nil {
		t.Fatalf("addFD failed: %v", err)
	}

	// Verify it's in the map
	p.mu.Lock()
	_, ok := p.fds[pipeFds[0]]
	p.mu.Unlock()
	if !ok {
		t.Error("fd should be in fds map after addFD")
	}

	// Delete the fd
	err = p.delFD(pipeFds[0])
	if err != nil {
		t.Fatalf("delFD failed: %v", err)
	}

	// Verify it's removed from the map
	p.mu.Lock()
	_, ok = p.fds[pipeFds[0]]
	p.mu.Unlock()
	if ok {
		t.Error("fd should not be in fds map after delFD")
	}
}

func TestPoller_ModFD(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	// Create a pipe for testing
	var pipeFds [2]int
	if err := createPipe(&pipeFds); err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer closeFd(pipeFds[0])
	defer closeFd(pipeFds[1])

	// Add the read end
	err = p.addFD(pipeFds[0], EPOLLIN, nil)
	if err != nil {
		t.Fatalf("addFD failed: %v", err)
	}

	// Modify events
	err = p.modFD(pipeFds[0], EPOLLIN|EPOLLOUT)
	if err != nil {
		t.Fatalf("modFD failed: %v", err)
	}

	// Verify events updated
	p.mu.Lock()
	desc := p.fds[pipeFds[0]]
	p.mu.Unlock()
	if desc.events != EPOLLIN|EPOLLOUT {
		t.Errorf("events = 0x%X, want 0x%X", desc.events, EPOLLIN|EPOLLOUT)
	}
}

func TestPoller_ModFD_NotFound(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	// Modify non-existent fd should fail
	err = p.modFD(9999, EPOLLIN)
	if err == nil {
		t.Error("modFD on non-existent fd should fail")
	}
}

func TestPoller_PollOnce(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	// Create a pipe
	var pipeFds [2]int
	if err := createPipe(&pipeFds); err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer closeFd(pipeFds[0])
	defer closeFd(pipeFds[1])

	// Track callback
	callCount := 0
	err = p.addFD(pipeFds[0], EPOLLIN, func(events uint32) {
		callCount++
	})
	if err != nil {
		t.Fatalf("addFD failed: %v", err)
	}

	// Write to pipe to trigger event
	if _, err := writeToPipe(pipeFds[1]); err != nil {
		t.Fatalf("write to pipe failed: %v", err)
	}

	// Poll should process the event
	n, err := p.pollOnce(100) // 100ms timeout
	if err != nil {
		t.Fatalf("pollOnce failed: %v", err)
	}

	if n != 1 {
		t.Errorf("pollOnce returned %d, want 1", n)
	}

	if callCount != 1 {
		t.Errorf("callback count = %d, want 1", callCount)
	}
}

func TestPoller_PollOnce_Timeout(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	// Poll with nothing ready should timeout
	n, err := p.pollOnce(1) // 1ms timeout
	if err != nil {
		t.Fatalf("pollOnce failed: %v", err)
	}

	if n != 0 {
		t.Errorf("pollOnce returned %d, want 0 (timeout)", n)
	}
}

func TestPoller_PollOnce_Wake(t *testing.T) {
	p, err := newPoller()
	if err != nil {
		t.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	// Wake the poller
	if err := p.wake(); err != nil {
		t.Fatalf("wake failed: %v", err)
	}

	// Poll should process the wake event (but not count it in n)
	n, err := p.pollOnce(100)
	if err != nil {
		t.Fatalf("pollOnce failed: %v", err)
	}

	// Wake event should be processed but not counted
	if n != 0 {
		t.Errorf("pollOnce returned %d, want 0 (wake doesn't count)", n)
	}
}

// =============================================================================
// Helper Functions for Testing
// =============================================================================

func createPipe(fds *[2]int) error {
	return syscall.Pipe2(fds[:], syscall.O_NONBLOCK|syscall.O_CLOEXEC)
}

func closeFd(fd int) {
	syscall.Close(fd)
}

func writeToPipe(fd int) (int, error) {
	return syscall.Write(fd, []byte{1})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkPoller_Wake(b *testing.B) {
	p, err := newPoller()
	if err != nil {
		b.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.wake()
		// Drain the eventfd to avoid filling it
		var buf [8]byte
		syscall.Read(p.wakefd, buf[:])
	}
}

func BenchmarkPoller_PollOnce_Empty(b *testing.B) {
	p, err := newPoller()
	if err != nil {
		b.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.pollOnce(0) // Non-blocking
	}
}

func BenchmarkPoller_AddDelFD(b *testing.B) {
	p, err := newPoller()
	if err != nil {
		b.Fatalf("newPoller failed: %v", err)
	}
	defer p.close()

	var pipeFds [2]int
	if err := createPipe(&pipeFds); err != nil {
		b.Fatalf("failed to create pipe: %v", err)
	}
	defer closeFd(pipeFds[0])
	defer closeFd(pipeFds[1])

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.addFD(pipeFds[0], EPOLLIN, nil)
		p.delFD(pipeFds[0])
	}
}
