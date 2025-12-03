package device

import (
	"context"
	"sync"

	"github.com/ardnew/softusb/device/hal"
	"github.com/ardnew/softusb/pkg"
)

// MaxEndpointAddresses is the number of possible endpoint addresses (0x00-0x0F IN and OUT).
const MaxEndpointAddresses = 32

// Stack manages the USB device stack.
type Stack struct {
	device  *Device
	hal     hal.DeviceHAL
	handler *StandardRequestHandler

	// State
	running bool
	mutex   sync.RWMutex

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Pending transfers - fixed-size arrays indexed by endpoint address
	// Address 0x00-0x0F = indices 0-15, 0x80-0x8F = indices 16-31
	pendingTransfers      [MaxEndpointAddresses][MaxPendingTransfersPerEndpoint]*Transfer
	pendingTransferCounts [MaxEndpointAddresses]int
	transferMutex         sync.Mutex

	// Reusable setup packet for zero-allocation reads
	setupBuf hal.SetupPacket

	// EP0 read buffer for control OUT data stage
	ep0ReadBuf [MaxControlDataSize]byte

	// Event callbacks
	onConnect    func()
	onDisconnect func()
}

// MaxControlDataSize is the maximum data size for control transfers.
const MaxControlDataSize = 512

// endpointIndex converts an endpoint address to an array index.
func endpointIndex(addr uint8) int {
	// OUT endpoints: 0x00-0x0F -> 0-15
	// IN endpoints: 0x80-0x8F -> 16-31
	if addr&0x80 != 0 {
		return int(addr&0x0F) + 16
	}
	return int(addr & 0x0F)
}

// halSpeedToDeviceSpeed converts hal.Speed to device.Speed.
func halSpeedToDeviceSpeed(s hal.Speed) Speed {
	switch s {
	case hal.SpeedLow:
		return SpeedLow
	case hal.SpeedFull:
		return SpeedFull
	case hal.SpeedHigh:
		return SpeedHigh
	default:
		return SpeedFull // Default to full speed
	}
}

// NewStack creates a new device stack.
func NewStack(dev *Device, h hal.DeviceHAL) *Stack {
	s := &Stack{
		device: dev,
		hal:    h,
		// pendingTransfers, pendingTransferCounts, setupBuf, ep0ReadBuf are zero-initialized
	}
	s.handler = NewStandardRequestHandler(dev)
	return s
}

// Start starts the device stack.
func (s *Stack) Start(ctx context.Context) error {
	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return pkg.ErrAlreadyRunning
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.mutex.Unlock()

	if err := s.hal.Init(s.ctx); err != nil {
		return err
	}

	if err := s.hal.Start(); err != nil {
		return err
	}

	s.mutex.Lock()
	s.running = true
	s.mutex.Unlock()

	pkg.LogDebug(pkg.ComponentStack, "device stack started")

	// Start the control transfer handler
	go s.controlLoop()

	return nil
}

// Stop stops the device stack.
func (s *Stack) Stop() error {
	s.mutex.Lock()
	if !s.running {
		s.mutex.Unlock()
		return nil
	}

	s.running = false
	if s.cancel != nil {
		s.cancel()
	}
	s.mutex.Unlock()

	if err := s.hal.Stop(); err != nil {
		return err
	}

	pkg.LogDebug(pkg.ComponentStack, "device stack stopped")
	return nil
}

// IsRunning returns true if the stack is running.
func (s *Stack) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}

// Device returns the underlying device.
func (s *Stack) Device() *Device {
	return s.device
}

