package fifo

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ardnew/softusb/host/hal"
	"github.com/ardnew/softusb/pkg"
)

// Message types for FIFO protocol.
const (
	msgSetup   = 0x01 // SETUP packet
	msgData    = 0x02 // DATA packet
	msgAck     = 0x03 // ACK response
	msgNak     = 0x04 // NAK response
	msgStall   = 0x05 // STALL response
	msgReset   = 0x12 // Port reset
	msgAddress = 0x13 // Set address
)

// Connection signal bytes (one-way signaling from device).
const (
	sigConnect    = 0x01 // Device connected
	sigDisconnect = 0x00 // Device disconnected
)

// Buffer sizes.
const (
	maxPacketSize   = 512  // Maximum USB packet size
	maxMessageSize  = 1024 // Maximum FIFO message size
	headerSize      = 3    // Message header size (type + length)
	setupPacketSize = 8    // USB SETUP packet size
)

// Timing constants.
const (
	pollInterval = 50 * time.Millisecond // Directory polling interval
)

// FIFO file names (inside each device subdirectory).
const (
	fifoHostToDevice = "host_to_device"
	fifoDeviceToHost = "device_to_host"
	fifoInterrupts   = "interrupts"
	fifoConnection   = "connection"
)

// Errors.
var (
	ErrNotConnected = errors.New("device not connected")
	ErrFIFOCreate   = errors.New("failed to create FIFO")
	ErrFIFOOpen     = errors.New("failed to open FIFO")
	ErrNoDevice     = errors.New("no device available")
)

// deviceConn represents a connected device.
type deviceConn struct {
	dir          string   // Device subdirectory path
	hostToDevice *os.File // Host writes to device (control transfers)
	deviceToHost *os.File // Device writes to host (control responses)
	interrupts   *os.File // Interrupt IN transfers
	connection   *os.File // Connection signaling
	// Endpoint FIFOs for data transfers (indices 0-14 = endpoints 1-15)
	epIn  [MaxEndpoints]*os.File // Host reads from device (IN endpoints)
	epOut [MaxEndpoints]*os.File // Host writes to device (OUT endpoints)
	speed hal.Speed
	port  int
}

// MaxEndpoints is the maximum number of data endpoints (1-15).
const MaxEndpoints = 15

