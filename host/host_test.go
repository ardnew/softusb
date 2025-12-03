package host

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ardnew/softusb/host/hal"
)

// =============================================================================
// Mock HAL for Testing
// =============================================================================

// mockHAL implements hal.HostHAL for testing.
type mockHAL struct {
	initErr    error
	startErr   error
	stopErr    error
	closeErr   error
	numPorts   int
	portStatus hal.PortStatus
	portSpeed  hal.Speed

	// Connection simulation
	connectCh    chan int
	disconnectCh chan int

	// Transfer results
	controlResult int
	controlErr    error
	bulkResult    int
	bulkErr       error
	interruptErr  error
	isoErr        error

	// State tracking
	running bool
	mu      sync.Mutex
}

func newMockHAL() *mockHAL {
	return &mockHAL{
		numPorts:     4,
		portSpeed:    hal.SpeedFull,
		connectCh:    make(chan int, 16),
		disconnectCh: make(chan int, 16),
	}
}

func (m *mockHAL) Init(ctx context.Context) error {
	return m.initErr
}

func (m *mockHAL) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = true
	return m.startErr
}

func (m *mockHAL) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
	return m.stopErr
}

func (m *mockHAL) Close() error {
	return m.closeErr
}

func (m *mockHAL) NumPorts() int {
	return m.numPorts
}

func (m *mockHAL) GetPortStatus(port int) (hal.PortStatus, error) {
	return m.portStatus, nil
}

func (m *mockHAL) PortSpeed(port int) hal.Speed {
	return m.portSpeed
}

func (m *mockHAL) ResetPort(port int) error {
	return nil
}

func (m *mockHAL) EnablePort(port int, enable bool) error {
	return nil
}

func (m *mockHAL) ControlTransfer(ctx context.Context, addr hal.DeviceAddress, setup *hal.SetupPacket, data []byte) (int, error) {
	return m.controlResult, m.controlErr
}

func (m *mockHAL) BulkTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	return m.bulkResult, m.bulkErr
}

func (m *mockHAL) InterruptTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	return 0, m.interruptErr
}

func (m *mockHAL) IsochronousTransfer(ctx context.Context, addr hal.DeviceAddress, endpoint uint8, data []byte) (int, error) {
	return 0, m.isoErr
}

func (m *mockHAL) SetDeviceAddress(ctx context.Context, newAddr hal.DeviceAddress) error {
	return nil
}

func (m *mockHAL) ClaimInterface(addr hal.DeviceAddress, iface uint8) error {
	return nil
}

func (m *mockHAL) ReleaseInterface(addr hal.DeviceAddress, iface uint8) error {
	return nil
}

func (m *mockHAL) WaitForConnection(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case port := <-m.connectCh:
		return port, nil
	}
}

func (m *mockHAL) WaitForDisconnection(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case port := <-m.disconnectCh:
		return port, nil
	}
}

// simulateConnect simulates a device connection.
func (m *mockHAL) simulateConnect(port int) {
	m.connectCh <- port
}

// simulateDisconnect simulates a device disconnection.
func (m *mockHAL) simulateDisconnect(port int) {
	m.disconnectCh <- port
}

// Ensure mockHAL implements hal.HostHAL
var _ hal.HostHAL = (*mockHAL)(nil)

// =============================================================================
// Host Tests
// =============================================================================

func TestNew(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	if h == nil {
		t.Fatal("New returned nil")
	}
	if h.hal != mock {
		t.Error("HAL not set correctly")
	}
	if h.nextAddress != 1 {
		t.Errorf("nextAddress = %d, want 1", h.nextAddress)
	}
	if h.deviceConnected == nil {
		t.Error("deviceConnected channel is nil")
	}
	if h.deviceDisconnected == nil {
		t.Error("deviceDisconnected channel is nil")
	}
}

