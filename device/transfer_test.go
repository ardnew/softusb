package device

import (
	"context"
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
