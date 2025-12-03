package device

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ardnew/softusb/pkg"
)

func TestNewControlTransfer(t *testing.T) {
	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, 18)
	data := make([]byte, 18)

	xfer := NewControlTransfer(&setup, data)

	if xfer.Type != EndpointTypeControl {
		t.Errorf("Type = %d, want %d", xfer.Type, EndpointTypeControl)
	}
	if xfer.Setup != &setup {
		t.Error("Setup not set correctly")
	}
	if len(xfer.Buffer) != 18 {
		t.Errorf("Buffer length = %d, want 18", len(xfer.Buffer))
	}
	if xfer.ctx == nil {
		t.Error("context should be initialized")
	}
}

func TestNewBulkTransfer(t *testing.T) {
	ep := &Endpoint{
		Address:       0x81,
		Attributes:    EndpointTypeBulk,
		MaxPacketSize: 512,
	}
	data := make([]byte, 1024)

	xfer := NewBulkTransfer(ep, data)

	if xfer.Type != EndpointTypeBulk {
		t.Errorf("Type = %d, want %d", xfer.Type, EndpointTypeBulk)
	}
	if xfer.Endpoint != ep {
		t.Error("Endpoint not set correctly")
	}
}

func TestNewInterruptTransfer(t *testing.T) {
	ep := &Endpoint{
		Address:       0x83,
		Attributes:    EndpointTypeInterrupt,
		MaxPacketSize: 8,
		Interval:      10,
	}
	data := make([]byte, 8)

	xfer := NewInterruptTransfer(ep, data)

	if xfer.Type != EndpointTypeInterrupt {
		t.Errorf("Type = %d, want %d", xfer.Type, EndpointTypeInterrupt)
	}
}

func TestNewIsochronousTransfer(t *testing.T) {
	ep := &Endpoint{
		Address:       0x04,
		Attributes:    EndpointTypeIsochronous | IsoSyncAsync,
		MaxPacketSize: 1023,
		Interval:      1,
	}
	data := make([]byte, 4092)

	xfer := NewIsochronousTransfer(ep, data, 4)

	if xfer.Type != EndpointTypeIsochronous {
		t.Errorf("Type = %d, want %d", xfer.Type, EndpointTypeIsochronous)
	}
	if xfer.NumIsoPackets != 4 {
		t.Errorf("NumIsoPackets = %d, want 4", xfer.NumIsoPackets)
	}
}

func TestTransferWithContext(t *testing.T) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	xfer.WithContext(ctx)

	// The transfer wraps the parent context with its own cancel, so we can't compare directly
	// Instead verify it's derived from the parent by cancelling parent
	cancel()
	select {
	case <-xfer.Context().Done():
		// Good - child context was cancelled when parent was
	default:
		t.Error("context should be cancelled when parent is cancelled")
	}
}

func TestTransferWithCallback(t *testing.T) {
	called := false
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	xfer.WithCallback(func(t *Transfer) {
		called = true
	})

	xfer.Complete(pkg.TransferStatusSuccess, 0, nil)

	if !called {
		t.Error("callback should have been called")
	}
}

func TestTransferCancel(t *testing.T) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)

	xfer.Cancel()

	if !xfer.IsCancelled() {
		t.Error("transfer should be cancelled")
	}
	if xfer.Status != pkg.TransferStatusCancelled {
		t.Errorf("Status = %v, want %v", xfer.Status, pkg.TransferStatusCancelled)
	}
	if xfer.Error != pkg.ErrCancelled {
		t.Errorf("Error = %v, want %v", xfer.Error, pkg.ErrCancelled)
	}
}

func TestTransferComplete(t *testing.T) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, make([]byte, 100))

	xfer.Complete(pkg.TransferStatusSuccess, 50, nil)

	if !xfer.IsCompleted() {
		t.Error("transfer should be completed")
	}
	if !xfer.IsSuccess() {
		t.Error("transfer should be successful")
	}
	if xfer.Length != 50 {
		t.Errorf("Length = %d, want 50", xfer.Length)
	}
}

