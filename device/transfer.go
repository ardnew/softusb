package device

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/ardnew/softusb/pkg"
)

// TransferCallback is called when a transfer completes.
type TransferCallback func(t *Transfer)

// Transfer represents a USB data transfer operation.
type Transfer struct {
	// Transfer type
	Type uint8 // EndpointTypeControl, Bulk, Interrupt, or Isochronous

	// Endpoint
	Endpoint *Endpoint

	// Setup packet for control transfers
	Setup *SetupPacket

	// Data buffer
	Buffer []byte
	Length int // Actual bytes transferred

	// Status
	Status pkg.TransferStatus
	Error  error

	// Callback
	Callback TransferCallback

	// Context for cancellation (stored by reference, not derived)
	ctx context.Context

	// Atomic cancellation flag (0 = not cancelled, 1 = cancelled)
	cancelled uint32

	// Isochronous-specific - fixed-size array for zero allocation
	isoPackets    [MaxIsoPackets]IsoPacket
	NumIsoPackets int // Number of isochronous packets in use

	// Internal state
	mutex     sync.Mutex
	completed bool
}

// IsoPacket describes a single isochronous packet within a transfer.
type IsoPacket struct {
	Offset       int // Offset in transfer buffer
	Length       int // Expected length
	ActualLength int // Actual bytes transferred
	Status       pkg.TransferStatus
}

// NewControlTransfer creates a new control transfer.
func NewControlTransfer(setup *SetupPacket, data []byte) *Transfer {
	return &Transfer{
		Type:   EndpointTypeControl,
		Setup:  setup,
		Buffer: data,
		ctx:    context.Background(),
	}
}

// NewBulkTransfer creates a new bulk transfer.
func NewBulkTransfer(ep *Endpoint, data []byte) *Transfer {
	return &Transfer{
		Type:     EndpointTypeBulk,
		Endpoint: ep,
		Buffer:   data,
		ctx:      context.Background(),
	}
}

// NewInterruptTransfer creates a new interrupt transfer.
func NewInterruptTransfer(ep *Endpoint, data []byte) *Transfer {
	return &Transfer{
		Type:     EndpointTypeInterrupt,
		Endpoint: ep,
		Buffer:   data,
		ctx:      context.Background(),
	}
}

// NewIsochronousTransfer creates a new isochronous transfer.
// numPackets must be <= MaxIsoPackets.
func NewIsochronousTransfer(ep *Endpoint, data []byte, numPackets int) *Transfer {
	if numPackets > MaxIsoPackets {
		numPackets = MaxIsoPackets
	}
	return &Transfer{
		Type:          EndpointTypeIsochronous,
		Endpoint:      ep,
		Buffer:        data,
		NumIsoPackets: numPackets,
		ctx:           context.Background(),
	}
}

// WithContext sets a custom context for the transfer.
// The context is stored by reference for cancellation checking.
func (t *Transfer) WithContext(ctx context.Context) *Transfer {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.ctx = ctx
	return t
}

// WithCallback sets the completion callback.
func (t *Transfer) WithCallback(cb TransferCallback) *Transfer {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Callback = cb
	return t
}

// Context returns the transfer's context.
func (t *Transfer) Context() context.Context {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if t.ctx == nil {
		return context.Background()
	}
	return t.ctx
}

// Cancel cancels the transfer.
func (t *Transfer) Cancel() {
	if atomic.CompareAndSwapUint32(&t.cancelled, 0, 1) {
		t.mutex.Lock()
		t.Status = pkg.TransferStatusCancelled
		t.Error = pkg.ErrCancelled
		t.mutex.Unlock()
	}
}

// Complete marks the transfer as completed with the given status.
func (t *Transfer) Complete(status pkg.TransferStatus, length int, err error) {
	t.mutex.Lock()
	if t.completed {
		t.mutex.Unlock()
		return
	}
	t.completed = true
	t.Status = status
	t.Length = length
	t.Error = err
	cb := t.Callback
	t.mutex.Unlock()

	if cb != nil {
		cb(t)
	}
}

