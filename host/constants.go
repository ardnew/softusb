package host

import "fmt"

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

// Device states as defined in USB 2.0 specification.
const (
	DeviceStateDetached   DeviceState = 0 // Device is not connected
	DeviceStateAttached   DeviceState = 1 // Device is attached but not addressed
	DeviceStateDefault    DeviceState = 2 // Device has been reset, at address 0
	DeviceStateAddress    DeviceState = 3 // Device has been assigned an address
	DeviceStateConfigured DeviceState = 4 // Device is configured
	DeviceStateSuspended  DeviceState = 5 // Device is in suspend mode
)

// DeviceState represents USB device state (from host perspective).
type DeviceState uint8

// String returns a human-readable state description.
func (s DeviceState) String() string {
	switch s {
	case DeviceStateDetached:
		return "Detached"
	case DeviceStateAttached:
		return "Attached"
	case DeviceStateDefault:
		return "Default"
	case DeviceStateAddress:
		return "Address"
	case DeviceStateConfigured:
		return "Configured"
	case DeviceStateSuspended:
		return "Suspended"
	default:
		return fmt.Sprintf("Unknown State (%d)", s)
	}
}

// Maximum limits for fixed-size arrays.
const (
	// MaxDevices is the maximum number of devices on the bus.
	MaxDevices = 16

	// MaxConfigurationsPerDevice is the maximum configurations per device.
	MaxConfigurationsPerDevice = 4

	// MaxInterfacesPerConfiguration is the maximum interfaces per configuration.
	MaxInterfacesPerConfiguration = 8

	// MaxEndpointsPerInterface is the maximum endpoints per interface.
	MaxEndpointsPerInterface = 16

	// MaxStringsPerDevice is the maximum string descriptors per device.
	MaxStringsPerDevice = 16

	// MaxDescriptorSize is the maximum size for descriptor buffers.
	MaxDescriptorSize = 512

	// MaxControlDataSize is the maximum data size for control transfers.
	MaxControlDataSize = 512
)

// Endpoint transfer types.
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

// Descriptor types.
const (
	DescriptorTypeDevice               = 0x01
	DescriptorTypeConfiguration        = 0x02
	DescriptorTypeString               = 0x03
	DescriptorTypeInterface            = 0x04
	DescriptorTypeEndpoint             = 0x05
	DescriptorTypeDeviceQualifier      = 0x06
	DescriptorTypeOtherSpeedConfig     = 0x07
	DescriptorTypeInterfacePower       = 0x08
	DescriptorTypeOTG                  = 0x09
	DescriptorTypeDebug                = 0x0A
	DescriptorTypeInterfaceAssociation = 0x0B
)

// Standard request codes.
const (
	RequestGetStatus        = 0x00
	RequestClearFeature     = 0x01
	RequestSetFeature       = 0x03
	RequestSetAddress       = 0x05
	RequestGetDescriptor    = 0x06
	RequestSetDescriptor    = 0x07
	RequestGetConfiguration = 0x08
	RequestSetConfiguration = 0x09
	RequestGetInterface     = 0x0A
	RequestSetInterface     = 0x0B
	RequestSynchFrame       = 0x0C
)

// Request types (bmRequestType).
const (
	RequestTypeOut       = 0x00 // Host to device
	RequestTypeIn        = 0x80 // Device to host
	RequestTypeStandard  = 0x00 // Standard request
	RequestTypeClass     = 0x20 // Class-specific request
	RequestTypeVendor    = 0x40 // Vendor-specific request
	RequestTypeDevice    = 0x00 // Recipient: device
	RequestTypeInterface = 0x01 // Recipient: interface
	RequestTypeEndpoint  = 0x02 // Recipient: endpoint
	RequestTypeOther     = 0x03 // Recipient: other
)

// LangIDUSEnglish is the default language ID.
const LangIDUSEnglish = 0x0409

// DeviceDescriptor represents a USB device descriptor.
type DeviceDescriptor struct {
	Length            uint8
	DescriptorType    uint8
	USBVersion        uint16
	DeviceClass       uint8
	DeviceSubClass    uint8
	DeviceProtocol    uint8
	MaxPacketSize0    uint8
	VendorID          uint16
	ProductID         uint16
	DeviceVersion     uint16
	ManufacturerIndex uint8
	ProductIndex      uint8
	SerialNumberIndex uint8
	NumConfigurations uint8
}

// DeviceDescriptorSize is the size of a device descriptor.
const DeviceDescriptorSize = 18

// ParseDeviceDescriptor parses device descriptor from data.
func ParseDeviceDescriptor(data []byte, out *DeviceDescriptor) bool {
	if len(data) < DeviceDescriptorSize {
		return false
	}
	out.Length = data[0]
	out.DescriptorType = data[1]
	out.USBVersion = uint16(data[2]) | uint16(data[3])<<8
	out.DeviceClass = data[4]
	out.DeviceSubClass = data[5]
	out.DeviceProtocol = data[6]
	out.MaxPacketSize0 = data[7]
	out.VendorID = uint16(data[8]) | uint16(data[9])<<8
	out.ProductID = uint16(data[10]) | uint16(data[11])<<8
	out.DeviceVersion = uint16(data[12]) | uint16(data[13])<<8
	out.ManufacturerIndex = data[14]
	out.ProductIndex = data[15]
	out.SerialNumberIndex = data[16]
	out.NumConfigurations = data[17]
	return true
}

