package device

import (
	"encoding/binary"
	"fmt"

	"github.com/ardnew/softusb/pkg"
)

// Standard USB request codes (USB 2.0 Spec Table 9-4).
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

// Feature selectors (USB 2.0 Spec Table 9-6).
const (
	FeatureEndpointHalt       = 0x00 // Endpoint halt feature
	FeatureDeviceRemoteWakeup = 0x01 // Device remote wakeup
	FeatureTestMode           = 0x02 // Test mode
)

// Request type masks (USB 2.0 Spec Table 9-2).
const (
	RequestTypeDirectionMask = 0x80 // Direction bit mask
	RequestTypeTypeMask      = 0x60 // Type bits mask
	RequestTypeRecipientMask = 0x1F // Recipient bits mask
)

// Request type direction values.
const (
	RequestDirectionHostToDevice = 0x00 // Host to device
	RequestDirectionDeviceToHost = 0x80 // Device to host
)

// Request type values.
const (
	RequestTypeStandard = 0x00 // Standard request
	RequestTypeClass    = 0x20 // Class-specific request
	RequestTypeVendor   = 0x40 // Vendor-specific request
)

// Request recipient values.
const (
	RequestRecipientDevice    = 0x00 // Device recipient
	RequestRecipientInterface = 0x01 // Interface recipient
	RequestRecipientEndpoint  = 0x02 // Endpoint recipient
	RequestRecipientOther     = 0x03 // Other recipient
)

// SetupPacket represents an 8-byte USB SETUP packet.
type SetupPacket struct {
	RequestType uint8  // bmRequestType: direction, type, recipient
	Request     uint8  // bRequest: specific request code
	Value       uint16 // wValue: request-specific parameter
	Index       uint16 // wIndex: request-specific index
	Length      uint16 // wLength: number of bytes to transfer
}

// SetupPacketSize is the size of a USB SETUP packet in bytes.
const SetupPacketSize = 8

// ParseSetupPacket parses a setup packet from 8 bytes into out.
// Returns an error if the data is too short.
func ParseSetupPacket(data []byte, out *SetupPacket) error {
	if len(data) < SetupPacketSize {
		return pkg.ErrSetupPacketTooShort
	}
	out.RequestType = data[0]
	out.Request = data[1]
	out.Value = binary.LittleEndian.Uint16(data[2:4])
	out.Index = binary.LittleEndian.Uint16(data[4:6])
	out.Length = binary.LittleEndian.Uint16(data[6:8])
	return nil
}

// MarshalTo serializes the setup packet to buf.
// Returns the number of bytes written (always 8 if buf is large enough).
func (s *SetupPacket) MarshalTo(buf []byte) int {
	if len(buf) < SetupPacketSize {
		return 0
	}
	buf[0] = s.RequestType
	buf[1] = s.Request
	binary.LittleEndian.PutUint16(buf[2:4], s.Value)
	binary.LittleEndian.PutUint16(buf[4:6], s.Index)
	binary.LittleEndian.PutUint16(buf[6:8], s.Length)
	return SetupPacketSize
}

// Direction returns the transfer direction.
func (s *SetupPacket) Direction() uint8 {
	return s.RequestType & RequestTypeDirectionMask
}

// IsDeviceToHost returns true if this is a device-to-host transfer.
func (s *SetupPacket) IsDeviceToHost() bool {
	return s.Direction() == RequestDirectionDeviceToHost
}

// IsHostToDevice returns true if this is a host-to-device transfer.
func (s *SetupPacket) IsHostToDevice() bool {
	return s.Direction() == RequestDirectionHostToDevice
}

// Type returns the request type (Standard, Class, or Vendor).
func (s *SetupPacket) Type() uint8 {
	return s.RequestType & RequestTypeTypeMask
}

// IsStandard returns true if this is a standard request.
func (s *SetupPacket) IsStandard() bool {
	return s.Type() == RequestTypeStandard
}

// IsClass returns true if this is a class-specific request.
func (s *SetupPacket) IsClass() bool {
	return s.Type() == RequestTypeClass
}

// IsVendor returns true if this is a vendor-specific request.
func (s *SetupPacket) IsVendor() bool {
	return s.Type() == RequestTypeVendor
}

// Recipient returns the request recipient.
func (s *SetupPacket) Recipient() uint8 {
	return s.RequestType & RequestTypeRecipientMask
}

// IsDeviceRecipient returns true if the recipient is the device.
func (s *SetupPacket) IsDeviceRecipient() bool {
	return s.Recipient() == RequestRecipientDevice
}

// IsInterfaceRecipient returns true if the recipient is an interface.
func (s *SetupPacket) IsInterfaceRecipient() bool {
	return s.Recipient() == RequestRecipientInterface
}

// IsEndpointRecipient returns true if the recipient is an endpoint.
func (s *SetupPacket) IsEndpointRecipient() bool {
	return s.Recipient() == RequestRecipientEndpoint
}

// DescriptorType returns the descriptor type from wValue high byte.
func (s *SetupPacket) DescriptorType() uint8 {
	return uint8(s.Value >> 8)
}

