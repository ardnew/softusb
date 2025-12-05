//go:build tinygo && atsamd51

// Package main provides an ATSAMD51-based USB MSC device HAL for TinyGo.
//
// This HAL implementation targets the Adafruit Grand Central M4 board and
// provides direct USB peripheral access for mass storage class devices.
// It uses fixed static arrays to avoid heap allocations and supports
// polling-based transfers suitable for TinyGo's execution model.
package main

import (
	"context"
	"unsafe"

	"github.com/ardnew/softusb/device/hal"
)

// MaxEndpoints is the number of USB endpoints supported by ATSAMD51.
const MaxEndpoints = 8

// MaxPacketSizeEP0 is the maximum packet size for the control endpoint.
const MaxPacketSizeEP0 = 64

// MaxPacketSizeBulk is the maximum packet size for bulk endpoints.
const MaxPacketSizeBulk = 64

// EndpointDescriptor represents a USB endpoint descriptor in SRAM.
// This structure matches the ATSAMD51 USB endpoint descriptor format.
// Each endpoint has two banks (OUT=bank0, IN=bank1).
type EndpointDescriptor struct {
	// Bank 0 (OUT direction - host to device)
	Bank0 EndpointBank
	// Bank 1 (IN direction - device to host)
	Bank1 EndpointBank
}

// EndpointBank represents one bank of an endpoint descriptor.
// This structure is 16 bytes and must be aligned to 4 bytes.
type EndpointBank struct {
	// ADDR: Data buffer address in SRAM
	Addr uint32
	// PCKSIZE: Packet size configuration
	//   Bits 0-13: BYTE_COUNT (bytes received/to send)
	//   Bits 14-27: MULTI_PACKET_SIZE
	//   Bits 28-30: SIZE (max packet size: 0=8, 1=16, 2=32, 3=64, 4=128, 5=256, 6=512, 7=1023)
	//   Bit 31: AUTO_ZLP
	PckSize uint32
	// EXTREG: Extended register (used for SETUP data)
	ExtReg uint16
	// STATUS_BK: Bank status
	StatusBK uint8
	// Reserved
	_ uint8
}

// HAL implements hal.DeviceHAL for ATSAMD51 USB peripheral.
type HAL struct {
	// Endpoint descriptors - must be aligned to 32-bit boundary
	// This is the USB descriptor table pointed to by DESCADD register
	epDescriptors [MaxEndpoints]EndpointDescriptor

	// Fixed endpoint buffers (zero-allocation)
	ep0OutBuf [MaxPacketSizeEP0]byte // EP0 OUT (SETUP/DATA from host)
	ep0InBuf  [MaxPacketSizeEP0]byte // EP0 IN (DATA to host)

	// Bulk endpoint buffers for MSC (EP1)
	bulkOutBuf [MaxPacketSizeBulk]byte // Bulk OUT (host to device)
	bulkInBuf  [MaxPacketSizeBulk]byte // Bulk IN (device to host)

	// Device address assigned by host
	address uint8

	// Connection speed (always Full Speed for ATSAMD51)
	speed hal.Speed

	// Running flag for polling loops (checked instead of context)
	running bool

	// Connected state
	connected bool

	// Configured endpoints tracking
	configuredEPs uint8
}

// New creates a new ATSAMD51 USB device HAL.
func New() *HAL {
	return &HAL{
		speed: hal.SpeedFull, // ATSAMD51 is Full Speed only
	}
}

// SetRunning sets the running flag for polling loops.
// Call with false to gracefully stop all blocking operations.
func (h *HAL) SetRunning(running bool) {
	h.running = running
}

// Init initializes the USB controller hardware.
func (h *HAL) Init(ctx context.Context) error {
	// Enable USB clocks
	enableUSBClocks()

	// Load USB pad calibration from NVM
	loadPadCalibration()

	// Set endpoint descriptor table address
	setDescriptorAddress(uintptr(unsafe.Pointer(&h.epDescriptors[0])))

	// Configure EP0 descriptor banks
	h.epDescriptors[0].Bank0.Addr = uint32(uintptr(unsafe.Pointer(&h.ep0OutBuf[0])))
	h.epDescriptors[0].Bank0.PckSize = packsizeField(MaxPacketSizeEP0) // SIZE=3 (64 bytes)

	h.epDescriptors[0].Bank1.Addr = uint32(uintptr(unsafe.Pointer(&h.ep0InBuf[0])))
	h.epDescriptors[0].Bank1.PckSize = packsizeField(MaxPacketSizeEP0)

	// Configure EP0 as control endpoint
	setEndpointConfig(0, epTypeCfgControl, epTypeCfgControl)

	// Enable USB peripheral
	enableUSB()

	h.running = true
	return nil
}

// Start enables the USB controller and attaches to the bus.
func (h *HAL) Start() error {
	// Clear detach bit to connect D+ pull-up (attach to bus)
	attachUSB()

	return nil
}

