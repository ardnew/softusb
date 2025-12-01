package device

import (
	"context"
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
