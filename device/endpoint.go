package device

import (
	"fmt"
	"sync"

	"github.com/ardnew/softusb/pkg"
)

// Endpoint transfer types (USB 2.0 Spec Table 9-13).
const (
	EndpointTypeControl     = 0x00 // Control transfer
	EndpointTypeIsochronous = 0x01 // Isochronous transfer
	EndpointTypeBulk        = 0x02 // Bulk transfer
	EndpointTypeInterrupt   = 0x03 // Interrupt transfer
)

// Endpoint directions.
const (
	EndpointDirectionOut = 0x00 // Host to device
	EndpointDirectionIn  = 0x80 // Device to host
)

// Isochronous synchronization types (bits 2-3 of Attributes).
const (
	IsoSyncNone     = 0x00 // No synchronization
	IsoSyncAsync    = 0x04 // Asynchronous
	IsoSyncAdaptive = 0x08 // Adaptive
	IsoSyncSync     = 0x0C // Synchronous
)

// Isochronous usage types (bits 4-5 of Attributes).
const (
	IsoUsageData     = 0x00 // Data endpoint
	IsoUsageFeedback = 0x10 // Feedback endpoint
	IsoUsageImplicit = 0x20 // Implicit feedback data endpoint
)

// Endpoint represents a USB endpoint.
type Endpoint struct {
	// Descriptor data
	Address       uint8  // Endpoint address including direction
	Attributes    uint8  // Transfer type and sync/usage for isochronous
	MaxPacketSize uint16 // Maximum packet size
	Interval      uint8  // Polling interval (interrupt/isochronous)

	// Runtime state
	stalled    bool // Endpoint is stalled
	dataToggle bool // DATA0/DATA1 toggle
	mutex      sync.Mutex

	// Isochronous-specific
	frameNumber uint16 // Current frame number for scheduling
}

// NewEndpoint creates a new endpoint from a descriptor.
func NewEndpoint(desc *EndpointDescriptor) *Endpoint {
	return &Endpoint{
		Address:       desc.EndpointAddress,
		Attributes:    desc.Attributes,
		MaxPacketSize: desc.MaxPacketSize,
		Interval:      desc.Interval,
	}
}

// Number returns the endpoint number (0-15).
func (e *Endpoint) Number() uint8 {
	return e.Address & 0x0F
}

// Direction returns the endpoint direction (EndpointDirectionIn or EndpointDirectionOut).
func (e *Endpoint) Direction() uint8 {
	return e.Address & 0x80
}

// IsIn returns true if this is an IN endpoint (device to host).
func (e *Endpoint) IsIn() bool {
	return e.Direction() == EndpointDirectionIn
}

// IsOut returns true if this is an OUT endpoint (host to device).
func (e *Endpoint) IsOut() bool {
	return e.Direction() == EndpointDirectionOut
}

// TransferType returns the transfer type (Control, Isochronous, Bulk, or Interrupt).
func (e *Endpoint) TransferType() uint8 {
	return e.Attributes & 0x03
}

// IsControl returns true if this is a control endpoint.
func (e *Endpoint) IsControl() bool {
	return e.TransferType() == EndpointTypeControl
}

// IsBulk returns true if this is a bulk endpoint.
func (e *Endpoint) IsBulk() bool {
	return e.TransferType() == EndpointTypeBulk
}

// IsInterrupt returns true if this is an interrupt endpoint.
func (e *Endpoint) IsInterrupt() bool {
	return e.TransferType() == EndpointTypeInterrupt
}

// IsIsochronous returns true if this is an isochronous endpoint.
func (e *Endpoint) IsIsochronous() bool {
	return e.TransferType() == EndpointTypeIsochronous
}

// IsoSyncType returns the isochronous synchronization type.
func (e *Endpoint) IsoSyncType() uint8 {
	return e.Attributes & 0x0C
}

// IsoUsageType returns the isochronous usage type.
func (e *Endpoint) IsoUsageType() uint8 {
	return e.Attributes & 0x30
}

// SetStall sets or clears the stall condition.
func (e *Endpoint) SetStall(stalled bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.stalled = stalled
	if stalled {
		pkg.LogDebug(pkg.ComponentEndpoint, "endpoint stalled",
			"address", fmt.Sprintf("0x%02X", e.Address))
	} else {
		pkg.LogDebug(pkg.ComponentEndpoint, "endpoint stall cleared",
			"address", fmt.Sprintf("0x%02X", e.Address))
	}
}

// IsStalled returns true if the endpoint is stalled.
func (e *Endpoint) IsStalled() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.stalled
}

// DataToggle returns the current data toggle state.
func (e *Endpoint) DataToggle() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.dataToggle
}

// ToggleData flips the data toggle state.
func (e *Endpoint) ToggleData() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.dataToggle = !e.dataToggle
}

// ResetDataToggle resets the data toggle to DATA0.
func (e *Endpoint) ResetDataToggle() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.dataToggle = false
}

// SetDataToggle sets the data toggle state explicitly.
func (e *Endpoint) SetDataToggle(toggle bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.dataToggle = toggle
}

// FrameNumber returns the current frame number for isochronous scheduling.
func (e *Endpoint) FrameNumber() uint16 {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return e.frameNumber
}

// SetFrameNumber sets the frame number for isochronous scheduling.
func (e *Endpoint) SetFrameNumber(frame uint16) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.frameNumber = frame
}

// IncrementFrame advances the frame number.
func (e *Endpoint) IncrementFrame() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.frameNumber++
}

// Descriptor returns the endpoint descriptor.
func (e *Endpoint) Descriptor() *EndpointDescriptor {
	return &EndpointDescriptor{
		Length:          7,
		DescriptorType:  DescriptorTypeEndpoint,
		EndpointAddress: e.Address,
		Attributes:      e.Attributes,
		MaxPacketSize:   e.MaxPacketSize,
		Interval:        e.Interval,
	}
}

// TransferTypeName returns a human-readable transfer type name.
func TransferTypeName(t uint8) string {
	switch t & 0x03 {
	case EndpointTypeControl:
		return "Control"
	case EndpointTypeIsochronous:
		return "Isochronous"
	case EndpointTypeBulk:
		return "Bulk"
	case EndpointTypeInterrupt:
		return "Interrupt"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

// DirectionName returns a human-readable direction name.
func DirectionName(dir uint8) string {
	if dir == EndpointDirectionIn {
		return "IN"
	}
	return "OUT"
}