func TestTransferCompleteOnce(t *testing.T) {
	callCount := 0
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	xfer.WithCallback(func(t *Transfer) {
		callCount++
	})

	xfer.Complete(pkg.TransferStatusSuccess, 0, nil)
	xfer.Complete(pkg.TransferStatusError, 0, pkg.ErrProtocol)

	if callCount != 1 {
		t.Errorf("callback called %d times, want 1", callCount)
	}
	// Status should remain from first completion
	if xfer.Status != pkg.TransferStatusSuccess {
		t.Errorf("Status = %v, want %v", xfer.Status, pkg.TransferStatusSuccess)
	}
}

func TestTransferReset(t *testing.T) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	xfer.Complete(pkg.TransferStatusSuccess, 100, nil)

	xfer.Reset()

	if xfer.IsCompleted() {
		t.Error("transfer should not be completed after reset")
	}
	if xfer.Length != 0 {
		t.Errorf("Length = %d, want 0", xfer.Length)
	}
}

func TestTransferDirection(t *testing.T) {
	var getDescSetup, setAddrSetup SetupPacket
	GetDescriptorSetup(&getDescSetup, DescriptorTypeDevice, 0, 18)
	GetSetAddressSetup(&setAddrSetup, 5)

	tests := []struct {
		name    string
		xfer    *Transfer
		wantIn  bool
		wantOut bool
	}{
		{
			name:    "control IN",
			xfer:    NewControlTransfer(&getDescSetup, nil),
			wantIn:  true,
			wantOut: false,
		},
		{
			name:    "control OUT",
			xfer:    NewControlTransfer(&setAddrSetup, nil),
			wantIn:  false,
			wantOut: true,
		},
		{
			name:    "bulk IN",
			xfer:    NewBulkTransfer(&Endpoint{Address: 0x81}, nil),
			wantIn:  true,
			wantOut: false,
		},
		{
			name:    "bulk OUT",
			xfer:    NewBulkTransfer(&Endpoint{Address: 0x02}, nil),
			wantIn:  false,
			wantOut: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.xfer.IsIn(); got != tt.wantIn {
				t.Errorf("IsIn() = %v, want %v", got, tt.wantIn)
			}
			if got := tt.xfer.IsOut(); got != tt.wantOut {
				t.Errorf("IsOut() = %v, want %v", got, tt.wantOut)
			}
		})
	}
}

