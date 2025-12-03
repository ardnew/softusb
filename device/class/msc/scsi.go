package msc

import "encoding/binary"

// InquiryResponse represents standard INQUIRY data.
type InquiryResponse struct {
	DeviceType       uint8    // Peripheral device type
	RMB              uint8    // Removable media bit (bit 7)
	Version          uint8    // SCSI version
	ResponseFormat   uint8    // Response data format
	AdditionalLength uint8    // Additional length (n-4)
	Flags            [3]uint8 // Various flags
	VendorID         [8]byte  // Vendor identification (ASCII)
	ProductID        [16]byte // Product identification (ASCII)
	ProductRev       [4]byte  // Product revision (ASCII)
}

// MarshalTo writes the INQUIRY response to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (r *InquiryResponse) MarshalTo(buf []byte) int {
	if len(buf) < InquiryStandardSize {
		return 0
	}

	buf[0] = r.DeviceType
	buf[1] = r.RMB
	buf[2] = r.Version
	buf[3] = r.ResponseFormat
	buf[4] = r.AdditionalLength
	copy(buf[5:8], r.Flags[:])
	copy(buf[8:16], r.VendorID[:])
	copy(buf[16:32], r.ProductID[:])
	copy(buf[32:36], r.ProductRev[:])

	return InquiryStandardSize
}

// NewInquiryResponse creates a standard INQUIRY response.
func NewInquiryResponse(deviceType uint8, removable bool, vendor, product, revision string) *InquiryResponse {
	resp := &InquiryResponse{
		DeviceType:       deviceType,
		Version:          InquiryVersionSPC4,
		ResponseFormat:   InquiryResponseFormatSPC,
		AdditionalLength: InquiryStandardSize - 5,
	}

	if removable {
		resp.RMB = InquiryRMB
	}

	// Copy strings with padding
	copy(resp.VendorID[:], padString(vendor, 8))
	copy(resp.ProductID[:], padString(product, 16))
	copy(resp.ProductRev[:], padString(revision, 4))

	return resp
}

// ReadCapacity10Response represents READ CAPACITY (10) response.
type ReadCapacity10Response struct {
	LastLBA     uint32 // Last logical block address
	BlockLength uint32 // Block length in bytes
}

// MarshalTo writes the response to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (r *ReadCapacity10Response) MarshalTo(buf []byte) int {
	if len(buf) < 8 {
		return 0
	}

	binary.BigEndian.PutUint32(buf[0:4], r.LastLBA)
	binary.BigEndian.PutUint32(buf[4:8], r.BlockLength)

	return 8
}

// ReadCapacity16Response represents READ CAPACITY (16) response.
type ReadCapacity16Response struct {
	LastLBA     uint64 // Last logical block address
	BlockLength uint32 // Block length in bytes
}

// MarshalTo writes the response to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (r *ReadCapacity16Response) MarshalTo(buf []byte) int {
	if len(buf) < 32 {
		return 0
	}

	binary.BigEndian.PutUint64(buf[0:8], r.LastLBA)
	binary.BigEndian.PutUint32(buf[8:12], r.BlockLength)
	// Remaining bytes are reserved/zero

	return 32
}

// RequestSenseResponse represents REQUEST SENSE response (fixed format).
type RequestSenseResponse struct {
	ResponseCode     uint8  // Response code (0x70 = current, 0x72 = descriptor)
	SenseKey         uint8  // Sense key (bits 0-3)
	Information      uint32 // Information field
	AdditionalLength uint8  // Additional sense length (n-7)
	ASC              uint8  // Additional sense code
	ASCQ             uint8  // Additional sense code qualifier
}

// MarshalTo writes the response to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (r *RequestSenseResponse) MarshalTo(buf []byte) int {
	const senseSize = 18
	if len(buf) < senseSize {
		return 0
	}

	// Clear buffer
	for i := 0; i < senseSize; i++ {
		buf[i] = 0
	}

	buf[0] = r.ResponseCode
	buf[2] = r.SenseKey & 0x0F
	binary.BigEndian.PutUint32(buf[3:7], r.Information)
	buf[7] = r.AdditionalLength
	buf[12] = r.ASC
	buf[13] = r.ASCQ

	return senseSize
}

// NewRequestSenseResponse creates a REQUEST SENSE response.
func NewRequestSenseResponse(key, asc, ascq uint8) *RequestSenseResponse {
	return &RequestSenseResponse{
		ResponseCode:     0x70, // Current errors, fixed format
		SenseKey:         key & 0x0F,
		AdditionalLength: 10, // Fixed format has 10 additional bytes
		ASC:              asc,
		ASCQ:             ascq,
	}
}

// ModeSense6Response represents MODE SENSE (6) response header.
type ModeSense6Response struct {
	ModeDataLength uint8  // Mode data length (excluding this field)
	MediumType     uint8  // Medium type
	DeviceParam    uint8  // Device-specific parameter
	BlockDescLen   uint8  // Block descriptor length
}

// MarshalTo writes the response header to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (r *ModeSense6Response) MarshalTo(buf []byte) int {
	if len(buf) < 4 {
		return 0
	}

	buf[0] = r.ModeDataLength
	buf[1] = r.MediumType
	buf[2] = r.DeviceParam
	buf[3] = r.BlockDescLen

	return 4
}

// ReadFormatCapacitiesHeader represents READ FORMAT CAPACITIES response header.
type ReadFormatCapacitiesHeader struct {
	Reserved       [3]uint8 // Reserved
	CapacityLength uint8    // Capacity list length
}

// MarshalTo writes the header to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (r *ReadFormatCapacitiesHeader) MarshalTo(buf []byte) int {
	if len(buf) < 4 {
		return 0
	}

	buf[0] = 0
	buf[1] = 0
	buf[2] = 0
	buf[3] = r.CapacityLength

	return 4
}

// CurrentMaximumCapacityDescriptor represents a capacity descriptor.
type CurrentMaximumCapacityDescriptor struct {
	BlockCount  uint32 // Number of blocks
	DescType    uint8  // Descriptor type (bits 0-1) / reserved (bits 2-7)
	BlockLength uint32 // Block length (24-bit, stored in bytes 1-3)
}

// MarshalTo writes the descriptor to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (d *CurrentMaximumCapacityDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < 8 {
		return 0
	}

	binary.BigEndian.PutUint32(buf[0:4], d.BlockCount)
	buf[4] = d.DescType
	// Block length is 24-bit in bytes 5-7
	buf[5] = uint8(d.BlockLength >> 16)
	buf[6] = uint8(d.BlockLength >> 8)
	buf[7] = uint8(d.BlockLength)

	return 8
}

// padString pads or truncates a string to the specified length.
func padString(s string, length int) []byte {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		if i < len(s) {
			result[i] = s[i]
		} else {
			result[i] = ' ' // Pad with spaces
		}
	}
	return result
}
