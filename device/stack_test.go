package device

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ardnew/softusb/device/hal"
	"github.com/ardnew/softusb/pkg"
)

// mockHAL implements hal.DeviceHAL for testing.
type mockHAL struct {
	initCalled   bool
	startCalled  bool
	stopCalled   bool
	connected    bool
	speed        hal.Speed
	setupPackets chan hal.SetupPacket
	address      uint8
	endpoints    []hal.EndpointConfig
	stalled      map[uint8]bool
	mutex        sync.Mutex
	readData     map[uint8][]byte
	writeData    map[uint8][]byte

	// Channels for connect/disconnect signaling
	connectChan    chan struct{}
	disconnectChan chan struct{}
}

func newMockHAL() *mockHAL {
	return &mockHAL{
		speed:          hal.SpeedFull,
		connected:      true,
		setupPackets:   make(chan hal.SetupPacket, 10),
		stalled:        make(map[uint8]bool),
		readData:       make(map[uint8][]byte),
		writeData:      make(map[uint8][]byte),
		connectChan:    make(chan struct{}),
		disconnectChan: make(chan struct{}),
	}
}

func (m *mockHAL) Init(ctx context.Context) error {
	m.initCalled = true
	return nil
}

func (m *mockHAL) Start() error {
	m.startCalled = true
	return nil
}

func (m *mockHAL) Stop() error {
	m.stopCalled = true
	return nil
}

func (m *mockHAL) SetAddress(address uint8) error {
	m.address = address
	return nil
}

func (m *mockHAL) ConfigureEndpoints(endpoints []hal.EndpointConfig) error {
	m.endpoints = endpoints
	return nil
}

func (m *mockHAL) ReadSetup(ctx context.Context, out *hal.SetupPacket) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case setup := <-m.setupPackets:
		*out = setup
		return nil
	}
}

func (m *mockHAL) WriteEP0(ctx context.Context, data []byte) error {
	return nil
}

func (m *mockHAL) ReadEP0(ctx context.Context, buf []byte) (int, error) {
	return 0, nil
}

func (m *mockHAL) StallEP0() error {
	m.mutex.Lock()
	m.stalled[0] = true
	m.mutex.Unlock()
	return nil
}

func (m *mockHAL) AckEP0() error {
	return nil
}

func (m *mockHAL) Read(ctx context.Context, address uint8, buf []byte) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if data, ok := m.readData[address]; ok {
		n := copy(buf, data)
		return n, nil
	}
	return 0, nil
}

func (m *mockHAL) Write(ctx context.Context, address uint8, data []byte) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.writeData[address] = append([]byte{}, data...)
	return len(data), nil
}

func (m *mockHAL) Stall(address uint8) error {
	m.mutex.Lock()
	m.stalled[address] = true
	m.mutex.Unlock()
	return nil
}

func (m *mockHAL) ClearStall(address uint8) error {
	m.mutex.Lock()
	m.stalled[address] = false
	m.mutex.Unlock()
	return nil
}

func (m *mockHAL) IsConnected() bool {
	return m.connected
}

func (m *mockHAL) GetSpeed() hal.Speed {
	return m.speed
}

func (m *mockHAL) WaitConnect(ctx context.Context) error {
	if m.connected {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.connectChan:
		return nil
	}
}

func (m *mockHAL) WaitDisconnect(ctx context.Context) error {
	if !m.connected {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.disconnectChan:
		return nil
	}
}

func (m *mockHAL) sendSetup(setup *SetupPacket) {
	m.setupPackets <- hal.SetupPacket{
		RequestType: setup.RequestType,
		Request:     setup.Request,
		Value:       setup.Value,
		Index:       setup.Index,
		Length:      setup.Length,
	}
}

func (m *mockHAL) setReadData(addr uint8, data []byte) {
	m.mutex.Lock()
	m.readData[addr] = data
	m.mutex.Unlock()
}

func TestNewStack(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()

	stack := NewStack(dev, hal)

	if stack.device != dev {
		t.Error("device not set")
	}
	if stack.hal != hal {
		t.Error("HAL not set")
	}
}

func TestStackStartStop(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	err := stack.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !hal.initCalled {
		t.Error("HAL Init() not called")
	}
	if !hal.startCalled {
		t.Error("HAL Start() not called")
	}
	if !stack.IsRunning() {
		t.Error("stack should be running")
	}

	// Double start should fail
	err = stack.Start(ctx)
	if err == nil {
		t.Error("double Start() should fail")
	}

	err = stack.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if !hal.stopCalled {
		t.Error("HAL Stop() not called")
	}
	if stack.IsRunning() {
		t.Error("stack should not be running")
	}
}

