package fifo

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ardnew/softusb/device/hal"
	"github.com/ardnew/softusb/pkg"
)

// MaxEndpoints is the maximum number of data endpoints (1-15 IN and OUT).
const MaxEndpoints = 15

// MaxPacketSize is the maximum packet size for any endpoint.
const MaxPacketSize = 512

// Message types for FIFO protocol (must match host HAL).
const (
	msgSetup   = 0x01 // SETUP packet from host
	msgData    = 0x02 // DATA packet
	msgAck     = 0x03 // ACK response
	msgNak     = 0x04 // NAK response
	msgStall   = 0x05 // STALL response
	msgReset   = 0x12 // Port reset
	msgAddress = 0x13 // Set address
)

// Header size for messages.
const headerSize = 3 // type (1) + length (2)

// Connection signal bytes (one-way signaling to host).
const (
	sigConnect    = 0x01 // Device connected
	sigDisconnect = 0x00 // Device disconnected
)

// FIFO file names.
const (
	fifoHostToDevice = "host_to_device"
	fifoDeviceToHost = "device_to_host"
	fifoInterrupts   = "interrupts"
	fifoConnection   = "connection"
)

// HAL implements hal.DeviceHAL using named pipes (FIFOs).
// Each device instance creates a unique subdirectory under the bus directory
// to enable hot-plugging and multiple device support.
type HAL struct {
	// Bus directory (root directory shared with host)
	busDir string

	// Device subdirectory (busDir/device-{uuid}/)
	deviceDir string
	uuid      string

	// Control endpoint FIFOs
	hostToDeviceRead  *os.File // Device reads commands from host
	deviceToHostWrite *os.File // Device writes responses to host
	interruptsWrite   *os.File // Device writes interrupt data
	connectionWrite   *os.File // Device signals connection status

	// Data endpoint FIFOs (indexed by endpoint number 1-15)
	epInWrite [MaxEndpoints]*os.File // Device writes IN data
	epOutRead [MaxEndpoints]*os.File // Device reads OUT data

	// State
	connected uint32 // Atomic: 1 = connected, 0 = disconnected
	speed     hal.Speed
	address   uint8

	// Configured endpoints
	endpoints     [MaxEndpoints * 2]hal.EndpointConfig // IN at [0-14], OUT at [15-29]
	endpointCount int

	// Synchronization
	mutex     sync.RWMutex
	initDone  bool
	connectCh chan struct{}
	disconnCh chan struct{}
	closeCh   chan struct{}
	closeOnce sync.Once

	// Internal buffers (zero-allocation)
	readBuf  [MaxPacketSize + headerSize + 16]byte // Extra space for protocol overhead
	writeBuf [MaxPacketSize + headerSize + 16]byte

	// Pending setup packet for ReadSetup
	pendingSetup    hal.SetupPacket
	hasPendingSetup bool
}

// New creates a new FIFO-based device HAL.
// The busDir parameter specifies the root bus directory shared with the host.
// The device will create its own subdirectory (device-{uuid}/) inside busDir.
func New(busDir string) *HAL {
	return &HAL{
		busDir:    busDir,
		speed:     hal.SpeedFull,
		connectCh: make(chan struct{}, 1),
		disconnCh: make(chan struct{}, 1),
		closeCh:   make(chan struct{}),
	}
}

// generateUUID generates a random UUID using crypto/rand.
func generateUUID() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}
	// Set version 4 (random) bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuid[:]), nil
}