func TestTransferMaxPacketSize(t *testing.T) {
	var getDescSetup SetupPacket
	GetDescriptorSetup(&getDescSetup, DescriptorTypeDevice, 0, 18)

	tests := []struct {
		name string
		xfer *Transfer
		want int
	}{
		{
			name: "with endpoint",
			xfer: NewBulkTransfer(&Endpoint{Address: 0x81, MaxPacketSize: 512}, nil),
			want: 512,
		},
		{
			name: "control (default)",
			xfer: NewControlTransfer(&getDescSetup, nil),
			want: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.xfer.MaxPacketSize(); got != tt.want {
				t.Errorf("MaxPacketSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIsoPacketSetup(t *testing.T) {
	ep := &Endpoint{
		Address:       0x04,
		Attributes:    EndpointTypeIsochronous,
		MaxPacketSize: 192,
	}
	xfer := NewIsochronousTransfer(ep, make([]byte, 768), 4)
	xfer.SetupIsoPackets(192)

	for i := 0; i < 4; i++ {
		expectedOffset := i * 192
		pkt := xfer.IsoPacket(i)
		if pkt == nil {
			t.Fatalf("IsoPacket(%d) returned nil", i)
		}
		if pkt.Offset != expectedOffset {
			t.Errorf("IsoPacket(%d).Offset = %d, want %d", i, pkt.Offset, expectedOffset)
		}
		if pkt.Length != 192 {
			t.Errorf("IsoPacket(%d).Length = %d, want 192", i, pkt.Length)
		}
	}
}

func TestIsoPacketSetupVariable(t *testing.T) {
	ep := &Endpoint{
		Address:    0x04,
		Attributes: EndpointTypeIsochronous,
	}
	xfer := NewIsochronousTransfer(ep, make([]byte, 500), 3)
	xfer.SetupIsoPacketsVariable([]int{100, 150, 200})

	expected := []struct {
		offset int
		length int
	}{
		{0, 100},
		{100, 150},
		{250, 200},
	}

	for i, exp := range expected {
		pkt := xfer.IsoPacket(i)
		if pkt == nil {
			t.Fatalf("IsoPacket(%d) returned nil", i)
		}
		if pkt.Offset != exp.offset {
			t.Errorf("IsoPacket(%d).Offset = %d, want %d", i, pkt.Offset, exp.offset)
		}
		if pkt.Length != exp.length {
			t.Errorf("IsoPacket(%d).Length = %d, want %d", i, pkt.Length, exp.length)
		}
	}
}

func TestIsoTotalLength(t *testing.T) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
	xfer := NewIsochronousTransfer(ep, make([]byte, 768), 4)
	xfer.SetupIsoPackets(192)

	if got := xfer.TotalIsoLength(); got != 768 {
		t.Errorf("TotalIsoLength() = %d, want 768", got)
	}
}

func TestIsoActualLength(t *testing.T) {
	ep := &Endpoint{Address: 0x84, Attributes: EndpointTypeIsochronous}
	xfer := NewIsochronousTransfer(ep, make([]byte, 768), 4)
	xfer.SetupIsoPackets(192)

	// Simulate actual transfer lengths using the IsoPacket accessor
	xfer.IsoPacket(0).ActualLength = 192
	xfer.IsoPacket(1).ActualLength = 100
	xfer.IsoPacket(2).ActualLength = 150
	xfer.IsoPacket(3).ActualLength = 0

	if got := xfer.ActualIsoLength(); got != 442 {
		t.Errorf("ActualIsoLength() = %d, want 442", got)
	}
}

func TestTransferPool(t *testing.T) {
	pool := NewTransferPool()

	// Get transfer from pool
	xfer := pool.Get()
	if xfer == nil {
		t.Fatal("Get() returned nil")
	}

	// Configure and use
	xfer.Type = EndpointTypeBulk
	xfer.Buffer = make([]byte, 100)
	xfer.Complete(pkg.TransferStatusSuccess, 50, nil)

	// Return to pool
	pool.Put(xfer)

	// Get again - should be reset
	xfer2 := pool.Get()
	if xfer2.IsCompleted() {
		t.Error("pooled transfer should be reset")
	}
	if xfer2.Buffer != nil {
		t.Error("pooled transfer buffer should be nil")
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestTransfer_ZeroValue tests zero-value Transfer behavior
func TestTransfer_ZeroValue(t *testing.T) {
	var xfer Transfer

	// Zero value should be safe to use
	if xfer.IsCompleted() {
		t.Error("zero-value transfer should not be completed")
	}
	if xfer.IsCancelled() {
		t.Error("zero-value transfer should not be cancelled")
	}
	if xfer.IsSuccess() {
		t.Error("zero-value transfer should not be success")
	}
	if xfer.IsIn() {
		t.Error("zero-value transfer should be OUT (default)")
	}
	if !xfer.IsOut() {
		t.Error("zero-value transfer IsOut() should be true")
	}

	// Direction with no endpoint or setup should default to OUT
	if xfer.Direction() != EndpointDirectionOut {
		t.Errorf("Direction() = 0x%02X, want 0x%02X", xfer.Direction(), EndpointDirectionOut)
	}

	// MaxPacketSize defaults to 64 (EP0 default)
	if xfer.MaxPacketSize() != 64 {
		t.Errorf("MaxPacketSize() = %d, want 64", xfer.MaxPacketSize())
	}
}

// TestTransfer_ContextNil tests Context() with nil internal ctx
func TestTransfer_ContextNil(t *testing.T) {
	xfer := &Transfer{} // ctx is nil
	ctx := xfer.Context()
	if ctx == nil {
		t.Error("Context() should return background context when internal is nil")
	}
}

// TestTransfer_CompleteTwice tests that Complete is idempotent
func TestTransfer_CompleteTwice(t *testing.T) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, make([]byte, 100))

	xfer.Complete(pkg.TransferStatusSuccess, 50, nil)
	firstStatus := xfer.Status
	firstLength := xfer.Length

	// Second complete should be ignored
	xfer.Complete(pkg.TransferStatusError, 100, pkg.ErrProtocol)

	if xfer.Status != firstStatus {
		t.Errorf("Status = %v, want %v (first)", xfer.Status, firstStatus)
	}
	if xfer.Length != firstLength {
		t.Errorf("Length = %d, want %d (first)", xfer.Length, firstLength)
	}
}

// TestTransfer_CancelTwice tests that Cancel is idempotent
func TestTransfer_CancelTwice(t *testing.T) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)

	xfer.Cancel()
	if !xfer.IsCancelled() {
		t.Error("transfer should be cancelled after first Cancel()")
	}

	// Second cancel should be no-op
	xfer.Cancel()
	if !xfer.IsCancelled() {
		t.Error("transfer should still be cancelled after second Cancel()")
	}
}

// TestTransfer_ResetClearsAllState tests Reset clears all fields
func TestTransfer_ResetClearsAllState(t *testing.T) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, make([]byte, 100))
	xfer.Complete(pkg.TransferStatusSuccess, 100, nil)
	xfer.Cancel()

	xfer.Reset()

	if xfer.IsCompleted() {
		t.Error("completed should be false after Reset")
	}
	if xfer.IsCancelled() {
		t.Error("cancelled should be false after Reset")
	}
	if xfer.Status != 0 {
		t.Errorf("Status = %v, want 0", xfer.Status)
	}
	if xfer.Length != 0 {
		t.Errorf("Length = %d, want 0", xfer.Length)
	}
	if xfer.Error != nil {
		t.Errorf("Error = %v, want nil", xfer.Error)
	}
}

