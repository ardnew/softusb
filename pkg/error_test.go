package pkg

import (
	"errors"
	"testing"
)

func TestTransferStatus_String(t *testing.T) {
	tests := []struct {
		status TransferStatus
		want   string
	}{
		{TransferStatusSuccess, "success"},
		{TransferStatusError, "error"},
		{TransferStatusStall, "stall"},
		{TransferStatusNAK, "nak"},
		{TransferStatusTimeout, "timeout"},
		{TransferStatusCancelled, "cancelled"},
		{TransferStatusOverrun, "overrun"},
		{TransferStatusUnderrun, "underrun"},
		{TransferStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("TransferStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransferStatus_Error(t *testing.T) {
	tests := []struct {
		status  TransferStatus
		wantErr error
	}{
		{TransferStatusSuccess, nil},
		{TransferStatusStall, ErrStall},
		{TransferStatusNAK, ErrNAK},
		{TransferStatusTimeout, ErrTimeout},
		{TransferStatusCancelled, ErrCancelled},
		{TransferStatusOverrun, ErrOverrun},
		{TransferStatusUnderrun, ErrUnderrun},
		{TransferStatusError, ErrProtocol},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			err := tt.status.Error()
			if tt.wantErr == nil && err != nil {
				t.Errorf("TransferStatus.Error() = %v, want nil", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("TransferStatus.Error() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are distinct
	errs := []error{
		ErrStall,
		ErrNAK,
		ErrTimeout,
		ErrCancelled,
		ErrOverrun,
		ErrUnderrun,
		ErrCRC,
		ErrBitStuff,
		ErrProtocol,
		ErrNoDevice,
		ErrNotConfigured,
		ErrInvalidEndpoint,
		ErrInvalidState,
		ErrInvalidRequest,
		ErrBufferTooSmall,
		ErrNotSupported,
		ErrBusy,
		ErrNoMemory,
		ErrBandwidth,
		ErrFrameOverrun,
	}

	for i, err1 := range errs {
		if err1 == nil {
			t.Errorf("error %d is nil", i)
			continue
		}
		for j, err2 := range errs {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("error %d and %d are equal", i, j)
			}
		}
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err     error
		wantMsg string
	}{
		{ErrStall, "endpoint stalled"},
		{ErrNAK, "NAK received"},
		{ErrTimeout, "transfer timeout"},
		{ErrNoDevice, "device not present"},
		{ErrBandwidth, "insufficient bandwidth"},
	}

	for _, tt := range tests {
		t.Run(tt.wantMsg, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("error.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}
