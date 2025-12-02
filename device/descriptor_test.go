package device

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestDeviceDescriptor_MarshalTo(t *testing.T) {
	desc := &DeviceDescriptor{
		USBVersion:        0x0200,
		DeviceClass:       ClassPerInterface,
		DeviceSubClass:    0,
		DeviceProtocol:    0,
		MaxPacketSize0:    64,
		VendorID:          0xCAFE,
		ProductID:         0xBABE,
		DeviceVersion:     0x0100,
		ManufacturerIndex: 1,
		ProductIndex:      2,
		SerialNumberIndex: 3,
		NumConfigurations: 1,
	}

	var buf [18]byte
	n := desc.MarshalTo(buf[:])
	if n != 18 {
		t.Fatalf("expected 18 bytes, got %d", n)
	}
	if buf[0] != 18 {
		t.Errorf("bLength = %d, want 18", buf[0])
	}
	if buf[1] != DescriptorTypeDevice {
		t.Errorf("bDescriptorType = 0x%02X, want 0x%02X", buf[1], DescriptorTypeDevice)
	}
}

func TestDeviceDescriptor_RoundTrip(t *testing.T) {
	original := &DeviceDescriptor{
		USBVersion:        0x0200,
		DeviceClass:       ClassCDC,
		DeviceSubClass:    0x02,
		DeviceProtocol:    0x01,
		MaxPacketSize0:    64,
		VendorID:          0x1234,
		ProductID:         0x5678,
		DeviceVersion:     0x0101,
		ManufacturerIndex: 1,
		ProductIndex:      2,
		SerialNumberIndex: 3,
		NumConfigurations: 1,
	}

	var buf [18]byte
	original.MarshalTo(buf[:])

	var parsed DeviceDescriptor
	err := ParseDeviceDescriptor(buf[:], &parsed)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.VendorID != original.VendorID {
		t.Errorf("VendorID = 0x%04X, want 0x%04X", parsed.VendorID, original.VendorID)
	}
	if parsed.ProductID != original.ProductID {
		t.Errorf("ProductID = 0x%04X, want 0x%04X", parsed.ProductID, original.ProductID)
	}
}

func TestParseDeviceDescriptor_TooShort(t *testing.T) {
	var parsed DeviceDescriptor
	err := ParseDeviceDescriptor(make([]byte, 10), &parsed)
	if err == nil {
		t.Error("expected error for short descriptor")
	}
}

func TestParseDeviceDescriptor_WrongType(t *testing.T) {
	data := make([]byte, 18)
	data[0] = 18
	data[1] = DescriptorTypeConfiguration // wrong type
	var parsed DeviceDescriptor
	err := ParseDeviceDescriptor(data, &parsed)
	if err == nil {
		t.Error("expected error for wrong descriptor type")
	}
}

func TestConfigurationDescriptor_MarshalTo(t *testing.T) {
	desc := &ConfigurationDescriptor{
		TotalLength:        32,
		NumInterfaces:      2,
		ConfigurationValue: 1,
		ConfigurationIndex: 0,
		Attributes:         ConfigAttrBusPowered,
		MaxPower:           50, // 100mA
	}

	var buf [9]byte
	n := desc.MarshalTo(buf[:])
	if n != 9 {
		t.Fatalf("expected 9 bytes, got %d", n)
	}
	if buf[0] != 9 {
		t.Errorf("bLength = %d, want 9", buf[0])
	}
}

func TestConfigurationDescriptor_RoundTrip(t *testing.T) {
	original := &ConfigurationDescriptor{
		TotalLength:        100,
		NumInterfaces:      3,
		ConfigurationValue: 1,
		ConfigurationIndex: 4,
		Attributes:         ConfigAttrBusPowered | ConfigAttrRemoteWakeup,
		MaxPower:           250,
	}

	var buf [9]byte
	original.MarshalTo(buf[:])

	var parsed ConfigurationDescriptor
	err := ParseConfigurationDescriptor(buf[:], &parsed)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.TotalLength != original.TotalLength {
		t.Errorf("TotalLength = %d, want %d", parsed.TotalLength, original.TotalLength)
	}
	if parsed.NumInterfaces != original.NumInterfaces {
		t.Errorf("NumInterfaces = %d, want %d", parsed.NumInterfaces, original.NumInterfaces)
	}
}