func TestHost_StartStop(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	ctx := context.Background()

	// Test Start
	if err := h.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !h.IsRunning() {
		t.Error("IsRunning() = false after Start")
	}

	// Test double Start
	if err := h.Start(ctx); err == nil {
		t.Error("second Start should return error")
	}

	// Test Stop
	if err := h.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if h.IsRunning() {
		t.Error("IsRunning() = true after Stop")
	}

	// Test double Stop (should be idempotent)
	if err := h.Stop(); err != nil {
		t.Fatalf("second Stop failed: %v", err)
	}
}

func TestHost_NumPorts(t *testing.T) {
	mock := newMockHAL()
	mock.numPorts = 8
	h := New(mock)

	if got := h.NumPorts(); got != 8 {
		t.Errorf("NumPorts() = %d, want 8", got)
	}
}

func TestHost_GetPortStatus(t *testing.T) {
	mock := newMockHAL()
	mock.portStatus = hal.PortStatus{
		Connected: true,
		Enabled:   true,
		PowerOn:   true,
		Speed:     hal.SpeedHigh,
	}
	h := New(mock)

	status, err := h.GetPortStatus(1)
	if err != nil {
		t.Fatalf("GetPortStatus failed: %v", err)
	}

	if !status.Connected {
		t.Error("Connected = false, want true")
	}
	if !status.Enabled {
		t.Error("Enabled = false, want true")
	}
	if status.Speed != hal.SpeedHigh {
		t.Errorf("Speed = %v, want SpeedHigh", status.Speed)
	}
}

func TestHost_Devices(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	// Initially no devices
	devices := h.Devices()
	if len(devices) != 0 {
		t.Errorf("len(Devices()) = %d, want 0", len(devices))
	}

	// Add a mock device directly
	h.mutex.Lock()
	h.devices[0] = &Device{address: 1}
	h.deviceCount = 1
	h.mutex.Unlock()

	devices = h.Devices()
	if len(devices) != 1 {
		t.Errorf("len(Devices()) = %d, want 1", len(devices))
	}
}

func TestHost_GetDevice(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	// Add a mock device
	dev := &Device{address: 1}
	h.mutex.Lock()
	h.devices[0] = dev
	h.deviceCount = 1
	h.mutex.Unlock()

	// Test valid address
	if got := h.GetDevice(1); got != dev {
		t.Errorf("GetDevice(1) returned wrong device")
	}

	// Test invalid addresses
	if got := h.GetDevice(0); got != nil {
		t.Error("GetDevice(0) should return nil")
	}
	if got := h.GetDevice(255); got != nil {
		t.Error("GetDevice(255) should return nil")
	}
}

func TestHost_AllocateAddress(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	// First allocation
	addr1 := h.allocateAddress()
	if addr1 == 0 {
		t.Error("allocateAddress returned 0")
	}

	// Mark address as used
	h.mutex.Lock()
	h.devices[addr1-1] = &Device{address: addr1}
	h.mutex.Unlock()

	// Second allocation should return different address
	addr2 := h.allocateAddress()
	if addr2 == 0 {
		t.Error("second allocateAddress returned 0")
	}
	if addr2 == addr1 {
		t.Errorf("second allocation returned same address: %d", addr2)
	}
}

func TestHost_SetCallbacks(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	h.SetOnDeviceConnect(func(d *Device) {
		// callback invoked
	})

	h.SetOnDeviceDisconnect(func(d *Device) {
		// callback invoked
	})

	// Verify callbacks are set
	h.mutex.RLock()
	if h.onDeviceConnect == nil {
		t.Error("onDeviceConnect not set")
	}
	if h.onDeviceDisconnect == nil {
		t.Error("onDeviceDisconnect not set")
	}
	h.mutex.RUnlock()
}

// =============================================================================
// Device Tests
// =============================================================================

