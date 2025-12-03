package linux

import (
	"testing"
)

// =============================================================================
// Limit Constant Tests
// =============================================================================

func TestMaxDevices(t *testing.T) {
	if MaxDevices < 1 {
		t.Errorf("MaxDevices = %d, should be at least 1", MaxDevices)
	}
	if MaxDevices > 127 {
		t.Errorf("MaxDevices = %d, should not exceed 127 (USB limit)", MaxDevices)
	}
}

func TestMaxEndpointsPerDevice(t *testing.T) {
	// USB 2.0 allows 16 endpoints per direction (32 total)
	if MaxEndpointsPerDevice != 32 {
		t.Errorf("MaxEndpointsPerDevice = %d, want 32", MaxEndpointsPerDevice)
	}
}

func TestMaxURBsPerEndpoint(t *testing.T) {
	if MaxURBsPerEndpoint < 1 {
		t.Errorf("MaxURBsPerEndpoint = %d, should be at least 1", MaxURBsPerEndpoint)
	}
}

func TestURBBufferSize(t *testing.T) {
	if URBBufferSize < 64 {
		t.Errorf("URBBufferSize = %d, should be at least 64 bytes", URBBufferSize)
	}
}

func TestMaxControlTransferSize(t *testing.T) {
	// Control transfers can have up to 4096 bytes in data phase
	if MaxControlTransferSize < 4096 {
		t.Errorf("MaxControlTransferSize = %d, should be at least 4096", MaxControlTransferSize)
	}
}

// =============================================================================
// Path Constant Tests
// =============================================================================

func TestSysfsUSBPath(t *testing.T) {
	expected := "/sys/bus/usb/devices"
	if SysfsUSBPath != expected {
		t.Errorf("SysfsUSBPath = %q, want %q", SysfsUSBPath, expected)
	}
}

func TestDevfsUSBPath(t *testing.T) {
	expected := "/dev/bus/usb"
	if DevfsUSBPath != expected {
		t.Errorf("DevfsUSBPath = %q, want %q", DevfsUSBPath, expected)
	}
}

// =============================================================================
// Errno Constant Tests
// =============================================================================

func TestErrnoConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"EPERM", EPERM, 1},
		{"ENOENT", ENOENT, 2},
		{"EIO", EIO, 5},
		{"ENXIO", ENXIO, 6},
		{"EBADF", EBADF, 9},
		{"EAGAIN", EAGAIN, 11},
		{"ENOMEM", ENOMEM, 12},
		{"EACCES", EACCES, 13},
		{"EFAULT", EFAULT, 14},
		{"EBUSY", EBUSY, 16},
		{"ENODEV", ENODEV, 19},
		{"EINVAL", EINVAL, 22},
		{"ENOSPC", ENOSPC, 28},
		{"EPIPE", EPIPE, 32},
		{"ENODATA", ENODATA, 61},
		{"ETIME", ETIME, 62},
		{"ENOSR", ENOSR, 63},
		{"EPROTO", EPROTO, 71},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// =============================================================================
// Speed Constant Tests
// =============================================================================

func TestSpeedConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"SpeedUnknown", SpeedUnknown, 0},
		{"SpeedLow", SpeedLow, 1},
		{"SpeedFull", SpeedFull, 2},
		{"SpeedHigh", SpeedHigh, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// =============================================================================
// URB Type Constant Tests
// =============================================================================

func TestURBTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"URBTypeISO", URBTypeISO, 0},
		{"URBTypeInterrupt", URBTypeInterrupt, 1},
		{"URBTypeControl", URBTypeControl, 2},
		{"URBTypeBulk", URBTypeBulk, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// =============================================================================
// URB Flag Constant Tests
// =============================================================================

func TestURBFlagConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"URBShortNotOK", URBShortNotOK, 0x01},
		{"URBISOAsap", URBISOAsap, 0x02},
		{"URBNoTransferDMA", URBNoTransferDMA, 0x04},
		{"URBNoFSBR", URBNoFSBR, 0x20},
		{"URBZeroPacket", URBZeroPacket, 0x40},
		{"URBNoInterrupt", URBNoInterrupt, 0x80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = 0x%02X, want 0x%02X", tt.name, tt.value, tt.expected)
			}
		})
	}
}

// =============================================================================
// HID Class Constant Tests
// =============================================================================

func TestHIDConstants(t *testing.T) {
	if USBClassHID != 0x03 {
		t.Errorf("USBClassHID = 0x%02X, want 0x03", USBClassHID)
	}
	if HIDSubclassNone != 0x00 {
		t.Errorf("HIDSubclassNone = 0x%02X, want 0x00", HIDSubclassNone)
	}
	if HIDSubclassBoot != 0x01 {
		t.Errorf("HIDSubclassBoot = 0x%02X, want 0x01", HIDSubclassBoot)
	}
	if HIDProtocolNone != 0x00 {
		t.Errorf("HIDProtocolNone = 0x%02X, want 0x00", HIDProtocolNone)
	}
	if HIDProtocolKeyboard != 0x01 {
		t.Errorf("HIDProtocolKeyboard = 0x%02X, want 0x01", HIDProtocolKeyboard)
	}
	if HIDProtocolMouse != 0x02 {
		t.Errorf("HIDProtocolMouse = 0x%02X, want 0x02", HIDProtocolMouse)
	}
}

// =============================================================================
// Netlink Constant Tests
// =============================================================================

func TestNetlinkConstants(t *testing.T) {
	// NETLINK_KOBJECT_UEVENT is 15
	if NetlinkKObjectUEvent != 15 {
		t.Errorf("NetlinkKObjectUEvent = %d, want 15", NetlinkKObjectUEvent)
	}
	if UEventBufferSize < 1024 {
		t.Errorf("UEventBufferSize = %d, should be at least 1024", UEventBufferSize)
	}
}

// =============================================================================
// Epoll Constant Tests
// =============================================================================

func TestEpollConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    uint32
		expected uint32
	}{
		{"EPOLLIN", EPOLLIN, 0x001},
		{"EPOLLOUT", EPOLLOUT, 0x004},
		{"EPOLLERR", EPOLLERR, 0x008},
		{"EPOLLHUP", EPOLLHUP, 0x010},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = 0x%03X, want 0x%03X", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestMaxEpollEvents(t *testing.T) {
	if MaxEpollEvents < 1 {
		t.Errorf("MaxEpollEvents = %d, should be at least 1", MaxEpollEvents)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkConstants(b *testing.B) {
	// This benchmark ensures constants are inlined efficiently
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MaxDevices
		_ = URBBufferSize
		_ = ENODEV
		_ = EPOLLIN
	}
}
