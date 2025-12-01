package hal

import (
	"context"
)

// Speed represents the USB connection speed.
type Speed uint8

// USB speed constants (USB 2.0 Specification).
const (
	SpeedUnknown Speed = iota // Not connected or unknown
	SpeedLow                  // Low Speed (1.5 Mbit/s)
	SpeedFull                 // Full Speed (12 Mbit/s)
	SpeedHigh                 // High Speed (480 Mbit/s)
)

// String returns a human-readable speed name.
func (s Speed) String() string {
	switch s {
	case SpeedLow:
		return "Low Speed"
	case SpeedFull:
		return "Full Speed"
	case SpeedHigh:
		return "High Speed"
	default:
		return "Unknown"
	}
}

// EndpointConfig describes an endpoint configuration for the HAL.
// This is a minimal, platform-agnostic representation used to configure
// hardware endpoints when a configuration is activated.
type EndpointConfig struct {
	Address       uint8  // Endpoint address including direction bit
	Attributes    uint8  // Transfer type and sync/usage flags
	MaxPacketSize uint16 // Maximum packet size
	Interval      uint8  // Polling interval for interrupt/isochronous
}

// Number returns the endpoint number (0-15).
func (e *EndpointConfig) Number() uint8 {
	return e.Address & 0x0F
}

// IsIn returns true if this is an IN endpoint (device to host).
func (e *EndpointConfig) IsIn() bool {
	return e.Address&0x80 != 0
}

// TransferType returns the transfer type (control, bulk, interrupt, isochronous).
func (e *EndpointConfig) TransferType() uint8 {
	return e.Attributes & 0x03
}

// SetupPacket represents a USB SETUP packet in the HAL layer.
// This is a fixed-size, zero-allocation structure for SETUP transactions.
type SetupPacket struct {
	RequestType uint8  // Request characteristics
	Request     uint8  // Specific request
	Value       uint16 // Request-specific value
	Index       uint16 // Request-specific index
	Length      uint16 // Number of bytes to transfer
}

// SetupPacketSize is the size of a USB SETUP packet in bytes.
const SetupPacketSize = 8

// ParseSetupPacket parses raw bytes into a SetupPacket.
// Returns false if data is too short.
func ParseSetupPacket(data []byte, out *SetupPacket) bool {
	if len(data) < SetupPacketSize {
		return false
	}
	out.RequestType = data[0]
	out.Request = data[1]
	out.Value = uint16(data[2]) | uint16(data[3])<<8
	out.Index = uint16(data[4]) | uint16(data[5])<<8
	out.Length = uint16(data[6]) | uint16(data[7])<<8
	return true
}

// MarshalTo writes the setup packet to buf.
// Returns the number of bytes written (8), or 0 if buf is too small.
func (s *SetupPacket) MarshalTo(buf []byte) int {
	if len(buf) < SetupPacketSize {
		return 0
	}
	buf[0] = s.RequestType
	buf[1] = s.Request
	buf[2] = byte(s.Value)
	buf[3] = byte(s.Value >> 8)
	buf[4] = byte(s.Index)
	buf[5] = byte(s.Index >> 8)
	buf[6] = byte(s.Length)
	buf[7] = byte(s.Length >> 8)
	return SetupPacketSize
}

// DeviceHAL defines the Hardware Abstraction Layer interface for USB device stacks.
//
// The HAL provides the low-level operations needed by the device stack to
// communicate with USB controller hardware. Platform vendors implement this
// interface to enable softusb on their specific platform and controller.
//
// All methods should be safe for concurrent use where applicable.
type DeviceHAL interface {
	// Init initializes the USB controller hardware.
	// The context can be used to cancel initialization.
	Init(ctx context.Context) error

	// Start enables the USB controller and attaches to the bus.
	// After Start returns, the device should be visible to the host.
	Start() error

	// Stop detaches from the bus and disables the USB controller.
	Stop() error

	// SetAddress sets the device address in hardware.
	// Called after the host assigns an address during enumeration.
	SetAddress(address uint8) error

	// ConfigureEndpoints configures hardware endpoints for the active configuration.
	// Called when the device transitions to the Configured state.
	// Pass nil or empty slice to unconfigure all endpoints.
	ConfigureEndpoints(endpoints []EndpointConfig) error

	// Control Endpoint (EP0) Operations

	// ReadSetup reads a SETUP packet from EP0.
	// Blocks until a SETUP packet is available or the context is cancelled.
	// The caller provides the output buffer to avoid allocation.
	ReadSetup(ctx context.Context, out *SetupPacket) error

	// WriteEP0 writes data to EP0 (control IN phase).
	// Blocks until the data is sent or the context is cancelled.
	WriteEP0(ctx context.Context, data []byte) error

	// ReadEP0 reads data from EP0 (control OUT phase).
	// Blocks until data is received or the context is cancelled.
	// Returns the number of bytes read into buf.
	ReadEP0(ctx context.Context, buf []byte) (int, error)

	// StallEP0 stalls the control endpoint to indicate an error.
	StallEP0() error

	// AckEP0 sends a zero-length packet to acknowledge a successful control transfer.
	AckEP0() error

	// Data Endpoint Operations

	// Read reads data from an OUT endpoint into buf.
	// Blocks until data is received or the context is cancelled.
	// Returns the number of bytes read.
	Read(ctx context.Context, address uint8, buf []byte) (int, error)

	// Write writes data to an IN endpoint.
	// Blocks until the data is sent or the context is cancelled.
	// Returns the number of bytes written.
	Write(ctx context.Context, address uint8, data []byte) (int, error)

	// Stall stalls the specified endpoint.
	Stall(address uint8) error

	// ClearStall clears a stall condition on the specified endpoint.
	ClearStall(address uint8) error

	// Connection State

	// IsConnected returns true if the device is connected to a host.
	IsConnected() bool

	// GetSpeed returns the negotiated USB connection speed.
	GetSpeed() Speed

	// WaitConnect blocks until the device connects to a host or the context is cancelled.
	WaitConnect(ctx context.Context) error

	// WaitDisconnect blocks until the device disconnects or the context is cancelled.
	WaitDisconnect(ctx context.Context) error
}