func TestDevice_Getters(t *testing.T) {
	dev := &Device{
		address: 5,
		port:    2,
		speed:   hal.SpeedHigh,
		descriptor: DeviceDescriptor{
			VendorID:          0x1234,
			ProductID:         0x5678,
			DeviceClass:       0x02,
			DeviceSubClass:    0x00,
			DeviceProtocol:    0x00,
			ManufacturerIndex: 1,
			ProductIndex:      2,
			SerialNumberIndex: 3,
		},
	}
	dev.strings[1] = "Test Manufacturer"
	dev.strings[2] = "Test Product"
	dev.strings[3] = "12345"

	if got := dev.Address(); got != 5 {
		t.Errorf("Address() = %d, want 5", got)
	}
	if got := dev.Port(); got != 2 {
		t.Errorf("Port() = %d, want 2", got)
	}
	if got := dev.Speed(); got != hal.SpeedHigh {
		t.Errorf("Speed() = %v, want SpeedHigh", got)
	}
	if got := dev.VendorID(); got != 0x1234 {
		t.Errorf("VendorID() = 0x%04X, want 0x1234", got)
	}
	if got := dev.ProductID(); got != 0x5678 {
		t.Errorf("ProductID() = 0x%04X, want 0x5678", got)
	}
	if got := dev.DeviceClass(); got != 0x02 {
		t.Errorf("DeviceClass() = 0x%02X, want 0x02", got)
	}
	if got := dev.Manufacturer(); got != "Test Manufacturer" {
		t.Errorf("Manufacturer() = %q, want %q", got, "Test Manufacturer")
	}
	if got := dev.Product(); got != "Test Product" {
		t.Errorf("Product() = %q, want %q", got, "Test Product")
	}
	if got := dev.SerialNumber(); got != "12345" {
		t.Errorf("SerialNumber() = %q, want %q", got, "12345")
	}
}

func TestDevice_GetString(t *testing.T) {
	dev := &Device{}
	dev.strings[1] = "Test String"

	// Valid index
	if got := dev.GetString(1); got != "Test String" {
		t.Errorf("GetString(1) = %q, want %q", got, "Test String")
	}

	// Index 0 should return empty
	if got := dev.GetString(0); got != "" {
		t.Errorf("GetString(0) = %q, want empty", got)
	}

	// Out of range should return empty
	if got := dev.GetString(255); got != "" {
		t.Errorf("GetString(255) = %q, want empty", got)
	}
}

func TestDevice_GetInterface(t *testing.T) {
	dev := &Device{
		interfaces: []InterfaceDescriptor{
			{InterfaceNumber: 0, InterfaceClass: 0x03},
			{InterfaceNumber: 1, InterfaceClass: 0x08},
		},
	}

	// Valid interface
	iface := dev.GetInterface(0)
	if iface == nil {
		t.Fatal("GetInterface(0) returned nil")
	}
	if iface.InterfaceClass != 0x03 {
		t.Errorf("InterfaceClass = 0x%02X, want 0x03", iface.InterfaceClass)
	}

	// Invalid interface
	if got := dev.GetInterface(5); got != nil {
		t.Error("GetInterface(5) should return nil")
	}
}

func TestDevice_GetEndpoint(t *testing.T) {
	dev := &Device{
		endpoints: []EndpointDescriptor{
			{EndpointAddress: 0x81, Attributes: EndpointTypeBulk},
			{EndpointAddress: 0x02, Attributes: EndpointTypeBulk},
		},
	}

	// Valid endpoint
	ep := dev.GetEndpoint(0x81)
	if ep == nil {
		t.Fatal("GetEndpoint(0x81) returned nil")
	}
	if ep.EndpointAddress != 0x81 {
		t.Errorf("EndpointAddress = 0x%02X, want 0x81", ep.EndpointAddress)
	}

	// Invalid endpoint
	if got := dev.GetEndpoint(0x83); got != nil {
		t.Error("GetEndpoint(0x83) should return nil")
	}
}

