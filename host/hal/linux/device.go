//go:build linux

package linux

import (
	"sync"
	"syscall"

	"github.com/ardnew/softusb/host/hal"
	"github.com/ardnew/softusb/pkg"
)

// =============================================================================
// URB Slot Management
// =============================================================================

// urbSlot represents a slot in the URB pool for an endpoint.
type urbSlot struct {
	urb      urb                 // The URB structure
	buffer   [URBBufferSize]byte // Data buffer for the URB
	inUse    bool                // Whether this slot is in use
	next     int8                // Next free slot index (-1 if none)
	complete chan error          // Channel for completion notification
}

// endpointState tracks the state of an endpoint.
type endpointState struct {
	slots    [MaxURBsPerEndpoint]urbSlot // URB slots
	freeHead int8                        // Head of free list (-1 if empty)
	pending  int                         // Number of pending URBs
	mu       sync.Mutex                  // Protects this endpoint state
}

// initEndpointState initializes endpoint state with all slots free.
func (e *endpointState) init() {
	e.freeHead = 0
	e.pending = 0
	for i := 0; i < MaxURBsPerEndpoint-1; i++ {
		e.slots[i].next = int8(i + 1)
		e.slots[i].inUse = false
		e.slots[i].complete = make(chan error, 1)
	}
	e.slots[MaxURBsPerEndpoint-1].next = -1
	e.slots[MaxURBsPerEndpoint-1].inUse = false
	e.slots[MaxURBsPerEndpoint-1].complete = make(chan error, 1)
}

// allocSlot allocates a URB slot from the pool.
// Returns the slot index, or -1 if no slots are available.
func (e *endpointState) allocSlot() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.freeHead < 0 {
		return -1
	}

	idx := int(e.freeHead)
	slot := &e.slots[idx]
	e.freeHead = slot.next
	slot.inUse = true
	slot.next = -1
	e.pending++

	return idx
}

// freeSlot returns a URB slot to the pool.
func (e *endpointState) freeSlot(idx int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if idx < 0 || idx >= MaxURBsPerEndpoint {
		return
	}

	slot := &e.slots[idx]
	if !slot.inUse {
		return
	}

	slot.inUse = false
	slot.next = e.freeHead
	e.freeHead = int8(idx)
	e.pending--
}

// getSlot returns the slot at the given index.
func (e *endpointState) getSlot(idx int) *urbSlot {
	if idx < 0 || idx >= MaxURBsPerEndpoint {
		return nil
	}
	return &e.slots[idx]
}

// =============================================================================
// Device Connection
// =============================================================================

// deviceConn represents a connection to a USB device.
type deviceConn struct {
	fd      int               // File descriptor for /dev/bus/usb/BBB/DDD
	info    usbDeviceInfo     // Device information
	address hal.DeviceAddress // Assigned USB address

	// Endpoint state with URB pools
	endpoints [MaxEndpointsPerDevice]endpointState

	// Interface claiming
	claimedMask uint16     // Bitmask of claimed interfaces
	claimMu     sync.Mutex // Protects claimedMask

	// State
	disconnected bool         // Set when device is disconnected
	mu           sync.RWMutex // Protects disconnected flag
}

// newDeviceConn creates a new device connection.
func newDeviceConn(info usbDeviceInfo) (*deviceConn, error) {
	// Open the device file
	fd, err := openDevice(info.devfsPath)
	if err != nil {
		return nil, err
	}

	conn := &deviceConn{
		fd:   fd,
		info: info,
	}

	// Initialize all endpoint states
	for i := range conn.endpoints {
		conn.endpoints[i].init()
	}

	return conn, nil
}

// close closes the device connection and releases all resources.
func (d *deviceConn) close() error {
	d.mu.Lock()
	d.disconnected = true
	d.mu.Unlock()

	// Release all claimed interfaces
	d.claimMu.Lock()
	for i := 0; i < MaxInterfacesPerDevice; i++ {
		if d.claimedMask&(1<<i) != 0 {
			releaseInterface(d.fd, uint8(i))
		}
	}
	d.claimedMask = 0
	d.claimMu.Unlock()

	// Discard any pending URBs
	d.discardAllURBs()

	return closeDevice(d.fd)
}

