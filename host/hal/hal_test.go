package hal

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
		{SpeedUnknown, "Unknown"},
		{SpeedLow, "Low Speed"},
		{SpeedFull, "Full Speed"},
		{SpeedHigh, "High Speed"},
		{Speed(255), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.speed.String(); got != tt.expected {
				t.Errorf("Speed(%d).String() = %q, want %q", tt.speed, got, tt.expected)
			}
		})
	}
}

// =============================================================================
// PortStatus Tests
// =============================================================================

func TestPortStatus_Fields(t *testing.T) {
	status := PortStatus{
		Connected:     true,
		Enabled:       true,
		Suspended:     false,
		OverCurrent:   false,
		Reset:         false,
		PowerOn:       true,
		Speed:         SpeedHigh,
		ConnectChange: true,
		EnableChange:  false,
		ResetChange:   false,
	}

	if !status.Connected {
		t.Error("Connected should be true")
	}
	if !status.Enabled {
		t.Error("Enabled should be true")
	}
	if status.Suspended {
		t.Error("Suspended should be false")
	}
	if !status.PowerOn {
		t.Error("PowerOn should be true")
	}
	if status.Speed != SpeedHigh {
		t.Errorf("Speed = %v, want SpeedHigh", status.Speed)
	}
	if !status.ConnectChange {
		t.Error("ConnectChange should be true")
	}
}

// =============================================================================
// SetupPacket Tests
// =============================================================================

func TestParseSetupPacket(t *testing.T) {
	data := []byte{
		0x80,       // RequestType (Device-to-Host, Standard, Device)
		0x06,       // Request (GET_DESCRIPTOR)
		0x00, 0x01, // Value (Device Descriptor)
		0x00, 0x00, // Index
		0x12, 0x00, // Length (18)
	}

	var setup SetupPacket
	if !ParseSetupPacket(data, &setup) {
		t.Fatal("ParseSetupPacket returned false")
	}

	if setup.RequestType != 0x80 {
		t.Errorf("RequestType = 0x%02X, want 0x80", setup.RequestType)
	}
	if setup.Request != 0x06 {
		t.Errorf("Request = 0x%02X, want 0x06", setup.Request)
	}
	if setup.Value != 0x0100 {
		t.Errorf("Value = 0x%04X, want 0x0100", setup.Value)
	}
	if setup.Index != 0x0000 {
		t.Errorf("Index = 0x%04X, want 0x0000", setup.Index)
	}
	if setup.Length != 0x0012 {
		t.Errorf("Length = 0x%04X, want 0x0012", setup.Length)
	}
}

func TestParseSetupPacket_TooShort(t *testing.T) {
	data := make([]byte, SetupPacketSize-1)
	var setup SetupPacket
	if ParseSetupPacket(data, &setup) {
		t.Error("ParseSetupPacket should return false for short data")
	}
}

func TestSetupPacket_MarshalTo(t *testing.T) {
	setup := SetupPacket{
		RequestType: 0x80,
		Request:     0x06,
		Value:       0x0100,
		Index:       0x0409,
		Length:      0x00FF,
	}

	buf := make([]byte, SetupPacketSize)
	n := setup.MarshalTo(buf)

	if n != SetupPacketSize {
		t.Errorf("MarshalTo returned %d, want %d", n, SetupPacketSize)
	}

	expected := []byte{0x80, 0x06, 0x00, 0x01, 0x09, 0x04, 0xFF, 0x00}
	for i, b := range expected {
		if buf[i] != b {
			t.Errorf("buf[%d] = 0x%02X, want 0x%02X", i, buf[i], b)
		}
	}
}

func TestSetupPacket_MarshalTo_TooSmall(t *testing.T) {
	setup := SetupPacket{}
	buf := make([]byte, SetupPacketSize-1)

	n := setup.MarshalTo(buf)
	if n != 0 {
		t.Errorf("MarshalTo to small buffer returned %d, want 0", n)
	}
}

func TestSetupPacket_RoundTrip(t *testing.T) {
	original := SetupPacket{
		RequestType: 0x21,
		Request:     0x09,
		Value:       0x0200,
		Index:       0x0001,
		Length:      0x0008,
	}

	buf := make([]byte, SetupPacketSize)
	original.MarshalTo(buf)

	var parsed SetupPacket
	if !ParseSetupPacket(buf, &parsed) {
		t.Fatal("ParseSetupPacket returned false")
	}

	if parsed != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", parsed, original)
	}
}

