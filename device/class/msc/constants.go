package msc

// USB Mass Storage Class codes.
const (
	ClassMSC = 0x08 // Mass Storage Class
)

// MSC Subclass codes.
const (
	SubclassRBC      = 0x01 // Reduced Block Commands
	SubclassMMC5     = 0x02 // Multi-Media Commands (CD/DVD)
	SubclassUFI      = 0x04 // USB Floppy Interface
	SubclassMMC2     = 0x05 // ATAPI/MMC-2
	SubclassSCSI     = 0x06 // SCSI Transparent Command Set
	SubclassLSDFS    = 0x07 // LSD FS
	SubclassIEEE1667 = 0x08 // IEEE 1667
)

// MSC Protocol codes.
const (
	ProtocolCBI      = 0x00 // Control/Bulk/Interrupt
	ProtocolCBICmpl  = 0x01 // CBI with command completion interrupt
	ProtocolBulkOnly = 0x50 // Bulk-Only Transport (BOT)
	ProtocolUAS      = 0x62 // USB Attached SCSI
)

// Bulk-Only Transport request codes.
const (
	RequestBulkOnlyMassStorageReset = 0xFF // Reset the MSC device
	RequestGetMaxLUN                = 0xFE // Get maximum Logical Unit Number
)

// Command Block Wrapper (CBW) constants.
const (
	CBWSignature   = 0x43425355 // "USBC" signature
	CBWSize        = 31         // Fixed CBW size in bytes
	CBWFlagDataOut = 0x00       // Data transfer: host to device
	CBWFlagDataIn  = 0x80       // Data transfer: device to host
)

// Command Status Wrapper (CSW) constants.
const (
	CSWSignature        = 0x53425355 // "USBS" signature
	CSWSize             = 13         // Fixed CSW size in bytes
	CSWStatusGood       = 0x00       // Command passed
	CSWStatusFailed     = 0x01       // Command failed
	CSWStatusPhaseError = 0x02       // Phase error occurred
)

// SCSI operation codes (commonly used subset).
const (
	SCSITestUnitReady        = 0x00 // Test if unit is ready
	SCSIRequestSense         = 0x03 // Request sense data
	SCSIInquiry              = 0x12 // Get device information
	SCSIModeSense6           = 0x1A // Get mode parameters (6-byte)
	SCSIModeSense10          = 0x5A // Get mode parameters (10-byte)
	SCSIStartStopUnit        = 0x1B // Start/stop unit
	SCSIPreventAllowRemoval  = 0x1E // Prevent/allow medium removal
	SCSIReadFormatCapacities = 0x23 // Read format capacities
	SCSIReadCapacity10       = 0x25 // Read capacity (10-byte)
	SCSIRead10               = 0x28 // Read blocks (10-byte)
	SCSIWrite10              = 0x2A // Write blocks (10-byte)
	SCSIVerify10             = 0x2F // Verify blocks (10-byte)
	SCSISynchronizeCache10   = 0x35 // Synchronize cache (10-byte)
	SCSIRead16               = 0x88 // Read blocks (16-byte)
	SCSIWrite16              = 0x8A // Write blocks (16-byte)
	SCSIServiceActionIn16    = 0x9E // Service action in (16-byte)
)

// Service action codes for SCSI_SERVICE_ACTION_IN_16.
const (
	ServiceActionReadCapacity16 = 0x10 // Read capacity (16-byte)
)

// SCSI sense keys.
const (
	SenseNoSense        = 0x00 // No error
	SenseRecoveredError = 0x01 // Recovered error
	SenseNotReady       = 0x02 // Device not ready
	SenseMediumError    = 0x03 // Medium error
	SenseHardwareError  = 0x04 // Hardware error
	SenseIllegalRequest = 0x05 // Illegal request
	SenseUnitAttention  = 0x06 // Unit attention
	SenseDataProtect    = 0x07 // Data protect
	SenseBlankCheck     = 0x08 // Blank check
	SenseAbortedCommand = 0x0B // Aborted command
)

// Additional Sense Codes (ASC).
const (
	ASCNoAdditionalInfo      = 0x00 // No additional sense information
	ASCInvalidCommand        = 0x20 // Invalid command operation code
	ASCLBAOutOfRange         = 0x21 // Logical block address out of range
	ASCInvalidFieldInCDB     = 0x24 // Invalid field in CDB
	ASCWriteProtected        = 0x27 // Write protected
	ASCNotReadyToReadyChange = 0x28 // Not ready to ready change
	ASCMediumNotPresent      = 0x3A // Medium not present
)

// SCSI device types (peripheral device type).
const (
	DeviceTypeDisk        = 0x00 // Direct access block device (disk)
	DeviceTypeTape        = 0x01 // Sequential access device (tape)
	DeviceTypePrinter     = 0x02 // Printer device
	DeviceTypeProcessor   = 0x03 // Processor device
	DeviceTypeWORM        = 0x04 // Write-once read-multiple
	DeviceTypeCDROM       = 0x05 // CD-ROM device
	DeviceTypeScanner     = 0x06 // Scanner device
	DeviceTypeOptical     = 0x07 // Optical memory device
	DeviceTypeChanger     = 0x08 // Medium changer device
	DeviceTypeComm        = 0x09 // Communications device
	DeviceTypeArray       = 0x0C // Storage array controller
	DeviceTypeEnclosure   = 0x0D // Enclosure services device
	DeviceTypeRBC         = 0x0E // Simplified direct-access device
	DeviceTypeOpticalCard = 0x0F // Optical card reader/writer
)

// INQUIRY response constants.
const (
	InquiryStandardSize = 36   // Standard INQUIRY data length
	InquiryVersionSPC3  = 0x05 // SPC-3 version
	InquiryVersionSPC4  = 0x06 // SPC-4 version
)

// INQUIRY response data format.
const (
	InquiryResponseFormatSPC = 0x02 // SPC-compliant response format
)

// INQUIRY flags.
const (
	InquiryRMB = 0x80 // Removable media bit
)

// Mode page codes.
const (
	ModePageCachingParameters = 0x08 // Caching parameters page
	ModePageAllPages          = 0x3F // All mode pages
)

// Mode sense control flags.
const (
	ModeSenseDBD = 0x08 // Disable block descriptors
)

// Default block size (512 bytes for most disks).
const DefaultBlockSize = 512

// Maximum transfer size (64 KB).
const MaxTransferSize = 65536