// TestIsoPacket_OutOfRange tests IsoPacket returns nil for invalid indices
func TestIsoPacket_OutOfRange(t *testing.T) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
	xfer := NewIsochronousTransfer(ep, make([]byte, 1024), 4)

	tests := []struct {
		name  string
		index int
		want  bool // true if should return non-nil
	}{
		{"negative", -1, false},
		{"zero (valid)", 0, true},
		{"last valid", 3, true},
		{"first invalid", 4, false},
		{"large invalid", 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := xfer.IsoPacket(tt.index)
			if (pkt != nil) != tt.want {
				t.Errorf("IsoPacket(%d) = %v, want non-nil=%v", tt.index, pkt, tt.want)
			}
		})
	}
}

// TestNewIsochronousTransfer_MaxPackets tests clamping to MaxIsoPackets
func TestNewIsochronousTransfer_MaxPackets(t *testing.T) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}

	tests := []struct {
		name     string
		request  int
		expected int
	}{
		{"zero", 0, 0},
		{"one", 1, 1},
		{"typical", 32, 32},
		{"max", MaxIsoPackets, MaxIsoPackets},
		{"over max", MaxIsoPackets + 1, MaxIsoPackets},
		{"large", 10000, MaxIsoPackets},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xfer := NewIsochronousTransfer(ep, nil, tt.request)
			if xfer.NumIsoPackets != tt.expected {
				t.Errorf("NumIsoPackets = %d, want %d", xfer.NumIsoPackets, tt.expected)
			}
		})
	}
}

// TestIsoTotalLength_EmptyPackets tests TotalIsoLength with no packets
func TestIsoTotalLength_EmptyPackets(t *testing.T) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
	xfer := NewIsochronousTransfer(ep, nil, 0)
	if got := xfer.TotalIsoLength(); got != 0 {
		t.Errorf("TotalIsoLength() = %d, want 0", got)
	}
}

