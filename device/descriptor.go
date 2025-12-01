package device

import (
	"encoding/binary"

	"github.com/ardnew/softusb/pkg"
)

// USB Descriptor Types (USB 2.0 Spec Table 9-5).
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
	DescriptorTypeBOS                  = 0x0F
	DescriptorTypeDeviceCapability     = 0x10
	DescriptorTypeHID                  = 0x21
	DescriptorTypeHIDReport            = 0x22
	DescriptorTypeHIDPhysical          = 0x23
	DescriptorTypeCSInterface          = 0x24 // Class-specific interface
	DescriptorTypeCSEndpoint           = 0x25 // Class-specific endpoint
)

// USB Class Codes.
const (
	ClassPerInterface = 0x00 // Class defined at interface level
	ClassAudio        = 0x01 // Audio class
	ClassCDC          = 0x02 // Communications Device Class
	ClassHID          = 0x03 // Human Interface Device
	ClassPhysical     = 0x05 // Physical
	ClassImage        = 0x06 // Still Imaging
	ClassPrinter      = 0x07 // Printer
	ClassMassStorage  = 0x08 // Mass Storage
	ClassHub          = 0x09 // Hub
	ClassCDCData      = 0x0A // CDC-Data
	ClassSmartCard    = 0x0B // Smart Card
	ClassContentSec   = 0x0D // Content Security
	ClassVideo        = 0x0E // Video
	ClassHealthcare   = 0x0F // Personal Healthcare
	ClassAudioVideo   = 0x10 // Audio/Video Devices
	ClassBillboard    = 0x11 // Billboard Device Class
	ClassDiagnostic   = 0xDC // Diagnostic Device
	ClassWireless     = 0xE0 // Wireless Controller
	ClassMisc         = 0xEF // Miscellaneous
	ClassAppSpecific  = 0xFE // Application Specific
	ClassVendor       = 0xFF // Vendor Specific
)

// DeviceDescriptor represents a USB device descriptor (18 bytes).
type DeviceDescriptor struct {
	Length            uint8  // Size of this descriptor (18)
	DescriptorType    uint8  // Device descriptor type (0x01)
	USBVersion        uint16 // USB specification version (BCD)
	DeviceClass       uint8  // Class code
	DeviceSubClass    uint8  // Subclass code
	DeviceProtocol    uint8  // Protocol code
	MaxPacketSize0    uint8  // Max packet size for EP0
	VendorID          uint16 // Vendor ID
	ProductID         uint16 // Product ID
	DeviceVersion     uint16 // Device release number (BCD)
	ManufacturerIndex uint8  // Index of manufacturer string
	ProductIndex      uint8  // Index of product string
	SerialNumberIndex uint8  // Index of serial number string
	NumConfigurations uint8  // Number of configurations
}

// DeviceDescriptorSize is the size of a device descriptor in bytes.
const DeviceDescriptorSize = 18

// MarshalTo serializes the device descriptor to buf.
// Returns the number of bytes written (always 18 if buf is large enough).
func (d *DeviceDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < DeviceDescriptorSize {
		return 0
	}
	buf[0] = DeviceDescriptorSize
	buf[1] = DescriptorTypeDevice
	binary.LittleEndian.PutUint16(buf[2:4], d.USBVersion)
	buf[4] = d.DeviceClass
	buf[5] = d.DeviceSubClass
	buf[6] = d.DeviceProtocol
	buf[7] = d.MaxPacketSize0
	binary.LittleEndian.PutUint16(buf[8:10], d.VendorID)
	binary.LittleEndian.PutUint16(buf[10:12], d.ProductID)
	binary.LittleEndian.PutUint16(buf[12:14], d.DeviceVersion)
	buf[14] = d.ManufacturerIndex
	buf[15] = d.ProductIndex
	buf[16] = d.SerialNumberIndex
	buf[17] = d.NumConfigurations
	return DeviceDescriptorSize
}

// ParseDeviceDescriptor parses a device descriptor from bytes into out.
// Returns an error if the data is too short or the descriptor type is wrong.
func ParseDeviceDescriptor(data []byte, out *DeviceDescriptor) error {
	if len(data) < DeviceDescriptorSize {
		return pkg.ErrDescriptorTooShort
	}
	if data[1] != DescriptorTypeDevice {
		return pkg.ErrDescriptorTypeMismatch
	}
	out.Length = data[0]
	out.DescriptorType = data[1]
	out.USBVersion = binary.LittleEndian.Uint16(data[2:4])
	out.DeviceClass = data[4]
	out.DeviceSubClass = data[5]
	out.DeviceProtocol = data[6]
	out.MaxPacketSize0 = data[7]
	out.VendorID = binary.LittleEndian.Uint16(data[8:10])
	out.ProductID = binary.LittleEndian.Uint16(data[10:12])
	out.DeviceVersion = binary.LittleEndian.Uint16(data[12:14])
	out.ManufacturerIndex = data[14]
	out.ProductIndex = data[15]
	out.SerialNumberIndex = data[16]
	out.NumConfigurations = data[17]
	return nil
}