// Init initializes the HAL by creating the device subdirectory and FIFO files.
func (h *HAL) Init(ctx context.Context) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.initDone {
		return pkg.ErrAlreadyRunning
	}

	// Generate UUID for this device
	uuid, err := generateUUID()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}
	h.uuid = uuid
	h.deviceDir = filepath.Join(h.busDir, "device-"+uuid)

	// Create bus directory if needed
	if err := os.MkdirAll(h.busDir, 0o755); err != nil {
		return fmt.Errorf("create bus dir: %w", err)
	}

	// Create device subdirectory
	if err := os.MkdirAll(h.deviceDir, 0o755); err != nil {
		return fmt.Errorf("create device dir: %w", err)
	}

	// Create FIFOs in device subdirectory
	if err := h.createFIFO(fifoHostToDevice); err != nil {
		return err
	}
	if err := h.createFIFO(fifoDeviceToHost); err != nil {
		return err
	}
	if err := h.createFIFO(fifoInterrupts); err != nil {
		return err
	}
	if err := h.createFIFO(fifoConnection); err != nil {
		return err
	}

	// Create data endpoint FIFOs (1-15)
	for i := 1; i <= MaxEndpoints; i++ {
		if err := h.createFIFO(fmt.Sprintf("ep%d_in", i)); err != nil {
			return err
		}
		if err := h.createFIFO(fmt.Sprintf("ep%d_out", i)); err != nil {
			return err
		}
	}

	// Open FIFOs with O_RDWR|O_NONBLOCK to avoid blocking
	// Connection FIFO - device writes
	h.connectionWrite, err = h.openFIFO(fifoConnection, os.O_RDWR|syscall.O_NONBLOCK)
	if err != nil {
		h.cleanup()
		return err
	}

	// Device to host - device writes
	h.deviceToHostWrite, err = h.openFIFO(fifoDeviceToHost, os.O_RDWR|syscall.O_NONBLOCK)
	if err != nil {
		h.cleanup()
		return err
	}

	// Interrupts - device writes
	h.interruptsWrite, err = h.openFIFO(fifoInterrupts, os.O_RDWR|syscall.O_NONBLOCK)
	if err != nil {
		h.cleanup()
		return err
	}

	// Host to device - device reads
	h.hostToDeviceRead, err = h.openFIFO(fifoHostToDevice, os.O_RDWR|syscall.O_NONBLOCK)
	if err != nil {
		h.cleanup()
		return err
	}

	// Open all endpoint FIFOs during init (so host can open them)
	// IN endpoints: device writes
	// OUT endpoints: device reads
	for i := 1; i <= MaxEndpoints; i++ {
		idx := i - 1
		h.epInWrite[idx], err = h.openFIFO(fmt.Sprintf("ep%d_in", i), os.O_RDWR|syscall.O_NONBLOCK)
		if err != nil {
			h.cleanup()
			return err
		}
		h.epOutRead[idx], err = h.openFIFO(fmt.Sprintf("ep%d_out", i), os.O_RDWR|syscall.O_NONBLOCK)
		if err != nil {
			h.cleanup()
			return err
		}
	}

	h.initDone = true
	pkg.LogInfo(pkg.ComponentHAL, "fifo device HAL initialized",
		"busDir", h.busDir,
		"deviceDir", h.deviceDir,
		"uuid", h.uuid)

	return nil
}

// Start enables the HAL and signals connection to host.
func (h *HAL) Start() error {
	h.mutex.Lock()
	if !h.initDone {
		h.mutex.Unlock()
		return pkg.ErrNotConfigured
	}
	h.mutex.Unlock()

	// Signal connection to host
	if _, err := h.connectionWrite.Write([]byte{sigConnect}); err != nil {
		pkg.LogWarn(pkg.ComponentHAL, "failed to signal connection", "error", err)
	}

	atomic.StoreUint32(&h.connected, 1)

	// Signal on connect channel
	select {
	case h.connectCh <- struct{}{}:
	default:
	}

	pkg.LogInfo(pkg.ComponentHAL, "fifo device HAL started")
	return nil
}

// Stop stops the HAL, signals disconnection, and cleans up.
func (h *HAL) Stop() error {
	// Signal disconnection to host
	h.mutex.RLock()
	if h.connectionWrite != nil {
		h.connectionWrite.Write([]byte{sigDisconnect})
	}
	h.mutex.RUnlock()

	atomic.StoreUint32(&h.connected, 0)

	// Signal on disconnect channel
	select {
	case h.disconnCh <- struct{}{}:
	default:
	}

	h.closeOnce.Do(func() {
		close(h.closeCh)
	})

	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.cleanup()

	h.initDone = false
	pkg.LogInfo(pkg.ComponentHAL, "fifo device HAL stopped")
	return nil
}

