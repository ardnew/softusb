package pkg

import "errors"

// USB protocol errors.
var (
	// ErrStall indicates an endpoint stall condition.
	ErrStall = errors.New("endpoint stalled")

	// ErrNAK indicates a NAK response (device busy).
	ErrNAK = errors.New("NAK received")

	// ErrTimeout indicates a transfer timeout.
	ErrTimeout = errors.New("transfer timeout")

	// ErrCancelled indicates a cancelled transfer.
	ErrCancelled = errors.New("transfer cancelled")

	// ErrOverrun indicates a data overrun condition.
	ErrOverrun = errors.New("data overrun")

	// ErrUnderrun indicates a data underrun condition.
	ErrUnderrun = errors.New("data underrun")

	// ErrCRC indicates a CRC error.
	ErrCRC = errors.New("CRC error")

	// ErrBitStuff indicates a bit stuffing error.
	ErrBitStuff = errors.New("bit stuffing error")

	// ErrProtocol indicates a protocol error.
	ErrProtocol = errors.New("protocol error")

	// ErrNoDevice indicates the device is not present.
	ErrNoDevice = errors.New("device not present")

	// ErrNotConfigured indicates the device is not configured.
	ErrNotConfigured = errors.New("device not configured")

	// ErrInvalidEndpoint indicates an invalid endpoint address.
	ErrInvalidEndpoint = errors.New("invalid endpoint")

	// ErrInvalidState indicates an invalid device state for the operation.
	ErrInvalidState = errors.New("invalid device state")

	// ErrInvalidRequest indicates an invalid or unsupported request.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrBufferTooSmall indicates the provided buffer is too small.
	ErrBufferTooSmall = errors.New("buffer too small")

	// ErrNotSupported indicates an unsupported operation or feature.
	ErrNotSupported = errors.New("not supported")

	// ErrBusy indicates the resource is busy.
	ErrBusy = errors.New("resource busy")

	// ErrNoMemory indicates insufficient memory.
	ErrNoMemory = errors.New("insufficient memory")

	// ErrBandwidth indicates insufficient bandwidth for isochronous transfer.
	ErrBandwidth = errors.New("insufficient bandwidth")

	// ErrFrameOverrun indicates a frame overrun for isochronous transfer.
	ErrFrameOverrun = errors.New("frame overrun")

	// ErrDescriptorTooShort indicates the descriptor data is too short.
	ErrDescriptorTooShort = errors.New("descriptor too short")

	// ErrDescriptorTypeMismatch indicates the descriptor type does not match expected.
	ErrDescriptorTypeMismatch = errors.New("descriptor type mismatch")

	// ErrSetupPacketTooShort indicates the setup packet data is too short.
	ErrSetupPacketTooShort = errors.New("setup packet too short")

	// ErrAlreadyRunning indicates the stack is already running.
	ErrAlreadyRunning = errors.New("already running")

	// ErrNotRunning indicates the stack is not running.
	ErrNotRunning = errors.New("not running")

	// ErrInvalidParameter indicates an invalid parameter was provided.
	ErrInvalidParameter = errors.New("invalid parameter")

	// ErrNoResources indicates insufficient resources (e.g., pending transfer slots).
	ErrNoResources = errors.New("no resources available")

	// ErrReset indicates a bus reset was received.
	ErrReset = errors.New("bus reset")
)

// TransferStatus represents the completion status of a USB transfer.
type TransferStatus int

// Transfer status values.
const (
	TransferStatusSuccess   TransferStatus = iota // Transfer completed successfully
	TransferStatusError                           // Transfer failed with error
	TransferStatusStall                           // Endpoint stalled
	TransferStatusNAK                             // NAK received
	TransferStatusTimeout                         // Transfer timed out
	TransferStatusCancelled                       // Transfer was cancelled
	TransferStatusOverrun                         // Data overrun
	TransferStatusUnderrun                        // Data underrun
)

// String returns a string representation of the transfer status.
func (s TransferStatus) String() string {
	switch s {
	case TransferStatusSuccess:
		return "success"
	case TransferStatusError:
		return "error"
	case TransferStatusStall:
		return "stall"
	case TransferStatusNAK:
		return "nak"
	case TransferStatusTimeout:
		return "timeout"
	case TransferStatusCancelled:
		return "cancelled"
	case TransferStatusOverrun:
		return "overrun"
	case TransferStatusUnderrun:
		return "underrun"
	default:
		return "unknown"
	}
}

// Error returns the corresponding error for the transfer status.
func (s TransferStatus) Error() error {
	switch s {
	case TransferStatusSuccess:
		return nil
	case TransferStatusStall:
		return ErrStall
	case TransferStatusNAK:
		return ErrNAK
	case TransferStatusTimeout:
		return ErrTimeout
	case TransferStatusCancelled:
		return ErrCancelled
	case TransferStatusOverrun:
		return ErrOverrun
	case TransferStatusUnderrun:
		return ErrUnderrun
	default:
		return ErrProtocol
	}
}