// Stop detaches from the bus and disables the USB controller.
func (h *HAL) Stop() error {
	h.running = false

	// Set detach bit to disconnect from bus
	detachUSB()

	// Disable USB peripheral
	disableUSB()

	return nil
}

// SetAddress sets the device address in hardware.
func (h *HAL) SetAddress(address uint8) error {
	h.address = address
	setDeviceAddress(address)
	return nil
}

// ConfigureEndpoints configures hardware endpoints for the active configuration.
func (h *HAL) ConfigureEndpoints(endpoints []hal.EndpointConfig) error {
	// Reset all non-control endpoints
	for i := 1; i < MaxEndpoints; i++ {
		setEndpointConfig(uint8(i), 0, 0)
	}
	h.configuredEPs = 0

	if len(endpoints) == 0 {
		return nil
	}

	for _, ep := range endpoints {
		num := ep.Number()
		if num == 0 || num >= MaxEndpoints {
			continue
		}

		// Get current endpoint config
		inCfg, outCfg := getEndpointConfig(num)

		if ep.IsIn() {
			// IN endpoint (device to host) - use bank 1
			h.epDescriptors[num].Bank1.Addr = uint32(uintptr(unsafe.Pointer(&h.bulkInBuf[0])))
			h.epDescriptors[num].Bank1.PckSize = packsizeField(uint16(ep.MaxPacketSize))
			inCfg = epTypeForTransfer(ep.TransferType())
		} else {
			// OUT endpoint (host to device) - use bank 0
			h.epDescriptors[num].Bank0.Addr = uint32(uintptr(unsafe.Pointer(&h.bulkOutBuf[0])))
			h.epDescriptors[num].Bank0.PckSize = packsizeField(uint16(ep.MaxPacketSize))
			outCfg = epTypeForTransfer(ep.TransferType())
		}

		setEndpointConfig(num, outCfg, inCfg)
		h.configuredEPs |= 1 << num
	}

	return nil
}

// ReadSetup reads a SETUP packet from EP0.
func (h *HAL) ReadSetup(ctx context.Context, out *hal.SetupPacket) error {
	for h.running {
		// Check for bus reset
		if hasUSBReset() {
			clearUSBReset()
			h.connected = true
			// Return reset error so stack can handle it
			return errReset
		}

		// Check for SETUP received on EP0
		if hasSetupReceived(0) {
			// Copy SETUP data from buffer
			if !hal.ParseSetupPacket(h.ep0OutBuf[:hal.SetupPacketSize], out) {
				clearSetupReceived(0)
				continue
			}
			clearSetupReceived(0)

			// Acknowledge SETUP - clear bank 0 ready for next packet
			clearOutReady(0)

			return nil
		}

		// Small delay to prevent tight polling
		delayMicroseconds(10)
	}

	return errCancelled
}

// WriteEP0 writes data to EP0 (control IN phase).
func (h *HAL) WriteEP0(ctx context.Context, data []byte) error {
	remaining := len(data)
	offset := 0

	for remaining > 0 && h.running {
		// Calculate chunk size
		chunk := remaining
		if chunk > MaxPacketSizeEP0 {
			chunk = MaxPacketSizeEP0
		}

		// Copy data to EP0 IN buffer
		copy(h.ep0InBuf[:chunk], data[offset:offset+chunk])

		// Set byte count
		setByteCount(0, true, uint16(chunk))

		// Set bank ready
		setInReady(0)

		// Wait for transfer complete
		for h.running {
			if hasTransferComplete(0, true) {
				clearTransferComplete(0, true)
				break
			}
			delayMicroseconds(1)
		}

		offset += chunk
		remaining -= chunk
	}

	if !h.running {
		return errCancelled
	}

	return nil
}

// ReadEP0 reads data from EP0 (control OUT phase).
func (h *HAL) ReadEP0(ctx context.Context, buf []byte) (int, error) {
	if len(buf) == 0 {
		// Status phase - just acknowledge
		return 0, nil
	}

	total := 0

	for h.running && total < len(buf) {
		// Wait for OUT data
		if hasTransferComplete(0, false) {
			// Get received byte count
			n := getByteCount(0, false)

			// Copy data
			if n > 0 {
				copyLen := int(n)
				if total+copyLen > len(buf) {
					copyLen = len(buf) - total
				}
				copy(buf[total:total+copyLen], h.ep0OutBuf[:copyLen])
				total += copyLen
			}

			clearTransferComplete(0, false)
			clearOutReady(0)

			// If short packet, we're done
			if n < MaxPacketSizeEP0 {
				break
			}
		}
		delayMicroseconds(1)
	}

	if !h.running && total == 0 {
		return 0, errCancelled
	}

	return total, nil
}

// StallEP0 stalls the control endpoint.
func (h *HAL) StallEP0() error {
	stallEndpoint(0, true)  // Stall IN
	stallEndpoint(0, false) // Stall OUT
	return nil
}

