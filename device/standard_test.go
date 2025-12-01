package device

import (
	"testing"

	"github.com/ardnew/softusb/pkg"
)

func setupTestDevice() *Device {
	dev := NewDevice(&DeviceDescriptor{
		Length:            18,
		DescriptorType:    DescriptorTypeDevice,
		USBVersion:        0x0200,
		DeviceClass:       ClassPerInterface,
		MaxPacketSize0:    64,
		VendorID:          0x1234,
		ProductID:         0x5678,
		NumConfigurations: 1,
	})

	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{
		InterfaceNumber: 0,
		InterfaceClass:  ClassCDC,
	})
	iface.AddEndpoint(&Endpoint{
		Address:       0x81,
		Attributes:    EndpointTypeBulk,
		MaxPacketSize: 512,
	})
	iface.AddEndpoint(&Endpoint{
		Address:       0x02,
		Attributes:    EndpointTypeBulk,
		MaxPacketSize: 512,
	})
	iface.AddEndpoint(&Endpoint{
		Address:       0x04,
		Attributes:    EndpointTypeIsochronous,
		MaxPacketSize: 192,
	})
	config.AddInterface(iface)
	dev.AddConfiguration(config)

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

	dev.Reset()
	dev.SetAddress(5)
	dev.SetConfiguration(1)

	return dev
}

func TestHandleGetDeviceStatus(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetStatusSetup(&setup, RequestRecipientDevice, 0)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 2 {
		t.Errorf("response length = %d, want 2", len(data))
	}
}

func TestHandleGetDeviceStatusWithRemoteWakeup(t *testing.T) {
	dev := setupTestDevice()
	dev.EnableRemoteWakeup(true)
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetStatusSetup(&setup, RequestRecipientDevice, 0)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if data[0]&0x02 == 0 {
		t.Error("remote wakeup bit should be set")
	}
}

func TestHandleClearDeviceFeature(t *testing.T) {
	dev := setupTestDevice()
	dev.EnableRemoteWakeup(true)
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetClearFeatureSetup(&setup, RequestRecipientDevice, FeatureDeviceRemoteWakeup, 0)
	_, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be disabled")
	}
}

func TestHandleSetDeviceFeature(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetSetFeatureSetup(&setup, RequestRecipientDevice, FeatureDeviceRemoteWakeup, 0)
	_, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if !dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be enabled")
	}
}

func TestHandleGetDescriptorDevice(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, 18)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 18 {
		t.Errorf("response length = %d, want 18", len(data))
	}
	if data[1] != DescriptorTypeDevice {
		t.Errorf("descriptor type = 0x%02X, want 0x%02X", data[1], DescriptorTypeDevice)
	}
}

func TestHandleGetDescriptorDeviceTruncated(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Request only 8 bytes
	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, 8)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 8 {
		t.Errorf("response length = %d, want 8", len(data))
	}
}

func TestHandleGetDescriptorConfiguration(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeConfiguration, 0, 255)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if data[1] != DescriptorTypeConfiguration {
		t.Errorf("descriptor type = 0x%02X, want 0x%02X", data[1], DescriptorTypeConfiguration)
	}
}

func TestHandleGetDescriptorString(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Get language descriptor (index 0)
	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeString, 0, 255)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if data[1] != DescriptorTypeString {
		t.Errorf("descriptor type = 0x%02X, want 0x%02X", data[1], DescriptorTypeString)
	}

	// Get manufacturer string (index 1)
	GetDescriptorSetup(&setup, DescriptorTypeString, 1, 255)
	data, err = handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if data[1] != DescriptorTypeString {
		t.Errorf("descriptor type = 0x%02X, want 0x%02X", data[1], DescriptorTypeString)
	}
}