// DescriptorIndex returns the descriptor index from wValue low byte.
func (s *SetupPacket) DescriptorIndex() uint8 {
	return uint8(s.Value & 0xFF)
}

// InterfaceNumber returns the interface number from wIndex.
func (s *SetupPacket) InterfaceNumber() uint8 {
	return uint8(s.Index & 0xFF)
}

// EndpointAddress returns the endpoint address from wIndex.
func (s *SetupPacket) EndpointAddress() uint8 {
	return uint8(s.Index & 0xFF)
}

// String returns a human-readable representation of the setup packet.
func (s *SetupPacket) String() string {
	dir := "OUT"
	if s.IsDeviceToHost() {
		dir = "IN"
	}
	reqType := "Standard"
	switch s.Type() {
	case RequestTypeClass:
		reqType = "Class"
	case RequestTypeVendor:
		reqType = "Vendor"
	}
	recip := "Device"
	switch s.Recipient() {
	case RequestRecipientInterface:
		recip = "Interface"
	case RequestRecipientEndpoint:
		recip = "Endpoint"
	case RequestRecipientOther:
		recip = "Other"
	}
	return fmt.Sprintf("SETUP[%s %s %s] Request=0x%02X Value=0x%04X Index=0x%04X Length=%d",
		dir, reqType, recip, s.Request, s.Value, s.Index, s.Length)
}

// GetDescriptorSetup initializes out as a GET_DESCRIPTOR setup packet.
func GetDescriptorSetup(out *SetupPacket, descType, descIndex uint8, length uint16) {
	out.RequestType = RequestDirectionDeviceToHost | RequestTypeStandard | RequestRecipientDevice
	out.Request = RequestGetDescriptor
	out.Value = uint16(descType)<<8 | uint16(descIndex)
	out.Index = 0
	out.Length = length
}

// GetSetAddressSetup initializes out as a SET_ADDRESS setup packet.
func GetSetAddressSetup(out *SetupPacket, address uint8) {
	out.RequestType = RequestDirectionHostToDevice | RequestTypeStandard | RequestRecipientDevice
	out.Request = RequestSetAddress
	out.Value = uint16(address)
	out.Index = 0
	out.Length = 0
}

// GetSetConfigurationSetup initializes out as a SET_CONFIGURATION setup packet.
func GetSetConfigurationSetup(out *SetupPacket, config uint8) {
	out.RequestType = RequestDirectionHostToDevice | RequestTypeStandard | RequestRecipientDevice
	out.Request = RequestSetConfiguration
	out.Value = uint16(config)
	out.Index = 0
	out.Length = 0
}

// GetConfigurationSetup initializes out as a GET_CONFIGURATION setup packet.
func GetConfigurationSetup(out *SetupPacket) {
	out.RequestType = RequestDirectionDeviceToHost | RequestTypeStandard | RequestRecipientDevice
	out.Request = RequestGetConfiguration
	out.Value = 0
	out.Index = 0
	out.Length = 1
}

// GetStatusSetup initializes out as a GET_STATUS setup packet.
func GetStatusSetup(out *SetupPacket, recipient uint8, index uint16) {
	out.RequestType = RequestDirectionDeviceToHost | RequestTypeStandard | recipient
	out.Request = RequestGetStatus
	out.Value = 0
	out.Index = index
	out.Length = 2
}

// GetSetFeatureSetup initializes out as a SET_FEATURE setup packet.
func GetSetFeatureSetup(out *SetupPacket, recipient uint8, feature uint16, index uint16) {
	out.RequestType = RequestDirectionHostToDevice | RequestTypeStandard | recipient
	out.Request = RequestSetFeature
	out.Value = feature
	out.Index = index
	out.Length = 0
}

// GetClearFeatureSetup initializes out as a CLEAR_FEATURE setup packet.
func GetClearFeatureSetup(out *SetupPacket, recipient uint8, feature uint16, index uint16) {
	out.RequestType = RequestDirectionHostToDevice | RequestTypeStandard | recipient
	out.Request = RequestClearFeature
	out.Value = feature
	out.Index = index
	out.Length = 0
}

// GetSetInterfaceSetup initializes out as a SET_INTERFACE setup packet.
func GetSetInterfaceSetup(out *SetupPacket, interfaceNum, alternateSetting uint8) {
	out.RequestType = RequestDirectionHostToDevice | RequestTypeStandard | RequestRecipientInterface
	out.Request = RequestSetInterface
	out.Value = uint16(alternateSetting)
	out.Index = uint16(interfaceNum)
	out.Length = 0
}

// GetInterfaceSetup initializes out as a GET_INTERFACE setup packet.
func GetInterfaceSetup(out *SetupPacket, interfaceNum uint8) {
	out.RequestType = RequestDirectionDeviceToHost | RequestTypeStandard | RequestRecipientInterface
	out.Request = RequestGetInterface
	out.Value = 0
	out.Index = uint16(interfaceNum)
	out.Length = 1
}
