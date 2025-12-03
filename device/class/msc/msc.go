package msc

import (
	"context"
	"encoding/binary"
	"sync"

	"github.com/ardnew/softusb/device"
	"github.com/ardnew/softusb/pkg"
)

// MSC implements the Mass Storage Class Bulk-Only Transport driver.
type MSC struct {
	// Interface
	iface *device.Interface

	// Endpoints
	bulkInEP  *device.Endpoint // Bulk IN (device to host)
	bulkOutEP *device.Endpoint // Bulk OUT (host to device)

	// Stack reference for data transfer
	stack *device.Stack

	// Storage backend
	storage Storage

	// Device information
	inquiry InquiryResponse

	// Current command state
	currentCBW  CommandBlockWrapper
	currentTag  uint32
	dataResidue uint32

	// Sense data (for REQUEST SENSE)
	senseKey uint8
	asc      uint8
	ascq     uint8

	// Buffers (zero-allocation pattern)
	cbwBuf  [CBWSize]byte
	cswBuf  [CSWSize]byte
	dataBuf [MaxTransferSize]byte
	senseBuf [18]byte

	// State
	mutex      sync.RWMutex
	configured bool

	// Logical Unit Number (typically 0)
	maxLUN uint8
}

// New creates a new MSC class driver with the given storage backend.
// vendorID and productID are 8 and 16 character strings respectively.
func New(storage Storage, vendorID, productID string) *MSC {
	m := &MSC{
		storage: storage,
		maxLUN:  0, // Single LUN by default
	}

	// Initialize INQUIRY response
	m.inquiry = *NewInquiryResponse(
		DeviceTypeDisk,
		storage.IsRemovable(),
		vendorID,
		productID,
		"1.0",
	)

	// Clear sense data (no error)
	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)

	return m
}

// SetStack sets the device stack reference for data transfer.
func (m *MSC) SetStack(stack *device.Stack) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.stack = stack
}

// SetMaxLUN sets the maximum Logical Unit Number (0-15).
func (m *MSC) SetMaxLUN(lun uint8) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if lun <= 15 {
		m.maxLUN = lun
	}
}

// Init initializes the class driver for the given interface.
func (m *MSC) Init(iface *device.Interface) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.iface = iface

	// Find bulk endpoints
	for _, ep := range iface.Endpoints() {
		if ep.IsBulk() {
			if ep.IsIn() {
				m.bulkInEP = ep
			} else {
				m.bulkOutEP = ep
			}
		}
	}

	if m.bulkInEP == nil || m.bulkOutEP == nil {
		return pkg.ErrInvalidEndpoint
	}

	m.configured = true
	pkg.LogDebug(pkg.ComponentDevice, "MSC configured",
		"bulkIn", m.bulkInEP.Address,
		"bulkOut", m.bulkOutEP.Address)

	return nil
}

// HandleSetup processes class-specific SETUP requests.
func (m *MSC) HandleSetup(iface *device.Interface, setup *device.SetupPacket, data []byte) (bool, error) {
	if !setup.IsClass() {
		return false, nil
	}

	switch setup.Request {
	case RequestBulkOnlyMassStorageReset:
		return m.handleReset(setup)

	case RequestGetMaxLUN:
		return m.handleGetMaxLUN(setup, data)

	default:
		return false, nil
	}
}

// handleReset handles the Bulk-Only Mass Storage Reset request.
func (m *MSC) handleReset(setup *device.SetupPacket) (bool, error) {
	pkg.LogDebug(pkg.ComponentDevice, "MSC reset requested")

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Reset sense data
	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)

	// Clear any stalled endpoints (would be done by stack)
	return true, nil
}

// handleGetMaxLUN handles the Get Max LUN request.
func (m *MSC) handleGetMaxLUN(setup *device.SetupPacket, data []byte) (bool, error) {
	m.mutex.RLock()
	maxLUN := m.maxLUN
	m.mutex.RUnlock()

	pkg.LogDebug(pkg.ComponentDevice, "Get Max LUN",
		"maxLUN", maxLUN)

	if len(data) > 0 {
		data[0] = maxLUN
	}

	return true, nil
}

// SetAlternate handles alternate setting changes.
func (m *MSC) SetAlternate(iface *device.Interface, alt uint8) error {
	pkg.LogDebug(pkg.ComponentDevice, "MSC alternate setting",
		"interface", iface.Number,
		"alt", alt)
	return nil
}

// Close releases resources held by the class driver.
func (m *MSC) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.iface = nil
	m.bulkInEP = nil
	m.bulkOutEP = nil
	m.stack = nil
	m.configured = false

	return nil
}