func TestDevice_State(t *testing.T) {
	dev := &Device{state: DeviceStateConfigured}

	if got := dev.State(); got != DeviceStateConfigured {
		t.Errorf("State() = %v, want DeviceStateConfigured", got)
	}
}

func TestDevice_Close(t *testing.T) {
	dev := &Device{state: DeviceStateConfigured}

	if err := dev.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if dev.State() != DeviceStateDetached {
		t.Errorf("State() = %v after Close, want DeviceStateDetached", dev.State())
	}
}

func TestDevice_ParseDeviceDescriptor(t *testing.T) {
	dev := &Device{}

	data := []byte{
		18, 0x01, // Length, Type
		0x00, 0x02, // USB 2.0
		0x00, 0x00, 0x00, // Class, SubClass, Protocol
		64,         // MaxPacketSize0
		0x34, 0x12, // VendorID
		0x78, 0x56, // ProductID
		0x01, 0x00, // DeviceVersion
		1, 2, 3, // String indices
		1, // NumConfigurations
	}

	if !dev.parseDeviceDescriptor(data) {
		t.Fatal("parseDeviceDescriptor returned false")
	}

	if dev.descriptor.VendorID != 0x1234 {
		t.Errorf("VendorID = 0x%04X, want 0x1234", dev.descriptor.VendorID)
	}
	if dev.descriptor.ProductID != 0x5678 {
		t.Errorf("ProductID = 0x%04X, want 0x5678", dev.descriptor.ProductID)
	}
}

func TestDevice_ParseConfigurationTree(t *testing.T) {
	dev := &Device{}

	// Configuration descriptor with interface and endpoint
	data := []byte{
		// Configuration descriptor
		9, 0x02, // Length, Type
		25, 0x00, // TotalLength = 25
		1,    // NumInterfaces
		1,    // ConfigurationValue
		0,    // ConfigurationIndex
		0x80, // Attributes
		50,   // MaxPower

		// Interface descriptor
		9, 0x04, // Length, Type
		0,    // InterfaceNumber
		0,    // AlternateSetting
		1,    // NumEndpoints
		0x03, // InterfaceClass (HID)
		0x00, // InterfaceSubClass
		0x00, // InterfaceProtocol
		0,    // InterfaceIndex

		// Endpoint descriptor
		7, 0x05, // Length, Type
		0x81,       // EndpointAddress (IN)
		0x03,       // Attributes (Interrupt)
		0x08, 0x00, // MaxPacketSize
		10, // Interval
	}

	dev.parseConfigurationTree(data)

	if dev.config.NumInterfaces != 1 {
		t.Errorf("NumInterfaces = %d, want 1", dev.config.NumInterfaces)
	}
	if len(dev.interfaces) != 1 {
		t.Fatalf("len(interfaces) = %d, want 1", len(dev.interfaces))
	}
	if dev.interfaces[0].InterfaceClass != 0x03 {
		t.Errorf("InterfaceClass = 0x%02X, want 0x03", dev.interfaces[0].InterfaceClass)
	}
	if len(dev.endpoints) != 1 {
		t.Fatalf("len(endpoints) = %d, want 1", len(dev.endpoints))
	}
	if dev.endpoints[0].EndpointAddress != 0x81 {
		t.Errorf("EndpointAddress = 0x%02X, want 0x81", dev.endpoints[0].EndpointAddress)
	}
}

// =============================================================================
// Transfer Tests
// =============================================================================

func TestTransfer_IsComplete(t *testing.T) {
	tr := &Transfer{}

	if tr.IsComplete() {
		t.Error("new transfer should not be complete")
	}

	tr.completed = 1
	if !tr.IsComplete() {
		t.Error("transfer with completed=1 should be complete")
	}
}

func TestTransfer_Result(t *testing.T) {
	tr := &Transfer{
		result: 64,
		err:    nil,
	}

	n, err := tr.Result()
	if n != 64 {
		t.Errorf("Result() n = %d, want 64", n)
	}
	if err != nil {
		t.Errorf("Result() err = %v, want nil", err)
	}
}

