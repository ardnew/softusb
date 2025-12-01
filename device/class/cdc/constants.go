package cdc

// CDC Class-specific descriptor types.
const (
	DescriptorTypeCSInterface = 0x24 // Class-specific Interface
	DescriptorTypeCSEndpoint  = 0x25 // Class-specific Endpoint
)

// CDC Functional Descriptor subtypes.
const (
	SubtypeHeader          = 0x00 // Header Functional Descriptor
	SubtypeCallManagement  = 0x01 // Call Management Functional Descriptor
	SubtypeACM             = 0x02 // Abstract Control Model Functional Descriptor
	SubtypeDLM             = 0x03 // Direct Line Management Functional Descriptor
	SubtypeTelephoneRinger = 0x04 // Telephone Ringer Functional Descriptor
	SubtypeTelephoneCall   = 0x05 // Telephone Call Functional Descriptor
	SubtypeUnion           = 0x06 // Union Functional Descriptor
	SubtypeCountrySelect   = 0x07 // Country Selection Functional Descriptor
	SubtypeTelephoneOpMode = 0x08 // Telephone Operational Modes Functional Descriptor
	SubtypeUSBTerminal     = 0x09 // USB Terminal Functional Descriptor
	SubtypeNetworkChannel  = 0x0A // Network Channel Terminal Functional Descriptor
	SubtypeProtocolUnit    = 0x0B // Protocol Unit Functional Descriptor
	SubtypeExtensionUnit   = 0x0C // Extension Unit Functional Descriptor
	SubtypeMCM             = 0x0D // Multi-Channel Management Functional Descriptor
	SubtypeCAPI            = 0x0E // CAPI Control Management Functional Descriptor
	SubtypeEthernet        = 0x0F // Ethernet Networking Functional Descriptor
	SubtypeATMNetworking   = 0x10 // ATM Networking Functional Descriptor
)

// CDC Class codes.
const (
	ClassCDC     = 0x02 // Communications Device Class
	ClassCDCData = 0x0A // CDC Data Class
)

// CDC Subclass codes.
const (
	SubclassNone = 0x00 // No subclass
	SubclassDLCM = 0x01 // Direct Line Control Model
	SubclassACM  = 0x02 // Abstract Control Model
	SubclassTCM  = 0x03 // Telephone Control Model
	SubclassMCCM = 0x04 // Multi-Channel Control Model
	SubclassCAPI = 0x05 // CAPI Control Model
	SubclassECM  = 0x06 // Ethernet Networking Control Model
	SubclassATM  = 0x07 // ATM Networking Control Model
)

// CDC Protocol codes.
const (
	ProtocolNone   = 0x00 // No protocol
	ProtocolAT     = 0x01 // AT Commands: V.250
	ProtocolVendor = 0xFF // Vendor-specific
)

// CDC Request codes.
const (
	RequestSendEncapsulatedCommand = 0x00
	RequestGetEncapsulatedResponse = 0x01
	RequestSetCommFeature          = 0x02
	RequestGetCommFeature          = 0x03
	RequestClearCommFeature        = 0x04
	RequestSetLineCoding           = 0x20
	RequestGetLineCoding           = 0x21
	RequestSetControlLineState     = 0x22
	RequestSendBreak               = 0x23
)

// CDC Notification codes.
const (
	NotificationNetworkConnection = 0x00
	NotificationResponseAvailable = 0x01
	NotificationSerialState       = 0x20
)

// LineCoding represents the serial line configuration.
type LineCoding struct {
	DTERate    uint32 // Data terminal rate (baud rate)
	CharFormat uint8  // Stop bits: 0=1, 1=1.5, 2=2
	ParityType uint8  // Parity: 0=None, 1=Odd, 2=Even, 3=Mark, 4=Space
	DataBits   uint8  // Data bits: 5, 6, 7, 8, or 16
}

// LineCodingSize is the size of LineCoding in bytes.
const LineCodingSize = 7

// Stop bit values.
const (
	StopBits1   = 0 // 1 stop bit
	StopBits1_5 = 1 // 1.5 stop bits
	StopBits2   = 2 // 2 stop bits
)

// Parity values.
const (
	ParityNone  = 0
	ParityOdd   = 1
	ParityEven  = 2
	ParityMark  = 3
	ParitySpace = 4
)

// Control line state bits (for SET_CONTROL_LINE_STATE).
const (
	ControlLineDTR = 1 << 0 // Data Terminal Ready
	ControlLineRTS = 1 << 1 // Request To Send
)

// Serial state bits (for SERIAL_STATE notification).
const (
	SerialStateRxCarrier  = 1 << 0 // DCD (Data Carrier Detect)
	SerialStateTxCarrier  = 1 << 1 // DSR (Data Set Ready)
	SerialStateBreak      = 1 << 2 // Break detected
	SerialStateRingSignal = 1 << 3 // Ring signal detected
	SerialStateFraming    = 1 << 4 // Framing error
	SerialStateParity     = 1 << 5 // Parity error
	SerialStateOverrun    = 1 << 6 // Overrun error
)