// ConfigurationDescriptor represents a USB configuration descriptor (9 bytes).
type ConfigurationDescriptor struct {
	Length             uint8  // Size of this descriptor (9)
	DescriptorType     uint8  // Configuration descriptor type (0x02)
	TotalLength        uint16 // Total length of configuration data
	NumInterfaces      uint8  // Number of interfaces
	ConfigurationValue uint8  // Configuration value for SET_CONFIGURATION
	ConfigurationIndex uint8  // Index of string descriptor
	Attributes         uint8  // Configuration attributes
	MaxPower           uint8  // Maximum power consumption (2mA units)
}

// Configuration attribute bits.
const (
	ConfigAttrBusPowered   = 0x80 // Bus-powered (required)
	ConfigAttrSelfPowered  = 0x40 // Self-powered
	ConfigAttrRemoteWakeup = 0x20 // Remote wakeup capable
)

// ConfigurationDescriptorSize is the size of a configuration descriptor in bytes.
const ConfigurationDescriptorSize = 9

// MarshalTo serializes the configuration descriptor to buf.
// Returns the number of bytes written (always 9 if buf is large enough).
func (c *ConfigurationDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < ConfigurationDescriptorSize {
		return 0
	}
	buf[0] = ConfigurationDescriptorSize
	buf[1] = DescriptorTypeConfiguration
	binary.LittleEndian.PutUint16(buf[2:4], c.TotalLength)
	buf[4] = c.NumInterfaces
	buf[5] = c.ConfigurationValue
	buf[6] = c.ConfigurationIndex
	buf[7] = c.Attributes
	buf[8] = c.MaxPower
	return ConfigurationDescriptorSize
}

// ParseConfigurationDescriptor parses a configuration descriptor from bytes into out.
// Returns an error if the data is too short or the descriptor type is wrong.
func ParseConfigurationDescriptor(data []byte, out *ConfigurationDescriptor) error {
	if len(data) < ConfigurationDescriptorSize {
		return pkg.ErrDescriptorTooShort
	}
	if data[1] != DescriptorTypeConfiguration {
		return pkg.ErrDescriptorTypeMismatch
	}
	out.Length = data[0]
	out.DescriptorType = data[1]
	out.TotalLength = binary.LittleEndian.Uint16(data[2:4])
	out.NumInterfaces = data[4]
	out.ConfigurationValue = data[5]
	out.ConfigurationIndex = data[6]
	out.Attributes = data[7]
	out.MaxPower = data[8]
	return nil
}

// InterfaceDescriptor represents a USB interface descriptor (9 bytes).
type InterfaceDescriptor struct {
	Length            uint8 // Size of this descriptor (9)
	DescriptorType    uint8 // Interface descriptor type (0x04)
	InterfaceNumber   uint8 // Interface number
	AlternateSetting  uint8 // Alternate setting number
	NumEndpoints      uint8 // Number of endpoints (excluding EP0)
	InterfaceClass    uint8 // Class code
	InterfaceSubClass uint8 // Subclass code
	InterfaceProtocol uint8 // Protocol code
	InterfaceIndex    uint8 // Index of string descriptor
}

// InterfaceDescriptorSize is the size of an interface descriptor in bytes.
const InterfaceDescriptorSize = 9

// MarshalTo serializes the interface descriptor to buf.
// Returns the number of bytes written (always 9 if buf is large enough).
func (i *InterfaceDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < InterfaceDescriptorSize {
		return 0
	}
	buf[0] = InterfaceDescriptorSize
	buf[1] = DescriptorTypeInterface
	buf[2] = i.InterfaceNumber
	buf[3] = i.AlternateSetting
	buf[4] = i.NumEndpoints
	buf[5] = i.InterfaceClass
	buf[6] = i.InterfaceSubClass
	buf[7] = i.InterfaceProtocol
	buf[8] = i.InterfaceIndex
	return InterfaceDescriptorSize
}

