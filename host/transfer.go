package host

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ardnew/softusb/host/hal"
	"github.com/ardnew/softusb/pkg"
)

// Transfer represents a USB transfer request.
type Transfer struct {
	// Device address
	Address uint8

	// Endpoint address (0x00-0x0F for OUT, 0x80-0x8F for IN)
	Endpoint uint8

	// Transfer type
	Type hal.TransferType

	// Data buffer (for all transfers)
	Data []byte

	// Setup packet (for control transfers only)
	Setup *hal.SetupPacket

	// Callback when transfer completes
	Callback func(*Transfer, int, error)

	// Context for cancellation
	Context context.Context

	// Internal state
	id        uint64
	completed int32
	result    int
	err       error
}

// IsComplete returns true if the transfer has completed.
func (t *Transfer) IsComplete() bool {
	return atomic.LoadInt32(&t.completed) != 0
}

// Result returns the transfer result.
func (t *Transfer) Result() (int, error) {
	return t.result, t.err
}

// TransferManager manages asynchronous transfers.
type TransferManager struct {
	host *Host

	// Pending transfers (by ID)
	pending   map[uint64]*Transfer
	pendingMu sync.RWMutex

	// Next transfer ID
	nextID uint64

	// Worker pool
	workers int
	jobs    chan *Transfer

	// State
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewTransferManager creates a new transfer manager.
func NewTransferManager(host *Host, workers int) *TransferManager {
	if workers < 1 {
		workers = 1
	}
	return &TransferManager{
		host:    host,
		pending: make(map[uint64]*Transfer),
		workers: workers,
		jobs:    make(chan *Transfer, 100),
	}
}

// Start starts the transfer manager.
func (tm *TransferManager) Start(ctx context.Context) error {
	tm.ctx, tm.cancel = context.WithCancel(ctx)
	tm.running = true

	// Start workers
	for i := 0; i < tm.workers; i++ {
		go tm.worker(i)
	}

	return nil
}

// Stop stops the transfer manager.
func (tm *TransferManager) Stop() error {
	if tm.cancel != nil {
		tm.cancel()
	}
	tm.running = false
	close(tm.jobs)
	return nil
}

// Submit submits a transfer for execution.
func (tm *TransferManager) Submit(t *Transfer) (uint64, error) {
	if !tm.running {
		return 0, pkg.ErrNotRunning
	}

	// Assign ID
	t.id = atomic.AddUint64(&tm.nextID, 1)

	// Add to pending
	tm.pendingMu.Lock()
	tm.pending[t.id] = t
	tm.pendingMu.Unlock()

	// Submit to worker pool
	select {
	case tm.jobs <- t:
		return t.id, nil
	case <-tm.ctx.Done():
		return 0, pkg.ErrCancelled
	}
}

// Cancel cancels a pending transfer.
func (tm *TransferManager) Cancel(id uint64) error {
	tm.pendingMu.Lock()
	t, ok := tm.pending[id]
	delete(tm.pending, id)
	tm.pendingMu.Unlock()

	if !ok {
		return nil
	}

	atomic.StoreInt32(&t.completed, 1)
	t.err = pkg.ErrCancelled
	return nil
}

// worker processes transfers.
func (tm *TransferManager) worker(id int) {
	pkg.LogDebug(pkg.ComponentHost, "transfer worker started", "id", id)

	for t := range tm.jobs {
		tm.executeTransfer(t)
	}

	pkg.LogDebug(pkg.ComponentHost, "transfer worker stopped", "id", id)
}

// executeTransfer executes a single transfer.
func (tm *TransferManager) executeTransfer(t *Transfer) {
	// Check if already cancelled
	if t.Context != nil {
		select {
		case <-t.Context.Done():
			t.err = t.Context.Err()
			atomic.StoreInt32(&t.completed, 1)
			tm.completeTransfer(t)
			return
		default:
		}
	}

	// Execute based on type
	var n int
	var err error

	ctx := t.Context
	if ctx == nil {
		ctx = tm.ctx
	}

	switch t.Type {
	case hal.TransferControl:
		if t.Setup == nil {
			err = pkg.ErrInvalidParameter
		} else {
			n, err = tm.host.hal.ControlTransfer(ctx, hal.DeviceAddress(t.Address), t.Setup, t.Data)
		}

	case hal.TransferBulk:
		n, err = tm.host.hal.BulkTransfer(ctx, hal.DeviceAddress(t.Address), t.Endpoint, t.Data)

	case hal.TransferInterrupt:
		n, err = tm.host.hal.InterruptTransfer(ctx, hal.DeviceAddress(t.Address), t.Endpoint, t.Data)

	case hal.TransferIsochronous:
		n, err = tm.host.hal.IsochronousTransfer(ctx, hal.DeviceAddress(t.Address), t.Endpoint, t.Data)

	default:
		err = pkg.ErrInvalidParameter
	}

	t.result = n
	t.err = err
	atomic.StoreInt32(&t.completed, 1)

	tm.completeTransfer(t)
}

// completeTransfer handles transfer completion.
func (tm *TransferManager) completeTransfer(t *Transfer) {
	// Remove from pending
	tm.pendingMu.Lock()
	delete(tm.pending, t.id)
	tm.pendingMu.Unlock()

	// Invoke callback
	if t.Callback != nil {
		t.Callback(t, t.result, t.err)
	}
}

// PendingCount returns the number of pending transfers.
func (tm *TransferManager) PendingCount() int {
	tm.pendingMu.RLock()
	defer tm.pendingMu.RUnlock()
	return len(tm.pending)
}

// WaitAll waits for all pending transfers to complete.
func (tm *TransferManager) WaitAll(ctx context.Context) error {
	for {
		tm.pendingMu.RLock()
		count := len(tm.pending)
		tm.pendingMu.RUnlock()

		if count == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue waiting
		}
	}
}