// setSense sets sense data for the next REQUEST SENSE command.
func (m *MSC) setSense(key, asc, ascq uint8) {
	m.senseKey = key
	m.asc = asc
	m.ascq = ascq
}

// ConfigureDevice adds the MSC interface to a device builder.
func (m *MSC) ConfigureDevice(builder *device.DeviceBuilder, bulkInEPAddr, bulkOutEPAddr uint8) *device.DeviceBuilder {
	builder.AddInterface(ClassMSC, SubclassSCSI, ProtocolBulkOnly)
	builder.AddEndpoint(bulkInEPAddr|device.EndpointDirectionIn, device.EndpointTypeBulk, 64)
	builder.AddEndpoint(bulkOutEPAddr&0x0F, device.EndpointTypeBulk, 64)
	return builder
}

// AttachToInterface attaches this class driver to the MSC interface.
func (m *MSC) AttachToInterface(dev *device.Device, configValue, ifaceNum uint8) error {
	config := dev.GetConfiguration(configValue)
	if config == nil {
		return pkg.ErrInvalidRequest
	}

	iface := config.GetInterface(ifaceNum)
	if iface == nil {
		return pkg.ErrInvalidRequest
	}

	return iface.SetClassDriver(m)
}

// Run is the main processing loop for MSC.
// It reads CBWs, processes SCSI commands, and sends CSWs.
// This should be called in a goroutine after the device is configured.
func (m *MSC) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Process one command
		if err := m.processCBW(ctx); err != nil {
			// Check if context was cancelled
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Log error and continue
			pkg.LogWarn(pkg.ComponentDevice, "CBW processing error",
				"error", err)
		}
	}
}

// processCBW reads and processes a Command Block Wrapper.
func (m *MSC) processCBW(ctx context.Context) error {
	m.mutex.RLock()
	stack := m.stack
	ep := m.bulkOutEP
	configured := m.configured
	m.mutex.RUnlock()

	if !configured || stack == nil || ep == nil {
		return pkg.ErrNotConfigured
	}

	// Read CBW from host
	n, err := stack.Read(ctx, ep, m.cbwBuf[:])
	if err != nil {
		return err
	}

	if n != CBWSize {
		pkg.LogWarn(pkg.ComponentDevice, "invalid CBW size",
			"expected", CBWSize,
			"got", n)
		return pkg.ErrInvalidRequest
	}

	// Parse CBW
	if !ParseCBW(m.cbwBuf[:], &m.currentCBW) {
		pkg.LogWarn(pkg.ComponentDevice, "invalid CBW signature")
		return pkg.ErrInvalidRequest
	}

	m.currentTag = m.currentCBW.Tag

	pkg.LogDebug(pkg.ComponentDevice, "CBW received",
		"tag", m.currentCBW.Tag,
		"dataLen", m.currentCBW.DataTransferLength,
		"flags", m.currentCBW.Flags,
		"lun", m.currentCBW.LUN,
		"cbLen", m.currentCBW.CBLength,
		"opcode", m.currentCBW.CB[0])

	// Handle SCSI command
	status, residue := m.handleSCSICommand(ctx, &m.currentCBW)

	// Send CSW
	return m.sendCSW(ctx, status, residue)
}

// sendCSW sends a Command Status Wrapper.
func (m *MSC) sendCSW(ctx context.Context, status uint8, residue uint32) error {
	m.mutex.RLock()
	stack := m.stack
	ep := m.bulkInEP
	m.mutex.RUnlock()

	if stack == nil || ep == nil {
		return pkg.ErrNotConfigured
	}

	csw := NewCSW(m.currentTag, residue, status)
	n := csw.MarshalTo(m.cswBuf[:])

	_, err := stack.Write(ctx, ep, m.cswBuf[:n])
	if err != nil {
		return err
	}

	pkg.LogDebug(pkg.ComponentDevice, "CSW sent",
		"tag", csw.Tag,
		"residue", residue,
		"status", status)

	return nil
}

// parseU16BE parses a big-endian uint16 from data at offset.
func parseU16BE(data []byte, offset int) uint16 {
	if offset+2 > len(data) {
		return 0
	}
	return binary.BigEndian.Uint16(data[offset:])
}

// parseU32BE parses a big-endian uint32 from data at offset.
func parseU32BE(data []byte, offset int) uint32 {
	if offset+4 > len(data) {
		return 0
	}
	return binary.BigEndian.Uint32(data[offset:])
}

// parseU64BE parses a big-endian uint64 from data at offset.
func parseU64BE(data []byte, offset int) uint64 {
	if offset+8 > len(data) {
		return 0
	}
	return binary.BigEndian.Uint64(data[offset:])
}

// Compile-time interface check
var _ device.ClassDriver = (*MSC)(nil)