// controlLoop handles control transfers on EP0.
func (s *Stack) controlLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if err := s.hal.ReadSetup(s.ctx, &s.setupBuf); err != nil {
			if s.ctx.Err() != nil {
				return
			}
			// Handle bus reset
			if err == pkg.ErrReset {
				s.device.Reset()
				continue
			}
			pkg.LogWarn(pkg.ComponentStack, "error reading setup",
				"error", err)
			continue
		}

		// Convert HAL setup packet to device setup packet
		var setup SetupPacket
		setup.RequestType = s.setupBuf.RequestType
		setup.Request = s.setupBuf.Request
		setup.Value = s.setupBuf.Value
		setup.Index = s.setupBuf.Index
		setup.Length = s.setupBuf.Length

		if err := s.handleSetup(&setup); err != nil {
			pkg.LogWarn(pkg.ComponentStack, "error handling setup",
				"error", err,
				"request", setup.String())
			s.hal.StallEP0()
		}
	}
}

// handleSetup processes a single SETUP transaction.
func (s *Stack) handleSetup(setup *SetupPacket) error {
	pkg.LogDebug(pkg.ComponentStack, "setup received",
		"request", setup.String())

	var responseData []byte
	var err error

	// Try standard request handler first
	if setup.IsStandard() {
		responseData, err = s.handler.HandleSetup(setup, nil)
		if err == nil {
			return s.completeSetup(setup, responseData)
		}
	}

	// Try class-specific handler
	if setup.IsClass() && setup.IsInterfaceRecipient() {
		iface := s.device.GetInterface(setup.InterfaceNumber())
		if iface != nil {
			handled, classErr := iface.HandleSetup(setup, nil)
			if handled {
				if classErr != nil {
					return classErr
				}
				return s.completeSetup(setup, nil)
			}
		}
	}

	// Request not handled
	if err != nil {
		return err
	}
	return pkg.ErrInvalidRequest
}

// completeSetup completes the control transfer.
func (s *Stack) completeSetup(setup *SetupPacket, data []byte) error {
	if setup.IsDeviceToHost() {
		// IN transfer - send data to host
		if len(data) > 0 {
			if err := s.hal.WriteEP0(s.ctx, data); err != nil {
				return err
			}
		}
		// Read status stage (zero-length OUT)
		_, err := s.hal.ReadEP0(s.ctx, s.ep0ReadBuf[:0])
		return err
	}

	// OUT transfer
	if setup.Length > 0 {
		// Read data stage
		maxLen := int(setup.Length)
		if maxLen > MaxControlDataSize {
			maxLen = MaxControlDataSize
		}
		_, err := s.hal.ReadEP0(s.ctx, s.ep0ReadBuf[:maxLen])
		if err != nil {
			return err
		}
	}
	// Send status stage
	return s.hal.AckEP0()
}

// SubmitTransfer submits a transfer for processing.
func (s *Stack) SubmitTransfer(t *Transfer) error {
	s.mutex.RLock()
	running := s.running
	s.mutex.RUnlock()

	if !running {
		return pkg.ErrNotConfigured
	}

	if !s.device.IsConfigured() {
		return pkg.ErrNotConfigured
	}

	if t.Endpoint == nil {
		return pkg.ErrInvalidEndpoint
	}

	s.transferMutex.Lock()
	idx := endpointIndex(t.Endpoint.Address)
	count := s.pendingTransferCounts[idx]
	if count >= MaxPendingTransfersPerEndpoint {
		s.transferMutex.Unlock()
		return pkg.ErrNoResources
	}
	s.pendingTransfers[idx][count] = t
	s.pendingTransferCounts[idx] = count + 1
	s.transferMutex.Unlock()

	// Process the transfer
	go s.processTransfer(t)

	return nil
}

// processTransfer processes a single transfer.
func (s *Stack) processTransfer(t *Transfer) {
	ctx := t.Context()

	select {
	case <-ctx.Done():
		t.Complete(pkg.TransferStatusCancelled, 0, pkg.ErrCancelled)
		s.removeTransfer(t)
		return
	default:
	}

	var n int
	var err error

	if t.IsIn() {
		// IN transfer - device to host (write to host)
		n, err = s.hal.Write(ctx, t.Endpoint.Address, t.Buffer)
	} else {
		// OUT transfer - host to device (read from host)
		n, err = s.hal.Read(ctx, t.Endpoint.Address, t.Buffer)
	}

	s.removeTransfer(t)

	if err != nil {
		status := errorToStatus(err)
		t.Complete(status, n, err)
		return
	}

	t.Endpoint.ToggleData()
	t.Complete(pkg.TransferStatusSuccess, n, nil)
}