// HostHAL implements the hal.HostHAL interface using named pipes.
// It monitors a bus directory for device subdirectories and manages
// connections to multiple devices.
type HostHAL struct {
	busDir string // Root bus directory

	// Active device connection (single device for now)
	device   *deviceConn
	deviceMu sync.RWMutex

	// Internal buffers (zero-allocation pattern)
	txBuf [maxMessageSize]byte
	rxBuf [maxMessageSize]byte

	// Channels for connection events
	connectCh    chan *deviceConn
	disconnectCh chan int

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewHostHAL creates a new FIFO-based host HAL.
// The busDir parameter specifies the root directory where device subdirectories
// will appear. Devices create their own subdirectories (e.g., device-{uuid}/).
func NewHostHAL(busDir string) *HostHAL {
	return &HostHAL{
		busDir:       busDir,
		connectCh:    make(chan *deviceConn, 8),
		disconnectCh: make(chan int, 8),
	}
}

// Init initializes the host HAL.
func (h *HostHAL) Init(ctx context.Context) error {
	h.ctx, h.cancel = context.WithCancel(ctx)

	// Ensure bus directory exists
	if err := os.MkdirAll(h.busDir, 0o755); err != nil {
		return fmt.Errorf("%w: %v", ErrFIFOCreate, err)
	}

	pkg.LogInfo(pkg.ComponentHAL, "host FIFO HAL initialized", "busDir", h.busDir)
	return nil
}

// Start starts the host HAL and begins monitoring for devices.
func (h *HostHAL) Start() error {
	// Start directory polling goroutine
	h.wg.Add(1)
	go h.pollDeviceDirectories()

	pkg.LogInfo(pkg.ComponentHAL, "host FIFO HAL started")
	return nil
}

// Stop stops the host HAL.
func (h *HostHAL) Stop() error {
	if h.cancel != nil {
		h.cancel()
	}

	// Wait for goroutines to finish
	h.wg.Wait()

	// Close any active device connection
	h.deviceMu.Lock()
	if h.device != nil {
		h.closeDevice(h.device)
		h.device = nil
	}
	h.deviceMu.Unlock()

	pkg.LogInfo(pkg.ComponentHAL, "host FIFO HAL stopped")
	return nil
}

// NumPorts returns the number of root hub ports (simulated as 1).
func (h *HostHAL) NumPorts() int {
	return 1
}

// GetPortStatus returns the status of a port.
func (h *HostHAL) GetPortStatus(port int) (hal.PortStatus, error) {
	h.deviceMu.RLock()
	defer h.deviceMu.RUnlock()

	if port != 1 {
		return hal.PortStatus{}, pkg.ErrInvalidEndpoint
	}

	connected := h.device != nil
	speed := hal.SpeedUnknown
	if connected {
		speed = h.device.speed
	}

	return hal.PortStatus{
		Connected: connected,
		Enabled:   connected,
		PowerOn:   true,
		Speed:     speed,
	}, nil
}

// PortSpeed returns the speed of a connected device.
func (h *HostHAL) PortSpeed(port int) hal.Speed {
	h.deviceMu.RLock()
	defer h.deviceMu.RUnlock()

	if port != 1 || h.device == nil {
		return hal.SpeedUnknown
	}
	return h.device.speed
}

// ResetPort initiates a port reset.
func (h *HostHAL) ResetPort(port int) error {
	if port != 1 {
		return pkg.ErrInvalidEndpoint
	}

	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()

	if h.device == nil {
		return ErrNotConnected
	}

	// Send reset message to device
	h.txBuf[0] = msgReset
	h.txBuf[1] = 0
	h.txBuf[2] = 0
	_, err := h.device.hostToDevice.Write(h.txBuf[:headerSize])
	if err != nil {
		return err
	}

	// Wait for acknowledgment with timeout
	h.device.deviceToHost.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := h.device.deviceToHost.Read(h.rxBuf[:])
	h.device.deviceToHost.SetReadDeadline(time.Time{})
	if err != nil {
		return err
	}
	if n < headerSize || h.rxBuf[0] != msgAck {
		return pkg.ErrProtocol
	}

	pkg.LogDebug(pkg.ComponentHAL, "port reset complete", "port", port)
	return nil
}

// EnablePort enables or disables a port.
func (h *HostHAL) EnablePort(port int, enable bool) error {
	if port != 1 {
		return pkg.ErrInvalidEndpoint
	}
	// FIFO HAL doesn't need explicit port enable
	return nil
}

// ControlTransfer performs a control transfer.
func (h *HostHAL) ControlTransfer(ctx context.Context, addr hal.DeviceAddress, setup *hal.SetupPacket, data []byte) (int, error) {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()

	if h.device == nil {
		return 0, ErrNotConnected
	}

	isIn := (setup.RequestType & 0x80) != 0

	// Build message: header + address + setup packet (+ data for OUT transfers)
	h.txBuf[0] = msgSetup
	h.txBuf[3] = byte(addr)
	setup.MarshalTo(h.txBuf[4:12])

	// Calculate payload length: address (1) + setup (8) + data (for OUT only)
	payloadLen := 1 + setupPacketSize
	if !isIn && len(data) > 0 {
		copy(h.txBuf[12:], data)
		payloadLen += len(data)
	}
	binary.LittleEndian.PutUint16(h.txBuf[1:3], uint16(payloadLen))

	// Write to FIFO
	msgLen := headerSize + payloadLen
	_, err := h.device.hostToDevice.Write(h.txBuf[:msgLen])
	if err != nil {
		return 0, err
	}

	// Read response with timeout
	h.device.deviceToHost.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := h.device.deviceToHost.Read(h.rxBuf[:])
	h.device.deviceToHost.SetReadDeadline(time.Time{})
	if err != nil {
		return 0, err
	}

	if n < headerSize {
		return 0, pkg.ErrProtocol
	}

	switch h.rxBuf[0] {
	case msgData:
		// Data phase response
		respLen := int(binary.LittleEndian.Uint16(h.rxBuf[1:3]))
		if respLen > 0 && isIn && len(data) > 0 {
			copied := copy(data, h.rxBuf[headerSize:headerSize+respLen])
			return copied, nil
		}
		return respLen, nil

	case msgAck:
		// Status phase only (no data)
		return 0, nil

	case msgNak:
		return 0, pkg.ErrNAK

	case msgStall:
		return 0, pkg.ErrStall

	default:
		return 0, pkg.ErrProtocol
	}
}

// BulkTransfer performs a bulk transfer.
func (h *HostHAL) BulkTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	return h.dataTransfer(ctx, addr, endpoint, data)
}