// IsCompleted returns true if the transfer is complete.
func (t *Transfer) IsCompleted() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.completed
}

// IsCancelled returns true if the transfer was cancelled.
func (t *Transfer) IsCancelled() bool {
	return atomic.LoadUint32(&t.cancelled) != 0
}

// IsSuccess returns true if the transfer completed successfully.
func (t *Transfer) IsSuccess() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.completed && t.Status == pkg.TransferStatusSuccess
}

// Reset resets the transfer for reuse.
func (t *Transfer) Reset() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Status = 0
	t.Error = nil
	t.Length = 0
	t.completed = false
	atomic.StoreUint32(&t.cancelled, 0)
	t.ctx = context.Background()
}

// Direction returns the transfer direction based on endpoint or setup packet.
func (t *Transfer) Direction() uint8 {
	if t.Type == EndpointTypeControl && t.Setup != nil {
		return t.Setup.Direction()
	}
	if t.Endpoint != nil {
		return t.Endpoint.Direction()
	}
	return EndpointDirectionOut
}

// IsIn returns true if this is an IN transfer (device to host).
func (t *Transfer) IsIn() bool {
	return t.Direction() == EndpointDirectionIn
}

// IsOut returns true if this is an OUT transfer (host to device).
func (t *Transfer) IsOut() bool {
	return t.Direction() == EndpointDirectionOut
}

// MaxPacketSize returns the maximum packet size for this transfer.
func (t *Transfer) MaxPacketSize() int {
	if t.Endpoint != nil {
		return int(t.Endpoint.MaxPacketSize)
	}
	// Default EP0 size
	return 64
}

// SetupIsoPackets initializes isochronous packet descriptors with uniform sizes.
func (t *Transfer) SetupIsoPackets(packetSize int) {
	offset := 0
	for i := 0; i < t.NumIsoPackets; i++ {
		t.isoPackets[i] = IsoPacket{
			Offset: offset,
			Length: packetSize,
		}
		offset += packetSize
	}
}

// SetupIsoPacketsVariable initializes isochronous packets with variable sizes.
func (t *Transfer) SetupIsoPacketsVariable(sizes []int) {
	offset := 0
	for i := 0; i < len(sizes) && i < t.NumIsoPackets; i++ {
		t.isoPackets[i] = IsoPacket{
			Offset: offset,
			Length: sizes[i],
		}
		offset += sizes[i]
	}
}

// IsoPacket returns a pointer to the isochronous packet at the given index.
// Returns nil if the index is out of range.
func (t *Transfer) IsoPacket(index int) *IsoPacket {
	if index < 0 || index >= t.NumIsoPackets {
		return nil
	}
	return &t.isoPackets[index]
}

// TotalIsoLength returns the total expected length of all isochronous packets.
func (t *Transfer) TotalIsoLength() int {
	total := 0
	for i := 0; i < t.NumIsoPackets; i++ {
		total += t.isoPackets[i].Length
	}
	return total
}

// ActualIsoLength returns the total actual length of all isochronous packets.
func (t *Transfer) ActualIsoLength() int {
	total := 0
	for i := 0; i < t.NumIsoPackets; i++ {
		total += t.isoPackets[i].ActualLength
	}
	return total
}

// TransferPool manages a pool of reusable transfer objects.
type TransferPool struct {
	pool sync.Pool
}

// NewTransferPool creates a new transfer pool.
func NewTransferPool() *TransferPool {
	return &TransferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Transfer{
					ctx: context.Background(),
				}
			},
		},
	}
}

// Get retrieves a transfer from the pool.
func (p *TransferPool) Get() *Transfer {
	t := p.pool.Get().(*Transfer)
	t.Reset()
	return t
}

// Put returns a transfer to the pool.
func (p *TransferPool) Put(t *Transfer) {
	t.Endpoint = nil
	t.Setup = nil
	t.Buffer = nil
	t.Callback = nil
	t.NumIsoPackets = 0
	p.pool.Put(t)
}