func TestInterfaceDescriptor_MarshalTo(t *testing.T) {
	desc := &InterfaceDescriptor{
		InterfaceNumber:   0,
		AlternateSetting:  0,
		NumEndpoints:      2,
		InterfaceClass:    ClassCDC,
		InterfaceSubClass: 0x02,
		InterfaceProtocol: 0x01,
		InterfaceIndex:    0,
	}

	var buf [9]byte
	n := desc.MarshalTo(buf[:])
	if n != 9 {
		t.Fatalf("expected 9 bytes, got %d", n)
	}
}

func TestInterfaceDescriptor_RoundTrip(t *testing.T) {
	original := &InterfaceDescriptor{
		InterfaceNumber:   1,
		AlternateSetting:  2,
		NumEndpoints:      3,
		InterfaceClass:    ClassHID,
		InterfaceSubClass: 0x01,
		InterfaceProtocol: 0x02,
		InterfaceIndex:    5,
	}

	var buf [9]byte
	original.MarshalTo(buf[:])

	var parsed InterfaceDescriptor
	err := ParseInterfaceDescriptor(buf[:], &parsed)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.InterfaceNumber != original.InterfaceNumber {
		t.Errorf("InterfaceNumber = %d, want %d", parsed.InterfaceNumber, original.InterfaceNumber)
	}
	if parsed.InterfaceClass != original.InterfaceClass {
		t.Errorf("InterfaceClass = 0x%02X, want 0x%02X", parsed.InterfaceClass, original.InterfaceClass)
	}
}

func TestEndpointDescriptor_MarshalTo(t *testing.T) {
	desc := &EndpointDescriptor{
		EndpointAddress: 0x81, // EP1 IN
		Attributes:      EndpointTypeBulk,
		MaxPacketSize:   512,
		Interval:        0,
	}

	var buf [7]byte
	n := desc.MarshalTo(buf[:])
	if n != 7 {
		t.Fatalf("expected 7 bytes, got %d", n)
	}
}

func TestEndpointDescriptor_RoundTrip(t *testing.T) {
	original := &EndpointDescriptor{
		EndpointAddress: 0x02, // EP2 OUT
		Attributes:      EndpointTypeInterrupt,
		MaxPacketSize:   64,
		Interval:        10,
	}

	var buf [7]byte
	original.MarshalTo(buf[:])

	var parsed EndpointDescriptor
	err := ParseEndpointDescriptor(buf[:], &parsed)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if parsed.EndpointAddress != original.EndpointAddress {
		t.Errorf("EndpointAddress = 0x%02X, want 0x%02X", parsed.EndpointAddress, original.EndpointAddress)
	}
	if parsed.MaxPacketSize != original.MaxPacketSize {
		t.Errorf("MaxPacketSize = %d, want %d", parsed.MaxPacketSize, original.MaxPacketSize)
	}
}

func TestInterfaceAssociationDescriptor_MarshalTo(t *testing.T) {
	iad := &InterfaceAssociationDescriptor{
		FirstInterface:   0,
		InterfaceCount:   2,
		FunctionClass:    ClassCDC,
		FunctionSubClass: 0x02,
		FunctionProtocol: 0x01,
		FunctionIndex:    0,
	}

	var buf [8]byte
	n := iad.MarshalTo(buf[:])
	if n != 8 {
		t.Fatalf("expected 8 bytes, got %d", n)
	}
	if buf[1] != DescriptorTypeInterfaceAssociation {
		t.Errorf("bDescriptorType = 0x%02X, want 0x%02X", buf[1], DescriptorTypeInterfaceAssociation)
	}
}

func TestStringDescriptorTo(t *testing.T) {
	tests := []struct {
		input string
		want  int // expected length
	}{
		{"", 2},
		{"A", 4},
		{"Hello", 12},
		{"Test Device", 24},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var buf [256]byte
			n := StringDescriptorTo(buf[:], tt.input)
			if n != tt.want {
				t.Errorf("len = %d, want %d", n, tt.want)
			}
			if buf[0] != uint8(tt.want) {
				t.Errorf("bLength = %d, want %d", buf[0], tt.want)
			}
			if buf[1] != DescriptorTypeString {
				t.Errorf("bDescriptorType = 0x%02X, want 0x%02X", buf[1], DescriptorTypeString)
			}
		})
	}
}