// ParseInterfaceDescriptor parses an interface descriptor from bytes into out.
// Returns an error if the data is too short or the descriptor type is wrong.
func ParseInterfaceDescriptor(data []byte, out *InterfaceDescriptor) error {
	if len(data) < InterfaceDescriptorSize {
		return pkg.ErrDescriptorTooShort
	}
	if data[1] != DescriptorTypeInterface {
		return pkg.ErrDescriptorTypeMismatch
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
	return nil
}

// EndpointDescriptor represents a USB endpoint descriptor (7 bytes).
type EndpointDescriptor struct {
	Length          uint8  // Size of this descriptor (7)
	DescriptorType  uint8  // Endpoint descriptor type (0x05)
	EndpointAddress uint8  // Endpoint address (including direction)
	Attributes      uint8  // Endpoint attributes (transfer type, etc.)
	MaxPacketSize   uint16 // Maximum packet size
	Interval        uint8  // Polling interval (for interrupt/isochronous)
}

// EndpointDescriptorSize is the size of an endpoint descriptor in bytes.
const EndpointDescriptorSize = 7

// MarshalTo serializes the endpoint descriptor to buf.
// Returns the number of bytes written (always 7 if buf is large enough).
func (e *EndpointDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < EndpointDescriptorSize {
		return 0
	}
	buf[0] = EndpointDescriptorSize
	buf[1] = DescriptorTypeEndpoint
	buf[2] = e.EndpointAddress
	buf[3] = e.Attributes
	binary.LittleEndian.PutUint16(buf[4:6], e.MaxPacketSize)
	buf[6] = e.Interval
	return EndpointDescriptorSize
}

// ParseEndpointDescriptor parses an endpoint descriptor from bytes into out.
// Returns an error if the data is too short or the descriptor type is wrong.
func ParseEndpointDescriptor(data []byte, out *EndpointDescriptor) error {
	if len(data) < EndpointDescriptorSize {
		return pkg.ErrDescriptorTooShort
	}
	if data[1] != DescriptorTypeEndpoint {
		return pkg.ErrDescriptorTypeMismatch
	}
	out.Length = data[0]
	out.DescriptorType = data[1]
	out.EndpointAddress = data[2]
	out.Attributes = data[3]
	out.MaxPacketSize = binary.LittleEndian.Uint16(data[4:6])
	out.Interval = data[6]
	return nil
}

// InterfaceAssociationDescriptor represents an IAD (8 bytes).
// Used for composite devices like CDC-ACM.
type InterfaceAssociationDescriptor struct {
	Length           uint8 // Size of this descriptor (8)
	DescriptorType   uint8 // IAD type (0x0B)
	FirstInterface   uint8 // First interface number
	InterfaceCount   uint8 // Number of contiguous interfaces
	FunctionClass    uint8 // Class code
	FunctionSubClass uint8 // Subclass code
	FunctionProtocol uint8 // Protocol code
	FunctionIndex    uint8 // Index of string descriptor
}

// IADSize is the size of an interface association descriptor in bytes.
const IADSize = 8

// MarshalTo serializes the IAD to buf.
// Returns the number of bytes written (always 8 if buf is large enough).
func (i *InterfaceAssociationDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < IADSize {
		return 0
	}
	buf[0] = IADSize
	buf[1] = DescriptorTypeInterfaceAssociation
	buf[2] = i.FirstInterface
	buf[3] = i.InterfaceCount
	buf[4] = i.FunctionClass
	buf[5] = i.FunctionSubClass
	buf[6] = i.FunctionProtocol
	buf[7] = i.FunctionIndex
	return IADSize
}

// StringDescriptorTo writes a USB string descriptor to buf.
// Returns the number of bytes written. The descriptor encodes the string
// as UTF-16LE. If buf is too small, returns 0.
func StringDescriptorTo(buf []byte, s string) int {
	runes := []rune(s)
	length := 2 + len(runes)*2
	if length > 255 {
		length = 255
		runes = runes[:(length-2)/2]
	}
	if len(buf) < length {
		return 0
	}
	buf[0] = uint8(length)
	buf[1] = DescriptorTypeString
	for i, r := range runes {
		binary.LittleEndian.PutUint16(buf[2+i*2:], uint16(r))
	}
	return length
}

// LanguageDescriptorTo writes the language ID string descriptor to buf.
// Standard language ID for US English is 0x0409.
// Returns the number of bytes written. If buf is too small, returns 0.
func LanguageDescriptorTo(buf []byte, langIDs ...uint16) int {
	length := 2 + len(langIDs)*2
	if len(buf) < length {
		return 0
	}
	buf[0] = uint8(length)
	buf[1] = DescriptorTypeString
	for i, id := range langIDs {
		binary.LittleEndian.PutUint16(buf[2+i*2:], id)
	}
	return length
}

// LangIDUSEnglish is the language ID for US English.
const LangIDUSEnglish = 0x0409
