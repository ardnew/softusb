package device

import (
	"bytes"
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