func TestLanguageDescriptorTo(t *testing.T) {
	var buf [4]byte
	n := LanguageDescriptorTo(buf[:], LangIDUSEnglish)
	if n != 4 {
		t.Fatalf("expected 4 bytes, got %d", n)
	}
	if buf[0] != 4 {
		t.Errorf("bLength = %d, want 4", buf[0])
	}
	if buf[1] != DescriptorTypeString {
		t.Errorf("bDescriptorType = 0x%02X, want 0x%02X", buf[1], DescriptorTypeString)
	}

	// Multi-language
	var buf2 [6]byte
	n = LanguageDescriptorTo(buf2[:], 0x0409, 0x0407)
	if n != 6 {
		t.Fatalf("expected 6 bytes, got %d", n)
	}
}

func TestStringDescriptorTo_UTF16(t *testing.T) {
	// Test that unicode characters are properly encoded
	var buf [256]byte
	n := StringDescriptorTo(buf[:], "日本語")
	if buf[1] != DescriptorTypeString {
		t.Errorf("bDescriptorType = 0x%02X, want 0x%02X", buf[1], DescriptorTypeString)
	}
	// Each character should be 2 bytes in UTF-16LE
	expectedLen := 2 + 3*2 // header + 3 characters
	if n != expectedLen {
		t.Errorf("len = %d, want %d", n, expectedLen)
	}
}

func TestStringDescriptorTo_MaxLength(t *testing.T) {
	// Create a very long string
	longStr := bytes.Repeat([]byte{'A'}, 300)
	var buf [256]byte
	n := StringDescriptorTo(buf[:], string(longStr))

	// Should be truncated to max 255 bytes
	if n > 255 {
		t.Errorf("descriptor too long: %d bytes", n)
	}
	if buf[0] != uint8(n) {
		t.Errorf("bLength = %d, actual len = %d", buf[0], n)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestDeviceDescriptor_MarshalTo_BufferTooSmall(t *testing.T) {
	desc := &DeviceDescriptor{
		USBVersion:     0x0200,
		MaxPacketSize0: 64,
		VendorID:       0xCAFE,
		ProductID:      0xBABE,
	}

	tests := []struct {
		name    string
		bufSize int
		wantN   int
	}{
		{"0 bytes", 0, 0},
		{"17 bytes", 17, 0},
		{"18 bytes (exact)", 18, 18},
		{"64 bytes", 64, 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n := desc.MarshalTo(buf)
			if n != tt.wantN {
				t.Errorf("MarshalTo() = %d, want %d", n, tt.wantN)
			}
		})
	}
}

func TestDeviceDescriptor_AllMaxValues(t *testing.T) {
	desc := &DeviceDescriptor{
		USBVersion:        0xFFFF,
		DeviceClass:       0xFF,
		DeviceSubClass:    0xFF,
		DeviceProtocol:    0xFF,
		MaxPacketSize0:    0xFF,
		VendorID:          0xFFFF,
		ProductID:         0xFFFF,
		DeviceVersion:     0xFFFF,
		ManufacturerIndex: 0xFF,
		ProductIndex:      0xFF,
		SerialNumberIndex: 0xFF,
		NumConfigurations: 0xFF,
	}

	var buf [DeviceDescriptorSize]byte
	n := desc.MarshalTo(buf[:])
	if n != DeviceDescriptorSize {
		t.Fatalf("MarshalTo() = %d, want %d", n, DeviceDescriptorSize)
	}

	var parsed DeviceDescriptor
	if err := ParseDeviceDescriptor(buf[:], &parsed); err != nil {
		t.Fatalf("ParseDeviceDescriptor() error = %v", err)
	}

	if parsed.VendorID != 0xFFFF {
		t.Errorf("VendorID = 0x%04X, want 0xFFFF", parsed.VendorID)
	}
	if parsed.ProductID != 0xFFFF {
		t.Errorf("ProductID = 0x%04X, want 0xFFFF", parsed.ProductID)
	}
}

func TestConfigurationDescriptor_MarshalTo_BufferTooSmall(t *testing.T) {
	desc := &ConfigurationDescriptor{
		TotalLength:        100,
		NumInterfaces:      2,
		ConfigurationValue: 1,
	}

	tests := []struct {
		name    string
		bufSize int
		wantN   int
	}{
		{"0 bytes", 0, 0},
		{"8 bytes", 8, 0},
		{"9 bytes (exact)", 9, 9},
		{"64 bytes", 64, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n := desc.MarshalTo(buf)
			if n != tt.wantN {
				t.Errorf("MarshalTo() = %d, want %d", n, tt.wantN)
			}
		})
	}
}