func TestStackDevice(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()
	stack := NewStack(dev, hal)

	if stack.Device() != dev {
		t.Error("Device() returned wrong device")
	}
}

func TestStackSubmitTransferNotRunning(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ep := &Endpoint{Address: 0x81}
	transfer := NewBulkTransfer(ep, make([]byte, 64))

	err := stack.SubmitTransfer(transfer)
	if err != pkg.ErrNotConfigured {
		t.Errorf("SubmitTransfer() error = %v, want %v", err, pkg.ErrNotConfigured)
	}
}

func TestStackSubmitTransferNotConfigured(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	ep := &Endpoint{Address: 0x81}
	transfer := NewBulkTransfer(ep, make([]byte, 64))

	err := stack.SubmitTransfer(transfer)
	if err != pkg.ErrNotConfigured {
		t.Errorf("SubmitTransfer() error = %v, want %v", err, pkg.ErrNotConfigured)
	}
}

func TestStackSubmitTransfer(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 64}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	data := []byte("test data")
	transfer := NewBulkTransfer(ep, data)

	done := make(chan struct{})
	transfer.WithCallback(func(t *Transfer) {
		close(done)
	})

	err := stack.SubmitTransfer(transfer)
	if err != nil {
		t.Fatalf("SubmitTransfer() error = %v", err)
	}

	select {
	case <-done:
		if !transfer.IsSuccess() {
			t.Errorf("transfer status = %v, want success", transfer.Status)
		}
	case <-time.After(time.Second):
		t.Error("transfer did not complete")
	}
}

func TestStackRead(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x02, Attributes: EndpointTypeBulk, MaxPacketSize: 64}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	hal := newMockHAL()
	hal.setReadData(0x02, []byte("hello"))
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	buf := make([]byte, 64)
	n, err := stack.Read(ctx, ep, buf)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Read() = %d, want 5", n)
	}
	if string(buf[:n]) != "hello" {
		t.Errorf("Read() data = %q, want %q", buf[:n], "hello")
	}
}

func TestStackWrite(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 64}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	data := []byte("world")
	n, err := stack.Write(ctx, ep, data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Write() = %d, want 5", n)
	}

	hal.mutex.Lock()
	written := hal.writeData[0x81]
	hal.mutex.Unlock()

	if string(written) != "world" {
		t.Errorf("written data = %q, want %q", written, "world")
	}
}

func TestStackReadNotConfigured(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	ep := &Endpoint{Address: 0x02}
	_, err := stack.Read(ctx, ep, make([]byte, 64))
	if err != pkg.ErrNotConfigured {
		t.Errorf("Read() error = %v, want %v", err, pkg.ErrNotConfigured)
	}
}

func TestStackCancelTransfers(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	// Add a transfer to pending using internal array structure
	// EP 0x81 (IN endpoint 1) maps to index 17 (16 + 1)
	transfer := NewBulkTransfer(ep, make([]byte, 64))
	idx := endpointIndex(0x81)
	stack.transferMutex.Lock()
	stack.pendingTransfers[idx][0] = transfer
	stack.pendingTransferCounts[idx] = 1
	stack.transferMutex.Unlock()

	stack.CancelTransfers(0x81)

	if !transfer.IsCancelled() {
		t.Error("transfer should be cancelled")
	}
}

