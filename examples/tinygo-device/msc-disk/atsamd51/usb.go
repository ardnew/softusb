//go:build tinygo && atsamd51

package main

import (
	"runtime/volatile"
	"unsafe"
)

// USB peripheral base address for ATSAMD51
const usbBase uintptr = 0x41000000

// USB device register offsets
const (
	offsetCTRLA       = 0x00 // Control A
	offsetSYNCBUSY    = 0x02 // Synchronization Busy
	offsetQOSCTRL     = 0x03 // QOS Control
	offsetCTRLB       = 0x08 // Control B
	offsetDADD        = 0x0A // Device Address
	offsetSTATUS      = 0x0C // Status
	offsetFSMSTATUS   = 0x0D // Finite State Machine Status
	offsetFNUM        = 0x10 // Frame Number
	offsetINTENSET    = 0x18 // Interrupt Enable Set
	offsetINTFLAG     = 0x1C // Interrupt Flag
	offsetEPINTSMRY   = 0x20 // Endpoint Interrupt Summary
	offsetDESCADD     = 0x24 // Descriptor Address
	offsetPADCAL      = 0x28 // Pad Calibration
	offsetEPCFG       = 0x100 // Endpoint Configuration (array, stride 0x20)
	offsetEPSTATUSCLR = 0x104 // Endpoint Status Clear (array, stride 0x20)
	offsetEPSTATUSSET = 0x105 // Endpoint Status Set (array, stride 0x20)
	offsetEPSTATUS    = 0x106 // Endpoint Status (array, stride 0x20)
	offsetEPINTFLAG   = 0x107 // Endpoint Interrupt Flag (array, stride 0x20)
	offsetEPINTENCLR  = 0x108 // Endpoint Interrupt Enable Clear (array, stride 0x20)
	offsetEPINTENSET  = 0x109 // Endpoint Interrupt Enable Set (array, stride 0x20)
)

// Endpoint register stride
const epStride = 0x20

// CTRLA register bits
const (
	ctrlaSWRST   = 1 << 0 // Software Reset
	ctrlaENABLE  = 1 << 1 // Enable
	ctrlaRUNSTBY = 1 << 2 // Run in Standby
	ctrlaMODE    = 1 << 7 // Operating Mode (0=device, 1=host)
)

// CTRLB register bits
const (
	ctrlbDETACH   = 1 << 0  // Detach
	ctrlbUPRSM    = 1 << 1  // Upstream Resume
	ctrlbSPDCONF  = 3 << 2  // Speed Configuration mask
	ctrlbNREPLY   = 1 << 4  // No Reply
	ctrlbGNAK     = 1 << 9  // Global NAK
	ctrlbLPMHDSK  = 3 << 10 // Link Power Management Handshake mask
)

// CTRLB speed configuration values
const (
	speedLowSpeed  = 0 << 2 // Low Speed
	speedFullSpeed = 1 << 2 // Full Speed
)

// INTFLAG register bits
const (
	intflagSUSPEND = 1 << 0 // Suspend
	intflagSOF     = 1 << 2 // Start of Frame
	intflagEORST   = 1 << 3 // End of Reset
	intflagWAKEUP  = 1 << 4 // Wake Up
	intflagEOFRSM  = 1 << 5 // End of Resume
	intflagUPRSM   = 1 << 6 // Upstream Resume
	intflagLPMNYET = 1 << 7 // LPM Not Yet
)

// EPINTFLAG register bits
const (
	epintflagTRCPT0 = 1 << 0 // Transfer Complete 0 (OUT)
	epintflagTRCPT1 = 1 << 1 // Transfer Complete 1 (IN)
	epintflagTRFAIL0 = 1 << 2 // Transfer Fail 0
	epintflagTRFAIL1 = 1 << 3 // Transfer Fail 1
	epintflagRXSTP  = 1 << 4 // Received Setup
	epintflagSTALL0 = 1 << 5 // Stall 0
	epintflagSTALL1 = 1 << 6 // Stall 1
)