// cleanup closes all FIFOs and removes the device directory.
func (h *HAL) cleanup() {
	// Close all open FIFOs
	if h.hostToDeviceRead != nil {
		h.hostToDeviceRead.Close()
		h.hostToDeviceRead = nil
	}
	if h.deviceToHostWrite != nil {
		h.deviceToHostWrite.Close()
		h.deviceToHostWrite = nil
	}
	if h.interruptsWrite != nil {
		h.interruptsWrite.Close()
		h.interruptsWrite = nil
	}
	if h.connectionWrite != nil {
		h.connectionWrite.Close()
		h.connectionWrite = nil
	}

	for i := 0; i < MaxEndpoints; i++ {
		if h.epInWrite[i] != nil {
			h.epInWrite[i].Close()
			h.epInWrite[i] = nil
		}
		if h.epOutRead[i] != nil {
			h.epOutRead[i].Close()
			h.epOutRead[i] = nil
		}
	}

	// Remove device directory
	if h.deviceDir != "" {
		os.RemoveAll(h.deviceDir)
	}
}

// SetAddress sets the device address (stored locally, not used for FIFO).
func (h *HAL) SetAddress(address uint8) error {
	h.mutex.Lock()
	h.address = address
	h.mutex.Unlock()
	pkg.LogDebug(pkg.ComponentHAL, "address set", "address", address)
	return nil
}

// ConfigureEndpoints configures the data endpoints.
// The endpoint FIFOs are already opened during Init, so this just tracks which endpoints are active.
func (h *HAL) ConfigureEndpoints(endpoints []hal.EndpointConfig) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.endpointCount = 0

	// Track which endpoints are configured
	for _, ep := range endpoints {
		num := ep.Number()
		if num == 0 || num > MaxEndpoints {
			continue
		}

		if h.endpointCount >= len(h.endpoints) {
			break
		}

		h.endpoints[h.endpointCount] = ep
		h.endpointCount++
	}

	pkg.LogDebug(pkg.ComponentHAL, "endpoints configured", "count", h.endpointCount)
	return nil
}

// ReadSetup reads a SETUP packet from EP0.
func (h *HAL) ReadSetup(ctx context.Context, out *hal.SetupPacket) error {
	// Check for pending setup from a previous message
	h.mutex.Lock()
	if h.hasPendingSetup {
		*out = h.pendingSetup
		h.hasPendingSetup = false
		h.mutex.Unlock()
		return nil
	}
	f := h.hostToDeviceRead
	h.mutex.Unlock()

	if f == nil {
		return pkg.ErrNotConfigured
	}

	// Read message header [type, length_lo, length_hi]
	for {
		header := h.readBuf[:headerSize]
		n, err := h.readWithContext(ctx, f, header)
		if err != nil {
			return err
		}
		if n < headerSize {
			return pkg.ErrSetupPacketTooShort
		}

		msgType := header[0]
		msgLen := int(binary.LittleEndian.Uint16(header[1:3]))

		// Read payload
		if msgLen > 0 {
			payload := h.readBuf[headerSize : headerSize+msgLen]
			n, err = h.readWithContext(ctx, f, payload)
			if err != nil {
				return err
			}
			if n < msgLen {
				return pkg.ErrSetupPacketTooShort
			}
		}

		switch msgType {
		case msgSetup:
			// Payload: [address, setup_packet(8), optional_data...]
			if msgLen < 1+hal.SetupPacketSize {
				return pkg.ErrSetupPacketTooShort
			}
			// address is at payload[0], setup starts at payload[1]
			if !hal.ParseSetupPacket(h.readBuf[headerSize+1:headerSize+1+hal.SetupPacketSize], out) {
				return pkg.ErrSetupPacketTooShort
			}

			pkg.LogDebug(pkg.ComponentHAL, "setup received",
				"reqType", out.RequestType,
				"req", out.Request,
				"value", out.Value,
				"index", out.Index,
				"length", out.Length)
			return nil

		case msgReset:
			// Port reset - send ACK and return ErrReset to notify stack
			h.sendAck()
			pkg.LogDebug(pkg.ComponentHAL, "port reset received")
			return pkg.ErrReset

		case msgAddress:
			// Set address - payload[0] is the new address
			if msgLen >= 1 {
				h.mutex.Lock()
				h.address = h.readBuf[headerSize]
				h.mutex.Unlock()
				h.sendAck()
				pkg.LogDebug(pkg.ComponentHAL, "address set", "address", h.readBuf[headerSize])
			}
			continue

		case msgData:
			// Data packet for a data endpoint - shouldn't happen on EP0
			// but handle it gracefully
			pkg.LogWarn(pkg.ComponentHAL, "unexpected DATA message on EP0")
			continue

		default:
			pkg.LogWarn(pkg.ComponentHAL, "unknown message type", "type", msgType)
			continue
		}
	}
}