// InterruptTransfer performs an interrupt transfer.
func (h *HostHAL) InterruptTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()

	// Use the same endpoint FIFOs as bulk transfers
	// (the device writes interrupt data to epN_in and reads from epN_out)
	return h.dataTransferLocked(addr, endpoint, data)
}

// IsochronousTransfer performs an isochronous transfer.
func (h *HostHAL) IsochronousTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	// FIFO HAL doesn't fully support isochronous - treat as bulk
	return h.dataTransfer(ctx, addr, endpoint, data)
}

// SetDeviceAddress sets the device address after reset.
func (h *HostHAL) SetDeviceAddress(ctx context.Context, newAddr hal.DeviceAddress) error {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()

	if h.device == nil {
		return ErrNotConnected
	}

	// Send address assignment message
	h.txBuf[0] = msgAddress
	binary.LittleEndian.PutUint16(h.txBuf[1:3], 1)
	h.txBuf[3] = byte(newAddr)

	_, err := h.device.hostToDevice.Write(h.txBuf[:headerSize+1])
	if err != nil {
		return err
	}

	// Wait for acknowledgment
	h.device.deviceToHost.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := h.device.deviceToHost.Read(h.rxBuf[:])
	h.device.deviceToHost.SetReadDeadline(time.Time{})
	if err != nil {
		return err
	}
	if n < headerSize || h.rxBuf[0] != msgAck {
		return pkg.ErrProtocol
	}

	pkg.LogDebug(pkg.ComponentHAL, "device address set", "address", newAddr)
	return nil
}

// WaitForConnection waits for a device to connect.
func (h *HostHAL) WaitForConnection(ctx context.Context) (int, error) {
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-h.ctx.Done():
			return 0, pkg.ErrCancelled
		case dev := <-h.connectCh:
			// Set as active device
			h.deviceMu.Lock()
			if h.device != nil {
				// Close previous device
				h.closeDevice(h.device)
			}
			h.device = dev
			h.deviceMu.Unlock()

			pkg.LogInfo(pkg.ComponentHAL, "device connected", "port", dev.port, "speed", dev.speed, "dir", dev.dir)
			return dev.port, nil
		}
	}
}

// WaitForDisconnection waits for a device to disconnect.
func (h *HostHAL) WaitForDisconnection(ctx context.Context) (int, error) {
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-h.ctx.Done():
			return 0, pkg.ErrCancelled
		case port := <-h.disconnectCh:
			h.deviceMu.Lock()
			if h.device != nil && h.device.port == port {
				h.closeDevice(h.device)
				h.device = nil
			}
			h.deviceMu.Unlock()

			pkg.LogInfo(pkg.ComponentHAL, "device disconnected", "port", port)
			return port, nil
		}
	}
}

// pollDeviceDirectories polls the bus directory for new device subdirectories.
func (h *HostHAL) pollDeviceDirectories() {
	defer h.wg.Done()

	knownDirs := make(map[string]bool)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			entries, err := os.ReadDir(h.busDir)
			if err != nil {
				continue
			}

			// Look for new device directories
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				// Device directories start with "device-"
				if !strings.HasPrefix(entry.Name(), "device-") {
					continue
				}

				dirPath := filepath.Join(h.busDir, entry.Name())
				if knownDirs[dirPath] {
					continue
				}

				// Check if connection FIFO exists
				connPath := filepath.Join(dirPath, fifoConnection)
				if _, err := os.Stat(connPath); os.IsNotExist(err) {
					continue
				}

				// Mark as known and try to connect
				knownDirs[dirPath] = true

				// Start a goroutine to handle this device
				h.wg.Add(1)
				go h.handleDeviceDirectory(dirPath)
			}

			// Clean up removed directories from knownDirs
			for dir := range knownDirs {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					delete(knownDirs, dir)
				}
			}
		}
	}
}

