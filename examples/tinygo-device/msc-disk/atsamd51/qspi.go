//go:build tinygo && atsamd51

package main

import (
	"runtime/volatile"
	"unsafe"
)

// QSPI flash constants for GD25Q64 (8MB) on Grand Central M4
const (
	// Flash geometry
	qspiFlashSize      = 8 * 1024 * 1024 // 8MB
	qspiSectorSize     = 4096            // 4KB sectors
	qspiPageSize       = 256             // 256 byte pages
	qspiBlockSize      = 512             // USB block size for MSC
	qspiBlockCount     = qspiFlashSize / qspiBlockSize

	// QSPI commands for GD25Q64
	cmdReadID          = 0x9F // Read JEDEC ID
	cmdReadStatus      = 0x05 // Read status register
	cmdWriteEnable     = 0x06 // Write enable
	cmdWriteDisable    = 0x04 // Write disable
	cmdRead            = 0x03 // Read data
	cmdFastRead        = 0x0B // Fast read (with dummy byte)
	cmdQuadRead        = 0xEB // Quad I/O fast read
	cmdPageProgram     = 0x02 // Page program
	cmdQuadPageProgram = 0x32 // Quad page program
	cmdSectorErase     = 0x20 // Sector erase (4KB)
	cmdBlockErase32    = 0x52 // Block erase (32KB)
	cmdBlockErase64    = 0xD8 // Block erase (64KB)
	cmdChipErase       = 0xC7 // Chip erase

	// Status register bits
	statusBusy         = 0x01 // Write in progress
	statusWEL          = 0x02 // Write enable latch
)

// QSPI peripheral base address
const qspiBase uintptr = 0x42003400

// QSPI register offsets
const (
	qspiCTRLA      = 0x00 // Control A
	qspiCTRLB      = 0x04 // Control B
	qspiBAUD       = 0x08 // Baud Rate
	qspiRXDATA     = 0x0C // Receive Data
	qspiTXDATA     = 0x10 // Transmit Data
	qspiINTENCLR   = 0x14 // Interrupt Enable Clear
	qspiINTENSET   = 0x18 // Interrupt Enable Set
	qspiINTFLAG    = 0x1C // Interrupt Flag
	qspiSTATUS     = 0x20 // Status
	qspiINSTRCTRL  = 0x30 // Instruction Code
	qspiINSTRADDR  = 0x34 // Instruction Address
	qspiINSTRFRAME = 0x38 // Instruction Frame
	qspiSCRSR      = 0x3C // Scrambling Seed
)

// QSPI AHB memory mapping
const qspiMemBase uintptr = 0x04000000

// MCLK bits for QSPI
const (
	mclkQSPIAHB = 1 << 13 // QSPI AHB clock
	mclkQSPIAPB = 1 << 13 // QSPI APB clock (in APBCMASK)
)

// QSPIStorage implements the msc.Storage interface for QSPI flash.
type QSPIStorage struct {
	// Sector buffer for read-modify-write operations
	sectorBuf [qspiSectorSize]byte

	// Cached sector number (-1 = invalid)
	cachedSector int32

	// Dirty flag for cached sector
	dirty bool

	// Read-only mode
	readOnly bool
}

// NewQSPIStorage creates a new QSPI flash storage backend.
func NewQSPIStorage() *QSPIStorage {
	s := &QSPIStorage{
		cachedSector: -1,
	}
	s.init()
	return s
}

// init initializes the QSPI peripheral.
func (s *QSPIStorage) init() {
	// Enable QSPI clocks
	enableQSPIClocks()

	// Reset QSPI
	qspiReg32(qspiCTRLA).Set(1 << 0) // SWRST
	for qspiReg32(qspiCTRLA).HasBits(1 << 0) {
	}

	// Configure QSPI in Serial Memory Mode
	// LOOPEN=0, CSMODE=NORELOAD, DATALEN=8 bits
	qspiReg32(qspiCTRLB).Set(
		(0 << 1) |  // LOOPEN = 0
		(0 << 4) |  // CSMODE = 0 (NORELOAD)
		(0 << 8))   // DATALEN = 0 (8 bits)

	// Set baud rate (divide by 2 for maximum speed)
	qspiReg32(qspiBAUD).Set(1) // BAUD = 1 (divide by 2)

	// Enable QSPI
	qspiReg32(qspiCTRLA).Set(1 << 1) // ENABLE
	for !qspiReg32(qspiCTRLA).HasBits(1 << 1) {
	}

	// Configure for memory-mapped mode with quad read
	s.configureMemoryMode()
}