// EPSTATUS register bits
const (
	epstatusDTGLIN   = 1 << 0 // Data Toggle IN
	epstatusDTGLOUT  = 1 << 1 // Data Toggle OUT
	epstatusCURBK    = 1 << 2 // Current Bank
	epstatusSTALLRQ0 = 1 << 4 // Stall Request 0 (OUT)
	epstatusSTALLRQ1 = 1 << 5 // Stall Request 1 (IN)
	epstatusBK0RDY   = 1 << 6 // Bank 0 Ready (OUT)
	epstatusBK1RDY   = 1 << 7 // Bank 1 Ready (IN)
)

// EPCFG register values
const (
	epTypeCfgDisabled    = 0x00 // Disabled
	epTypeCfgControl     = 0x01 // Control
	epTypeCfgIsochronous = 0x02 // Isochronous
	epTypeCfgBulk        = 0x03 // Bulk
	epTypeCfgInterrupt   = 0x04 // Interrupt
	epTypeCfgDualBank    = 0x05 // Dual Bank (Isochronous)
)

// NVM calibration addresses for USB pad
const (
	nvmUSBTransN = 0x00806020 // USB TRANSN calibration value
	nvmUSBTransP = 0x00806020 // USB TRANSP calibration value (different bits)
	nvmUSBTrim   = 0x00806020 // USB TRIM calibration value (different bits)
)

// MCLK register addresses
const (
	mclkBase       uintptr = 0x40000800
	mclkAHBMASK            = mclkBase + 0x10
	mclkAPBBMASK           = mclkBase + 0x18
)

// GCLK register addresses
const (
	gclkBase     uintptr = 0x40001C00
	gclkPCHCTRL          = gclkBase + 0x80 // Peripheral channel control (array)
)

// GCLK peripheral IDs
const (
	gclkUSB = 10 // USB peripheral clock
)

// Error values (simple error type for TinyGo compatibility)
type simpleError string

func (e simpleError) Error() string { return string(e) }

var (
	errReset           = simpleError("reset")
	errCancelled       = simpleError("cancelled")
	errInvalidEndpoint = simpleError("invalid endpoint")
	errTimeout         = simpleError("timeout")
)

// USB register accessors

func usbReg8(offset uintptr) *volatile.Register8 {
	return (*volatile.Register8)(unsafe.Pointer(usbBase + offset))
}

func usbReg16(offset uintptr) *volatile.Register16 {
	return (*volatile.Register16)(unsafe.Pointer(usbBase + offset))
}

func usbReg32(offset uintptr) *volatile.Register32 {
	return (*volatile.Register32)(unsafe.Pointer(usbBase + offset))
}

func epReg8(ep uint8, offset uintptr) *volatile.Register8 {
	return (*volatile.Register8)(unsafe.Pointer(usbBase + offset + uintptr(ep)*epStride))
}

// enableUSBClocks enables the clocks required for USB operation.
func enableUSBClocks() {
	// Enable USB AHB clock
	ahbmask := (*volatile.Register32)(unsafe.Pointer(mclkAHBMASK))
	ahbmask.SetBits(1 << 10) // USB AHB clock

	// Enable USB APB clock
	apbbmask := (*volatile.Register32)(unsafe.Pointer(mclkAPBBMASK))
	apbbmask.SetBits(1 << 10) // USB APB clock

	// Configure GCLK for USB (use GCLK0 = 48MHz from DFLL)
	pchctrl := (*volatile.Register32)(unsafe.Pointer(gclkPCHCTRL + uintptr(gclkUSB)*4))
	// GCLK0 (generator 0), enable
	pchctrl.Set((0 << 0) | (1 << 6)) // GEN=0, CHEN=1
}