// WriteEP0 writes data to EP0 (control IN phase).
func (h *HAL) WriteEP0(ctx context.Context, data []byte) error {
	h.mutex.RLock()
	f := h.deviceToHostWrite
	h.mutex.RUnlock()

	if f == nil {
		return pkg.ErrNotConfigured
	}

	// Send DATA message: [msgData, len_lo, len_hi, data...]
	return h.sendMessage(ctx, f, msgData, data)
}

// ReadEP0 reads data from EP0 (control OUT phase).
// For the FIFO HAL, OUT data is included in the SETUP message payload,
// so this is typically a no-op or reads zero bytes for status phase.
func (h *HAL) ReadEP0(ctx context.Context, buf []byte) (int, error) {
	// Status phase - nothing to read, the host sends a zero-length packet
	// which we don't need to explicitly read in our protocol
	return 0, nil
}

// StallEP0 stalls the control endpoint.
func (h *HAL) StallEP0() error {
	h.mutex.RLock()
	f := h.deviceToHostWrite
	h.mutex.RUnlock()

	if f != nil {
		// Send STALL response
		h.sendMessage(context.Background(), f, msgStall, nil)
	}
	pkg.LogDebug(pkg.ComponentHAL, "EP0 stalled")
	return nil
}

// AckEP0 sends a zero-length packet to acknowledge.
func (h *HAL) AckEP0() error {
	return h.sendAck()
}

// sendAck sends an ACK message to the host.
func (h *HAL) sendAck() error {
	h.mutex.RLock()
	f := h.deviceToHostWrite
	h.mutex.RUnlock()

	if f == nil {
		return pkg.ErrNotConfigured
	}
	return h.sendMessage(context.Background(), f, msgAck, nil)
}

// Read reads data from an OUT endpoint.
func (h *HAL) Read(ctx context.Context, address uint8, buf []byte) (int, error) {
	num := address & 0x0F
	if num == 0 || num > MaxEndpoints {
		return 0, pkg.ErrInvalidEndpoint
	}

	h.mutex.RLock()
	f := h.epOutRead[num-1]
	h.mutex.RUnlock()

	if f == nil {
		return 0, pkg.ErrInvalidEndpoint
	}

	return h.readPacket(ctx, f, buf)
}

// Write writes data to an IN endpoint.
func (h *HAL) Write(ctx context.Context, address uint8, data []byte) (int, error) {
	num := address & 0x0F
	if num == 0 || num > MaxEndpoints {
		return 0, pkg.ErrInvalidEndpoint
	}

	h.mutex.RLock()
	f := h.epInWrite[num-1]
	h.mutex.RUnlock()

	if f == nil {
		return 0, pkg.ErrInvalidEndpoint
	}

	if err := h.writePacket(ctx, f, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

// Stall stalls the specified endpoint.
func (h *HAL) Stall(address uint8) error {
	pkg.LogDebug(pkg.ComponentHAL, "endpoint stalled", "address", address)
	return nil
}

// ClearStall clears a stall condition.
func (h *HAL) ClearStall(address uint8) error {
	pkg.LogDebug(pkg.ComponentHAL, "endpoint stall cleared", "address", address)
	return nil
}

// IsConnected returns true if connected to a host.
func (h *HAL) IsConnected() bool {
	return atomic.LoadUint32(&h.connected) == 1
}

// GetSpeed returns the negotiated connection speed.
func (h *HAL) GetSpeed() hal.Speed {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.speed
}

// WaitConnect blocks until connected or context is cancelled.
func (h *HAL) WaitConnect(ctx context.Context) error {
	if h.IsConnected() {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.connectCh:
		return nil
	case <-h.closeCh:
		return pkg.ErrCancelled
	}
}

// WaitDisconnect blocks until disconnected or context is cancelled.
func (h *HAL) WaitDisconnect(ctx context.Context) error {
	if !h.IsConnected() {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.disconnCh:
		return nil
	case <-h.closeCh:
		return pkg.ErrCancelled
	}
}

// DeviceDir returns the device subdirectory path.
// This can be useful for debugging or for tests that need to know the path.
func (h *HAL) DeviceDir() string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.deviceDir
}

// UUID returns the device's unique identifier.
func (h *HAL) UUID() string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.uuid
}