// configureMemoryMode sets up QSPI for memory-mapped read access.
func (s *QSPIStorage) configureMemoryMode() {
	// Configure instruction for quad read
	// INSTR = 0xEB (quad read), OPTCODE, etc.
	instrctrl := qspiReg32(qspiINSTRCTRL)
	instrctrl.Set(
		uint32(cmdQuadRead) | // Instruction code
		(0 << 8))             // Option code

	// Configure instruction frame
	// WIDTH=QUAD_IO, INSTREN, ADDREN, DATAEN, etc.
	instrframe := qspiReg32(qspiINSTRFRAME)
	instrframe.Set(
		(4 << 0) |  // WIDTH = QUAD_IO
		(1 << 4) |  // INSTREN = 1
		(1 << 5) |  // ADDREN = 1
		(0 << 6) |  // OPTCODEEN = 0
		(1 << 7) |  // DATAEN = 1
		(0 << 8) |  // OPTCODELEN = 0
		(3 << 12) | // ADDRLEN = 3 (24-bit address)
		(1 << 14) | // TFRTYPE = 1 (read)
		(0 << 15) | // CRMODE = 0
		(1 << 16) | // DDREN = 0
		(4 << 17))  // DUMMYLEN = 4 dummy cycles
}

// BlockSize returns the block size in bytes.
func (s *QSPIStorage) BlockSize() uint32 {
	return qspiBlockSize
}

// BlockCount returns the total number of blocks.
func (s *QSPIStorage) BlockCount() uint64 {
	return qspiBlockCount
}

// Read reads blocks from flash.
func (s *QSPIStorage) Read(lba uint64, blocks uint32, buf []byte) (uint32, error) {
	if lba+uint64(blocks) > qspiBlockCount {
		return 0, errInvalidAddress
	}

	required := uint64(blocks) * qspiBlockSize
	if uint64(len(buf)) < required {
		return 0, errBufferTooSmall
	}

	offset := lba * qspiBlockSize
	totalBytes := blocks * qspiBlockSize

	// Use memory-mapped read
	s.readMemoryMapped(uint32(offset), buf[:totalBytes])

	return blocks, nil
}

// Write writes blocks to flash.
func (s *QSPIStorage) Write(lba uint64, blocks uint32, buf []byte) (uint32, error) {
	if s.readOnly {
		return 0, errReadOnly
	}

	if lba+uint64(blocks) > qspiBlockCount {
		return 0, errInvalidAddress
	}

	required := uint64(blocks) * qspiBlockSize
	if uint64(len(buf)) < required {
		return 0, errBufferTooSmall
	}

	written := uint32(0)
	offset := uint32(lba * qspiBlockSize)

	for written < blocks {
		// Calculate which sector this block falls into
		sectorNum := int32(offset / qspiSectorSize)
		sectorOffset := offset % qspiSectorSize

		// Load sector if not cached
		if s.cachedSector != sectorNum {
			// Flush any dirty sector first
			if s.dirty && s.cachedSector >= 0 {
				s.flushSector()
			}

			// Read entire sector into buffer
			s.readMemoryMapped(uint32(sectorNum)*qspiSectorSize, s.sectorBuf[:])
			s.cachedSector = sectorNum
			s.dirty = false
		}

		// Copy data into sector buffer
		copyLen := qspiBlockSize
		if sectorOffset+copyLen > qspiSectorSize {
			copyLen = qspiSectorSize - sectorOffset
		}

		srcOffset := written * qspiBlockSize
		copy(s.sectorBuf[sectorOffset:sectorOffset+copyLen], buf[srcOffset:srcOffset+copyLen])
		s.dirty = true

		offset += copyLen
		if copyLen == qspiBlockSize {
			written++
		} else {
			// Partial block, need to continue in next sector
			offset = uint32((sectorNum + 1)) * qspiSectorSize
		}
	}

	return blocks, nil
}

// Sync flushes cached writes to flash.
func (s *QSPIStorage) Sync() error {
	if s.dirty && s.cachedSector >= 0 {
		return s.flushSector()
	}
	return nil
}

// IsReadOnly returns whether the storage is read-only.
func (s *QSPIStorage) IsReadOnly() bool {
	return s.readOnly
}

// SetReadOnly sets the read-only flag.
func (s *QSPIStorage) SetReadOnly(readOnly bool) {
	s.readOnly = readOnly
}

// IsRemovable returns whether the media is removable.
func (s *QSPIStorage) IsRemovable() bool {
	return false // QSPI flash is not removable
}

// IsPresent returns whether media is present.
func (s *QSPIStorage) IsPresent() bool {
	return true // Always present
}

// Eject is not supported for QSPI flash.
func (s *QSPIStorage) Eject() error {
	return errNotSupported
}

// flushSector erases and rewrites the cached sector.
func (s *QSPIStorage) flushSector() error {
	if s.cachedSector < 0 {
		return nil
	}

	sectorAddr := uint32(s.cachedSector) * qspiSectorSize

	// Erase sector
	if err := s.eraseSector(sectorAddr); err != nil {
		return err
	}

	// Program sector page by page
	for pageOffset := uint32(0); pageOffset < qspiSectorSize; pageOffset += qspiPageSize {
		if err := s.programPage(sectorAddr+pageOffset, s.sectorBuf[pageOffset:pageOffset+qspiPageSize]); err != nil {
			return err
		}
	}

	s.dirty = false
	return nil
}