func TestErrorToStatus(t *testing.T) {
	tests := []struct {
		err  error
		want pkg.TransferStatus
	}{
		{nil, pkg.TransferStatusSuccess},
		{pkg.ErrStall, pkg.TransferStatusStall},
		{pkg.ErrNAK, pkg.TransferStatusNAK},
		{pkg.ErrTimeout, pkg.TransferStatusTimeout},
		{pkg.ErrCancelled, pkg.TransferStatusCancelled},
		{pkg.ErrOverrun, pkg.TransferStatusOverrun},
		{pkg.ErrUnderrun, pkg.TransferStatusUnderrun},
		{pkg.ErrProtocol, pkg.TransferStatusError},
	}

	for _, tt := range tests {
		if got := errorToStatus(tt.err); got != tt.want {
			t.Errorf("errorToStatus(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

// =============================================================================
// benchHAL - Minimal mock HAL for benchmarking
// =============================================================================

// benchHAL is a minimal HAL implementation for benchmarking.
// It has zero latency by default but can simulate configurable delays.
type benchHAL struct {
	connected bool
	speed     hal.Speed
	latency   time.Duration // Optional latency for realistic I/O simulation
}

func newBenchHAL(latency time.Duration) *benchHAL {
	return &benchHAL{
		connected: true,
		speed:     hal.SpeedHigh,
		latency:   latency,
	}
}

func (b *benchHAL) Init(ctx context.Context) error {
	return nil
}

func (b *benchHAL) Start() error {
	return nil
}

func (b *benchHAL) Stop() error {
	return nil
}

func (b *benchHAL) SetAddress(address uint8) error {
	return nil
}

func (b *benchHAL) ConfigureEndpoints(endpoints []hal.EndpointConfig) error {
	return nil
}

func (b *benchHAL) ReadSetup(ctx context.Context, out *hal.SetupPacket) error {
	if b.latency > 0 {
		time.Sleep(b.latency)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Return a minimal setup packet
		out.RequestType = 0x80 // Device-to-host
		out.Request = 0x06     // GET_DESCRIPTOR
		out.Value = 0x0100     // Device descriptor
		out.Index = 0
		out.Length = 18
		return nil
	}
}

func (b *benchHAL) WriteEP0(ctx context.Context, data []byte) error {
	if b.latency > 0 {
		time.Sleep(b.latency)
	}
	return nil
}

func (b *benchHAL) ReadEP0(ctx context.Context, buf []byte) (int, error) {
	if b.latency > 0 {
		time.Sleep(b.latency)
	}
	return 0, nil
}

func (b *benchHAL) StallEP0() error {
	return nil
}

func (b *benchHAL) AckEP0() error {
	return nil
}

func (b *benchHAL) Read(ctx context.Context, address uint8, buf []byte) (int, error) {
	if b.latency > 0 {
		time.Sleep(b.latency)
	}
	// Simulate reading data - fill buffer with test pattern
	for i := range buf {
		buf[i] = byte(i)
	}
	return len(buf), nil
}

func (b *benchHAL) Write(ctx context.Context, address uint8, data []byte) (int, error) {
	if b.latency > 0 {
		time.Sleep(b.latency)
	}
	return len(data), nil
}

func (b *benchHAL) Stall(address uint8) error {
	return nil
}

func (b *benchHAL) ClearStall(address uint8) error {
	return nil
}

func (b *benchHAL) IsConnected() bool {
	return b.connected
}

func (b *benchHAL) GetSpeed() hal.Speed {
	return b.speed
}

func (b *benchHAL) WaitConnect(ctx context.Context) error {
	if b.latency > 0 {
		time.Sleep(b.latency)
	}
	return nil
}

func (b *benchHAL) WaitDisconnect(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestStack_EndpointIndex tests endpointIndex calculation for all addresses
func TestStack_EndpointIndex(t *testing.T) {
	tests := []struct {
		address uint8
		want    int
	}{
		// OUT endpoints (0-15)
		{0x00, 0},
		{0x01, 1},
		{0x07, 7},
		{0x0F, 15},

		// IN endpoints (16-31)
		{0x80, 16},
		{0x81, 17},
		{0x87, 23},
		{0x8F, 31},
	}

	for _, tt := range tests {
		got := endpointIndex(tt.address)
		if got != tt.want {
			t.Errorf("endpointIndex(0x%02X) = %d, want %d", tt.address, got, tt.want)
		}
	}
}

// TestStack_SubmitTransfer_PendingLimit tests MaxPendingTransfersPerEndpoint limit
func TestStack_SubmitTransfer_PendingLimit(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 64}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	idx := endpointIndex(0x81)

	// Fill pending queue to capacity
	stack.transferMutex.Lock()
	for i := 0; i < MaxPendingTransfersPerEndpoint; i++ {
		stack.pendingTransfers[idx][i] = NewBulkTransfer(ep, nil)
	}
	stack.pendingTransferCounts[idx] = MaxPendingTransfersPerEndpoint
	stack.transferMutex.Unlock()

	// Should fail with pending limit exceeded
	transfer := NewBulkTransfer(ep, nil)
	err := stack.SubmitTransfer(transfer)
	if err != pkg.ErrNoResources {
		t.Errorf("error = %v, want %v", err, pkg.ErrNoResources)
	}
}

// TestStack_DoubleStop tests that Stop is idempotent
func TestStack_DoubleStop(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)

	err := stack.Stop()
	if err != nil {
		t.Fatalf("first Stop() error = %v", err)
	}

	// Second stop should be fine (no panic)
	err = stack.Stop()
	if err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
}

// TestStack_StopNotStarted tests Stop when not started
func TestStack_StopNotStarted(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newMockHAL()
	stack := NewStack(dev, hal)

	err := stack.Stop()
	// Should be a no-op or error but not panic
	_ = err
}

// TestStack_CancelAllTransfers tests CancelTransfers on multiple endpoints
func TestStack_CancelAllTransfers(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep1 := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	ep2 := &Endpoint{Address: 0x02, Attributes: EndpointTypeBulk}
	iface.AddEndpoint(ep1)
	iface.AddEndpoint(ep2)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	hal := newMockHAL()
	stack := NewStack(dev, hal)

	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	// Add transfers to different endpoints
	t1 := NewBulkTransfer(ep1, nil)
	t2 := NewBulkTransfer(ep2, nil)

	stack.transferMutex.Lock()
	stack.pendingTransfers[endpointIndex(0x81)][0] = t1
	stack.pendingTransferCounts[endpointIndex(0x81)] = 1
	stack.pendingTransfers[endpointIndex(0x02)][0] = t2
	stack.pendingTransferCounts[endpointIndex(0x02)] = 1
	stack.transferMutex.Unlock()

	// Cancel transfers for each endpoint
	stack.CancelTransfers(0x81)
	stack.CancelTransfers(0x02)

	if !t1.IsCancelled() {
		t.Error("transfer 1 should be cancelled")
	}
	if !t2.IsCancelled() {
		t.Error("transfer 2 should be cancelled")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewStack(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newBenchHAL(0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStack(dev, hal)
	}
}

func BenchmarkStack_StartStop(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	hal := newBenchHAL(0)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stack := NewStack(dev, hal)
		_ = stack.Start(ctx)
		_ = stack.Stop()
	}
}

func BenchmarkStack_SubmitTransfer(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 512}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	bufferSizes := []int{8, 64, 512, 1024}

	b.Run("benchHAL", func(b *testing.B) {
		for _, size := range bufferSizes {
			b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
				hal := newBenchHAL(0)
				stack := NewStack(dev, hal)
				ctx := context.Background()
				stack.Start(ctx)
				defer stack.Stop()

				data := make([]byte, size)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					transfer := NewBulkTransfer(ep, data)
					_ = stack.SubmitTransfer(transfer)
				}
			})
		}
	})
}