// createFIFO creates a named pipe at the given path.
func (h *HAL) createFIFO(name string) error {
	path := filepath.Join(h.deviceDir, name)

	// Remove existing file if any
	os.Remove(path)

	if err := syscall.Mkfifo(path, 0o666); err != nil {
		return fmt.Errorf("mkfifo %s: %w", name, err)
	}

	return nil
}

// openFIFO opens a named pipe with the given flags.
func (h *HAL) openFIFO(name string, flag int) (*os.File, error) {
	path := filepath.Join(h.deviceDir, name)
	f, err := os.OpenFile(path, flag, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	return f, nil
}

// readWithContext reads exactly len(buf) bytes from a file with context cancellation support.
// It handles partial reads and retries with timeouts.
func (h *HAL) readWithContext(ctx context.Context, f *os.File, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		case <-h.closeCh:
			return total, pkg.ErrCancelled
		default:
		}

		// Set a read deadline
		f.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := f.Read(buf[total:])
		if n > 0 {
			total += n
		}
		if err != nil {
			if os.IsTimeout(err) {
				// Timeout, retry
				continue
			}
			// Other error
			return total, err
		}
		if n == 0 && err == nil {
			// EOF or no data - treat as timeout and retry
			continue
		}
	}
	return total, nil
}

// sendMessage sends a protocol message with header [type, len_lo, len_hi, data...].
func (h *HAL) sendMessage(ctx context.Context, f *os.File, msgType byte, data []byte) error {
	// Check for cancellation first
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-h.closeCh:
		return pkg.ErrCancelled
	default:
	}

	h.mutex.Lock()
	buf := h.writeBuf[:]
	h.mutex.Unlock()

	n := len(data)
	if n > MaxPacketSize {
		n = MaxPacketSize
	}

	// Build message: [type, len_lo, len_hi, data...]
	buf[0] = msgType
	binary.LittleEndian.PutUint16(buf[1:3], uint16(n))

	// Copy data
	if n > 0 {
		copy(buf[headerSize:], data[:n])
	}

	total := headerSize + n

	// Write all bytes
	written := 0
	for written < total {
		m, err := f.Write(buf[written:total])
		if m > 0 {
			written += m
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// writePacket writes a DATA message for data endpoints.
func (h *HAL) writePacket(ctx context.Context, f *os.File, data []byte) error {
	return h.sendMessage(ctx, f, msgData, data)
}

// readPacket reads a DATA message from a data endpoint.
func (h *HAL) readPacket(ctx context.Context, f *os.File, buf []byte) (int, error) {
	// Read header
	header := h.readBuf[:headerSize]
	n, err := h.readWithContext(ctx, f, header)
	if err != nil {
		pkg.LogDebug(pkg.ComponentHAL, "readPacket header error", "error", err)
		return 0, err
	}
	if n < headerSize {
		pkg.LogDebug(pkg.ComponentHAL, "readPacket header too short", "got", n)
		return 0, io.ErrUnexpectedEOF
	}

	msgType := header[0]
	length := int(binary.LittleEndian.Uint16(header[1:3]))

	pkg.LogDebug(pkg.ComponentHAL, "readPacket header", "type", msgType, "length", length)

	if msgType != msgData {
		return 0, pkg.ErrProtocol
	}

	if length == 0 {
		return 0, nil // Zero-length packet
	}

	if length > len(buf) {
		return 0, pkg.ErrBufferTooSmall
	}

	// Read data
	return h.readWithContext(ctx, f, buf[:length])
}

// Compile-time interface check
var _ hal.DeviceHAL = (*HAL)(nil)