// AckEP0 sends a zero-length packet to acknowledge.
func (h *HAL) AckEP0() error {
	// Send ZLP on EP0 IN
	setByteCount(0, true, 0)
	setInReady(0)

	// Wait for transfer complete
	for h.running {
		if hasTransferComplete(0, true) {
			clearTransferComplete(0, true)
			return nil
		}
		delayMicroseconds(1)
	}

	return errCancelled
}

// Read reads data from an OUT endpoint.
func (h *HAL) Read(ctx context.Context, address uint8, buf []byte) (int, error) {
	epNum := address & 0x0F
	if epNum == 0 || epNum >= MaxEndpoints {
		return 0, errInvalidEndpoint
	}

	// Wait for data
	for h.running {
		if hasTransferComplete(epNum, false) {
			// Get received byte count
			n := getByteCount(epNum, false)

			// Copy data from buffer
			copyLen := int(n)
			if copyLen > len(buf) {
				copyLen = len(buf)
			}
			copy(buf[:copyLen], h.bulkOutBuf[:copyLen])

			clearTransferComplete(epNum, false)

			// Ready for next packet
			clearOutReady(epNum)

			return copyLen, nil
		}
		delayMicroseconds(1)
	}

	return 0, errCancelled
}

// Write writes data to an IN endpoint.
func (h *HAL) Write(ctx context.Context, address uint8, data []byte) (int, error) {
	epNum := address & 0x0F
	if epNum == 0 || epNum >= MaxEndpoints {
		return 0, errInvalidEndpoint
	}

	remaining := len(data)
	offset := 0

	for remaining > 0 && h.running {
		// Calculate chunk size
		chunk := remaining
		if chunk > MaxPacketSizeBulk {
			chunk = MaxPacketSizeBulk
		}

		// Copy data to buffer
		copy(h.bulkInBuf[:chunk], data[offset:offset+chunk])

		// Set byte count
		setByteCount(epNum, true, uint16(chunk))

		// Set bank ready
		setInReady(epNum)

		// Wait for transfer complete
		for h.running {
			if hasTransferComplete(epNum, true) {
				clearTransferComplete(epNum, true)
				break
			}
			delayMicroseconds(1)
		}

		offset += chunk
		remaining -= chunk
	}

	if !h.running {
		return offset, errCancelled
	}

	return len(data), nil
}

// Stall stalls the specified endpoint.
func (h *HAL) Stall(address uint8) error {
	epNum := address & 0x0F
	isIn := address&0x80 != 0
	stallEndpoint(epNum, isIn)
	return nil
}

// ClearStall clears a stall condition on the specified endpoint.
func (h *HAL) ClearStall(address uint8) error {
	epNum := address & 0x0F
	isIn := address&0x80 != 0
	clearStall(epNum, isIn)
	return nil
}

// IsConnected returns true if the device is connected to a host.
func (h *HAL) IsConnected() bool {
	return h.connected && h.running
}

// GetSpeed returns the negotiated USB connection speed.
func (h *HAL) GetSpeed() hal.Speed {
	return h.speed
}

// WaitConnect blocks until the device connects to a host.
func (h *HAL) WaitConnect(ctx context.Context) error {
	for h.running {
		if h.connected {
			return nil
		}
		// Check for bus reset which indicates connection
		if hasUSBReset() {
			clearUSBReset()
			h.connected = true
			return nil
		}
		delayMicroseconds(100)
	}
	return errCancelled
}

// WaitDisconnect blocks until the device disconnects.
func (h *HAL) WaitDisconnect(ctx context.Context) error {
	for h.running {
		if !h.connected {
			return nil
		}
		// Check for suspend which may indicate disconnection
		if hasSuspend() {
			h.connected = false
			return nil
		}
		delayMicroseconds(100)
	}
	return errCancelled
}

// packsizeField creates the PCKSIZE field value for the given max packet size.
func packsizeField(maxPacketSize uint16) uint32 {
	// SIZE field encoding: 0=8, 1=16, 2=32, 3=64, 4=128, 5=256, 6=512, 7=1023
	var size uint32
	switch {
	case maxPacketSize <= 8:
		size = 0
	case maxPacketSize <= 16:
		size = 1
	case maxPacketSize <= 32:
		size = 2
	case maxPacketSize <= 64:
		size = 3
	case maxPacketSize <= 128:
		size = 4
	case maxPacketSize <= 256:
		size = 5
	case maxPacketSize <= 512:
		size = 6
	default:
		size = 7
	}
	return size << 28
}

// epTypeForTransfer returns the endpoint type config value for a transfer type.
func epTypeForTransfer(transferType uint8) uint8 {
	switch transferType & 0x03 {
	case 0x00: // Control
		return epTypeCfgControl
	case 0x01: // Isochronous
		return epTypeCfgIsochronous
	case 0x02: // Bulk
		return epTypeCfgBulk
	case 0x03: // Interrupt
		return epTypeCfgInterrupt
	default:
		return epTypeCfgDisabled
	}
}

// Compile-time interface check
var _ hal.DeviceHAL = (*HAL)(nil)