// isDisconnected returns true if the device has been disconnected.
func (d *deviceConn) isDisconnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.disconnected
}

// markDisconnected marks the device as disconnected.
func (d *deviceConn) markDisconnected() {
	d.mu.Lock()
	d.disconnected = true
	d.mu.Unlock()
}

// =============================================================================
// Interface Claiming (Lazy)
// =============================================================================

// ensureInterfaceClaimed ensures an interface is claimed, using lazy claiming.
func (d *deviceConn) ensureInterfaceClaimed(iface uint8) error {
	if iface >= MaxInterfacesPerDevice {
		return pkg.ErrInvalidEndpoint
	}

	d.claimMu.Lock()
	defer d.claimMu.Unlock()

	mask := uint16(1) << iface
	if d.claimedMask&mask != 0 {
		// Already claimed
		return nil
	}

	// Disconnect any kernel driver first
	if err := disconnectDriver(d.fd, iface); err != nil {
		// ENODATA means no driver was attached, which is fine
		_ = isNoData(err)
	}

	// Claim the interface
	if err := claimInterface(d.fd, iface); err != nil {
		return err
	}

	d.claimedMask |= mask
	return nil
}

// releaseInterfaceClaim releases a previously claimed interface.
func (d *deviceConn) releaseInterfaceClaim(iface uint8) error {
	if iface >= MaxInterfacesPerDevice {
		return pkg.ErrInvalidEndpoint
	}

	d.claimMu.Lock()
	defer d.claimMu.Unlock()

	mask := uint16(1) << iface
	if d.claimedMask&mask == 0 {
		// Not claimed
		return nil
	}

	if err := releaseInterface(d.fd, iface); err != nil {
		return err
	}

	d.claimedMask &= ^mask
	return nil
}

// =============================================================================
// URB Operations
// =============================================================================

// submitControlURB submits a control transfer URB.
func (d *deviceConn) submitControlURB(setup []byte, data []byte, timeout uint32) (int, error) {
	// Control transfers use endpoint 0
	return doControlTransfer(d.fd, setup[0], setup[1],
		uint16(setup[2])|uint16(setup[3])<<8,
		uint16(setup[4])|uint16(setup[5])<<8,
		data, timeout)
}

// submitBulkURB submits a bulk transfer URB.
func (d *deviceConn) submitBulkURB(endpoint uint8, data []byte, timeout uint32) (int, error) {
	return doBulkTransfer(d.fd, endpoint, data, timeout)
}

// submitAsyncURB submits an async URB and returns immediately.
func (d *deviceConn) submitAsyncURB(endpoint uint8, urbType uint8, data []byte) (*urb, error) {
	epIdx := endpointIndex(endpoint)
	if epIdx < 0 || epIdx >= MaxEndpointsPerDevice {
		return nil, pkg.ErrInvalidEndpoint
	}

	ep := &d.endpoints[epIdx]
	slotIdx := ep.allocSlot()
	if slotIdx < 0 {
		return nil, pkg.ErrNoMemory
	}

	slot := ep.getSlot(slotIdx)
	u := &slot.urb

	// Initialize URB based on type
	switch urbType {
	case URBTypeBulk:
		initBulkURB(u, endpoint, slot.buffer[:len(data)])
	case URBTypeInterrupt:
		initInterruptURB(u, endpoint, slot.buffer[:len(data)])
	default:
		ep.freeSlot(slotIdx)
		return nil, pkg.ErrInvalidRequest
	}

	// Copy data for OUT transfers
	if endpoint&0x80 == 0 {
		copy(slot.buffer[:], data)
	}

	// Submit the URB
	if err := submitURB(d.fd, u); err != nil {
		ep.freeSlot(slotIdx)
		return nil, err
	}

	return u, nil
}

// reapAsyncURB waits for an async URB to complete.
func (d *deviceConn) reapAsyncURB() (*urb, error) {
	return reapURBNDelay(d.fd)
}

