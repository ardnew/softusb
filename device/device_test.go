package device

import (
	"context"
	"testing"

	"github.com/ardnew/softusb/pkg"
)

func TestNewDevice(t *testing.T) {
	desc := &DeviceDescriptor{
		Length:            18,
		DescriptorType:    DescriptorTypeDevice,
		USBVersion:        0x0200,
		DeviceClass:       ClassPerInterface,
		MaxPacketSize0:    64,
		VendorID:          0x1234,
		ProductID:         0x5678,
		NumConfigurations: 1,
	}

	dev := NewDevice(desc)

	if dev.Descriptor != desc {
		t.Error("Descriptor not set")
	}
	if dev.State() != StateAttached {
		t.Errorf("State() = %v, want %v", dev.State(), StateAttached)
	}
	if dev.Speed() != SpeedFull {
		t.Errorf("Speed() = %v, want %v", dev.Speed(), SpeedFull)
	}
	if dev.ep0 == nil {
		t.Error("EP0 not initialized")
	}
	if dev.ep0.MaxPacketSize != 64 {
		t.Errorf("EP0 MaxPacketSize = %d, want 64", dev.ep0.MaxPacketSize)
	}
}

func TestDeviceConfiguration(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)

	err := dev.AddConfiguration(config)
	if err != nil {
		t.Fatalf("AddConfiguration() error = %v", err)
	}

	// Adding same config again should fail
	err = dev.AddConfiguration(config)
	if err == nil {
		t.Error("AddConfiguration() should fail for duplicate")
	}

	if got := dev.GetConfiguration(1); got != config {
		t.Error("GetConfiguration(1) returned wrong config")
	}
	if got := dev.GetConfiguration(2); got != nil {
		t.Error("GetConfiguration(2) should return nil")
	}
}

func TestDeviceStrings(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})

	// Create language descriptor
	var langBuf [4]byte
	langLen := LanguageDescriptorTo(langBuf[:], LangIDUSEnglish)
	dev.SetLanguages(langBuf[:langLen])

	// Create string descriptors
	var mfrBuf [256]byte
	mfrLen := StringDescriptorTo(mfrBuf[:], "Test Manufacturer")
	dev.SetString(1, mfrBuf[:mfrLen])

	var prodBuf [256]byte
	prodLen := StringDescriptorTo(prodBuf[:], "Test Product")
	dev.SetString(2, prodBuf[:prodLen])

	lang := dev.GetString(0)
	if lang == nil {
		t.Fatal("GetString(0) returned nil")
	}
	if lang[0] != 4 { // 2 bytes header + 2 bytes lang ID
		t.Errorf("language descriptor length = %d, want 4", lang[0])
	}

	mfr := dev.GetString(1)
	if mfr == nil {
		t.Fatal("GetString(1) returned nil")
	}
	if mfr[1] != DescriptorTypeString {
		t.Errorf("string descriptor type = 0x%02X, want 0x%02X", mfr[1], DescriptorTypeString)
	}
}

func TestDeviceStateTransitions(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)

	// Initial state
	if dev.State() != StateAttached {
		t.Errorf("initial state = %v, want %v", dev.State(), StateAttached)
	}

	// Reset transitions to Default
	dev.Reset()
	if dev.State() != StateDefault {
		t.Errorf("after reset, state = %v, want %v", dev.State(), StateDefault)
	}

	// SetAddress transitions to Address
	err := dev.SetAddress(5)
	if err != nil {
		t.Fatalf("SetAddress() error = %v", err)
	}
	if dev.State() != StateAddress {
		t.Errorf("after set address, state = %v, want %v", dev.State(), StateAddress)
	}
	if dev.Address() != 5 {
		t.Errorf("Address() = %d, want 5", dev.Address())
	}

	// SetConfiguration transitions to Configured
	err = dev.SetConfiguration(1)
	if err != nil {
		t.Fatalf("SetConfiguration() error = %v", err)
	}
	if dev.State() != StateConfigured {
		t.Errorf("after configure, state = %v, want %v", dev.State(), StateConfigured)
	}
	if !dev.IsConfigured() {
		t.Error("IsConfigured() should return true")
	}

	// SetConfiguration(0) unconfigures
	err = dev.SetConfiguration(0)
	if err != nil {
		t.Fatalf("SetConfiguration(0) error = %v", err)
	}
	if dev.State() != StateAddress {
		t.Errorf("after unconfigure, state = %v, want %v", dev.State(), StateAddress)
	}
}