// TestTransfer_ConcurrentComplete tests concurrent Complete calls
func TestTransfer_ConcurrentComplete(t *testing.T) {
	const goroutines = 100
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)

	callbackCount := int32(0)
	xfer.WithCallback(func(*Transfer) {
		atomic.AddInt32(&callbackCount, 1)
	})

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			xfer.Complete(pkg.TransferStatus(n%3), n, nil)
		}(i)
	}

	wg.Wait()

	// Callback should be called exactly once
	if callbackCount != 1 {
		t.Errorf("callback called %d times, want 1", callbackCount)
	}
	if !xfer.IsCompleted() {
		t.Error("transfer should be completed")
	}
}

// TestTransfer_ConcurrentCancel tests concurrent Cancel calls
func TestTransfer_ConcurrentCancel(t *testing.T) {
	const goroutines = 100
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			xfer.Cancel()
		}()
	}

	wg.Wait()

	// Should be cancelled
	if !xfer.IsCancelled() {
		t.Error("transfer should be cancelled")
	}
	if xfer.Status != pkg.TransferStatusCancelled {
		t.Errorf("Status = %v, want %v", xfer.Status, pkg.TransferStatusCancelled)
	}
}

// TestTransfer_ConcurrentReset tests concurrent Reset with other operations
func TestTransfer_ConcurrentReset(t *testing.T) {
	const iterations = 1000
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)

	var wg sync.WaitGroup
	wg.Add(4)

	// Goroutine completing
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			xfer.Complete(pkg.TransferStatusSuccess, i, nil)
		}
	}()

	// Goroutine resetting
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			xfer.Reset()
		}
	}()

	// Goroutine checking status
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = xfer.IsCompleted()
		}
	}()

	// Goroutine cancelling
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			xfer.Cancel()
		}
	}()

	wg.Wait()
	// Success if no race/panic
}

// TestTransferPool_StressTest tests pool under high contention
func TestTransferPool_StressTest(t *testing.T) {
	pool := NewTransferPool()
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				xfer := pool.Get()
				if xfer == nil {
					t.Error("Get() returned nil")
					return
				}
				// Use transfer
				xfer.Type = EndpointTypeBulk
				xfer.Buffer = make([]byte, 64)
				xfer.Complete(pkg.TransferStatusSuccess, 32, nil)
				// Return to pool
				pool.Put(xfer)
			}
		}()
	}

	wg.Wait()
}

// TestTransferPool_GetReturnsReset tests that Get always returns reset transfer
func TestTransferPool_GetReturnsReset(t *testing.T) {
	pool := NewTransferPool()

	for i := 0; i < 100; i++ {
		xfer := pool.Get()
		if xfer.IsCompleted() {
			t.Fatalf("iteration %d: Get() returned completed transfer", i)
		}
		if xfer.IsCancelled() {
			t.Fatalf("iteration %d: Get() returned cancelled transfer", i)
		}
		if xfer.Buffer != nil {
			t.Fatalf("iteration %d: Get() returned transfer with buffer", i)
		}

		// Dirty it up
		xfer.Type = EndpointTypeBulk
		xfer.Buffer = make([]byte, 64)
		xfer.Complete(pkg.TransferStatusSuccess, 32, nil)
		xfer.Cancel()

		pool.Put(xfer)
	}
}

// TestAllTransferTypes tests creation of all transfer types
func TestAllTransferTypes(t *testing.T) {
	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, 18)

	tests := []struct {
		name     string
		xfer     *Transfer
		wantType uint8
	}{
		{"Control", NewControlTransfer(&setup, nil), EndpointTypeControl},
		{"Bulk", NewBulkTransfer(&Endpoint{Attributes: EndpointTypeBulk}, nil), EndpointTypeBulk},
		{"Interrupt", NewInterruptTransfer(&Endpoint{Attributes: EndpointTypeInterrupt}, nil), EndpointTypeInterrupt},
		{"Isochronous", NewIsochronousTransfer(&Endpoint{Attributes: EndpointTypeIsochronous}, nil, 4), EndpointTypeIsochronous},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xfer.Type != tt.wantType {
				t.Errorf("Type = %d, want %d", tt.xfer.Type, tt.wantType)
			}
		})
	}
}

