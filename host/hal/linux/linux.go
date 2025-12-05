//go:build linux

package linux

import (
	"context"
	"fmt"
	"sync"

	"github.com/ardnew/softusb/host/hal"
	"github.com/ardnew/softusb/pkg"
)

// =============================================================================
// HostHAL Implementation
// =============================================================================

// HostHAL implements the hal.HostHAL interface for Linux using usbfs.
type HostHAL struct {
	// Device pool for tracking connected devices
	devices devicePool

	// Poller for async I/O
	poller *poller

	// Hotplug monitor
	hotplug *hotplugMonitor

	// Channels for connection events
	connectCh    chan int // Port number of connected device
	disconnectCh chan int // Port number of disconnected device

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// State
	running bool
	mu      sync.Mutex

	// Transfer timeout in milliseconds
	transferTimeout uint32
}

// NewHostHAL creates a new Linux host HAL.
func NewHostHAL() *HostHAL {
	return &HostHAL{
		connectCh:       make(chan int, 16),
		disconnectCh:    make(chan int, 16),
		transferTimeout: 5000, // 5 second default
	}
}

// SetTransferTimeout sets the timeout for USB transfers in milliseconds.
func (h *HostHAL) SetTransferTimeout(ms uint32) {
	h.transferTimeout = ms
}

// =============================================================================
// Lifecycle Methods
// =============================================================================

// Init initializes the USB host controller hardware.
func (h *HostHAL) Init(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return pkg.ErrBusy
	}

	// Initialize device pool
	h.devices.init()

	// Create poller
	var err error
	h.poller, err = newPoller()
	if err != nil {
		return err
	}

	// Create hotplug monitor
	h.hotplug, err = newHotplugMonitor()
	if err != nil {
		h.poller.close()
		return err
	}

	// Add hotplug socket to poller
	if err := h.poller.addFD(h.hotplug.socketFD(), EPOLLIN, h.onHotplugEvent); err != nil {
		h.hotplug.close()
		h.poller.close()
		return err
	}

	h.ctx, h.cancel = context.WithCancel(ctx)

	pkg.LogDebug(pkg.ComponentHAL, "Linux host HAL initialized")
	return nil
}

// Start enables the host controller and applies power to ports.
func (h *HostHAL) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return pkg.ErrBusy
	}

	h.running = true

	// Start polling goroutine
	h.wg.Add(1)
	go h.pollLoop()

	// Start hotplug event processing goroutine
	h.wg.Add(1)
	go h.hotplugLoop()

	// Scan for existing devices
	h.wg.Add(1)
	go h.initialScan()

	pkg.LogDebug(pkg.ComponentHAL, "Linux host HAL started")
	return nil
}

// Stop disables the host controller and removes power from ports.
func (h *HostHAL) Stop() error {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return nil
	}
	h.running = false
	h.mu.Unlock()

	// Signal cancellation
	if h.cancel != nil {
		h.cancel()
	}

	// Wake the poller
	if h.poller != nil {
		h.poller.wake()
	}

	// Wait for goroutines
	h.wg.Wait()

	pkg.LogDebug(pkg.ComponentHAL, "Linux host HAL stopped")
	return nil
}

// Close releases all resources associated with the HAL.
func (h *HostHAL) Close() error {
	h.Stop()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all device connections
	for i := 0; i < MaxDevices; i++ {
		if h.devices.slots[i].conn != nil {
			h.devices.free(i)
		}
	}

	// Close hotplug monitor
	if h.hotplug != nil {
		h.hotplug.close()
		h.hotplug = nil
	}

	// Close poller
	if h.poller != nil {
		h.poller.close()
		h.poller = nil
	}

	pkg.LogDebug(pkg.ComponentHAL, "Linux host HAL closed")
	return nil
}

// =============================================================================
// Port Operations
// =============================================================================

// NumPorts returns the number of root hub ports.
// For Linux usbfs, we report a virtual single port per device.
func (h *HostHAL) NumPorts() int {
	return MaxDevices
}

// GetPortStatus returns the status of a port.
func (h *HostHAL) GetPortStatus(port int) (hal.PortStatus, error) {
	if port < 1 || port > MaxDevices {
		return hal.PortStatus{}, pkg.ErrInvalidEndpoint
	}

	conn := h.devices.findByPort(port)
	if conn == nil {
		return hal.PortStatus{
			PowerOn: true,
		}, nil
	}

	return hal.PortStatus{
		Connected: !conn.isDisconnected(),
		Enabled:   true,
		PowerOn:   true,
		Speed:     conn.info.speed,
	}, nil
}

// PortSpeed returns the connection speed of a device on the given port.
func (h *HostHAL) PortSpeed(port int) hal.Speed {
	if port < 1 || port > MaxDevices {
		return hal.SpeedUnknown
	}

	conn := h.devices.findByPort(port)
	if conn == nil {
		return hal.SpeedUnknown
	}

	return conn.info.speed
}

