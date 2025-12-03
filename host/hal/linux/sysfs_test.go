//go:build linux

package linux

import (
	"testing"

	"github.com/ardnew/softusb/host/hal"
)

// =============================================================================
// formatPadded Tests
// =============================================================================

func TestFormatPadded(t *testing.T) {
	tests := []struct {
		val      uint8
		width    int
		expected string
	}{
		{0, 3, "000"},
		{1, 3, "001"},
		{12, 3, "012"},
		{123, 3, "123"},
		{0, 1, "0"},
		{9, 1, "9"},
		{255, 3, "255"},
	}

	for _, tt := range tests {
		buf := make([]byte, 10)
		n := formatPadded(buf, tt.val, tt.width)
		got := string(buf[:n])
		if got != tt.expected {
			t.Errorf("formatPadded(%d, %d) = %q, want %q", tt.val, tt.width, got, tt.expected)
		}
	}
}

// =============================================================================
// formatDevfsPath Tests
// =============================================================================

func TestFormatDevfsPath(t *testing.T) {
	tests := []struct {
		busNum   uint8
		devNum   uint8
		expected string
	}{
		{1, 1, "/dev/bus/usb/001/001"},
		{1, 123, "/dev/bus/usb/001/123"},
		{12, 34, "/dev/bus/usb/012/034"},
		{255, 255, "/dev/bus/usb/255/255"},
	}

	for _, tt := range tests {
		got := formatDevfsPath(tt.busNum, tt.devNum)
		if got != tt.expected {
			t.Errorf("formatDevfsPath(%d, %d) = %q, want %q",
				tt.busNum, tt.devNum, got, tt.expected)
		}
	}
}

// =============================================================================
// parseSpeed Tests
// =============================================================================

func TestParseSpeed(t *testing.T) {
	tests := []struct {
		input    string
		expected hal.Speed
	}{
		{"1.5", hal.SpeedLow},
		{"12", hal.SpeedFull},
		{"480", hal.SpeedHigh},
		{"", hal.SpeedUnknown},
		{"5000", hal.SpeedUnknown}, // SuperSpeed not supported
		{"invalid", hal.SpeedUnknown},
	}

	for _, tt := range tests {
		got := parseSpeed(tt.input)
		if got != tt.expected {
			t.Errorf("parseSpeed(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

// =============================================================================
// usbDeviceInfo Tests
// =============================================================================

func TestUSBDeviceInfo_hasHIDInterface(t *testing.T) {
	tests := []struct {
		name       string
		interfaces []usbInterfaceInfo
		expected   bool
	}{
		{
			name:       "no interfaces",
			interfaces: nil,
			expected:   false,
		},
		{
			name: "no HID interface",
			interfaces: []usbInterfaceInfo{
				{class: 0x08}, // Mass storage
				{class: 0x02}, // CDC
			},
			expected: false,
		},
		{
			name: "has HID interface",
			interfaces: []usbInterfaceInfo{
				{class: 0x03}, // HID
			},
			expected: true,
		},
		{
			name: "mixed interfaces with HID",
			interfaces: []usbInterfaceInfo{
				{class: 0x08},
				{class: 0x03}, // HID
				{class: 0x02},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := usbDeviceInfo{interfaces: tt.interfaces}
			got := info.hasHIDInterface()
			if got != tt.expected {
				t.Errorf("hasHIDInterface() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUSBDeviceInfo_getHIDInterfaces(t *testing.T) {
	info := usbDeviceInfo{
		interfaces: []usbInterfaceInfo{
			{number: 0, class: 0x08},
			{number: 1, class: 0x03, subclass: 0x01, protocol: 0x01}, // HID keyboard
			{number: 2, class: 0x03, subclass: 0x01, protocol: 0x02}, // HID mouse
			{number: 3, class: 0x02},
		},
	}

	hidInterfaces := info.getHIDInterfaces()
	if len(hidInterfaces) != 2 {
		t.Fatalf("len(getHIDInterfaces()) = %d, want 2", len(hidInterfaces))
	}

	if hidInterfaces[0].number != 1 {
		t.Errorf("hidInterfaces[0].number = %d, want 1", hidInterfaces[0].number)
	}
	if hidInterfaces[1].number != 2 {
		t.Errorf("hidInterfaces[1].number = %d, want 2", hidInterfaces[1].number)
	}
}

// =============================================================================
// usbInterfaceInfo Tests
// =============================================================================

func TestUSBInterfaceInfo_Fields(t *testing.T) {
	iface := usbInterfaceInfo{
		number:   1,
		class:    0x03,
		subclass: 0x01,
		protocol: 0x02,
	}

	if iface.number != 1 {
		t.Errorf("number = %d, want 1", iface.number)
	}
	if iface.class != 0x03 {
		t.Errorf("class = 0x%02X, want 0x03", iface.class)
	}
	if iface.subclass != 0x01 {
		t.Errorf("subclass = 0x%02X, want 0x01", iface.subclass)
	}
	if iface.protocol != 0x02 {
		t.Errorf("protocol = 0x%02X, want 0x02", iface.protocol)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkFormatPadded(b *testing.B) {
	buf := make([]byte, 10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatPadded(buf, uint8(i%256), 3)
	}
}

func BenchmarkFormatDevfsPath(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = formatDevfsPath(uint8(i%256), uint8((i+1)%256))
	}
}

func BenchmarkParseSpeed(b *testing.B) {
	speeds := []string{"1.5", "12", "480", "5000"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseSpeed(speeds[i%4])
	}
}

func BenchmarkHasHIDInterface(b *testing.B) {
	info := usbDeviceInfo{
		interfaces: []usbInterfaceInfo{
			{class: 0x08},
			{class: 0x03},
			{class: 0x02},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.hasHIDInterface()
	}
}