func TestTransferManager_StartStop(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	tm := NewTransferManager(h, 2)
	if tm == nil {
		t.Fatal("NewTransferManager returned nil")
	}

	ctx := context.Background()

	if err := tm.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !tm.running {
		t.Error("running = false after Start")
	}

	if err := tm.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if tm.running {
		t.Error("running = true after Stop")
	}
}

func TestTransferManager_PendingCount(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)
	tm := NewTransferManager(h, 1)

	if got := tm.PendingCount(); got != 0 {
		t.Errorf("PendingCount() = %d, want 0", got)
	}
}

// =============================================================================
// Pipe Tests
// =============================================================================

func TestPipe_New(t *testing.T) {
	dev := &Device{address: 1}
	pipe := NewPipe(dev, 0x81, 0x02, 64)

	if pipe == nil {
		t.Fatal("NewPipe returned nil")
	}
	if pipe.device != dev {
		t.Error("device not set correctly")
	}
	if pipe.epIn != 0x81 {
		t.Errorf("epIn = 0x%02X, want 0x81", pipe.epIn)
	}
	if pipe.epOut != 0x02 {
		t.Errorf("epOut = 0x%02X, want 0x02", pipe.epOut)
	}
	if pipe.maxSize != 64 {
		t.Errorf("maxSize = %d, want 64", pipe.maxSize)
	}
}

func TestPipe_Device(t *testing.T) {
	dev := &Device{address: 1}
	pipe := NewPipe(dev, 0x81, 0x02, 64)

	if got := pipe.Device(); got != dev {
		t.Error("Device() returned wrong device")
	}
}

func TestPipe_Close(t *testing.T) {
	dev := &Device{address: 1}
	pipe := NewPipe(dev, 0x81, 0x02, 64)

	if err := pipe.Close(); err != nil {
		t.Errorf("Close() = %v, want nil", err)
	}
}

// =============================================================================
// WaitDevice Tests
// =============================================================================

func TestHost_WaitDevice_Timeout(t *testing.T) {
	mock := newMockHAL()
	h := New(mock)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	h.ctx, h.cancel = context.WithCancel(context.Background())
	defer h.cancel()

	_, err := h.WaitDevice(ctx)
	if err == nil {
		t.Error("WaitDevice should return error on timeout")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkHost_GetDevice(b *testing.B) {
	mock := newMockHAL()
	h := New(mock)

	// Add some devices
	for i := 0; i < 10; i++ {
		h.devices[i] = &Device{address: uint8(i + 1)}
	}
	h.deviceCount = 10

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.GetDevice(uint8((i % 10) + 1))
	}
}

func BenchmarkHost_Devices(b *testing.B) {
	mock := newMockHAL()
	h := New(mock)

	// Add some devices
	for i := 0; i < 5; i++ {
		h.devices[i] = &Device{address: uint8(i + 1)}
	}
	h.deviceCount = 5

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Devices()
	}
}

func BenchmarkDevice_GetInterface(b *testing.B) {
	dev := &Device{
		interfaces: make([]InterfaceDescriptor, 4),
	}
	for i := range dev.interfaces {
		dev.interfaces[i].InterfaceNumber = uint8(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.GetInterface(uint8(i % 4))
	}
}

func BenchmarkDevice_GetEndpoint(b *testing.B) {
	dev := &Device{
		endpoints: []EndpointDescriptor{
			{EndpointAddress: 0x81},
			{EndpointAddress: 0x82},
			{EndpointAddress: 0x01},
			{EndpointAddress: 0x02},
		},
	}

	addrs := []uint8{0x81, 0x82, 0x01, 0x02}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.GetEndpoint(addrs[i%4])
	}
}

func BenchmarkTransfer_IsComplete(b *testing.B) {
	tr := &Transfer{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tr.IsComplete()
	}
}