// ResetPort initiates a port reset.
func (h *HostHAL) ResetPort(port int) error {
	if port < 1 || port > MaxDevices {
		return pkg.ErrInvalidEndpoint
	}

	conn := h.devices.findByPort(port)
	if conn == nil {
		return pkg.ErrNoDevice
	}

	return resetDevice(conn.fd)
}

// EnablePort enables or disables a port.
func (h *HostHAL) EnablePort(port int, enable bool) error {
	// Linux usbfs doesn't require explicit port enable
	return nil
}

// =============================================================================
// Control Transfers
// =============================================================================

// ControlTransfer performs a control transfer to a device.
func (h *HostHAL) ControlTransfer(ctx context.Context, addr hal.DeviceAddress, setup *hal.SetupPacket, data []byte) (int, error) {
	conn := h.devices.findByAddress(addr)
	if conn == nil {
		return 0, pkg.ErrNoDevice
	}

	if conn.isDisconnected() {
		return 0, pkg.ErrNoDevice
	}

	n, err := doControlTransfer(conn.fd,
		setup.RequestType,
		setup.Request,
		setup.Value,
		setup.Index,
		data,
		h.transferTimeout,
	)

	if isNoDevice(err) {
		conn.handleENODEV()
		return 0, pkg.ErrNoDevice
	}

	return n, err
}

// =============================================================================
// Data Transfers
// =============================================================================

// BulkTransfer performs a bulk transfer to/from an endpoint.
func (h *HostHAL) BulkTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	conn := h.devices.findByAddress(addr)
	if conn == nil {
		return 0, pkg.ErrNoDevice
	}

	if conn.isDisconnected() {
		return 0, pkg.ErrNoDevice
	}

	n, err := conn.submitBulkURB(endpoint, data, h.transferTimeout)

	if isNoDevice(err) {
		conn.handleENODEV()
		return 0, pkg.ErrNoDevice
	}

	if isPipe(err) {
		return 0, pkg.ErrStall
	}

	return n, err
}

// InterruptTransfer performs an interrupt transfer to/from an endpoint.
func (h *HostHAL) InterruptTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	// For interrupt transfers, we use the same bulk transfer mechanism
	// The kernel handles the polling interval based on the endpoint descriptor
	return h.BulkTransfer(ctx, addr, endpoint, data)
}

// IsochronousTransfer performs an isochronous transfer.
func (h *HostHAL) IsochronousTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	// Isochronous transfers are not fully implemented
	return 0, pkg.ErrNotSupported
}

// =============================================================================
// Device Management
// =============================================================================

// SetDeviceAddress assigns an address to a device at address 0.
func (h *HostHAL) SetDeviceAddress(ctx context.Context, newAddr hal.DeviceAddress) error {
	// In Linux usbfs, address assignment is handled by the kernel during enumeration
	// We just need to update our internal tracking

	// Find the device at address 0 (most recently connected)
	conn := h.devices.findByAddress(0)
	if conn == nil {
		// Try to find any device without an address assigned
		for i := 0; i < MaxDevices; i++ {
			if h.devices.slots[i].conn != nil && h.devices.slots[i].conn.address == 0 {
				conn = h.devices.slots[i].conn
				break
			}
		}
	}

	if conn == nil {
		return pkg.ErrNoDevice
	}

	conn.address = newAddr
	pkg.LogDebug(pkg.ComponentHAL, "device address set", "address", newAddr)
	return nil
}

// ClaimInterface claims exclusive access to an interface on a device.
func (h *HostHAL) ClaimInterface(addr hal.DeviceAddress, iface uint8) error {
	conn := h.devices.findByAddress(addr)
	if conn == nil {
		return pkg.ErrNoDevice
	}

	return conn.ensureInterfaceClaimed(iface)
}

// ReleaseInterface releases a previously claimed interface.
func (h *HostHAL) ReleaseInterface(addr hal.DeviceAddress, iface uint8) error {
	conn := h.devices.findByAddress(addr)
	if conn == nil {
		return pkg.ErrNoDevice
	}

	return conn.releaseInterfaceClaim(iface)
}

// =============================================================================
// Connection Events
// =============================================================================

// WaitForConnection blocks until a device connects or context is cancelled.
func (h *HostHAL) WaitForConnection(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-h.ctx.Done():
		return 0, pkg.ErrCancelled
	case port := <-h.connectCh:
		return port, nil
	}
}

// WaitForDisconnection blocks until a device disconnects or context is cancelled.
func (h *HostHAL) WaitForDisconnection(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-h.ctx.Done():
		return 0, pkg.ErrCancelled
	case port := <-h.disconnectCh:
		return port, nil
	}
}

// =============================================================================
// Internal Methods
// =============================================================================

// pollLoop runs the epoll wait loop.
func (h *HostHAL) pollLoop() {
	defer h.wg.Done()

	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		// Poll with timeout to allow checking context
		_, err := h.poller.pollOnce(100) // 100ms timeout
		if err != nil {
			if isAgain(err) {
				continue
			}
			pkg.LogWarn(pkg.ComponentHAL, "poll error", "error", err)
		}
	}
}