func TestDeviceSetAddressInvalidState(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	// Device is in Attached state, SetAddress should fail
	err := dev.SetAddress(5)
	if err != pkg.ErrInvalidState {
		t.Errorf("SetAddress() error = %v, want %v", err, pkg.ErrInvalidState)
	}
}

func TestDeviceSetConfigurationInvalidState(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	dev.Reset()
	// Device is in Default state, SetConfiguration should fail
	err := dev.SetConfiguration(1)
	if err != pkg.ErrInvalidState {
		t.Errorf("SetConfiguration() error = %v, want %v", err, pkg.ErrInvalidState)
	}
}

func TestDeviceSetConfigurationInvalid(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	dev.Reset()
	dev.SetAddress(5)

	// No configuration 2 exists
	err := dev.SetConfiguration(2)
	if err != pkg.ErrInvalidRequest {
		t.Errorf("SetConfiguration() error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}

func TestDeviceSuspendResume(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(5)
	dev.SetConfiguration(1)

	// Suspend
	dev.Suspend()
	if !dev.IsSuspended() {
		t.Error("IsSuspended() should return true after suspend")
	}
	if dev.State() != StateSuspended {
		t.Errorf("state = %v, want %v", dev.State(), StateSuspended)
	}

	// Resume should restore previous state
	dev.Resume()
	if dev.IsSuspended() {
		t.Error("IsSuspended() should return false after resume")
	}
	if dev.State() != StateConfigured {
		t.Errorf("after resume, state = %v, want %v", dev.State(), StateConfigured)
	}
}

func TestDeviceCallbacks(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)

	var stateChanges []string
	var suspendCalled, resumeCalled, resetCalled bool
	var addressSet uint8
	var configSet uint8

	dev.SetOnStateChange(func(old, new State) {
		stateChanges = append(stateChanges, old.String()+"->"+new.String())
	})
	dev.SetOnSuspend(func() { suspendCalled = true })
	dev.SetOnResume(func() { resumeCalled = true })
	dev.SetOnReset(func() { resetCalled = true })
	dev.SetOnSetAddress(func(addr uint8) { addressSet = addr })
	dev.SetOnSetConfiguration(func(cfg uint8) { configSet = cfg })

	dev.Reset()
	if !resetCalled {
		t.Error("reset callback not called")
	}

	dev.SetAddress(5)
	if addressSet != 5 {
		t.Errorf("address callback got %d, want 5", addressSet)
	}

	dev.SetConfiguration(1)
	if configSet != 1 {
		t.Errorf("configuration callback got %d, want 1", configSet)
	}

	dev.Suspend()
	if !suspendCalled {
		t.Error("suspend callback not called")
	}

	dev.Resume()
	if !resumeCalled {
		t.Error("resume callback not called")
	}

	if len(stateChanges) == 0 {
		t.Error("state change callback not called")
	}
}

func TestDeviceRemoteWakeup(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})

	if dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be disabled by default")
	}

	dev.EnableRemoteWakeup(true)
	if !dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be enabled")
	}

	dev.EnableRemoteWakeup(false)
	if dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be disabled")
	}
}

func TestDeviceGetInterface(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	config.AddInterface(iface)
	dev.AddConfiguration(config)

	// Not configured yet
	if got := dev.GetInterface(0); got != nil {
		t.Error("GetInterface should return nil when not configured")
	}

	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	if got := dev.GetInterface(0); got != iface {
		t.Error("GetInterface(0) returned wrong interface")
	}
	if got := dev.GetInterface(1); got != nil {
		t.Error("GetInterface(1) should return nil")
	}
}