// eraseSector erases a 4KB sector.
func (s *QSPIStorage) eraseSector(addr uint32) error {
	s.writeEnable()

	// Send sector erase command
	s.sendCommand(cmdSectorErase, addr, nil, 0)

	// Wait for completion
	return s.waitBusy()
}

// programPage programs a 256-byte page.
func (s *QSPIStorage) programPage(addr uint32, data []byte) error {
	if len(data) > qspiPageSize {
		data = data[:qspiPageSize]
	}

	s.writeEnable()

	// Send page program command
	s.sendCommand(cmdPageProgram, addr, data, 0)

	// Wait for completion
	return s.waitBusy()
}

// writeEnable sends the write enable command.
func (s *QSPIStorage) writeEnable() {
	s.sendCommand(cmdWriteEnable, 0, nil, 0)
}

// waitBusy waits for the flash to complete an operation.
func (s *QSPIStorage) waitBusy() error {
	timeout := uint32(1000000) // ~1 second at 120MHz
	for timeout > 0 {
		status := s.readStatus()
		if status&statusBusy == 0 {
			return nil
		}
		timeout--
	}
	return errTimeout
}

// readStatus reads the flash status register.
func (s *QSPIStorage) readStatus() uint8 {
	var status [1]byte
	s.sendCommand(cmdReadStatus, 0, nil, 1)
	status[0] = uint8(qspiReg32(qspiRXDATA).Get())
	return status[0]
}

// sendCommand sends a command to the flash.
func (s *QSPIStorage) sendCommand(cmd uint8, addr uint32, data []byte, readLen int) {
	// Exit memory mode temporarily
	qspiReg32(qspiCTRLA).ClearBits(1 << 1) // Disable
	for qspiReg32(qspiCTRLA).HasBits(1 << 1) {
	}

	// Configure for SPI mode command
	instrframe := uint32(0)
	instrframe |= (0 << 0)  // WIDTH = SINGLE
	instrframe |= (1 << 4)  // INSTREN = 1

	// Add address if non-zero
	if addr != 0 || cmd == cmdSectorErase || cmd == cmdPageProgram || cmd == cmdRead {
		instrframe |= (1 << 5)  // ADDREN = 1
		instrframe |= (3 << 12) // ADDRLEN = 3 (24-bit)
		qspiReg32(qspiINSTRADDR).Set(addr)
	}

	// Add data if present
	if len(data) > 0 || readLen > 0 {
		instrframe |= (1 << 7) // DATAEN = 1
		if readLen > 0 {
			instrframe |= (1 << 14) // TFRTYPE = READ
		} else {
			instrframe |= (2 << 14) // TFRTYPE = WRITE
		}
	}

	qspiReg32(qspiINSTRCTRL).Set(uint32(cmd))
	qspiReg32(qspiINSTRFRAME).Set(instrframe)

	// Enable and trigger
	qspiReg32(qspiCTRLA).Set(1 << 1) // ENABLE
	for !qspiReg32(qspiCTRLA).HasBits(1 << 1) {
	}

	// Write data if present
	for i := 0; i < len(data); i++ {
		qspiReg32(qspiTXDATA).Set(uint32(data[i]))
		// Wait for TX complete
		for !qspiReg32(qspiINTFLAG).HasBits(1 << 1) {
		}
	}

	// Read data if requested
	for i := 0; i < readLen; i++ {
		// Trigger read
		qspiReg32(qspiTXDATA).Set(0)
		// Wait for RX complete
		for !qspiReg32(qspiINTFLAG).HasBits(1 << 0) {
		}
	}

	// Restore memory mode
	s.configureMemoryMode()
}

// readMemoryMapped reads data using memory-mapped access.
func (s *QSPIStorage) readMemoryMapped(offset uint32, buf []byte) {
	// Direct memory access to QSPI memory region
	src := (*[qspiFlashSize]byte)(unsafe.Pointer(qspiMemBase))
	copy(buf, src[offset:offset+uint32(len(buf))])
}

// enableQSPIClocks enables clocks for the QSPI peripheral.
func enableQSPIClocks() {
	// Enable QSPI AHB clock
	ahbmask := (*volatile.Register32)(unsafe.Pointer(mclkAHBMASK))
	ahbmask.SetBits(mclkQSPIAHB)

	// Enable QSPI APB clock (in APBCMASK at offset 0x1C)
	apbcmask := (*volatile.Register32)(unsafe.Pointer(mclkBase + 0x1C))
	apbcmask.SetBits(mclkQSPIAPB)
}

// qspiReg32 returns a 32-bit register accessor.
func qspiReg32(offset uintptr) *volatile.Register32 {
	return (*volatile.Register32)(unsafe.Pointer(qspiBase + offset))
}

// Additional error types for storage operations
var (
	errInvalidAddress = simpleError("invalid address")
	errBufferTooSmall = simpleError("buffer too small")
	errReadOnly       = simpleError("read only")
	errNotSupported   = simpleError("not supported")
)