// discardAllURBs cancels all pending URBs (used during ENODEV recovery).
func (d *deviceConn) discardAllURBs() {
	for epIdx := range d.endpoints {
		ep := &d.endpoints[epIdx]
		ep.mu.Lock()
		for i := 0; i < MaxURBsPerEndpoint; i++ {
			if ep.slots[i].inUse {
				discardURB(d.fd, &ep.slots[i].urb)
			}
		}
		ep.mu.Unlock()
	}

	// Reap all discarded URBs
	for {
		_, err := reapURBNDelay(d.fd)
		if err != nil {
			break
		}
	}

	// Reset all endpoint states
	for epIdx := range d.endpoints {
		d.endpoints[epIdx].init()
	}
}

// handleENODEV handles ENODEV error by cleaning up URBs.
func (d *deviceConn) handleENODEV() {
	d.markDisconnected()
	d.discardAllURBs()
}

// =============================================================================
// Device Pool
// =============================================================================

// deviceSlot represents a slot in the device pool.
type deviceSlot struct {
	conn *deviceConn // Device connection (nil if slot is free)
	port int         // Port number (1-indexed)
	next int8        // Next free slot index (-1 if none)
}

// devicePool manages a fixed-size pool of device connections.
type devicePool struct {
	slots    [MaxDevices]deviceSlot // Device slots
	freeHead int8                   // Head of free list (-1 if empty)
	count    int                    // Number of active devices
	mu       sync.Mutex             // Protects the pool
}

// initDevicePool initializes the device pool.
func (p *devicePool) init() {
	p.freeHead = 0
	p.count = 0
	for i := 0; i < MaxDevices-1; i++ {
		p.slots[i].next = int8(i + 1)
		p.slots[i].conn = nil
	}
	p.slots[MaxDevices-1].next = -1
	p.slots[MaxDevices-1].conn = nil
}

// alloc allocates a device slot and returns its index.
// Returns -1 if no slots are available.
func (p *devicePool) alloc(port int) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.freeHead < 0 {
		return -1
	}

	idx := int(p.freeHead)
	slot := &p.slots[idx]
	p.freeHead = slot.next
	slot.next = -1
	slot.port = port
	p.count++

	return idx
}

// free returns a device slot to the pool.
func (p *devicePool) free(idx int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if idx < 0 || idx >= MaxDevices {
		return
	}

	slot := &p.slots[idx]
	if slot.conn != nil {
		slot.conn.close()
		slot.conn = nil
	}

	slot.next = p.freeHead
	p.freeHead = int8(idx)
	p.count--
}

// get returns the device connection at the given slot index.
func (p *devicePool) get(idx int) *deviceConn {
	if idx < 0 || idx >= MaxDevices {
		return nil
	}
	return p.slots[idx].conn
}

// set sets the device connection at the given slot index.
func (p *devicePool) set(idx int, conn *deviceConn) {
	if idx < 0 || idx >= MaxDevices {
		return
	}
	p.slots[idx].conn = conn
}

// findByAddress finds a device by its USB address.
func (p *devicePool) findByAddress(addr hal.DeviceAddress) *deviceConn {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := 0; i < MaxDevices; i++ {
		if p.slots[i].conn != nil && p.slots[i].conn.address == addr {
			return p.slots[i].conn
		}
	}
	return nil
}

// findByPort finds a device by its port number.
func (p *devicePool) findByPort(port int) *deviceConn {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i := 0; i < MaxDevices; i++ {
		if p.slots[i].conn != nil && p.slots[i].port == port {
			return p.slots[i].conn
		}
	}
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

// endpointIndex converts an endpoint address to an array index.
// IN endpoints (0x81-0x8F) and OUT endpoints (0x01-0x0F) use separate indices.
func endpointIndex(addr uint8) int {
	epNum := int(addr & 0x0F)
	if addr&0x80 != 0 {
		// IN endpoint
		return epNum + 16
	}
	// OUT endpoint
	return epNum
}

// isNoData returns true if the error indicates no data (ENODATA).
func isNoData(err error) bool {
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.Errno(ENODATA)
	}
	return false
}