func TestDeviceGetEndpoint(t *testing.T) {
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

	// EP0
	if got := dev.GetEndpoint(0); got != dev.ControlEndpoint() {
		t.Error("GetEndpoint(0) should return EP0")
	}
	if got := dev.GetEndpoint(0x80); got != dev.ControlEndpoint() {
		t.Error("GetEndpoint(0x80) should return EP0")
	}

	// Non-control endpoint
	if got := dev.GetEndpoint(0x81); got != ep {
		t.Error("GetEndpoint(0x81) returned wrong endpoint")
	}
	if got := dev.GetEndpoint(0x82); got != nil {
		t.Error("GetEndpoint(0x82) should return nil")
	}
}

func TestDeviceSetEndpointStall(t *testing.T) {
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

	err := dev.SetEndpointStall(0x81, true)
	if err != nil {
		t.Fatalf("SetEndpointStall() error = %v", err)
	}
	if !ep.IsStalled() {
		t.Error("endpoint should be stalled")
	}

	err = dev.SetEndpointStall(0x81, false)
	if err != nil {
		t.Fatalf("SetEndpointStall() error = %v", err)
	}
	if ep.IsStalled() {
		t.Error("endpoint should not be stalled")
	}

	// Invalid endpoint
	err = dev.SetEndpointStall(0x82, true)
	if err != pkg.ErrInvalidEndpoint {
		t.Errorf("SetEndpointStall() error = %v, want %v", err, pkg.ErrInvalidEndpoint)
	}
}

func TestDeviceGetStatus(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	config.SetSelfPowered(true)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	status := dev.GetStatus()
	if status&DeviceStatusSelfPowered == 0 {
		t.Error("status should indicate self-powered")
	}
	if status&DeviceStatusRemoteWakeup != 0 {
		t.Error("status should not indicate remote wakeup")
	}

	dev.EnableRemoteWakeup(true)
	status = dev.GetStatus()
	if status&DeviceStatusRemoteWakeup == 0 {
		t.Error("status should indicate remote wakeup")
	}
}

func TestDeviceClose(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	driver := &mockClassDriver{}
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	iface.SetClassDriver(driver)
	config := NewConfiguration(1)
	config.AddInterface(iface)
	dev.AddConfiguration(config)

	err := dev.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !driver.closeCalled {
		t.Error("class driver Close() should be called")
	}
}

func TestDeviceBuilder(t *testing.T) {
	dev, err := NewDeviceBuilder().
		WithVendorProduct(0x1234, 0x5678).
		WithStrings("Test Mfr", "Test Prod", "12345").
		AddConfiguration(1).
		AddInterface(ClassCDC, 0x02, 0x01).
		AddEndpoint(0x81, EndpointTypeBulk, 512).
		AddEndpoint(0x02, EndpointTypeBulk, 512).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if dev.Descriptor.VendorID != 0x1234 {
		t.Errorf("VendorID = 0x%04X, want 0x1234", dev.Descriptor.VendorID)
	}
	if dev.Descriptor.ProductID != 0x5678 {
		t.Errorf("ProductID = 0x%04X, want 0x5678", dev.Descriptor.ProductID)
	}

	config := dev.GetConfiguration(1)
	if config == nil {
		t.Fatal("configuration 1 not found")
	}

	iface := config.GetInterface(0)
	if iface == nil {
		t.Fatal("interface 0 not found")
	}

	if iface.Class != ClassCDC {
		t.Errorf("interface class = 0x%02X, want 0x%02X", iface.Class, ClassCDC)
	}

	if iface.NumEndpoints() != 2 {
		t.Errorf("interface has %d endpoints, want 2", iface.NumEndpoints())
	}
}

func TestDeviceBuilderNoDevice(t *testing.T) {
	_, err := NewDeviceBuilder().
		AddConfiguration(1).
		Build(context.Background())

	if err == nil {
		t.Error("Build() should fail without device initialization")
	}
}