// loadPadCalibration reads USB pad calibration from NVM and configures PADCAL.
func loadPadCalibration() {
	// Read calibration values from NVM
	// TRANSN: bits 31:27 of 0x00806020
	// TRANSP: bits 36:32 of 0x00806020 (i.e., bits 4:0 of 0x00806024)
	// TRIM: bits 41:37 of 0x00806020 (i.e., bits 9:5 of 0x00806024)

	nvmWord0 := (*volatile.Register32)(unsafe.Pointer(uintptr(0x00806020)))
	nvmWord1 := (*volatile.Register32)(unsafe.Pointer(uintptr(0x00806024)))

	transn := (nvmWord0.Get() >> 27) & 0x1F
	transp := nvmWord1.Get() & 0x1F
	trim := (nvmWord1.Get() >> 5) & 0x07

	// Write to PADCAL register
	// TRANSP: bits 4:0
	// TRANSN: bits 10:6
	// TRIM: bits 14:12
	padcal := (transp << 0) | (transn << 6) | (trim << 12)
	usbReg16(offsetPADCAL).Set(uint16(padcal))
}

// setDescriptorAddress sets the endpoint descriptor table base address.
func setDescriptorAddress(addr uintptr) {
	usbReg32(offsetDESCADD).Set(uint32(addr))
}

// enableUSB enables the USB peripheral in device mode.
func enableUSB() {
	// Set device mode and enable
	ctrla := usbReg8(offsetCTRLA)

	// First ensure SWRST is not set
	ctrla.ClearBits(ctrlaSWRST)

	// Wait for sync
	for usbReg8(offsetSYNCBUSY).HasBits(ctrlaSWRST) {
	}

	// Enable in device mode (MODE=0)
	ctrla.Set(ctrlaENABLE)

	// Wait for enable to complete
	for usbReg8(offsetSYNCBUSY).HasBits(ctrlaENABLE) {
	}

	// Configure for full speed
	ctrlb := usbReg16(offsetCTRLB)
	ctrlb.Set(speedFullSpeed | ctrlbDETACH) // Keep detached initially
}

// disableUSB disables the USB peripheral.
func disableUSB() {
	usbReg8(offsetCTRLA).ClearBits(ctrlaENABLE)
	for usbReg8(offsetSYNCBUSY).HasBits(ctrlaENABLE) {
	}
}

// attachUSB attaches the device to the bus (enables D+ pull-up).
func attachUSB() {
	usbReg16(offsetCTRLB).ClearBits(ctrlbDETACH)
}

// detachUSB detaches the device from the bus (disables D+ pull-up).
func detachUSB() {
	usbReg16(offsetCTRLB).SetBits(ctrlbDETACH)
}

// setDeviceAddress sets the device address.
func setDeviceAddress(addr uint8) {
	// DADD register: bits 6:0 = address, bit 7 = ADDEN (address enable)
	usbReg8(offsetDADD).Set(addr | 0x80)
}

// setEndpointConfig configures an endpoint's type for both banks.
// outType: bank 0 (OUT) configuration
// inType: bank 1 (IN) configuration
func setEndpointConfig(ep uint8, outType, inType uint8) {
	// EPCFG: bits 2:0 = EPTYPE0 (OUT), bits 6:4 = EPTYPE1 (IN)
	epReg8(ep, offsetEPCFG).Set((inType << 4) | outType)
}

// getEndpointConfig returns the current endpoint configuration.
func getEndpointConfig(ep uint8) (outType, inType uint8) {
	cfg := epReg8(ep, offsetEPCFG).Get()
	return cfg & 0x07, (cfg >> 4) & 0x07
}

// hasUSBReset returns true if a USB reset has occurred.
func hasUSBReset() bool {
	return usbReg16(offsetINTFLAG).HasBits(intflagEORST)
}

// clearUSBReset clears the USB reset flag.
func clearUSBReset() {
	usbReg16(offsetINTFLAG).Set(intflagEORST)
}

// hasSuspend returns true if USB is in suspend state.
func hasSuspend() bool {
	return usbReg16(offsetINTFLAG).HasBits(intflagSUSPEND)
}

// hasSetupReceived returns true if a SETUP packet was received on the endpoint.
func hasSetupReceived(ep uint8) bool {
	return epReg8(ep, offsetEPINTFLAG).HasBits(epintflagRXSTP)
}

// clearSetupReceived clears the SETUP received flag.
func clearSetupReceived(ep uint8) {
	epReg8(ep, offsetEPINTFLAG).Set(epintflagRXSTP)
}