func TestConfigurationDescriptor_MaxTotalLength(t *testing.T) {
	desc := &ConfigurationDescriptor{
		TotalLength:        0xFFFF, // Max possible total length
		NumInterfaces:      0xFF,
		ConfigurationValue: 0xFF,
		Attributes:         0xFF,
		MaxPower:           0xFF,
	}

	var buf [ConfigurationDescriptorSize]byte
	n := desc.MarshalTo(buf[:])
	if n != ConfigurationDescriptorSize {
		t.Fatalf("MarshalTo() = %d, want %d", n, ConfigurationDescriptorSize)
	}

	var parsed ConfigurationDescriptor
	if err := ParseConfigurationDescriptor(buf[:], &parsed); err != nil {
		t.Fatalf("ParseConfigurationDescriptor() error = %v", err)
	}

	if parsed.TotalLength != 0xFFFF {
		t.Errorf("TotalLength = 0x%04X, want 0xFFFF", parsed.TotalLength)
	}
}

func TestInterfaceDescriptor_MarshalTo_BufferTooSmall(t *testing.T) {
	desc := &InterfaceDescriptor{
		InterfaceNumber: 0,
		NumEndpoints:    2,
		InterfaceClass:  ClassCDC,
	}

	buf := make([]byte, 8)
	n := desc.MarshalTo(buf)
	if n != 0 {
		t.Errorf("MarshalTo() with small buffer = %d, want 0", n)
	}
}

func TestEndpointDescriptor_MarshalTo_BufferTooSmall(t *testing.T) {
	desc := &EndpointDescriptor{
		EndpointAddress: 0x81,
		Attributes:      EndpointTypeBulk,
		MaxPacketSize:   512,
	}

	buf := make([]byte, 6)
	n := desc.MarshalTo(buf)
	if n != 0 {
		t.Errorf("MarshalTo() with small buffer = %d, want 0", n)
	}
}

func TestEndpointDescriptor_AllEndpointTypes(t *testing.T) {
	types := []struct {
		name       string
		attributes uint8
	}{
		{"Control", EndpointTypeControl},
		{"Isochronous", EndpointTypeIsochronous},
		{"Bulk", EndpointTypeBulk},
		{"Interrupt", EndpointTypeInterrupt},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			desc := &EndpointDescriptor{
				EndpointAddress: 0x81,
				Attributes:      tt.attributes,
				MaxPacketSize:   64,
				Interval:        1,
			}

			var buf [EndpointDescriptorSize]byte
			n := desc.MarshalTo(buf[:])
			if n != EndpointDescriptorSize {
				t.Fatalf("MarshalTo() = %d, want %d", n, EndpointDescriptorSize)
			}

			var parsed EndpointDescriptor
			if err := ParseEndpointDescriptor(buf[:], &parsed); err != nil {
				t.Fatalf("ParseEndpointDescriptor() error = %v", err)
			}

			if parsed.Attributes != tt.attributes {
				t.Errorf("Attributes = 0x%02X, want 0x%02X", parsed.Attributes, tt.attributes)
			}
		})
	}
}

func TestEndpointDescriptor_AllAddresses(t *testing.T) {
	// Test all valid endpoint addresses (0x00-0x0F OUT, 0x80-0x8F IN)
	addresses := []uint8{
		0x00, 0x01, 0x02, 0x0F, // OUT endpoints
		0x80, 0x81, 0x82, 0x8F, // IN endpoints
	}

	for _, addr := range addresses {
		t.Run(fmt.Sprintf("EP_%02X", addr), func(t *testing.T) {
			desc := &EndpointDescriptor{
				EndpointAddress: addr,
				Attributes:      EndpointTypeBulk,
				MaxPacketSize:   64,
			}

			var buf [EndpointDescriptorSize]byte
			desc.MarshalTo(buf[:])

			var parsed EndpointDescriptor
			if err := ParseEndpointDescriptor(buf[:], &parsed); err != nil {
				t.Fatalf("ParseEndpointDescriptor() error = %v", err)
			}

			if parsed.EndpointAddress != addr {
				t.Errorf("EndpointAddress = 0x%02X, want 0x%02X", parsed.EndpointAddress, addr)
			}
		})
	}
}