// hotplugLoop processes hotplug events.
func (h *HostHAL) hotplugLoop() {
	defer h.wg.Done()

	for {
		select {
		case <-h.ctx.Done():
			return
		case info := <-h.hotplug.addChannel():
			h.handleDeviceAdd(info)
		case info := <-h.hotplug.removeChannel():
			h.handleDeviceRemove(info)
		}
	}
}

// initialScan scans for existing devices at startup.
func (h *HostHAL) initialScan() {
	defer h.wg.Done()

	devices, err := scanUSBDevices()
	if err != nil {
		pkg.LogWarn(pkg.ComponentHAL, "initial scan failed", "error", err)
		return
	}

	for _, info := range devices {
		h.handleDeviceAdd(info)
	}
}

// onHotplugEvent is called when the hotplug socket has data.
func (h *HostHAL) onHotplugEvent(events uint32) {
	if events&EPOLLIN == 0 {
		return
	}

	// Process all available events
	for {
		processed, err := h.hotplug.processEvent()
		if err != nil || !processed {
			break
		}
	}
}

// handleDeviceAdd handles a device add event.
func (h *HostHAL) handleDeviceAdd(info usbDeviceInfo) {
	// Check if device is already tracked (avoid duplicates from initial scan + hotplug)
	h.devices.mu.Lock()
	for i := 0; i < MaxDevices; i++ {
		conn := h.devices.slots[i].conn
		if conn != nil && conn.info.busNum == info.busNum && conn.info.devNum == info.devNum {
			h.devices.mu.Unlock()
			pkg.LogDebug(pkg.ComponentHAL, "device already tracked, skipping",
				"bus", info.busNum,
				"dev", info.devNum)
			return
		}
	}
	h.devices.mu.Unlock()

	// Allocate a device slot - returns slot index
	slotIdx := h.devices.alloc(0) // Port will be set below
	if slotIdx < 0 {
		pkg.LogWarn(pkg.ComponentHAL, "no device slots available")
		return
	}

	// The port number is the slot index + 1 (1-indexed)
	port := slotIdx + 1

	// Update slot with actual port number
	h.devices.slots[slotIdx].port = port

	// Open the device
	conn, err := newDeviceConn(info)
	if err != nil {
		pkg.LogWarn(pkg.ComponentHAL, "failed to open device", "error", err, "path", info.devfsPath)
		h.devices.free(slotIdx)
		return
	}

	// Set the device address to match the port number returned by WaitForConnection().
	// This ensures findByAddress() can locate the device when ControlTransfer() is called.
	conn.address = hal.DeviceAddress(port)

	// Store connection
	h.devices.set(slotIdx, conn)

	// Add device fd to poller for URB completion
	if err := h.poller.addFD(conn.fd, EPOLLIN, func(events uint32) {
		h.onDeviceEvent(conn, events)
	}); err != nil {
		pkg.LogWarn(pkg.ComponentHAL, "failed to add device to poller", "error", err)
	}

	pkg.LogDebug(pkg.ComponentHAL, "device connected",
		"port", port,
		"bus", info.busNum,
		"dev", info.devNum,
		"vid", fmt.Sprintf("0x%04x", info.vendorID),
		"pid", fmt.Sprintf("0x%04x", info.productID),
	)

	// Signal connection
	select {
	case h.connectCh <- port:
	default:
	}
}

// handleDeviceRemove handles a device remove event.
func (h *HostHAL) handleDeviceRemove(info usbDeviceInfo) {
	// Find and free the device
	h.devices.mu.Lock()
	for i := 0; i < MaxDevices; i++ {
		conn := h.devices.slots[i].conn
		if conn != nil && conn.info.busNum == info.busNum && conn.info.devNum == info.devNum {
			port := h.devices.slots[i].port

			// Remove from poller
			h.poller.delFD(conn.fd)

			// Mark disconnected and close
			conn.markDisconnected()
			h.devices.mu.Unlock()
			h.devices.free(i)

			pkg.LogDebug(pkg.ComponentHAL, "device disconnected", "port", port)

			// Signal disconnection
			select {
			case h.disconnectCh <- port:
			default:
			}
			return
		}
	}
	h.devices.mu.Unlock()
}

// onDeviceEvent is called when a device fd has events.
func (h *HostHAL) onDeviceEvent(conn *deviceConn, events uint32) {
	if events&EPOLLERR != 0 || events&EPOLLHUP != 0 {
		// Device error or hangup - likely disconnected
		conn.handleENODEV()
		return
	}

	if events&EPOLLIN != 0 {
		// URB completed - try to reap it
		for {
			u, err := conn.reapAsyncURB()
			if err != nil {
				if isAgain(err) {
					break
				}
				if isNoDevice(err) {
					conn.handleENODEV()
				}
				break
			}
			if u == nil {
				break
			}
			// URB completed - notify waiters
			// This is handled by the endpoint slot's complete channel
		}
	}
}

// =============================================================================
// Interface Compliance
// =============================================================================

// Ensure HostHAL implements hal.HostHAL.
var _ hal.HostHAL = (*HostHAL)(nil)