// ConfigurationDescriptor represents a USB configuration descriptor.
type ConfigurationDescriptor struct {
	Length             uint8
	DescriptorType     uint8
	TotalLength        uint16
	NumInterfaces      uint8
	ConfigurationValue uint8
	ConfigurationIndex uint8
	Attributes         uint8
	MaxPower           uint8
}

// ConfigurationDescriptorSize is the size of a configuration descriptor header.
const ConfigurationDescriptorSize = 9

// ParseConfigurationDescriptor parses configuration descriptor from data.
func ParseConfigurationDescriptor(data []byte, out *ConfigurationDescriptor) bool {
	if len(data) < ConfigurationDescriptorSize {
		return false
	}
	out.Length = data[0]
	out.DescriptorType = data[1]
	out.TotalLength = uint16(data[2]) | uint16(data[3])<<8
	out.NumInterfaces = data[4]
	out.ConfigurationValue = data[5]
	out.ConfigurationIndex = data[6]
	out.Attributes = data[7]
	out.MaxPower = data[8]
	return true
}

// InterfaceDescriptor represents a USB interface descriptor.
type InterfaceDescriptor struct {
	Length            uint8
	DescriptorType    uint8
	InterfaceNumber   uint8
	AlternateSetting  uint8
	NumEndpoints      uint8
	InterfaceClass    uint8
	InterfaceSubClass uint8
	InterfaceProtocol uint8
	InterfaceIndex    uint8
}

// InterfaceDescriptorSize is the size of an interface descriptor.
const InterfaceDescriptorSize = 9

// ParseInterfaceDescriptor parses interface descriptor from data.
func ParseInterfaceDescriptor(data []byte, out *InterfaceDescriptor) bool {
	if len(data) < InterfaceDescriptorSize {
		return false
	}
	out.Length = data[0]
	out.DescriptorType = data[1]
	out.InterfaceNumber = data[2]
	out.AlternateSetting = data[3]
	out.NumEndpoints = data[4]
	out.InterfaceClass = data[5]
	out.InterfaceSubClass = data[6]
	out.InterfaceProtocol = data[7]
	out.InterfaceIndex = data[8]
	return true
}

// EndpointDescriptor represents a USB endpoint descriptor.
type EndpointDescriptor struct {
	Length          uint8
	DescriptorType  uint8
	EndpointAddress uint8
	Attributes      uint8
	MaxPacketSize   uint16
	Interval        uint8
}

// EndpointDescriptorSize is the size of an endpoint descriptor.
const EndpointDescriptorSize = 7

// ParseEndpointDescriptor parses endpoint descriptor from data.
func ParseEndpointDescriptor(data []byte, out *EndpointDescriptor) bool {
	if len(data) < EndpointDescriptorSize {
		return false
	}
	out.Length = data[0]
	out.DescriptorType = data[1]
	out.EndpointAddress = data[2]
	out.Attributes = data[3]
	out.MaxPacketSize = uint16(data[4]) | uint16(data[5])<<8
	out.Interval = data[6]
	return true
}

// Number returns the endpoint number (0-15).
func (e *EndpointDescriptor) Number() uint8 {
	return e.EndpointAddress & 0x0F
}

// Direction returns the endpoint direction.
func (e *EndpointDescriptor) Direction() uint8 {
	return e.EndpointAddress & 0x80
}

// IsIn returns true if this is an IN endpoint.
func (e *EndpointDescriptor) IsIn() bool {
	return e.Direction() == EndpointDirectionIn
}

// IsOut returns true if this is an OUT endpoint.
func (e *EndpointDescriptor) IsOut() bool {
	return e.Direction() == EndpointDirectionOut
}

// TransferType returns the transfer type.
func (e *EndpointDescriptor) TransferType() uint8 {
	return e.Attributes & 0x03
}

// IsControl returns true if this is a control endpoint.
func (e *EndpointDescriptor) IsControl() bool {
	return e.TransferType() == EndpointTypeControl
}

// IsBulk returns true if this is a bulk endpoint.
func (e *EndpointDescriptor) IsBulk() bool {
	return e.TransferType() == EndpointTypeBulk
}

// IsInterrupt returns true if this is an interrupt endpoint.
func (e *EndpointDescriptor) IsInterrupt() bool {
	return e.TransferType() == EndpointTypeInterrupt
}

// IsIsochronous returns true if this is an isochronous endpoint.
func (e *EndpointDescriptor) IsIsochronous() bool {
	return e.TransferType() == EndpointTypeIsochronous
}