// DefaultLineCoding provides sensible defaults (115200 8N1).
var DefaultLineCoding = LineCoding{
	DTERate:    115200,
	CharFormat: StopBits1,
	ParityType: ParityNone,
	DataBits:   8,
}

// MarshalTo writes the LineCoding to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (lc *LineCoding) MarshalTo(buf []byte) int {
	if len(buf) < LineCodingSize {
		return 0
	}
	buf[0] = byte(lc.DTERate)
	buf[1] = byte(lc.DTERate >> 8)
	buf[2] = byte(lc.DTERate >> 16)
	buf[3] = byte(lc.DTERate >> 24)
	buf[4] = lc.CharFormat
	buf[5] = lc.ParityType
	buf[6] = lc.DataBits
	return LineCodingSize
}

// ParseLineCoding parses LineCoding from data.
// Returns false if data is too short.
func ParseLineCoding(data []byte, out *LineCoding) bool {
	if len(data) < LineCodingSize {
		return false
	}
	out.DTERate = uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	out.CharFormat = data[4]
	out.ParityType = data[5]
	out.DataBits = data[6]
	return true
}

// HeaderDescriptor is the CDC Header Functional Descriptor.
type HeaderDescriptor struct {
	Length         uint8  // Size of this descriptor (5)
	DescriptorType uint8  // CS_INTERFACE (0x24)
	SubType        uint8  // Header (0x00)
	CDCVersion     uint16 // CDC specification release number (0x0110 for 1.10)
}

// HeaderDescriptorSize is the size of the Header Functional Descriptor.
const HeaderDescriptorSize = 5

// MarshalTo writes the descriptor to buf.
func (d *HeaderDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < HeaderDescriptorSize {
		return 0
	}
	buf[0] = HeaderDescriptorSize
	buf[1] = DescriptorTypeCSInterface
	buf[2] = SubtypeHeader
	buf[3] = byte(d.CDCVersion)
	buf[4] = byte(d.CDCVersion >> 8)
	return HeaderDescriptorSize
}

// CallManagementDescriptor is the Call Management Functional Descriptor.
type CallManagementDescriptor struct {
	Length         uint8 // Size of this descriptor (5)
	DescriptorType uint8 // CS_INTERFACE (0x24)
	SubType        uint8 // Call Management (0x01)
	Capabilities   uint8 // Call management capabilities
	DataInterface  uint8 // Interface number of the Data Class interface
}

// CallManagementDescriptorSize is the size of the Call Management Descriptor.
const CallManagementDescriptorSize = 5

// Call management capability bits.
const (
	CallMgmtHandlesCallManagement = 1 << 0 // Device handles call management
	CallMgmtCallMgmtOverDataClass = 1 << 1 // Call management over Data Class interface
)

// MarshalTo writes the descriptor to buf.
func (d *CallManagementDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < CallManagementDescriptorSize {
		return 0
	}
	buf[0] = CallManagementDescriptorSize
	buf[1] = DescriptorTypeCSInterface
	buf[2] = SubtypeCallManagement
	buf[3] = d.Capabilities
	buf[4] = d.DataInterface
	return CallManagementDescriptorSize
}

// ACMDescriptor is the Abstract Control Management Functional Descriptor.
type ACMDescriptor struct {
	Length         uint8 // Size of this descriptor (4)
	DescriptorType uint8 // CS_INTERFACE (0x24)
	SubType        uint8 // ACM (0x02)
	Capabilities   uint8 // ACM capabilities
}

// ACMDescriptorSize is the size of the ACM Functional Descriptor.
const ACMDescriptorSize = 4

// ACM capability bits.
const (
	ACMCapCommFeature = 1 << 0 // Supports Set/Get/Clear Comm Feature
	ACMCapLineCoding  = 1 << 1 // Supports Set/Get Line Coding and Set Control Line State
	ACMCapSendBreak   = 1 << 2 // Supports Send Break
	ACMCapNetworkConn = 1 << 3 // Supports Network Connection notification
)

// MarshalTo writes the descriptor to buf.
func (d *ACMDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < ACMDescriptorSize {
		return 0
	}
	buf[0] = ACMDescriptorSize
	buf[1] = DescriptorTypeCSInterface
	buf[2] = SubtypeACM
	buf[3] = d.Capabilities
	return ACMDescriptorSize
}

// UnionDescriptor is the Union Functional Descriptor.
type UnionDescriptor struct {
	Length          uint8 // Size of this descriptor (5 for 1 subordinate)
	DescriptorType  uint8 // CS_INTERFACE (0x24)
	SubType         uint8 // Union (0x06)
	MasterInterface uint8 // Control interface number
	SlaveInterface0 uint8 // First subordinate interface (Data interface)
}

// UnionDescriptorSize is the size of the Union Descriptor with one subordinate.
const UnionDescriptorSize = 5

// MarshalTo writes the descriptor to buf.
func (d *UnionDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < UnionDescriptorSize {
		return 0
	}
	buf[0] = UnionDescriptorSize
	buf[1] = DescriptorTypeCSInterface
	buf[2] = SubtypeUnion
	buf[3] = d.MasterInterface
	buf[4] = d.SlaveInterface0
	return UnionDescriptorSize
}
