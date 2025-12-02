//go:build linux

package linux

import (
	"sync"
	"syscall"
	"unsafe"
)

// =============================================================================
// Epoll Types
// =============================================================================

// epollEvent matches the kernel's struct epoll_event.
type epollEvent struct {
	events uint32
	data   [8]byte // union: ptr, fd, u32, u64
}

// pollDesc describes a file descriptor being polled.
type pollDesc struct {
	fd       int          // File descriptor
	events   uint32       // Events to watch for
	callback func(uint32) // Callback when events occur
}

// =============================================================================
// Poller
// =============================================================================

// poller manages epoll-based I/O multiplexing for USB devices.
type poller struct {
	epfd    int               // epoll file descriptor
	wakefd  int               // eventfd for waking the poller
	mu      sync.Mutex        // Protects fds map
	fds     map[int]*pollDesc // Tracked file descriptors
	running bool              // Whether poll loop is running
	done    chan struct{}     // Signal to stop polling
}

// newPoller creates a new poller instance.
func newPoller() (*poller, error) {
	// Create epoll instance
	epfd, err := epollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	// Create eventfd for wakeup signaling
	wakefd, err := eventfdCreate(0, syscall.O_NONBLOCK|syscall.O_CLOEXEC)
	if err != nil {
		syscall.Close(epfd)
		return nil, err
	}

	p := &poller{
		epfd:   epfd,
		wakefd: wakefd,
		fds:    make(map[int]*pollDesc),
		done:   make(chan struct{}),
	}

	// Add wakefd to epoll
	if err := p.addFD(wakefd, EPOLLIN, nil); err != nil {
		syscall.Close(wakefd)
		syscall.Close(epfd)
		return nil, err
	}

	return p, nil
}

// close shuts down the poller.
func (p *poller) close() error {
	p.mu.Lock()
	if p.running {
		close(p.done)
		p.wake()
	}
	p.mu.Unlock()

	if p.wakefd >= 0 {
		syscall.Close(p.wakefd)
	}
	if p.epfd >= 0 {
		syscall.Close(p.epfd)
	}
	return nil
}

// addFD adds a file descriptor to the poller.
func (p *poller) addFD(fd int, events uint32, callback func(uint32)) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	event := epollEvent{
		events: events,
	}
	*(*int)(unsafe.Pointer(&event.data)) = fd

	err := epollCtl(p.epfd, syscall.EPOLL_CTL_ADD, fd, &event)
	if err != nil {
		return err
	}

	p.fds[fd] = &pollDesc{
		fd:       fd,
		events:   events,
		callback: callback,
	}
	return nil
}

// modFD modifies the events for a file descriptor.
func (p *poller) modFD(fd int, events uint32) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	desc, ok := p.fds[fd]
	if !ok {
		return syscall.ENOENT
	}

	event := epollEvent{
		events: events,
	}
	*(*int)(unsafe.Pointer(&event.data)) = fd

	err := epollCtl(p.epfd, syscall.EPOLL_CTL_MOD, fd, &event)
	if err != nil {
		return err
	}

	desc.events = events
	return nil
}

// delFD removes a file descriptor from the poller.
func (p *poller) delFD(fd int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.fds, fd)
	return epollCtl(p.epfd, syscall.EPOLL_CTL_DEL, fd, nil)
}

// wake signals the poller to wake up.
func (p *poller) wake() error {
	var buf [8]byte
	buf[0] = 1 // Write value 1
	_, err := syscall.Write(p.wakefd, buf[:])
	return err
}

// poll runs the epoll wait loop.
// It blocks until an event occurs or the poller is closed.
func (p *poller) poll() error {
	var events [MaxEpollEvents]epollEvent

	p.mu.Lock()
	p.running = true
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()

	for {
		select {
		case <-p.done:
			return nil
		default:
		}

		n, err := epollWait(p.epfd, events[:], -1)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return err
		}

		for i := 0; i < n; i++ {
			fd := *(*int)(unsafe.Pointer(&events[i].data))
			evts := events[i].events

			if fd == p.wakefd {
				// Drain the eventfd
				var buf [8]byte
				syscall.Read(p.wakefd, buf[:])
				continue
			}

			p.mu.Lock()
			desc, ok := p.fds[fd]
			p.mu.Unlock()

			if ok && desc.callback != nil {
				desc.callback(evts)
			}
		}
	}
}

// pollOnce performs a single poll iteration with timeout.
// timeout is in milliseconds, -1 for infinite, 0 for non-blocking.
func (p *poller) pollOnce(timeout int) (int, error) {
	var events [MaxEpollEvents]epollEvent

	n, err := epollWait(p.epfd, events[:], timeout)
	if err != nil {
		return 0, err
	}

	processed := 0
	for i := 0; i < n; i++ {
		fd := *(*int)(unsafe.Pointer(&events[i].data))
		evts := events[i].events

		if fd == p.wakefd {
			// Drain the eventfd
			var buf [8]byte
			syscall.Read(p.wakefd, buf[:])
			continue
		}

		p.mu.Lock()
		desc, ok := p.fds[fd]
		p.mu.Unlock()

		if ok && desc.callback != nil {
			desc.callback(evts)
			processed++
		}
	}

	return processed, nil
}

// =============================================================================
// Syscall Wrappers
// =============================================================================

// epollCreate1 creates an epoll instance.
func epollCreate1(flags int) (int, error) {
	fd, _, errno := syscall.Syscall(syscall.SYS_EPOLL_CREATE1, uintptr(flags), 0, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}

// epollCtl controls an epoll instance.
func epollCtl(epfd, op, fd int, event *epollEvent) error {
	var eventPtr uintptr
	if event != nil {
		eventPtr = uintptr(unsafe.Pointer(event))
	}
	_, _, errno := syscall.Syscall6(
		syscall.SYS_EPOLL_CTL,
		uintptr(epfd),
		uintptr(op),
		uintptr(fd),
		eventPtr,
		0, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// epollWait waits for events on an epoll instance.
func epollWait(epfd int, events []epollEvent, timeout int) (int, error) {
	n, _, errno := syscall.Syscall6(
		syscall.SYS_EPOLL_WAIT,
		uintptr(epfd),
		uintptr(unsafe.Pointer(&events[0])),
		uintptr(len(events)),
		uintptr(timeout),
		0, 0,
	)
	if errno != 0 {
		return 0, errno
	}
	return int(n), nil
}

// eventfdCreate creates an eventfd.
func eventfdCreate(initval uint, flags int) (int, error) {
	fd, _, errno := syscall.Syscall(syscall.SYS_EVENTFD2, uintptr(initval), uintptr(flags), 0)
	if errno != 0 {
		return -1, errno
	}
	return int(fd), nil
}