// =============================================================================
// TransferType Tests
// =============================================================================

func TestTransferType_Values(t *testing.T) {
	if TransferControl != 0 {
		t.Errorf("TransferControl = %d, want 0", TransferControl)
	}
	if TransferIsochronous != 1 {
		t.Errorf("TransferIsochronous = %d, want 1", TransferIsochronous)
	}
	if TransferBulk != 2 {
		t.Errorf("TransferBulk = %d, want 2", TransferBulk)
	}
	if TransferInterrupt != 3 {
		t.Errorf("TransferInterrupt = %d, want 3", TransferInterrupt)
	}
}

// =============================================================================
// EndpointDescriptor Tests
// =============================================================================

func TestEndpointDescriptor_Number(t *testing.T) {
	tests := []struct {
		address  uint8
		expected uint8
	}{
		{0x00, 0},
		{0x01, 1},
		{0x0F, 15},
		{0x81, 1},
		{0x8F, 15},
	}

	for _, tt := range tests {
		ep := EndpointDescriptor{Address: tt.address}
		if got := ep.Number(); got != tt.expected {
			t.Errorf("EndpointDescriptor{Address: 0x%02X}.Number() = %d, want %d",
				tt.address, got, tt.expected)
		}
	}
}

func TestEndpointDescriptor_IsIn(t *testing.T) {
	tests := []struct {
		address  uint8
		expected bool
	}{
		{0x00, false},
		{0x01, false},
		{0x0F, false},
		{0x80, true},
		{0x81, true},
		{0x8F, true},
	}

	for _, tt := range tests {
		ep := EndpointDescriptor{Address: tt.address}
		if got := ep.IsIn(); got != tt.expected {
			t.Errorf("EndpointDescriptor{Address: 0x%02X}.IsIn() = %v, want %v",
				tt.address, got, tt.expected)
		}
	}
}

func TestEndpointDescriptor_TransferType(t *testing.T) {
	tests := []struct {
		attributes uint8
		expected   TransferType
	}{
		{0x00, TransferControl},
		{0x01, TransferIsochronous},
		{0x02, TransferBulk},
		{0x03, TransferInterrupt},
		{0x80, TransferControl},   // Other bits should be masked
		{0xFF, TransferInterrupt}, // All bits set
	}

	for _, tt := range tests {
		ep := EndpointDescriptor{Attributes: tt.attributes}
		if got := ep.TransferType(); got != tt.expected {
			t.Errorf("EndpointDescriptor{Attributes: 0x%02X}.TransferType() = %d, want %d",
				tt.attributes, got, tt.expected)
		}
	}
}

// =============================================================================
// DeviceAddress Tests
// =============================================================================

func TestDeviceAddress_Range(t *testing.T) {
	// Valid addresses are 0-127
	for i := 0; i <= 127; i++ {
		addr := DeviceAddress(i)
		if uint8(addr) != uint8(i) {
			t.Errorf("DeviceAddress(%d) = %d, want %d", i, addr, i)
		}
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkParseSetupPacket(b *testing.B) {
	data := []byte{0x80, 0x06, 0x00, 0x01, 0x00, 0x00, 0x12, 0x00}
	var setup SetupPacket

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseSetupPacket(data, &setup)
	}
}

func BenchmarkSetupPacket_MarshalTo(b *testing.B) {
	setup := SetupPacket{
		RequestType: 0x80,
		Request:     0x06,
		Value:       0x0100,
		Index:       0x0000,
		Length:      0x0012,
	}
	buf := make([]byte, SetupPacketSize)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		setup.MarshalTo(buf)
	}
}

func BenchmarkSpeed_String(b *testing.B) {
	speeds := []Speed{SpeedUnknown, SpeedLow, SpeedFull, SpeedHigh}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = speeds[i%4].String()
	}
}

func BenchmarkEndpointDescriptor_Number(b *testing.B) {
	ep := EndpointDescriptor{Address: 0x81}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.Number()
	}
}

func BenchmarkEndpointDescriptor_IsIn(b *testing.B) {
	ep := EndpointDescriptor{Address: 0x81}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.IsIn()
	}
}

func BenchmarkEndpointDescriptor_TransferType(b *testing.B) {
	ep := EndpointDescriptor{Attributes: 0x02}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.TransferType()
	}
}