// TestIsoPacketsVariable_MismatchedSizes tests SetupIsoPacketsVariable with mismatched counts
func TestIsoPacketsVariable_MismatchedSizes(t *testing.T) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}

	// More sizes than packets
	xfer := NewIsochronousTransfer(ep, make([]byte, 1000), 3)
	xfer.SetupIsoPacketsVariable([]int{100, 200, 300, 400, 500}) // 5 sizes, 3 packets

	// Only first 3 should be set
	if pkt := xfer.IsoPacket(0); pkt.Length != 100 {
		t.Errorf("IsoPacket(0).Length = %d, want 100", pkt.Length)
	}
	if pkt := xfer.IsoPacket(2); pkt.Length != 300 {
		t.Errorf("IsoPacket(2).Length = %d, want 300", pkt.Length)
	}

	// Fewer sizes than packets
	xfer2 := NewIsochronousTransfer(ep, make([]byte, 500), 5)
	xfer2.SetupIsoPacketsVariable([]int{100, 200}) // 2 sizes, 5 packets

	// Only first 2 should be set
	if pkt := xfer2.IsoPacket(0); pkt.Length != 100 {
		t.Errorf("IsoPacket(0).Length = %d, want 100", pkt.Length)
	}
	if pkt := xfer2.IsoPacket(1); pkt.Length != 200 {
		t.Errorf("IsoPacket(1).Length = %d, want 200", pkt.Length)
	}
	// Rest should be zero
	if pkt := xfer2.IsoPacket(2); pkt.Length != 0 {
		t.Errorf("IsoPacket(2).Length = %d, want 0", pkt.Length)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewControlTransfer(b *testing.B) {
	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, 18)
	data := make([]byte, 18)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewControlTransfer(&setup, data)
	}
}

func BenchmarkNewBulkTransfer(b *testing.B) {
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 512}
	data := make([]byte, 1024)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewBulkTransfer(ep, data)
	}
}

func BenchmarkNewInterruptTransfer(b *testing.B) {
	ep := &Endpoint{Address: 0x83, Attributes: EndpointTypeInterrupt, MaxPacketSize: 8}
	data := make([]byte, 8)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewInterruptTransfer(ep, data)
	}
}

func BenchmarkNewIsochronousTransfer(b *testing.B) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous, MaxPacketSize: 1023}
	data := make([]byte, 4092)
	numPackets := []int{1, 4, 32, 128, 256}

	for _, n := range numPackets {
		b.Run(fmt.Sprintf("packets=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = NewIsochronousTransfer(ep, data, n)
			}
		})
	}
}

func BenchmarkTransfer_WithContext(b *testing.B) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xfer.WithContext(ctx)
	}
}

func BenchmarkTransfer_WithCallback(b *testing.B) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	cb := func(*Transfer) {}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xfer.WithCallback(cb)
	}
}

func BenchmarkTransfer_Complete(b *testing.B) {
	b.Run("NoCallback", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
			xfer.Complete(pkg.TransferStatusSuccess, 100, nil)
		}
	})

	b.Run("WithCallback", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
			xfer.WithCallback(func(*Transfer) {})
			xfer.Complete(pkg.TransferStatusSuccess, 100, nil)
		}
	})
}

func BenchmarkTransfer_Cancel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
		xfer.Cancel()
	}
}

func BenchmarkTransfer_IsCancelled(b *testing.B) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = xfer.IsCancelled()
	}
}

func BenchmarkTransfer_IsCompleted(b *testing.B) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = xfer.IsCompleted()
	}
}

