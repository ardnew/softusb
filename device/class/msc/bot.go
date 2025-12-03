package msc

import "encoding/binary"

// CommandBlockWrapper represents a Command Block Wrapper in Bulk-Only Transport.
type CommandBlockWrapper struct {
	Signature          uint32   // Must be CBWSignature (0x43425355)
	Tag                uint32   // Command block tag
	DataTransferLength uint32   // Number of bytes to transfer in data phase
	Flags              uint8    // Direction flag (bit 7: 0=Out, 1=In)
	LUN                uint8    // Logical Unit Number (bits 0-3)
	CBLength           uint8    // Command block length (1-16)
	CB                 [16]byte // Command block (SCSI CDB)
}

// ParseCBW parses a Command Block Wrapper from raw bytes.
// Returns false if data is too short or signature is invalid.
func ParseCBW(data []byte, out *CommandBlockWrapper) bool {
	if len(data) < CBWSize {
		return false
	}

	out.Signature = binary.LittleEndian.Uint32(data[0:4])
	if out.Signature != CBWSignature {
		return false
	}

	out.Tag = binary.LittleEndian.Uint32(data[4:8])
	out.DataTransferLength = binary.LittleEndian.Uint32(data[8:12])
	out.Flags = data[12]
	out.LUN = data[13] & 0x0F // Only bits 0-3
	out.CBLength = data[14] & 0x1F // Only bits 0-4
	copy(out.CB[:], data[15:31])

	return true
}

// IsDataIn returns true if the data phase is device-to-host (IN).
func (cbw *CommandBlockWrapper) IsDataIn() bool {
	return cbw.Flags&CBWFlagDataIn != 0
}

// IsDataOut returns true if the data phase is host-to-device (OUT).
func (cbw *CommandBlockWrapper) IsDataOut() bool {
	return cbw.Flags&CBWFlagDataIn == 0
}

// CommandStatusWrapper represents a Command Status Wrapper in Bulk-Only Transport.
type CommandStatusWrapper struct {
	Signature   uint32 // Must be CSWSignature (0x53425355)
	Tag         uint32 // Must match the CBW tag
	DataResidue uint32 // Difference between expected and actual data transfer
	Status      uint8  // Command status (CSWStatus*)
}

// MarshalTo writes the Command Status Wrapper to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (csw *CommandStatusWrapper) MarshalTo(buf []byte) int {
	if len(buf) < CSWSize {
		return 0
	}

	binary.LittleEndian.PutUint32(buf[0:4], csw.Signature)
	binary.LittleEndian.PutUint32(buf[4:8], csw.Tag)
	binary.LittleEndian.PutUint32(buf[8:12], csw.DataResidue)
	buf[12] = csw.Status

	return CSWSize
}

// NewCSW creates a new Command Status Wrapper with the given parameters.
func NewCSW(tag uint32, residue uint32, status uint8) *CommandStatusWrapper {
	return &CommandStatusWrapper{
		Signature:   CSWSignature,
		Tag:         tag,
		DataResidue: residue,
		Status:      status,
	}
}