// handleDeviceDirectory monitors a device directory for connection signals.
func (h *HostHAL) handleDeviceDirectory(dirPath string) {
	defer h.wg.Done()

	pkg.LogDebug(pkg.ComponentHAL, "monitoring device directory", "dir", dirPath)

	// Open connection FIFO with O_RDWR to avoid blocking
	connPath := filepath.Join(dirPath, fifoConnection)
	connFile, err := os.OpenFile(connPath, os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		pkg.LogWarn(pkg.ComponentHAL, "failed to open connection FIFO", "path", connPath, "error", err)
		return
	}
	defer connFile.Close()

	var buf [1]byte
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		// Set read timeout
		connFile.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := connFile.Read(buf[:])
		if err != nil {
			if os.IsTimeout(err) {
				continue
			}
			if errors.Is(err, io.EOF) {
				// Device directory was removed
				return
			}
			continue
		}

		if n == 0 {
			continue
		}

		if buf[0] == sigConnect {
			// Device connected - open all FIFOs
			dev, err := h.openDeviceFIFOs(dirPath)
			if err != nil {
				pkg.LogWarn(pkg.ComponentHAL, "failed to open device FIFOs", "dir", dirPath, "error", err)
				continue
			}

			// Send to connect channel
			select {
			case h.connectCh <- dev:
			case <-h.ctx.Done():
				h.closeDevice(dev)
				return
			}

			// Monitor for disconnect
			h.monitorDeviceConnection(dev, connFile)
		}
	}
}

// openDeviceFIFOs opens all FIFOs for a device.
func (h *HostHAL) openDeviceFIFOs(dirPath string) (*deviceConn, error) {
	dev := &deviceConn{
		dir:   dirPath,
		speed: hal.SpeedFull,
		port:  1,
	}

	var err error

	// Open host_to_device for writing (non-blocking initially, then blocking)
	dev.hostToDevice, err = os.OpenFile(
		filepath.Join(dirPath, fifoHostToDevice),
		os.O_WRONLY|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("open host_to_device: %w", err)
	}

	// Open device_to_host for reading
	dev.deviceToHost, err = os.OpenFile(
		filepath.Join(dirPath, fifoDeviceToHost),
		os.O_RDONLY|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		dev.hostToDevice.Close()
		return nil, fmt.Errorf("open device_to_host: %w", err)
	}

	// Open interrupts for reading
	dev.interrupts, err = os.OpenFile(
		filepath.Join(dirPath, fifoInterrupts),
		os.O_RDONLY|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		dev.hostToDevice.Close()
		dev.deviceToHost.Close()
		return nil, fmt.Errorf("open interrupts: %w", err)
	}

	// Open endpoint FIFOs (1-15)
	for i := 1; i <= MaxEndpoints; i++ {
		idx := i - 1

		// Open epN_in for reading (device writes, host reads)
		dev.epIn[idx], err = os.OpenFile(
			filepath.Join(dirPath, fmt.Sprintf("ep%d_in", i)),
			os.O_RDONLY|syscall.O_NONBLOCK,
			0,
		)
		if err != nil {
			h.closeDeviceFIFOs(dev)
			return nil, fmt.Errorf("open ep%d_in: %w", i, err)
		}

		// Open epN_out for writing (host writes, device reads)
		dev.epOut[idx], err = os.OpenFile(
			filepath.Join(dirPath, fmt.Sprintf("ep%d_out", i)),
			os.O_WRONLY|syscall.O_NONBLOCK,
			0,
		)
		if err != nil {
			h.closeDeviceFIFOs(dev)
			return nil, fmt.Errorf("open ep%d_out: %w", i, err)
		}
	}

	return dev, nil
}

