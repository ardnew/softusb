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

// PortStatus represents the status of a host port.
type PortStatus struct {
	Connected     bool  // Device is connected
	Enabled       bool  // Port is enabled
	Suspended     bool  // Port is suspended
	OverCurrent   bool  // Over-current condition detected
	Reset         bool  // Port is being reset
	PowerOn       bool  // Port has power applied
	Speed         Speed // Connected device speed
	ConnectChange bool  // Connection status has changed
	EnableChange  bool  // Enable status has changed
	ResetChange   bool  // Reset has completed
}

// SetupPacket represents a USB SETUP packet in the HAL layer.
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

// TransferType indicates the type of USB transfer.
type TransferType uint8

// Transfer type constants.
const (
	TransferControl     TransferType = 0 // Control transfer
	TransferIsochronous TransferType = 1 // Isochronous transfer
	TransferBulk        TransferType = 2 // Bulk transfer
	TransferInterrupt   TransferType = 3 // Interrupt transfer
)

// EndpointDescriptor describes an endpoint for HAL configuration.
type EndpointDescriptor struct {
	Address       uint8  // Endpoint address including direction bit
	Attributes    uint8  // Transfer type and sync/usage flags
	MaxPacketSize uint16 // Maximum packet size
	Interval      uint8  // Polling interval for interrupt/isochronous
}

// Number returns the endpoint number (0-15).
func (e *EndpointDescriptor) Number() uint8 {
	return e.Address & 0x0F
}

// IsIn returns true if this is an IN endpoint (device to host).
func (e *EndpointDescriptor) IsIn() bool {
	return e.Address&0x80 != 0
}

// TransferType returns the transfer type.
func (e *EndpointDescriptor) TransferType() TransferType {
	return TransferType(e.Attributes & 0x03)
}

// DeviceAddress represents a USB device address (1-127).
type DeviceAddress uint8

// HostHAL defines the Hardware Abstraction Layer interface for USB host stacks.
//
// The HAL provides the low-level operations needed by the host stack to
// communicate with USB controller hardware. Platform vendors implement this
// interface to enable softusb on their specific platform and controller.
//
// All methods should be safe for concurrent use where applicable.
type HostHAL interface {
	// Initialization and Lifecycle

	// Init initializes the USB host controller hardware.
	// The context can be used to cancel initialization.
	Init(ctx context.Context) error

	// Start enables the host controller and applies power to ports.
	// After Start returns, the controller should be ready to detect devices.
	Start() error

	// Stop disables the host controller and removes power from ports.
	Stop() error

	// Close releases all resources associated with the HAL.
	// After Close returns, the HAL should not be used.
	Close() error

	// Port Operations

	// NumPorts returns the number of root hub ports.
	NumPorts() int

	// GetPortStatus returns the status of a port (1-indexed).
	GetPortStatus(port int) (PortStatus, error)

	// PortSpeed returns the connection speed of a device on the given port.
	PortSpeed(port int) Speed

	// ResetPort initiates a port reset (1-indexed).
	// After reset completes, the device will be at address 0.
	ResetPort(port int) error

	// EnablePort enables or disables a port.
	EnablePort(port int, enable bool) error

	// Control Transfers

	// ControlTransfer performs a control transfer to a device.
	// The setup packet and data buffer are provided by the caller.
	// For OUT transfers, data contains the data to send.
	// For IN transfers, data is filled with received data.
	// Returns the number of bytes transferred in the data phase.
	ControlTransfer(ctx context.Context, addr DeviceAddress, setup *SetupPacket, data []byte) (int, error)

	// Data Transfers

	// BulkTransfer performs a bulk transfer to/from an endpoint.
	// For IN endpoints, data is filled with received data.
	// For OUT endpoints, data contains the data to send.
	// Returns the number of bytes transferred.
	BulkTransfer(ctx context.Context, addr DeviceAddress, endpoint uint8, data []byte) (int, error)

	// InterruptTransfer performs an interrupt transfer to/from an endpoint.
	// Returns the number of bytes transferred.
	InterruptTransfer(ctx context.Context, addr DeviceAddress, endpoint uint8, data []byte) (int, error)

	// IsochronousTransfer performs an isochronous transfer.
	// Returns the number of bytes transferred.
	IsochronousTransfer(ctx context.Context, addr DeviceAddress, endpoint uint8, data []byte) (int, error)

	// Device Management

	// SetDeviceAddress assigns an address to a device at address 0.
	// This is called after port reset during enumeration.
	SetDeviceAddress(ctx context.Context, newAddr DeviceAddress) error

	// Interface Management

	// ClaimInterface claims exclusive access to an interface on a device.
	// For HALs that require kernel driver detachment (e.g., Linux usbfs),
	// this should detach any existing driver before claiming.
	ClaimInterface(addr DeviceAddress, iface uint8) error

	// ReleaseInterface releases a previously claimed interface.
	ReleaseInterface(addr DeviceAddress, iface uint8) error

	// Connection Events

	// WaitForConnection blocks until a device connects or context is cancelled.
	// Returns the port number (1-indexed) where the device connected.
	WaitForConnection(ctx context.Context) (int, error)

	// WaitForDisconnection blocks until a device disconnects or context is cancelled.
	// Returns the port number (1-indexed) where the device disconnected.
	WaitForDisconnection(ctx context.Context) (int, error)
}