func TestInterfaceAssociationDescriptor_MarshalTo_BufferTooSmall(t *testing.T) {
	iad := &InterfaceAssociationDescriptor{
		FirstInterface:   0,
		InterfaceCount:   2,
		FunctionClass:    ClassCDC,
		FunctionSubClass: 0x02,
	}

	buf := make([]byte, 7)
	n := iad.MarshalTo(buf)
	if n != 0 {
		t.Errorf("MarshalTo() with small buffer = %d, want 0", n)
	}
}

func TestStringDescriptorTo_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		bufSize    int
		wantN      int
		wantLength uint8
	}{
		{"empty string, exact buffer", "", 2, 2, 2},
		{"empty string, large buffer", "", 256, 2, 2},
		{"single char", "A", 4, 4, 4},
		{"buffer too small for header", "test", 1, 0, 0},
		{"buffer too small for content", "ABCD", 4, 0, 0},
		{"buffer exactly fits", "AB", 6, 6, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n := StringDescriptorTo(buf, tt.input)
			if n != tt.wantN {
				t.Errorf("StringDescriptorTo() = %d, want %d", n, tt.wantN)
			}
			if n > 0 && buf[0] != tt.wantLength {
				t.Errorf("bLength = %d, want %d", buf[0], tt.wantLength)
			}
		})
	}
}

func TestStringDescriptorTo_UnicodeVariants(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ASCII", "Hello World"},
		{"Japanese", "日本語"},
		{"German", "Größe"},
		{"Emoji", "USB⚡"},
		{"Mixed", "Device™"},
		{"Chinese", "设备"},
		{"Korean", "장치"},
		{"Arabic", "جهاز"},
		{"Cyrillic", "Устройство"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf [256]byte
			n := StringDescriptorTo(buf[:], tt.input)
			if n < 2 {
				t.Errorf("StringDescriptorTo() = %d, want >= 2", n)
			}
			if buf[1] != DescriptorTypeString {
				t.Errorf("bDescriptorType = 0x%02X, want 0x%02X", buf[1], DescriptorTypeString)
			}
			// Verify length matches header
			if buf[0] != uint8(n) {
				t.Errorf("bLength = %d, actual = %d", buf[0], n)
			}
		})
	}
}

func TestStringDescriptorTo_MaxLengthBoundary(t *testing.T) {
	// Test strings near the 255-byte boundary
	// Max descriptor length is 255 bytes = 2 header + 253 bytes of UTF-16
	// Since UTF-16 uses 2 bytes per char, max chars = (255-2)/2 = 126 (truncated)
	// But the code uses length > 255 check, so 127 chars = 256 bytes gets truncated to 126 chars = 254+2 header
	tests := []struct {
		name       string
		charCount  int
		wantLength int
	}{
		{"126 chars (fits)", 126, 254},      // 2 + 126*2 = 254, fits
		{"127 chars (max)", 127, 255},       // Would be 256, truncated to 255 (the code allows up to 255)
		{"200 chars (truncated)", 200, 255}, // Truncated to max 255
		{"1 char", 1, 4},                    // 2 + 1*2 = 4
		{"0 chars", 0, 2},                   // Just header
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.Repeat("A", tt.charCount)
			var buf [256]byte
			n := StringDescriptorTo(buf[:], input)
			if n != tt.wantLength {
				t.Errorf("StringDescriptorTo() = %d, want %d", n, tt.wantLength)
			}
		})
	}
}

func TestLanguageDescriptorTo_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		langIDs []uint16
		bufSize int
		wantN   int
	}{
		{"single language", []uint16{0x0409}, 4, 4},
		{"two languages", []uint16{0x0409, 0x0407}, 6, 6},
		{"buffer too small", []uint16{0x0409}, 3, 0},
		{"empty (no languages)", []uint16{}, 2, 2},
		{"many languages", []uint16{0x0409, 0x0407, 0x0809, 0x040C}, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n := LanguageDescriptorTo(buf, tt.langIDs...)
			if n != tt.wantN {
				t.Errorf("LanguageDescriptorTo() = %d, want %d", n, tt.wantN)
			}
		})
	}
}

