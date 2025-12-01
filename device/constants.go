package device

import "fmt"

// Maximum limits for fixed-size arrays (zero-allocation support).
const (
	// MaxEndpointsPerInterface is the maximum number of endpoints per interface.
	// USB 2.0 allows up to 16 IN + 16 OUT endpoints, but typically far fewer are used.
	MaxEndpointsPerInterface = 16

	// MaxInterfacesPerConfiguration is the maximum number of interfaces per configuration.
	MaxInterfacesPerConfiguration = 8

	// MaxConfigurations is the maximum number of configurations per device.
	MaxConfigurations = 4

	// MaxStrings is the maximum number of string descriptors per device.
	MaxStrings = 16

	// MaxPendingTransfersPerEndpoint is the maximum pending transfers per endpoint.
	MaxPendingTransfersPerEndpoint = 8

	// MaxIsoPackets is the maximum number of isochronous packets per transfer.
	MaxIsoPackets = 256
)

// USB Speeds as defined in USB 2.0 specification.
const (
	SpeedLow   Speed = 0 // 1.5 Mbps (USB 1.0)
	SpeedFull  Speed = 1 // 12 Mbps (USB 1.1)
	SpeedHigh  Speed = 2 // 480 Mbps (USB 2.0)
	SpeedSuper Speed = 3 // 5 Gbps (USB 3.0)
)

// Speed represents USB connection speed.
type Speed uint8

// String returns a human-readable speed description.
func (s Speed) String() string {
	switch s {
	case SpeedLow:
		return "Low Speed (1.5 Mbps)"
	case SpeedFull:
		return "Full Speed (12 Mbps)"
	case SpeedHigh:
		return "High Speed (480 Mbps)"
	case SpeedSuper:
		return "Super Speed (5 Gbps)"
	default:
		return fmt.Sprintf("Unknown Speed (%d)", s)
	}
}

// MaxPacketSize0 returns the maximum packet size for endpoint 0 at this speed.
func (s Speed) MaxPacketSize0() uint16 {
	switch s {
	case SpeedLow:
		return 8
	case SpeedFull:
		return 64
	case SpeedHigh:
		return 64
	case SpeedSuper:
		return 512
	default:
		return 8
	}
}

// Device states as defined in USB 2.0 specification section 9.1.
const (
	StateAttached   State = 0 // Device is attached but not powered
	StatePowered    State = 1 // Device is powered
	StateDefault    State = 2 // Device has been reset, using default address
	StateAddress    State = 3 // Device has been assigned a unique address
	StateConfigured State = 4 // Device is configured and operational
	StateSuspended  State = 5 // Device is in suspend mode
)

// State represents USB device state.
type State uint8

// String returns a human-readable state description.
func (s State) String() string {
	switch s {
	case StateAttached:
		return "Attached"
	case StatePowered:
		return "Powered"
	case StateDefault:
		return "Default"
	case StateAddress:
		return "Address"
	case StateConfigured:
		return "Configured"
	case StateSuspended:
		return "Suspended"
	default:
		return fmt.Sprintf("Unknown State (%d)", s)
	}
}