// closeDeviceFIFOs closes all FIFOs for a device.
func (h *HostHAL) closeDeviceFIFOs(dev *deviceConn) {
	if dev.hostToDevice != nil {
		dev.hostToDevice.Close()
	}
	if dev.deviceToHost != nil {
		dev.deviceToHost.Close()
	}
	if dev.interrupts != nil {
		dev.interrupts.Close()
	}
	for i := 0; i < MaxEndpoints; i++ {
		if dev.epIn[i] != nil {
			dev.epIn[i].Close()
		}
		if dev.epOut[i] != nil {
			dev.epOut[i].Close()
		}
	}
}

// monitorDeviceConnection monitors for device disconnection.
func (h *HostHAL) monitorDeviceConnection(dev *deviceConn, connFile *os.File) {
	var buf [1]byte
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		connFile.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := connFile.Read(buf[:])
		if err != nil {
			if os.IsTimeout(err) {
				continue
			}
			// Connection lost
			select {
			case h.disconnectCh <- dev.port:
			case <-h.ctx.Done():
			}
			return
		}

		if n > 0 && buf[0] == sigDisconnect {
			select {
			case h.disconnectCh <- dev.port:
			case <-h.ctx.Done():
			}
			return
		}
	}
}

// closeDevice closes all FIFOs for a device.
func (h *HostHAL) closeDevice(dev *deviceConn) {
	h.closeDeviceFIFOs(dev)
	if dev.connection != nil {
		dev.connection.Close()
	}
}

// dataTransfer performs a bulk/interrupt data transfer.
func (h *HostHAL) dataTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	h.deviceMu.Lock()
	defer h.deviceMu.Unlock()

	return h.dataTransferLocked(addr, endpoint, data)
}

// dataTransferLocked performs a data transfer (caller must hold lock).
func (h *HostHAL) dataTransferLocked(addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	if h.device == nil {
		return 0, ErrNotConnected
	}

	epNum := int(endpoint & 0x0F)
	if epNum == 0 || epNum > MaxEndpoints {
		return 0, pkg.ErrInvalidEndpoint
	}
	idx := epNum - 1

	isIn := (endpoint & 0x80) != 0

	if isIn {
		// IN transfer - read from device's IN endpoint FIFO
		// The device writes DATA messages to epN_in
		epFile := h.device.epIn[idx]
		if epFile == nil {
			return 0, pkg.ErrInvalidEndpoint
		}

		// Read with timeout
		epFile.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := epFile.Read(h.rxBuf[:])
		epFile.SetReadDeadline(time.Time{})
		if err != nil {
			return 0, err
		}

		// Parse DATA message: [type, len_lo, len_hi, data...]
		if n < headerSize {
			return 0, pkg.ErrProtocol
		}
		if h.rxBuf[0] != msgData {
			return 0, pkg.ErrProtocol
		}
		respLen := int(binary.LittleEndian.Uint16(h.rxBuf[1:3]))
		if respLen > 0 {
			copied := copy(data, h.rxBuf[headerSize:headerSize+respLen])
			return copied, nil
		}
		return 0, nil
	}

	// OUT transfer - write to device's OUT endpoint FIFO
	// The device reads DATA messages from epN_out
	epFile := h.device.epOut[idx]
	if epFile == nil {
		return 0, pkg.ErrInvalidEndpoint
	}

	// Build DATA message: [type, len_lo, len_hi, data...]
	h.txBuf[0] = msgData
	binary.LittleEndian.PutUint16(h.txBuf[1:3], uint16(len(data)))
	copy(h.txBuf[headerSize:], data)

	// Write message
	total := headerSize + len(data)
	_, err := epFile.Write(h.txBuf[:total])
	if err != nil {
		return 0, err
	}

	// For OUT transfers, we expect an ACK from the device on the main FIFO
	// But in this simple FIFO model, we assume success if write completed
	return len(data), nil
}

// Ensure HostHAL implements hal.HostHAL.
var _ hal.HostHAL = (*HostHAL)(nil)