func TestHandleGetDescriptorInvalid(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Invalid string index
	var setup SetupPacket
	GetDescriptorSetup(&setup, DescriptorTypeString, 99, 255)
	_, err := handler.HandleSetup(&setup, nil)

	if err != pkg.ErrInvalidRequest {
		t.Errorf("error = %v, want %v", err, pkg.ErrInvalidRequest)
	}

	// Invalid configuration index
	GetDescriptorSetup(&setup, DescriptorTypeConfiguration, 99, 255)
	_, err = handler.HandleSetup(&setup, nil)

	if err != pkg.ErrInvalidRequest {
		t.Errorf("error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}

func TestHandleGetConfiguration(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetConfigurationSetup(&setup)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 1 {
		t.Errorf("response length = %d, want 1", len(data))
	}
	if data[0] != 1 {
		t.Errorf("configuration value = %d, want 1", data[0])
	}
}

func TestHandleSetConfiguration(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(5)
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetSetConfigurationSetup(&setup, 1)
	_, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if !dev.IsConfigured() {
		t.Error("device should be configured")
	}
}

func TestHandleGetInterfaceStatus(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetStatusSetup(&setup, RequestRecipientInterface, 0)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 2 {
		t.Errorf("response length = %d, want 2", len(data))
	}
	// Interface status is always 0
	if data[0] != 0 || data[1] != 0 {
		t.Error("interface status should be 0")
	}
}

func TestHandleGetInterface(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetInterfaceSetup(&setup, 0)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 1 {
		t.Errorf("response length = %d, want 1", len(data))
	}
	// Default alternate setting is 0
	if data[0] != 0 {
		t.Errorf("alternate setting = %d, want 0", data[0])
	}
}

func TestHandleSetInterface(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetSetInterfaceSetup(&setup, 0, 1)
	_, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}

	iface := dev.GetInterface(0)
	if iface.AlternateSetting != 1 {
		t.Errorf("alternate setting = %d, want 1", iface.AlternateSetting)
	}
}

func TestHandleGetEndpointStatus(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetStatusSetup(&setup, RequestRecipientEndpoint, 0x81)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 2 {
		t.Errorf("response length = %d, want 2", len(data))
	}
	// Endpoint should not be halted
	if data[0] != 0 {
		t.Error("endpoint should not be halted")
	}
}

func TestHandleGetEndpointStatusStalled(t *testing.T) {
	dev := setupTestDevice()
	dev.SetEndpointStall(0x81, true)
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetStatusSetup(&setup, RequestRecipientEndpoint, 0x81)
	data, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if data[0]&0x01 == 0 {
		t.Error("endpoint halt bit should be set")
	}
}

func TestHandleClearEndpointFeature(t *testing.T) {
	dev := setupTestDevice()
	dev.SetEndpointStall(0x81, true)
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetClearFeatureSetup(&setup, RequestRecipientEndpoint, FeatureEndpointHalt, 0x81)
	_, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}

	ep := dev.GetEndpoint(0x81)
	if ep.IsStalled() {
		t.Error("endpoint should not be stalled")
	}
}

func TestHandleSetEndpointFeature(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	var setup SetupPacket
	GetSetFeatureSetup(&setup, RequestRecipientEndpoint, FeatureEndpointHalt, 0x81)
	_, err := handler.HandleSetup(&setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}

	ep := dev.GetEndpoint(0x81)
	if !ep.IsStalled() {
		t.Error("endpoint should be stalled")
	}
}

func TestHandleSynchFrame(t *testing.T) {
	dev := setupTestDevice()
	ep := dev.GetEndpoint(0x04)
	ep.SetFrameNumber(1000)
	handler := NewStandardRequestHandler(dev)

	setup := &SetupPacket{
		RequestType: RequestDirectionDeviceToHost | RequestTypeStandard | RequestRecipientEndpoint,
		Request:     RequestSynchFrame,
		Value:       0,
		Index:       0x04,
		Length:      2,
	}
	data, err := handler.HandleSetup(setup, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if len(data) != 2 {
		t.Errorf("response length = %d, want 2", len(data))
	}
	// Frame number should be 1000
	frame := uint16(data[0]) | uint16(data[1])<<8
	if frame != 1000 {
		t.Errorf("frame number = %d, want 1000", frame)
	}
}

func TestHandleSynchFrameNonIsochronous(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Try on bulk endpoint
	setup := &SetupPacket{
		RequestType: RequestDirectionDeviceToHost | RequestTypeStandard | RequestRecipientEndpoint,
		Request:     RequestSynchFrame,
		Value:       0,
		Index:       0x81,
		Length:      2,
	}
	_, err := handler.HandleSetup(setup, nil)

	if err != pkg.ErrInvalidRequest {
		t.Errorf("error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}

func TestHandleNonStandardRequest(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Class request should fail
	setup := &SetupPacket{
		RequestType: RequestDirectionDeviceToHost | RequestTypeClass | RequestRecipientDevice,
		Request:     0x01,
		Length:      1,
	}
	_, err := handler.HandleSetup(setup, nil)

	if err != pkg.ErrInvalidRequest {
		t.Errorf("error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}

func TestHandleInvalidEndpoint(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Non-existent endpoint
	var setup SetupPacket
	GetStatusSetup(&setup, RequestRecipientEndpoint, 0x85)
	_, err := handler.HandleSetup(&setup, nil)

	if err != pkg.ErrInvalidEndpoint {
		t.Errorf("error = %v, want %v", err, pkg.ErrInvalidEndpoint)
	}
}

func TestHandleInvalidInterface(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Non-existent interface
	var setup SetupPacket
	GetStatusSetup(&setup, RequestRecipientInterface, 99)
	_, err := handler.HandleSetup(&setup, nil)

	if err != pkg.ErrInvalidRequest {
		t.Errorf("error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}