// removeTransfer removes a transfer from the pending list.
func (s *Stack) removeTransfer(t *Transfer) {
	s.transferMutex.Lock()
	defer s.transferMutex.Unlock()

	idx := endpointIndex(t.Endpoint.Address)
	count := s.pendingTransferCounts[idx]
	for i := 0; i < count; i++ {
		if s.pendingTransfers[idx][i] == t {
			// Shift remaining transfers down
			copy(s.pendingTransfers[idx][i:count-1], s.pendingTransfers[idx][i+1:count])
			s.pendingTransfers[idx][count-1] = nil // Clear last slot
			s.pendingTransferCounts[idx] = count - 1
			break
		}
	}
}

// CancelTransfers cancels all pending transfers for an endpoint.
func (s *Stack) CancelTransfers(address uint8) {
	s.transferMutex.Lock()
	idx := endpointIndex(address)
	count := s.pendingTransferCounts[idx]
	// Copy pointers to local array to cancel outside lock
	var toCancel [MaxPendingTransfersPerEndpoint]*Transfer
	copy(toCancel[:count], s.pendingTransfers[idx][:count])
	// Clear the pending transfers
	for i := 0; i < count; i++ {
		s.pendingTransfers[idx][i] = nil
	}
	s.pendingTransferCounts[idx] = 0
	s.transferMutex.Unlock()

	for i := 0; i < count; i++ {
		toCancel[i].Cancel()
	}
}

// SetOnConnect sets the connect callback.
func (s *Stack) SetOnConnect(cb func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onConnect = cb
}

// SetOnDisconnect sets the disconnect callback.
func (s *Stack) SetOnDisconnect(cb func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onDisconnect = cb
}

// Speed returns the negotiated USB connection speed.
func (s *Stack) Speed() Speed {
	return halSpeedToDeviceSpeed(s.hal.GetSpeed())
}

// IsConnected returns true if the device is connected to a host.
func (s *Stack) IsConnected() bool {
	return s.hal.IsConnected()
}

// WaitConnect blocks until the device connects to a host or the context is cancelled.
func (s *Stack) WaitConnect(ctx context.Context) error {
	return s.hal.WaitConnect(ctx)
}

// WaitDisconnect blocks until the device disconnects or the context is cancelled.
func (s *Stack) WaitDisconnect(ctx context.Context) error {
	return s.hal.WaitDisconnect(ctx)
}

// Read performs a blocking read on an endpoint.
func (s *Stack) Read(ctx context.Context, ep *Endpoint, buf []byte) (int, error) {
	if !s.device.IsConfigured() {
		return 0, pkg.ErrNotConfigured
	}
	return s.hal.Read(ctx, ep.Address, buf)
}

// Write performs a blocking write on an endpoint.
func (s *Stack) Write(ctx context.Context, ep *Endpoint, data []byte) (int, error) {
	if !s.device.IsConfigured() {
		return 0, pkg.ErrNotConfigured
	}
	return s.hal.Write(ctx, ep.Address, data)
}

// errorToStatus converts an error to a transfer status.
func errorToStatus(err error) pkg.TransferStatus {
	switch err {
	case nil:
		return pkg.TransferStatusSuccess
	case pkg.ErrStall:
		return pkg.TransferStatusStall
	case pkg.ErrNAK:
		return pkg.TransferStatusNAK
	case pkg.ErrTimeout:
		return pkg.TransferStatusTimeout
	case pkg.ErrCancelled:
		return pkg.TransferStatusCancelled
	case pkg.ErrOverrun:
		return pkg.TransferStatusOverrun
	case pkg.ErrUnderrun:
		return pkg.TransferStatusUnderrun
	default:
		return pkg.TransferStatusError
	}
}
