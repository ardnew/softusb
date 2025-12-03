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

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestTransferStatus_AllValues tests all TransferStatus values
func TestTransferStatus_AllValues(t *testing.T) {
	statuses := []TransferStatus{
		TransferStatusSuccess,
		TransferStatusError,
		TransferStatusStall,
		TransferStatusNAK,
		TransferStatusTimeout,
		TransferStatusCancelled,
		TransferStatusOverrun,
		TransferStatusUnderrun,
	}

	for _, s := range statuses {
		// String should not panic and not be empty
		str := s.String()
		if str == "" {
			t.Errorf("TransferStatus(%d).String() is empty", s)
		}
		if str == "unknown" && s <= TransferStatusUnderrun {
			t.Errorf("TransferStatus(%d).String() = 'unknown' but should be known", s)
		}
	}
}

// TestTransferStatus_UnknownValues tests unknown TransferStatus values
func TestTransferStatus_UnknownValues(t *testing.T) {
	unknowns := []TransferStatus{100, 255, -1}
	for _, s := range unknowns {
		if got := s.String(); got != "unknown" {
			t.Errorf("TransferStatus(%d).String() = %q, want 'unknown'", s, got)
		}
		// Error() should return ErrProtocol for unknown statuses
		if !errors.Is(s.Error(), ErrProtocol) {
			t.Errorf("TransferStatus(%d).Error() = %v, want %v", s, s.Error(), ErrProtocol)
		}
	}
}

// TestErrorIs tests errors.Is behavior with sentinel errors
func TestErrorIs(t *testing.T) {
	tests := []struct {
		err    error
		target error
		want   bool
	}{
		{ErrStall, ErrStall, true},
		{ErrStall, ErrNAK, false},
		{ErrTimeout, ErrTimeout, true},
		{nil, ErrStall, false},
	}

	for _, tt := range tests {
		got := errors.Is(tt.err, tt.target)
		if got != tt.want {
			t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err, tt.target, got, tt.want)
		}
	}
}

// TestAllSentinelErrors tests all sentinel errors are non-nil and have messages
func TestAllSentinelErrors(t *testing.T) {
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
		ErrDescriptorTooShort,
		ErrDescriptorTypeMismatch,
		ErrSetupPacketTooShort,
		ErrAlreadyRunning,
		ErrNotRunning,
		ErrInvalidParameter,
		ErrNoResources,
		ErrReset,
	}

	for i, err := range errs {
		if err == nil {
			t.Errorf("sentinel error %d is nil", i)
			continue
		}
		msg := err.Error()
		if msg == "" {
			t.Errorf("sentinel error %d has empty message", i)
		}
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkTransferStatus_String(b *testing.B) {
	statuses := []TransferStatus{
		TransferStatusSuccess,
		TransferStatusError,
		TransferStatusStall,
		TransferStatusNAK,
		TransferStatusTimeout,
		TransferStatusCancelled,
		TransferStatusOverrun,
		TransferStatusUnderrun,
	}

	for _, s := range statuses {
		b.Run(s.String(), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.String()
			}
		})
	}

	b.Run("Unknown", func(b *testing.B) {
		s := TransferStatus(99)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = s.String()
		}
	})
}

func BenchmarkTransferStatus_Error(b *testing.B) {
	statuses := []TransferStatus{
		TransferStatusSuccess,
		TransferStatusStall,
		TransferStatusNAK,
		TransferStatusTimeout,
		TransferStatusCancelled,
		TransferStatusOverrun,
		TransferStatusUnderrun,
		TransferStatusError,
	}

	for _, s := range statuses {
		b.Run(s.String(), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = s.Error()
			}
		})
	}
}

func BenchmarkError_Error(b *testing.B) {
	errors := []error{
		ErrStall,
		ErrNAK,
		ErrTimeout,
		ErrCancelled,
		ErrProtocol,
		ErrNoDevice,
		ErrInvalidRequest,
	}

	for _, err := range errors {
		b.Run(err.Error(), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = err.Error()
			}
		})
	}
}

func BenchmarkErrorsIs(b *testing.B) {
	testCases := []struct {
		name   string
		err    error
		target error
	}{
		{"match", ErrStall, ErrStall},
		{"nomatch", ErrStall, ErrNAK},
		{"nil", nil, ErrStall},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = errors.Is(tc.err, tc.target)
			}
		})
	}
}