func TestParseDescriptor_AllTypeMismatches(t *testing.T) {
	tests := []struct {
		name      string
		parseFunc func([]byte) error
		wrongType uint8
		bufSize   int
	}{
		{
			"DeviceDescriptor with config type",
			func(data []byte) error { var d DeviceDescriptor; return ParseDeviceDescriptor(data, &d) },
			DescriptorTypeConfiguration,
			DeviceDescriptorSize,
		},
		{
			"ConfigurationDescriptor with device type",
			func(data []byte) error { var c ConfigurationDescriptor; return ParseConfigurationDescriptor(data, &c) },
			DescriptorTypeDevice,
			ConfigurationDescriptorSize,
		},
		{
			"InterfaceDescriptor with endpoint type",
			func(data []byte) error { var i InterfaceDescriptor; return ParseInterfaceDescriptor(data, &i) },
			DescriptorTypeEndpoint,
			InterfaceDescriptorSize,
		},
		{
			"EndpointDescriptor with interface type",
			func(data []byte) error { var e EndpointDescriptor; return ParseEndpointDescriptor(data, &e) },
			DescriptorTypeInterface,
			EndpointDescriptorSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.bufSize)
			data[0] = uint8(tt.bufSize)
			data[1] = tt.wrongType
			err := tt.parseFunc(data)
			if err == nil {
				t.Error("expected error for wrong descriptor type")
			}
		})
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDeviceDescriptor_MarshalTo(b *testing.B) {
	desc := &DeviceDescriptor{
		USBVersion:        0x0200,
		DeviceClass:       ClassPerInterface,
		MaxPacketSize0:    64,
		VendorID:          0xCAFE,
		ProductID:         0xBABE,
		DeviceVersion:     0x0100,
		ManufacturerIndex: 1,
		ProductIndex:      2,
		SerialNumberIndex: 3,
		NumConfigurations: 1,
	}

	b.ReportAllocs()
	var buf [DeviceDescriptorSize]byte
	for i := 0; i < b.N; i++ {
		desc.MarshalTo(buf[:])
	}
}

func BenchmarkDeviceDescriptor_Parse(b *testing.B) {
	data := []byte{
		18, 0x01, 0x00, 0x02, 0x00, 0x00, 0x00, 64,
		0xFE, 0xCA, 0xBE, 0xBA, 0x00, 0x01, 1, 2, 3, 1,
	}

	b.ReportAllocs()
	var desc DeviceDescriptor
	for i := 0; i < b.N; i++ {
		_ = ParseDeviceDescriptor(data, &desc)
	}
}

func BenchmarkConfigurationDescriptor_MarshalTo(b *testing.B) {
	desc := &ConfigurationDescriptor{
		TotalLength:        100,
		NumInterfaces:      2,
		ConfigurationValue: 1,
		Attributes:         ConfigAttrBusPowered,
		MaxPower:           50,
	}

	b.ReportAllocs()
	var buf [ConfigurationDescriptorSize]byte
	for i := 0; i < b.N; i++ {
		desc.MarshalTo(buf[:])
	}
}

func BenchmarkConfigurationDescriptor_Parse(b *testing.B) {
	data := []byte{9, 0x02, 100, 0, 2, 1, 0, 0x80, 50}

	b.ReportAllocs()
	var desc ConfigurationDescriptor
	for i := 0; i < b.N; i++ {
		_ = ParseConfigurationDescriptor(data, &desc)
	}
}

func BenchmarkInterfaceDescriptor_MarshalTo(b *testing.B) {
	desc := &InterfaceDescriptor{
		InterfaceNumber:   0,
		NumEndpoints:      2,
		InterfaceClass:    ClassCDC,
		InterfaceSubClass: 0x02,
		InterfaceProtocol: 0x01,
	}

	b.ReportAllocs()
	var buf [InterfaceDescriptorSize]byte
	for i := 0; i < b.N; i++ {
		desc.MarshalTo(buf[:])
	}
}

func BenchmarkInterfaceDescriptor_Parse(b *testing.B) {
	data := []byte{9, 0x04, 0, 0, 2, 0x02, 0x02, 0x01, 0}

	b.ReportAllocs()
	var desc InterfaceDescriptor
	for i := 0; i < b.N; i++ {
		_ = ParseInterfaceDescriptor(data, &desc)
	}
}

