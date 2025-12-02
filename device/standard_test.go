package device

import (
	"fmt"
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

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestHandleSetup_AllRecipients tests handling all recipient types
func TestHandleSetup_AllRecipients(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	tests := []struct {
		name      string
		recipient uint8
		index     uint16
		wantErr   error
	}{
		{"Device", RequestRecipientDevice, 0, nil},
		{"Interface", RequestRecipientInterface, 0, nil},
		{"Endpoint", RequestRecipientEndpoint, 0x81, nil},
		{"Other (invalid)", RequestRecipientOther, 0, pkg.ErrInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var setup SetupPacket
			GetStatusSetup(&setup, tt.recipient, tt.index)
			_, err := handler.HandleSetup(&setup, nil)
			if err != tt.wantErr {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestHandleGetDescriptor_AllTypes tests getting all descriptor types
func TestHandleGetDescriptor_AllTypes(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	tests := []struct {
		name    string
		descTyp uint8
		index   uint8
		wantErr bool
	}{
		{"Device", DescriptorTypeDevice, 0, false},
		{"Configuration", DescriptorTypeConfiguration, 0, false},
		{"String_Language", DescriptorTypeString, 0, false},
		{"String_Manufacturer", DescriptorTypeString, 1, false},
		{"String_Product", DescriptorTypeString, 2, false},
		{"String_Invalid", DescriptorTypeString, 99, true},
		{"Configuration_Invalid", DescriptorTypeConfiguration, 99, true},
		{"Unknown_Type", 0xFF, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var setup SetupPacket
			GetDescriptorSetup(&setup, tt.descTyp, tt.index, 255)
			_, err := handler.HandleSetup(&setup, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestHandleGetDescriptor_TruncatedResponses tests length limiting
func TestHandleGetDescriptor_TruncatedResponses(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	sizes := []uint16{1, 4, 8, 18, 64, 255, 512}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("maxLen=%d", size), func(t *testing.T) {
			var setup SetupPacket
			GetDescriptorSetup(&setup, DescriptorTypeDevice, 0, size)
			data, err := handler.HandleSetup(&setup, nil)
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			// Device descriptor is 18 bytes
			expected := int(size)
			if expected > 18 {
				expected = 18
			}
			if len(data) != expected {
				t.Errorf("len = %d, want %d", len(data), expected)
			}
		})
	}
}

// TestHandleSetAddress_Boundary tests SET_ADDRESS boundary values
func TestHandleSetAddress_Boundary(t *testing.T) {
	tests := []struct {
		name    string
		address uint16
		wantErr bool
	}{
		{"Address_0", 0, false},
		{"Address_1", 1, false},
		{"Address_127", 127, false},
		// Address is masked to 7 bits per spec
		{"Address_128 (masked)", 128, false},
		{"Address_255 (masked)", 255, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
			dev.AddConfiguration(NewConfiguration(1))
			dev.Reset()
			handler := NewStandardRequestHandler(dev)

			var setup SetupPacket
			setup.RequestType = RequestDirectionHostToDevice | RequestTypeStandard | RequestRecipientDevice
			setup.Request = RequestSetAddress
			setup.Value = tt.address
			setup.Index = 0
			setup.Length = 0

			_, err := handler.HandleSetup(&setup, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestHandleSetConfiguration_Values tests SET_CONFIGURATION with various values
func TestHandleSetConfiguration_Values(t *testing.T) {
	tests := []struct {
		name    string
		config  uint16
		wantErr bool
	}{
		{"Config_0 (unconfigure)", 0, false},
		{"Config_1", 1, false},
		{"Config_Invalid", 99, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
			dev.AddConfiguration(NewConfiguration(1))
			dev.Reset()
			dev.SetAddress(5)
			handler := NewStandardRequestHandler(dev)

			var setup SetupPacket
			GetSetConfigurationSetup(&setup, uint8(tt.config))
			_, err := handler.HandleSetup(&setup, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestHandleFeature_InvalidValues tests feature requests with invalid values
func TestHandleFeature_InvalidValues(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Invalid device feature
	setup := &SetupPacket{
		RequestType: RequestDirectionHostToDevice | RequestTypeStandard | RequestRecipientDevice,
		Request:     RequestSetFeature,
		Value:       0xFF, // Invalid feature
	}
	_, err := handler.HandleSetup(setup, nil)
	if err != pkg.ErrInvalidRequest {
		t.Errorf("SetFeature invalid device feature: error = %v, want %v", err, pkg.ErrInvalidRequest)
	}

	// Invalid endpoint feature
	setup = &SetupPacket{
		RequestType: RequestDirectionHostToDevice | RequestTypeStandard | RequestRecipientEndpoint,
		Request:     RequestSetFeature,
		Value:       0xFF, // Invalid feature
		Index:       0x81,
	}
	_, err = handler.HandleSetup(setup, nil)
	if err != pkg.ErrInvalidRequest {
		t.Errorf("SetFeature invalid endpoint feature: error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}

// TestHandleGetStatus_ShortLength tests GET_STATUS with insufficient length
func TestHandleGetStatus_ShortLength(t *testing.T) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Length < 2 for GET_STATUS should fail
	setup := &SetupPacket{
		RequestType: RequestDirectionDeviceToHost | RequestTypeStandard | RequestRecipientDevice,
		Request:     RequestGetStatus,
		Length:      1, // Too short
	}
	_, err := handler.HandleSetup(setup, nil)
	if err != pkg.ErrInvalidRequest {
		t.Errorf("error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewStandardRequestHandler(b *testing.B) {
	dev := setupTestDevice()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStandardRequestHandler(dev)
	}
}

func BenchmarkHandleGetStatus(b *testing.B) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	recipients := []struct {
		name  string
		recip uint8
		index uint16
	}{
		{"Device", RequestRecipientDevice, 0},
		{"Interface", RequestRecipientInterface, 0},
		{"Endpoint", RequestRecipientEndpoint, 0x81},
	}

	for _, r := range recipients {
		b.Run(r.name, func(b *testing.B) {
			var setup SetupPacket
			GetStatusSetup(&setup, r.recip, r.index)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = handler.HandleSetup(&setup, nil)
			}
		})
	}
}

func BenchmarkHandleGetDescriptor(b *testing.B) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	descriptors := []struct {
		name    string
		descTyp uint8
		index   uint8
		maxLen  uint16
	}{
		{"Device", DescriptorTypeDevice, 0, 18},
		{"Device_Truncated", DescriptorTypeDevice, 0, 8},
		{"Configuration", DescriptorTypeConfiguration, 0, 255},
		{"String_Language", DescriptorTypeString, 0, 255},
		{"String_Manufacturer", DescriptorTypeString, 1, 255},
	}

	for _, d := range descriptors {
		b.Run(d.name, func(b *testing.B) {
			var setup SetupPacket
			GetDescriptorSetup(&setup, d.descTyp, d.index, d.maxLen)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = handler.HandleSetup(&setup, nil)
			}
		})
	}
}

func BenchmarkHandleSetAddress(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		dev.AddConfiguration(NewConfiguration(1))
		dev.Reset()
		handler := NewStandardRequestHandler(dev)
		var setup SetupPacket
		setup.RequestType = RequestDirectionHostToDevice | RequestTypeStandard | RequestRecipientDevice
		setup.Request = RequestSetAddress
		setup.Value = uint16(i % 127)
		b.StartTimer()

		_, _ = handler.HandleSetup(&setup, nil)
	}
}

func BenchmarkHandleSetConfiguration(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		dev.AddConfiguration(NewConfiguration(1))
		dev.Reset()
		dev.SetAddress(5)
		handler := NewStandardRequestHandler(dev)
		var setup SetupPacket
		GetSetConfigurationSetup(&setup, 1)
		b.StartTimer()

		_, _ = handler.HandleSetup(&setup, nil)
	}
}

func BenchmarkHandleGetConfiguration(b *testing.B) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)
	var setup SetupPacket
	GetConfigurationSetup(&setup)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.HandleSetup(&setup, nil)
	}
}

func BenchmarkHandleSetFeature(b *testing.B) {
	b.Run("RemoteWakeup", func(b *testing.B) {
		dev := setupTestDevice()
		handler := NewStandardRequestHandler(dev)
		var setup SetupPacket
		GetSetFeatureSetup(&setup, RequestRecipientDevice, FeatureDeviceRemoteWakeup, 0)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = handler.HandleSetup(&setup, nil)
		}
	})

	b.Run("EndpointHalt", func(b *testing.B) {
		dev := setupTestDevice()
		handler := NewStandardRequestHandler(dev)
		var setup SetupPacket
		GetSetFeatureSetup(&setup, RequestRecipientEndpoint, FeatureEndpointHalt, 0x81)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = handler.HandleSetup(&setup, nil)
		}
	})
}

func BenchmarkHandleClearFeature(b *testing.B) {
	b.Run("RemoteWakeup", func(b *testing.B) {
		dev := setupTestDevice()
		handler := NewStandardRequestHandler(dev)
		var setup SetupPacket
		GetClearFeatureSetup(&setup, RequestRecipientDevice, FeatureDeviceRemoteWakeup, 0)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = handler.HandleSetup(&setup, nil)
		}
	})

	b.Run("EndpointHalt", func(b *testing.B) {
		dev := setupTestDevice()
		handler := NewStandardRequestHandler(dev)
		var setup SetupPacket
		GetClearFeatureSetup(&setup, RequestRecipientEndpoint, FeatureEndpointHalt, 0x81)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = handler.HandleSetup(&setup, nil)
		}
	})
}

func BenchmarkHandleGetInterface(b *testing.B) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)
	var setup SetupPacket
	GetInterfaceSetup(&setup, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.HandleSetup(&setup, nil)
	}
}

func BenchmarkHandleSetInterface(b *testing.B) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)
	var setup SetupPacket
	GetSetInterfaceSetup(&setup, 0, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.HandleSetup(&setup, nil)
	}
}

func BenchmarkHandleSynchFrame(b *testing.B) {
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)
	setup := &SetupPacket{
		RequestType: RequestDirectionDeviceToHost | RequestTypeStandard | RequestRecipientEndpoint,
		Request:     RequestSynchFrame,
		Value:       0,
		Index:       0x04,
		Length:      2,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.HandleSetup(setup, nil)
	}
}

func BenchmarkHandleSetup_FullDispatch(b *testing.B) {
	// Test full dispatch path through all code paths
	dev := setupTestDevice()
	handler := NewStandardRequestHandler(dev)

	// Various setup packets for different paths
	setups := make([]*SetupPacket, 0, 10)

	var s1 SetupPacket
	GetStatusSetup(&s1, RequestRecipientDevice, 0)
	setups = append(setups, &s1)

	var s2 SetupPacket
	GetDescriptorSetup(&s2, DescriptorTypeDevice, 0, 18)
	setups = append(setups, &s2)

	var s3 SetupPacket
	GetDescriptorSetup(&s3, DescriptorTypeConfiguration, 0, 255)
	setups = append(setups, &s3)

	var s4 SetupPacket
	GetConfigurationSetup(&s4)
	setups = append(setups, &s4)

	var s5 SetupPacket
	GetInterfaceSetup(&s5, 0)
	setups = append(setups, &s5)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.HandleSetup(setups[i%len(setups)], nil)
	}
}