func BenchmarkStack_Write(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 512}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	bufferSizes := []int{8, 64, 512, 1024}
	latencies := []time.Duration{0, 10 * time.Microsecond}

	for _, latency := range latencies {
		latencyName := "noLatency"
		if latency > 0 {
			latencyName = "10µs"
		}
		b.Run(latencyName, func(b *testing.B) {
			for _, size := range bufferSizes {
				b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
					hal := newBenchHAL(latency)
					stack := NewStack(dev, hal)
					ctx := context.Background()
					stack.Start(ctx)
					defer stack.Stop()

					data := make([]byte, size)
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = stack.Write(ctx, ep, data)
					}
				})
			}
		})
	}
}

func BenchmarkStack_Read(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x02, Attributes: EndpointTypeBulk, MaxPacketSize: 512}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	bufferSizes := []int{8, 64, 512, 1024}

	for _, size := range bufferSizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			hal := newBenchHAL(0)
			stack := NewStack(dev, hal)
			ctx := context.Background()
			stack.Start(ctx)
			defer stack.Stop()

			buf := make([]byte, size)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = stack.Read(ctx, ep, buf)
			}
		})
	}
}

func BenchmarkStack_CancelTransfers(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	hal := newBenchHAL(0)
	stack := NewStack(dev, hal)
	ctx := context.Background()
	stack.Start(ctx)
	defer stack.Stop()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stack.CancelTransfers(0x81)
	}
}

func BenchmarkEndpointIndex(b *testing.B) {
	addresses := []uint8{0x00, 0x01, 0x0F, 0x80, 0x81, 0x8F}
	for _, addr := range addresses {
		b.Run(fmt.Sprintf("0x%02X", addr), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = endpointIndex(addr)
			}
		})
	}
}

func BenchmarkErrorToStatus(b *testing.B) {
	errors := []error{nil, pkg.ErrStall, pkg.ErrNAK, pkg.ErrTimeout, pkg.ErrCancelled}
	for _, err := range errors {
		name := "nil"
		if err != nil {
			name = err.Error()
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = errorToStatus(err)
			}
		})
	}
}