func BenchmarkTransfer_Reset(b *testing.B) {
	xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xfer.Complete(pkg.TransferStatusSuccess, 100, nil)
		xfer.Reset()
	}
}

func BenchmarkTransfer_Direction(b *testing.B) {
	b.Run("Control", func(b *testing.B) {
		var setup SetupPacket
		GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, 18)
		xfer := NewControlTransfer(&setup, nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xfer.Direction()
		}
	})

	b.Run("Bulk", func(b *testing.B) {
		xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xfer.Direction()
		}
	})
}

func BenchmarkTransfer_MaxPacketSize(b *testing.B) {
	b.Run("WithEndpoint", func(b *testing.B) {
		xfer := NewBulkTransfer(&Endpoint{Address: 0x81, MaxPacketSize: 512}, nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xfer.MaxPacketSize()
		}
	})

	b.Run("NoEndpoint", func(b *testing.B) {
		var setup SetupPacket
		GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, 18)
		xfer := NewControlTransfer(&setup, nil)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xfer.MaxPacketSize()
		}
	})
}

func BenchmarkIsoPacket_Setup(b *testing.B) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous, MaxPacketSize: 192}
	numPackets := []int{4, 32, 128, 256}

	b.Run("Uniform", func(b *testing.B) {
		for _, n := range numPackets {
			b.Run(fmt.Sprintf("packets=%d", n), func(b *testing.B) {
				xfer := NewIsochronousTransfer(ep, make([]byte, 192*n), n)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					xfer.SetupIsoPackets(192)
				}
			})
		}
	})

	b.Run("Variable", func(b *testing.B) {
		for _, n := range numPackets {
			sizes := make([]int, n)
			for i := range sizes {
				sizes[i] = 192
			}
			b.Run(fmt.Sprintf("packets=%d", n), func(b *testing.B) {
				xfer := NewIsochronousTransfer(ep, make([]byte, 192*n), n)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					xfer.SetupIsoPacketsVariable(sizes)
				}
			})
		}
	})
}

func BenchmarkIsoPacket_Access(b *testing.B) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
	xfer := NewIsochronousTransfer(ep, make([]byte, 49152), 256)
	xfer.SetupIsoPackets(192)

	b.Run("Get", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xfer.IsoPacket(i % 256)
		}
	})

	b.Run("TotalLength", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xfer.TotalIsoLength()
		}
	})

	b.Run("ActualLength", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = xfer.ActualIsoLength()
		}
	})
}

func BenchmarkTransferPool_GetPut(b *testing.B) {
	pool := NewTransferPool()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xfer := pool.Get()
		pool.Put(xfer)
	}
}

func BenchmarkTransferPool_Concurrent(b *testing.B) {
	pool := NewTransferPool()
	goroutineCounts := []int{1, 2, 4, 8}

	for _, g := range goroutineCounts {
		b.Run(fmt.Sprintf("goroutines=%d", g), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.SetParallelism(g)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					xfer := pool.Get()
					xfer.Type = EndpointTypeBulk
					pool.Put(xfer)
				}
			})
		})
	}
}

func BenchmarkTransfer_Concurrent(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8}

	b.Run("IsCancelled", func(b *testing.B) {
		for _, g := range goroutineCounts {
			b.Run(fmt.Sprintf("goroutines=%d", g), func(b *testing.B) {
				xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
				b.ReportAllocs()
				b.ResetTimer()
				b.SetParallelism(g)
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_ = xfer.IsCancelled()
					}
				})
			})
		}
	})

	b.Run("IsCompleted", func(b *testing.B) {
		for _, g := range goroutineCounts {
			b.Run(fmt.Sprintf("goroutines=%d", g), func(b *testing.B) {
				xfer := NewBulkTransfer(&Endpoint{Address: 0x81}, nil)
				b.ReportAllocs()
				b.ResetTimer()
				b.SetParallelism(g)
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_ = xfer.IsCompleted()
					}
				})
			})
		}
	})
}
