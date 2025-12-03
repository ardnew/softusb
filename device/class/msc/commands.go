package msc

import (
	"context"
	"io"

	"github.com/ardnew/softusb/pkg"
)

// handleSCSICommand processes a SCSI command from CBW.
// Returns command status and data residue.
func (m *MSC) handleSCSICommand(ctx context.Context, cbw *CommandBlockWrapper) (status uint8, residue uint32) {
	opcode := cbw.CB[0]

	pkg.LogDebug(pkg.ComponentDevice, "SCSI command",
		"opcode", opcode,
		"lun", cbw.LUN)

	// Check LUN
	if cbw.LUN > m.maxLUN {
		m.setSense(SenseIllegalRequest, ASCInvalidFieldInCDB, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	// Dispatch to command handler
	switch opcode {
	case SCSITestUnitReady:
		return m.handleTestUnitReady(cbw)

	case SCSIRequestSense:
		return m.handleRequestSense(ctx, cbw)

	case SCSIInquiry:
		return m.handleInquiry(ctx, cbw)

	case SCSIReadCapacity10:
		return m.handleReadCapacity10(ctx, cbw)

	case SCSIRead10:
		return m.handleRead10(ctx, cbw)

	case SCSIWrite10:
		return m.handleWrite10(ctx, cbw)

	case SCSIModeSense6:
		return m.handleModeSense6(ctx, cbw)

	case SCSIPreventAllowRemoval:
		return m.handlePreventAllowRemoval(cbw)

	case SCSIStartStopUnit:
		return m.handleStartStopUnit(cbw)

	case SCSISynchronizeCache10:
		return m.handleSynchronizeCache10(cbw)

	case SCSIVerify10:
		return m.handleVerify10(cbw)

	case SCSIReadFormatCapacities:
		return m.handleReadFormatCapacities(ctx, cbw)

	case SCSIServiceActionIn16:
		// Check service action
		serviceAction := cbw.CB[1] & 0x1F
		if serviceAction == ServiceActionReadCapacity16 {
			return m.handleReadCapacity16(ctx, cbw)
		}
		fallthrough

	default:
		pkg.LogWarn(pkg.ComponentDevice, "unsupported SCSI command",
			"opcode", opcode)
		m.setSense(SenseIllegalRequest, ASCInvalidCommand, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}
}

// handleTestUnitReady processes TEST UNIT READY command.
func (m *MSC) handleTestUnitReady(cbw *CommandBlockWrapper) (uint8, uint32) {
	if !m.storage.IsPresent() {
		m.setSense(SenseNotReady, ASCMediumNotPresent, 0)
		return CSWStatusFailed, 0
	}

	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)
	return CSWStatusGood, 0
}

// handleRequestSense processes REQUEST SENSE command.
func (m *MSC) handleRequestSense(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	allocLength := cbw.CB[4]
	if allocLength == 0 {
		allocLength = 18
	}

	resp := NewRequestSenseResponse(m.senseKey, m.asc, m.ascq)
	n := resp.MarshalTo(m.senseBuf[:])

	// Send data
	sendLen := int(allocLength)
	if sendLen > n {
		sendLen = n
	}

	if err := m.sendData(ctx, m.senseBuf[:sendLen]); err != nil {
		return CSWStatusFailed, cbw.DataTransferLength
	}

	// Clear sense data after successful REQUEST SENSE
	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)

	residue := cbw.DataTransferLength - uint32(sendLen)
	return CSWStatusGood, residue
}

// handleInquiry processes INQUIRY command.
func (m *MSC) handleInquiry(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	allocLength := parseU16BE(cbw.CB[:], 3)
	if allocLength == 0 {
		return CSWStatusGood, 0
	}

	n := m.inquiry.MarshalTo(m.dataBuf[:])

	// Send data
	sendLen := int(allocLength)
	if sendLen > n {
		sendLen = n
	}

	if err := m.sendData(ctx, m.dataBuf[:sendLen]); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	residue := cbw.DataTransferLength - uint32(sendLen)
	return CSWStatusGood, residue
}

// handleReadCapacity10 processes READ CAPACITY (10) command.
func (m *MSC) handleReadCapacity10(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	if !m.storage.IsPresent() {
		m.setSense(SenseNotReady, ASCMediumNotPresent, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	blockCount := m.storage.BlockCount()
	blockSize := m.storage.BlockSize()

	// READ CAPACITY (10) returns last LBA (max 0xFFFFFFFF)
	lastLBA := uint32(blockCount - 1)
	if blockCount > 0xFFFFFFFF {
		lastLBA = 0xFFFFFFFF
	}

	resp := ReadCapacity10Response{
		LastLBA:     lastLBA,
		BlockLength: blockSize,
	}

	n := resp.MarshalTo(m.dataBuf[:])

	if err := m.sendData(ctx, m.dataBuf[:n]); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	residue := cbw.DataTransferLength - uint32(n)
	return CSWStatusGood, residue
}

// handleReadCapacity16 processes READ CAPACITY (16) command.
func (m *MSC) handleReadCapacity16(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	if !m.storage.IsPresent() {
		m.setSense(SenseNotReady, ASCMediumNotPresent, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	blockCount := m.storage.BlockCount()
	blockSize := m.storage.BlockSize()

	resp := ReadCapacity16Response{
		LastLBA:     blockCount - 1,
		BlockLength: blockSize,
	}

	n := resp.MarshalTo(m.dataBuf[:])

	allocLength := parseU32BE(cbw.CB[:], 10)
	sendLen := int(allocLength)
	if sendLen > n {
		sendLen = n
	}

	if err := m.sendData(ctx, m.dataBuf[:sendLen]); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	residue := cbw.DataTransferLength - uint32(sendLen)
	return CSWStatusGood, residue
}

// handleRead10 processes READ (10) command.
func (m *MSC) handleRead10(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	if !m.storage.IsPresent() {
		m.setSense(SenseNotReady, ASCMediumNotPresent, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	lba := parseU32BE(cbw.CB[:], 2)
	transferBlocks := parseU16BE(cbw.CB[:], 7)

	if transferBlocks == 0 {
		return CSWStatusGood, 0
	}

	blockSize := m.storage.BlockSize()
	transferLength := uint32(transferBlocks) * blockSize

	// Check LBA range
	if uint64(lba)+uint64(transferBlocks) > m.storage.BlockCount() {
		m.setSense(SenseIllegalRequest, ASCLBAOutOfRange, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	pkg.LogDebug(pkg.ComponentDevice, "READ(10)",
		"lba", lba,
		"blocks", transferBlocks)

	// Read blocks
	blocksRead, err := m.storage.Read(uint64(lba), uint32(transferBlocks), m.dataBuf[:transferLength])
	if err != nil {
		pkg.LogWarn(pkg.ComponentDevice, "read error", "error", err)
		m.setSense(SenseMediumError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	actualLength := blocksRead * blockSize

	// Send data
	if err := m.sendData(ctx, m.dataBuf[:actualLength]); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	residue := cbw.DataTransferLength - actualLength
	return CSWStatusGood, residue
}

// handleWrite10 processes WRITE (10) command.
func (m *MSC) handleWrite10(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	if !m.storage.IsPresent() {
		m.setSense(SenseNotReady, ASCMediumNotPresent, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	if m.storage.IsReadOnly() {
		m.setSense(SenseDataProtect, ASCWriteProtected, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	lba := parseU32BE(cbw.CB[:], 2)
	transferBlocks := parseU16BE(cbw.CB[:], 7)

	if transferBlocks == 0 {
		return CSWStatusGood, 0
	}

	blockSize := m.storage.BlockSize()
	transferLength := uint32(transferBlocks) * blockSize

	// Check LBA range
	if uint64(lba)+uint64(transferBlocks) > m.storage.BlockCount() {
		m.setSense(SenseIllegalRequest, ASCLBAOutOfRange, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	pkg.LogDebug(pkg.ComponentDevice, "WRITE(10)",
		"lba", lba,
		"blocks", transferBlocks)

	// Receive data from host
	if err := m.receiveData(ctx, m.dataBuf[:transferLength]); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	// Write blocks
	blocksWritten, err := m.storage.Write(uint64(lba), uint32(transferBlocks), m.dataBuf[:transferLength])
	if err != nil {
		pkg.LogWarn(pkg.ComponentDevice, "write error", "error", err)
		m.setSense(SenseMediumError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	actualLength := blocksWritten * blockSize
	residue := cbw.DataTransferLength - actualLength
	return CSWStatusGood, residue
}

// handleModeSense6 processes MODE SENSE (6) command.
func (m *MSC) handleModeSense6(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	allocLength := cbw.CB[4]
	if allocLength == 0 {
		return CSWStatusGood, 0
	}

	// Simple response with no mode pages
	resp := ModeSense6Response{
		ModeDataLength: 3, // Header only (excluding this field)
		MediumType:     0,
		DeviceParam:    0,
		BlockDescLen:   0,
	}

	if m.storage.IsReadOnly() {
		resp.DeviceParam = 0x80 // Write protect bit
	}

	n := resp.MarshalTo(m.dataBuf[:])

	sendLen := int(allocLength)
	if sendLen > n {
		sendLen = n
	}

	if err := m.sendData(ctx, m.dataBuf[:sendLen]); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	residue := cbw.DataTransferLength - uint32(sendLen)
	return CSWStatusGood, residue
}

// handlePreventAllowRemoval processes PREVENT/ALLOW MEDIUM REMOVAL command.
func (m *MSC) handlePreventAllowRemoval(cbw *CommandBlockWrapper) (uint8, uint32) {
	prevent := cbw.CB[4] & 0x01
	pkg.LogDebug(pkg.ComponentDevice, "PREVENT/ALLOW MEDIUM REMOVAL",
		"prevent", prevent)

	// We don't actually prevent removal, just acknowledge the command
	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)
	return CSWStatusGood, 0
}

// handleStartStopUnit processes START/STOP UNIT command.
func (m *MSC) handleStartStopUnit(cbw *CommandBlockWrapper) (uint8, uint32) {
	start := cbw.CB[4]&0x01 != 0
	loej := cbw.CB[4]&0x02 != 0

	pkg.LogDebug(pkg.ComponentDevice, "START/STOP UNIT",
		"start", start,
		"loej", loej)

	// Handle eject if requested
	if loej && !start {
		if m.storage.IsRemovable() {
			if err := m.storage.Eject(); err != nil {
				m.setSense(SenseIllegalRequest, ASCInvalidFieldInCDB, 0)
				return CSWStatusFailed, 0
			}
		}
	}

	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)
	return CSWStatusGood, 0
}

// handleSynchronizeCache10 processes SYNCHRONIZE CACHE (10) command.
func (m *MSC) handleSynchronizeCache10(cbw *CommandBlockWrapper) (uint8, uint32) {
	if err := m.storage.Sync(); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, 0
	}

	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)
	return CSWStatusGood, 0
}

// handleVerify10 processes VERIFY (10) command.
func (m *MSC) handleVerify10(cbw *CommandBlockWrapper) (uint8, uint32) {
	// We don't actually verify, just acknowledge success
	m.setSense(SenseNoSense, ASCNoAdditionalInfo, 0)
	return CSWStatusGood, 0
}

// handleReadFormatCapacities processes READ FORMAT CAPACITIES command.
func (m *MSC) handleReadFormatCapacities(ctx context.Context, cbw *CommandBlockWrapper) (uint8, uint32) {
	if !m.storage.IsPresent() {
		m.setSense(SenseNotReady, ASCMediumNotPresent, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	allocLength := parseU16BE(cbw.CB[:], 7)
	if allocLength == 0 {
		return CSWStatusGood, 0
	}

	blockCount := m.storage.BlockCount()
	blockSize := m.storage.BlockSize()

	// Build response
	offset := 0

	// Header
	header := ReadFormatCapacitiesHeader{
		CapacityLength: 8, // One descriptor
	}
	offset += header.MarshalTo(m.dataBuf[offset:])

	// Current/Maximum capacity descriptor
	desc := CurrentMaximumCapacityDescriptor{
		BlockCount:  uint32(blockCount),
		DescType:    0x02, // Formatted media
		BlockLength: blockSize,
	}
	offset += desc.MarshalTo(m.dataBuf[offset:])

	sendLen := int(allocLength)
	if sendLen > offset {
		sendLen = offset
	}

	if err := m.sendData(ctx, m.dataBuf[:sendLen]); err != nil {
		m.setSense(SenseHardwareError, ASCNoAdditionalInfo, 0)
		return CSWStatusFailed, cbw.DataTransferLength
	}

	residue := cbw.DataTransferLength - uint32(sendLen)
	return CSWStatusGood, residue
}

// sendData sends data to the host via bulk IN endpoint.
func (m *MSC) sendData(ctx context.Context, data []byte) error {
	m.mutex.RLock()
	stack := m.stack
	ep := m.bulkInEP
	m.mutex.RUnlock()

	if stack == nil || ep == nil {
		return pkg.ErrNotConfigured
	}

	_, err := stack.Write(ctx, ep, data)
	return err
}

// receiveData receives data from the host via bulk OUT endpoint.
func (m *MSC) receiveData(ctx context.Context, buf []byte) error {
	m.mutex.RLock()
	stack := m.stack
	ep := m.bulkOutEP
	m.mutex.RUnlock()

	if stack == nil || ep == nil {
		return pkg.ErrNotConfigured
	}

	totalRead := 0
	for totalRead < len(buf) {
		n, err := stack.Read(ctx, ep, buf[totalRead:])
		if err != nil {
			if err == io.EOF && totalRead > 0 {
				break
			}
			return err
		}
		totalRead += n
		if n == 0 {
			break
		}
	}

	return nil
}