// Pipe provides a buffered, bidirectional communication channel over USB endpoints.
type Pipe struct {
	device  *Device
	epIn    uint8
	epOut   uint8
	maxSize int

	readBuf  []byte
	readPos  int
	readLen  int
	writeBuf []byte

	mu sync.Mutex
}

// NewPipe creates a new pipe for the given endpoints.
func NewPipe(dev *Device, epIn, epOut uint8, maxPacketSize int) *Pipe {
	return &Pipe{
		device:   dev,
		epIn:     epIn,
		epOut:    epOut,
		maxSize:  maxPacketSize,
		readBuf:  make([]byte, maxPacketSize),
		writeBuf: make([]byte, maxPacketSize),
	}
}

// Read reads data from the IN endpoint.
func (p *Pipe) Read(ctx context.Context, data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If we have buffered data, return it
	if p.readPos < p.readLen {
		n := copy(data, p.readBuf[p.readPos:p.readLen])
		p.readPos += n
		return n, nil
	}

	// Read from device
	n, err := p.device.BulkTransfer(ctx, p.epIn, p.readBuf)
	if err != nil {
		return 0, err
	}

	// Buffer the data
	p.readPos = 0
	p.readLen = n

	// Return what we can
	copied := copy(data, p.readBuf[:n])
	p.readPos = copied
	return copied, nil
}

// Write writes data to the OUT endpoint.
func (p *Pipe) Write(ctx context.Context, data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	total := 0
	for len(data) > 0 {
		n := len(data)
		if n > p.maxSize {
			n = p.maxSize
		}

		copy(p.writeBuf, data[:n])
		written, err := p.device.BulkTransfer(ctx, p.epOut, p.writeBuf[:n])
		if err != nil {
			return total, err
		}

		total += written
		data = data[n:]
	}

	return total, nil
}

// Close closes the pipe.
func (p *Pipe) Close() error {
	return nil
}

// Device returns the device this pipe is connected to.
func (p *Pipe) Device() *Device {
	return p.device
}