func BenchmarkEndpointDescriptor_MarshalTo(b *testing.B) {
	desc := &EndpointDescriptor{
		EndpointAddress: 0x81,
		Attributes:      EndpointTypeBulk,
		MaxPacketSize:   512,
		Interval:        0,
	}

	b.ReportAllocs()
	var buf [EndpointDescriptorSize]byte
	for i := 0; i < b.N; i++ {
		desc.MarshalTo(buf[:])
	}
}

func BenchmarkEndpointDescriptor_Parse(b *testing.B) {
	data := []byte{7, 0x05, 0x81, 0x02, 0x00, 0x02, 0}

	b.ReportAllocs()
	var desc EndpointDescriptor
	for i := 0; i < b.N; i++ {
		_ = ParseEndpointDescriptor(data, &desc)
	}
}

func BenchmarkInterfaceAssociationDescriptor_MarshalTo(b *testing.B) {
	iad := &InterfaceAssociationDescriptor{
		FirstInterface:   0,
		InterfaceCount:   2,
		FunctionClass:    ClassCDC,
		FunctionSubClass: 0x02,
		FunctionProtocol: 0x01,
	}

	b.ReportAllocs()
	var buf [IADSize]byte
	for i := 0; i < b.N; i++ {
		iad.MarshalTo(buf[:])
	}
}

func BenchmarkStringDescriptorTo(b *testing.B) {
	lengths := []int{0, 32, 64, 127, 254}

	for _, length := range lengths {
		name := fmt.Sprintf("len=%d", length)
		input := strings.Repeat("A", length)

		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			var buf [256]byte
			for i := 0; i < b.N; i++ {
				StringDescriptorTo(buf[:], input)
			}
		})
	}
}

func BenchmarkStringDescriptorTo_Unicode(b *testing.B) {
	inputs := []struct {
		name  string
		input string
	}{
		{"ASCII", "USB Device"},
		{"Japanese", "日本語デバイス"},
		{"Mixed", "Device™ ⚡"},
	}

	for _, tt := range inputs {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			var buf [256]byte
			for i := 0; i < b.N; i++ {
				StringDescriptorTo(buf[:], tt.input)
			}
		})
	}
}

func BenchmarkLanguageDescriptorTo(b *testing.B) {
	counts := []int{1, 2, 4}

	for _, count := range counts {
		name := fmt.Sprintf("langs=%d", count)
		langs := make([]uint16, count)
		for i := range langs {
			langs[i] = 0x0409
		}

		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			var buf [16]byte
			for i := 0; i < b.N; i++ {
				LanguageDescriptorTo(buf[:], langs...)
			}
		})
	}
}

func BenchmarkDescriptor_RoundTrip(b *testing.B) {
	b.Run("DeviceDescriptor", func(b *testing.B) {
		b.ReportAllocs()
		desc := &DeviceDescriptor{
			USBVersion:        0x0200,
			MaxPacketSize0:    64,
			VendorID:          0xCAFE,
			ProductID:         0xBABE,
			NumConfigurations: 1,
		}
		var buf [DeviceDescriptorSize]byte
		var parsed DeviceDescriptor
		for i := 0; i < b.N; i++ {
			desc.MarshalTo(buf[:])
			_ = ParseDeviceDescriptor(buf[:], &parsed)
		}
	})

	b.Run("ConfigurationDescriptor", func(b *testing.B) {
		b.ReportAllocs()
		desc := &ConfigurationDescriptor{
			TotalLength:        100,
			NumInterfaces:      2,
			ConfigurationValue: 1,
		}
		var buf [ConfigurationDescriptorSize]byte
		var parsed ConfigurationDescriptor
		for i := 0; i < b.N; i++ {
			desc.MarshalTo(buf[:])
			_ = ParseConfigurationDescriptor(buf[:], &parsed)
		}
	})

	b.Run("EndpointDescriptor", func(b *testing.B) {
		b.ReportAllocs()
		desc := &EndpointDescriptor{
			EndpointAddress: 0x81,
			Attributes:      EndpointTypeBulk,
			MaxPacketSize:   512,
		}
		var buf [EndpointDescriptorSize]byte
		var parsed EndpointDescriptor
		for i := 0; i < b.N; i++ {
			desc.MarshalTo(buf[:])
			_ = ParseEndpointDescriptor(buf[:], &parsed)
		}
	})
}
