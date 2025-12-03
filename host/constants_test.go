package host

import (
	"testing"
)

// =============================================================================
// Speed Tests
// =============================================================================

func TestSpeed_String(t *testing.T) {
	tests := []struct {
		speed    Speed
		expected string
	}{
		{SpeedLow, "Low Speed (1.5 Mbps)"},
		{SpeedFull, "Full Speed (12 Mbps)"},
		{SpeedHigh, "High Speed (480 Mbps)"},
		{SpeedSuper, "Super Speed (5 Gbps)"},
		{Speed(255), "Unknown Speed (255)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.speed.String(); got != tt.expected {
				t.Errorf("Speed.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSpeed_MaxPacketSize0(t *testing.T) {
	tests := []struct {
		speed    Speed
		expected uint16
	}{
		{SpeedLow, 8},
		{SpeedFull, 64},
		{SpeedHigh, 64},
		{SpeedSuper, 512},
		{Speed(255), 8}, // Unknown defaults to 8
	}

	for _, tt := range tests {
		t.Run(tt.speed.String(), func(t *testing.T) {
			if got := tt.speed.MaxPacketSize0(); got != tt.expected {
				t.Errorf("Speed.MaxPacketSize0() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// DeviceState Tests
// =============================================================================

func TestDeviceState_String(t *testing.T) {
	tests := []struct {
		state    DeviceState
		expected string
	}{
		{DeviceStateDetached, "Detached"},
		{DeviceStateAttached, "Attached"},
		{DeviceStateDefault, "Default"},
		{DeviceStateAddress, "Address"},
		{DeviceStateConfigured, "Configured"},
		{DeviceStateSuspended, "Suspended"},
		{DeviceState(255), "Unknown State (255)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("DeviceState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Descriptor Parsing Tests
// =============================================================================

func TestParseDeviceDescriptor(t *testing.T) {
	data := []byte{
		18, 0x01, // Length, Type
		0x00, 0x02, // USB Version 2.0 (little-endian)
		0x00, 0x00, 0x00, // Class, SubClass, Protocol
		64,         // MaxPacketSize0
		0x34, 0x12, // VendorID (little-endian)
		0x78, 0x56, // ProductID (little-endian)
		0x01, 0x00, // DeviceVersion
		1, 2, 3, // Manufacturer, Product, SerialNumber indices
		1, // NumConfigurations
	}

	var desc DeviceDescriptor
	if !ParseDeviceDescriptor(data, &desc) {
		t.Fatal("ParseDeviceDescriptor returned false")
	}

	if desc.Length != 18 {
		t.Errorf("Length = %d, want 18", desc.Length)
	}
	if desc.DescriptorType != 0x01 {
		t.Errorf("DescriptorType = 0x%02X, want 0x01", desc.DescriptorType)
	}
	if desc.USBVersion != 0x0200 {
		t.Errorf("USBVersion = 0x%04X, want 0x0200", desc.USBVersion)
	}
	if desc.MaxPacketSize0 != 64 {
		t.Errorf("MaxPacketSize0 = %d, want 64", desc.MaxPacketSize0)
	}
	if desc.VendorID != 0x1234 {
		t.Errorf("VendorID = 0x%04X, want 0x1234", desc.VendorID)
	}
	if desc.ProductID != 0x5678 {
		t.Errorf("ProductID = 0x%04X, want 0x5678", desc.ProductID)
	}
	if desc.ManufacturerIndex != 1 {
		t.Errorf("ManufacturerIndex = %d, want 1", desc.ManufacturerIndex)
	}
	if desc.ProductIndex != 2 {
		t.Errorf("ProductIndex = %d, want 2", desc.ProductIndex)
	}
	if desc.SerialNumberIndex != 3 {
		t.Errorf("SerialNumberIndex = %d, want 3", desc.SerialNumberIndex)
	}
	if desc.NumConfigurations != 1 {
		t.Errorf("NumConfigurations = %d, want 1", desc.NumConfigurations)
	}
}

func TestParseDeviceDescriptor_TooShort(t *testing.T) {
	data := make([]byte, DeviceDescriptorSize-1)
	var desc DeviceDescriptor
	if ParseDeviceDescriptor(data, &desc) {
		t.Error("ParseDeviceDescriptor should return false for short data")
	}
}

func TestParseConfigurationDescriptor(t *testing.T) {
	data := []byte{
		9, 0x02, // Length, Type
		0x20, 0x00, // TotalLength (little-endian)
		2,    // NumInterfaces
		1,    // ConfigurationValue
		4,    // ConfigurationIndex
		0xA0, // Attributes
		50,   // MaxPower
	}

	var desc ConfigurationDescriptor
	if !ParseConfigurationDescriptor(data, &desc) {
		t.Fatal("ParseConfigurationDescriptor returned false")
	}

	if desc.Length != 9 {
		t.Errorf("Length = %d, want 9", desc.Length)
	}
	if desc.TotalLength != 0x0020 {
		t.Errorf("TotalLength = %d, want 32", desc.TotalLength)
	}
	if desc.NumInterfaces != 2 {
		t.Errorf("NumInterfaces = %d, want 2", desc.NumInterfaces)
	}
	if desc.ConfigurationValue != 1 {
		t.Errorf("ConfigurationValue = %d, want 1", desc.ConfigurationValue)
	}
}

func TestParseConfigurationDescriptor_TooShort(t *testing.T) {
	data := make([]byte, ConfigurationDescriptorSize-1)
	var desc ConfigurationDescriptor
	if ParseConfigurationDescriptor(data, &desc) {
		t.Error("ParseConfigurationDescriptor should return false for short data")
	}
}

func TestParseInterfaceDescriptor(t *testing.T) {
	data := []byte{
		9, 0x04, // Length, Type
		0,    // InterfaceNumber
		0,    // AlternateSetting
		2,    // NumEndpoints
		0x02, // InterfaceClass (CDC)
		0x02, // InterfaceSubClass
		0x01, // InterfaceProtocol
		5,    // InterfaceIndex
	}

	var desc InterfaceDescriptor
	if !ParseInterfaceDescriptor(data, &desc) {
		t.Fatal("ParseInterfaceDescriptor returned false")
	}

	if desc.InterfaceNumber != 0 {
		t.Errorf("InterfaceNumber = %d, want 0", desc.InterfaceNumber)
	}
	if desc.NumEndpoints != 2 {
		t.Errorf("NumEndpoints = %d, want 2", desc.NumEndpoints)
	}
	if desc.InterfaceClass != 0x02 {
		t.Errorf("InterfaceClass = 0x%02X, want 0x02", desc.InterfaceClass)
	}
}

func TestParseInterfaceDescriptor_TooShort(t *testing.T) {
	data := make([]byte, InterfaceDescriptorSize-1)
	var desc InterfaceDescriptor
	if ParseInterfaceDescriptor(data, &desc) {
		t.Error("ParseInterfaceDescriptor should return false for short data")
	}
}

func TestParseEndpointDescriptor(t *testing.T) {
	data := []byte{
		7, 0x05, // Length, Type
		0x81,       // EndpointAddress (EP1 IN)
		0x02,       // Attributes (Bulk)
		0x00, 0x02, // MaxPacketSize (512)
		0, // Interval
	}

	var desc EndpointDescriptor
	if !ParseEndpointDescriptor(data, &desc) {
		t.Fatal("ParseEndpointDescriptor returned false")
	}

	if desc.EndpointAddress != 0x81 {
		t.Errorf("EndpointAddress = 0x%02X, want 0x81", desc.EndpointAddress)
	}
	if desc.Attributes != 0x02 {
		t.Errorf("Attributes = 0x%02X, want 0x02", desc.Attributes)
	}
	if desc.MaxPacketSize != 512 {
		t.Errorf("MaxPacketSize = %d, want 512", desc.MaxPacketSize)
	}
}

func TestParseEndpointDescriptor_TooShort(t *testing.T) {
	data := make([]byte, EndpointDescriptorSize-1)
	var desc EndpointDescriptor
	if ParseEndpointDescriptor(data, &desc) {
		t.Error("ParseEndpointDescriptor should return false for short data")
	}
}

func TestEndpointDescriptor_Methods(t *testing.T) {
	tests := []struct {
		name   string
		desc   EndpointDescriptor
		number uint8
		isIn   bool
		isOut  bool
		isBulk bool
		isIntr bool
		isIso  bool
		isCtrl bool
	}{
		{
			name:   "BulkIN",
			desc:   EndpointDescriptor{EndpointAddress: 0x81, Attributes: EndpointTypeBulk},
			number: 1, isIn: true, isBulk: true,
		},
		{
			name:   "BulkOUT",
			desc:   EndpointDescriptor{EndpointAddress: 0x02, Attributes: EndpointTypeBulk},
			number: 2, isOut: true, isBulk: true,
		},
		{
			name:   "InterruptIN",
			desc:   EndpointDescriptor{EndpointAddress: 0x83, Attributes: EndpointTypeInterrupt},
			number: 3, isIn: true, isIntr: true,
		},
		{
			name:   "IsochronousIN",
			desc:   EndpointDescriptor{EndpointAddress: 0x84, Attributes: EndpointTypeIsochronous},
			number: 4, isIn: true, isIso: true,
		},
		{
			name:   "ControlOUT",
			desc:   EndpointDescriptor{EndpointAddress: 0x00, Attributes: EndpointTypeControl},
			number: 0, isOut: true, isCtrl: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.desc.Number(); got != tt.number {
				t.Errorf("Number() = %d, want %d", got, tt.number)
			}
			if got := tt.desc.IsIn(); got != tt.isIn {
				t.Errorf("IsIn() = %v, want %v", got, tt.isIn)
			}
			if got := tt.desc.IsOut(); got != tt.isOut {
				t.Errorf("IsOut() = %v, want %v", got, tt.isOut)
			}
			if got := tt.desc.IsBulk(); got != tt.isBulk {
				t.Errorf("IsBulk() = %v, want %v", got, tt.isBulk)
			}
			if got := tt.desc.IsInterrupt(); got != tt.isIntr {
				t.Errorf("IsInterrupt() = %v, want %v", got, tt.isIntr)
			}
			if got := tt.desc.IsIsochronous(); got != tt.isIso {
				t.Errorf("IsIsochronous() = %v, want %v", got, tt.isIso)
			}
			if got := tt.desc.IsControl(); got != tt.isCtrl {
				t.Errorf("IsControl() = %v, want %v", got, tt.isCtrl)
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkSpeed_String(b *testing.B) {
	speeds := []Speed{SpeedLow, SpeedFull, SpeedHigh, SpeedSuper}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = speeds[i%len(speeds)].String()
	}
}

func BenchmarkSpeed_MaxPacketSize0(b *testing.B) {
	speeds := []Speed{SpeedLow, SpeedFull, SpeedHigh, SpeedSuper}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = speeds[i%len(speeds)].MaxPacketSize0()
	}
}

func BenchmarkDeviceState_String(b *testing.B) {
	states := []DeviceState{DeviceStateDetached, DeviceStateAttached, DeviceStateDefault, DeviceStateAddress, DeviceStateConfigured, DeviceStateSuspended}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = states[i%len(states)].String()
	}
}

func BenchmarkParseDeviceDescriptor(b *testing.B) {
	data := []byte{
		18, 0x01,
		0x00, 0x02,
		0x00, 0x00, 0x00,
		64,
		0x34, 0x12,
		0x78, 0x56,
		0x01, 0x00,
		1, 2, 3,
		1,
	}
	var desc DeviceDescriptor
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseDeviceDescriptor(data, &desc)
	}
}

func BenchmarkParseConfigurationDescriptor(b *testing.B) {
	data := []byte{9, 0x02, 0x20, 0x00, 2, 1, 4, 0xA0, 50}
	var desc ConfigurationDescriptor
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseConfigurationDescriptor(data, &desc)
	}
}

func BenchmarkParseInterfaceDescriptor(b *testing.B) {
	data := []byte{9, 0x04, 0, 0, 2, 0x02, 0x02, 0x01, 5}
	var desc InterfaceDescriptor
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseInterfaceDescriptor(data, &desc)
	}
}

func BenchmarkParseEndpointDescriptor(b *testing.B) {
	data := []byte{7, 0x05, 0x81, 0x02, 0x00, 0x02, 0}
	var desc EndpointDescriptor
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseEndpointDescriptor(data, &desc)
	}
}

func BenchmarkEndpointDescriptor_Number(b *testing.B) {
	desc := EndpointDescriptor{EndpointAddress: 0x81}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = desc.Number()
	}
}

func BenchmarkEndpointDescriptor_TransferType(b *testing.B) {
	desc := EndpointDescriptor{Attributes: EndpointTypeBulk}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = desc.TransferType()
	}
}

func BenchmarkEndpointDescriptor_IsIn(b *testing.B) {
	desc := EndpointDescriptor{EndpointAddress: 0x81}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = desc.IsIn()
	}
}

func BenchmarkEndpointDescriptor_IsBulk(b *testing.B) {
	desc := EndpointDescriptor{Attributes: EndpointTypeBulk}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = desc.IsBulk()
	}
}