// hasTransferComplete returns true if a transfer is complete.
// isIn: true for IN (bank 1), false for OUT (bank 0)
func hasTransferComplete(ep uint8, isIn bool) bool {
	if isIn {
		return epReg8(ep, offsetEPINTFLAG).HasBits(epintflagTRCPT1)
	}
	return epReg8(ep, offsetEPINTFLAG).HasBits(epintflagTRCPT0)
}

// clearTransferComplete clears the transfer complete flag.
func clearTransferComplete(ep uint8, isIn bool) {
	if isIn {
		epReg8(ep, offsetEPINTFLAG).Set(epintflagTRCPT1)
	} else {
		epReg8(ep, offsetEPINTFLAG).Set(epintflagTRCPT0)
	}
}

// setInReady sets the IN bank as ready (data available for host).
func setInReady(ep uint8) {
	epReg8(ep, offsetEPSTATUSSET).Set(epstatusBK1RDY)
}

// clearOutReady clears the OUT bank ready flag (ready to receive).
func clearOutReady(ep uint8) {
	epReg8(ep, offsetEPSTATUSCLR).Set(epstatusBK0RDY)
}

// stallEndpoint stalls an endpoint bank.
func stallEndpoint(ep uint8, isIn bool) {
	if isIn {
		epReg8(ep, offsetEPSTATUSSET).Set(epstatusSTALLRQ1)
	} else {
		epReg8(ep, offsetEPSTATUSSET).Set(epstatusSTALLRQ0)
	}
}

// clearStall clears a stall condition.
func clearStall(ep uint8, isIn bool) {
	if isIn {
		epReg8(ep, offsetEPSTATUSCLR).Set(epstatusSTALLRQ1)
		// Also clear data toggle
		epReg8(ep, offsetEPSTATUSCLR).Set(epstatusDTGLIN)
	} else {
		epReg8(ep, offsetEPSTATUSCLR).Set(epstatusSTALLRQ0)
		epReg8(ep, offsetEPSTATUSCLR).Set(epstatusDTGLOUT)
	}
}

// setByteCount sets the byte count in the endpoint descriptor.
// This function accesses the descriptor table directly.
func setByteCount(ep uint8, isIn bool, count uint16) {
	// Access the endpoint descriptor in the HAL's descriptor table
	// This is a bit awkward - we need to access the global HAL instance
	// For now, we'll use the descriptor address from DESCADD register
	descAddr := uintptr(usbReg32(offsetDESCADD).Get())
	epDesc := (*EndpointDescriptor)(unsafe.Pointer(descAddr + uintptr(ep)*unsafe.Sizeof(EndpointDescriptor{})))

	if isIn {
		// Clear existing byte count (bits 13:0) and set new value
		pcksize := epDesc.Bank1.PckSize
		pcksize = (pcksize &^ 0x3FFF) | uint32(count)
		epDesc.Bank1.PckSize = pcksize
	} else {
		pcksize := epDesc.Bank0.PckSize
		pcksize = (pcksize &^ 0x3FFF) | uint32(count)
		epDesc.Bank0.PckSize = pcksize
	}
}

// getByteCount gets the byte count from the endpoint descriptor.
func getByteCount(ep uint8, isIn bool) uint16 {
	descAddr := uintptr(usbReg32(offsetDESCADD).Get())
	epDesc := (*EndpointDescriptor)(unsafe.Pointer(descAddr + uintptr(ep)*unsafe.Sizeof(EndpointDescriptor{})))

	if isIn {
		return uint16(epDesc.Bank1.PckSize & 0x3FFF)
	}
	return uint16(epDesc.Bank0.PckSize & 0x3FFF)
}

// delayMicroseconds provides a simple delay.
// This is a busy-wait loop; in production you might use a timer.
func delayMicroseconds(us uint32) {
	// Rough approximation for 120MHz CPU
	// Each loop iteration is approximately 8-10 cycles
	cycles := us * 12 // ~12 cycles per microsecond at 120MHz
	for i := uint32(0); i < cycles; i++ {
		// Prevent optimizer from removing the loop
		volatile.Asm("nop")
	}
}
