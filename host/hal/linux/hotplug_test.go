//go:build linux

package linux

import (
	"testing"
)

// =============================================================================
// uevent Parsing Tests
// =============================================================================

func TestParseUEvent_Add(t *testing.T) {
	// Simulated uevent message for device add
	data := []byte(
		"add@/devices/pci0000:00/0000:00:14.0/usb1/1-1\x00" +
			"ACTION=add\x00" +
			"DEVPATH=/devices/pci0000:00/0000:00:14.0/usb1/1-1\x00" +
			"SUBSYSTEM=usb\x00" +
			"DEVTYPE=usb_device\x00" +
			"BUSNUM=001\x00" +
			"DEVNUM=002\x00",
	)

	evt := parseUEvent(data)

	if evt.action != ueventAdd {
		t.Errorf("action = %d, want ueventAdd (%d)", evt.action, ueventAdd)
	}
	if evt.devpath != "/devices/pci0000:00/0000:00:14.0/usb1/1-1" {
		t.Errorf("devpath = %q, unexpected value", evt.devpath)
	}
	if evt.subsystem != "usb" {
		t.Errorf("subsystem = %q, want %q", evt.subsystem, "usb")
	}
	if evt.devtype != "usb_device" {
		t.Errorf("devtype = %q, want %q", evt.devtype, "usb_device")
	}
	if evt.busnum != "001" {
		t.Errorf("busnum = %q, want %q", evt.busnum, "001")
	}
	if evt.devnum != "002" {
		t.Errorf("devnum = %q, want %q", evt.devnum, "002")
	}
}

func TestParseUEvent_Remove(t *testing.T) {
	data := []byte(
		"remove@/devices/pci0000:00/0000:00:14.0/usb1/1-1\x00" +
			"ACTION=remove\x00" +
			"DEVPATH=/devices/pci0000:00/0000:00:14.0/usb1/1-1\x00" +
			"SUBSYSTEM=usb\x00" +
			"DEVTYPE=usb_device\x00",
	)

	evt := parseUEvent(data)

	if evt.action != ueventRemove {
		t.Errorf("action = %d, want ueventRemove (%d)", evt.action, ueventRemove)
	}
}

func TestParseUEvent_Change(t *testing.T) {
	data := []byte(
		"ACTION=change\x00" +
			"DEVPATH=/devices/pci0000:00/usb1/1-1\x00" +
			"SUBSYSTEM=usb\x00",
	)

	evt := parseUEvent(data)

	if evt.action != ueventChange {
		t.Errorf("action = %d, want ueventChange (%d)", evt.action, ueventChange)
	}
}

func TestParseUEvent_Bind(t *testing.T) {
	data := []byte(
		"bind@/devices/pci0000:00/usb1/1-1:1.0\x00" +
			"ACTION=bind\x00" +
			"SUBSYSTEM=usb\x00",
	)

	evt := parseUEvent(data)

	if evt.action != ueventBind {
		t.Errorf("action = %d, want ueventBind (%d)", evt.action, ueventBind)
	}
}

func TestParseUEvent_Unbind(t *testing.T) {
	data := []byte(
		"unbind@/devices/pci0000:00/usb1/1-1:1.0\x00" +
			"ACTION=unbind\x00" +
			"SUBSYSTEM=usb\x00",
	)

	evt := parseUEvent(data)

	if evt.action != ueventUnbind {
		t.Errorf("action = %d, want ueventUnbind (%d)", evt.action, ueventUnbind)
	}
}

func TestParseUEvent_VendorProduct(t *testing.T) {
	data := []byte(
		"ACTION=add\x00" +
			"SUBSYSTEM=usb\x00" +
			"ID_VENDOR_ID=046d\x00" +
			"ID_MODEL_ID=c52b\x00",
	)

	evt := parseUEvent(data)

	if evt.vendorID != "046d" {
		t.Errorf("vendorID = %q, want %q", evt.vendorID, "046d")
	}
	if evt.productID != "c52b" {
		t.Errorf("productID = %q, want %q", evt.productID, "c52b")
	}
}

func TestParseUEvent_Interface(t *testing.T) {
	data := []byte(
		"ACTION=add\x00" +
			"SUBSYSTEM=usb\x00" +
			"INTERFACE=3/1/1\x00",
	)

	evt := parseUEvent(data)

	if evt.interfaceClass != "3/1/1" {
		t.Errorf("interfaceClass = %q, want %q", evt.interfaceClass, "3/1/1")
	}
}

func TestParseUEvent_EmptyData(t *testing.T) {
	evt := parseUEvent([]byte{})

	if evt.action != ueventUnknown {
		t.Errorf("action = %d, want ueventUnknown (%d)", evt.action, ueventUnknown)
	}
	if evt.devpath != "" {
		t.Errorf("devpath should be empty")
	}
}

func TestParseUEvent_OnlyAction(t *testing.T) {
	data := []byte("add@/devices/usb1/1-1\x00")

	evt := parseUEvent(data)

	if evt.action != ueventAdd {
		t.Errorf("action = %d, want ueventAdd (%d)", evt.action, ueventAdd)
	}
	if evt.devpath != "/devices/usb1/1-1" {
		t.Errorf("devpath = %q, want %q", evt.devpath, "/devices/usb1/1-1")
	}
}

// =============================================================================
// ueventAction Tests
// =============================================================================

func TestUeventAction_Values(t *testing.T) {
	if ueventUnknown != 0 {
		t.Errorf("ueventUnknown = %d, want 0", ueventUnknown)
	}
	if ueventAdd != 1 {
		t.Errorf("ueventAdd = %d, want 1", ueventAdd)
	}
	if ueventRemove != 2 {
		t.Errorf("ueventRemove = %d, want 2", ueventRemove)
	}
	if ueventChange != 3 {
		t.Errorf("ueventChange = %d, want 3", ueventChange)
	}
	if ueventBind != 4 {
		t.Errorf("ueventBind = %d, want 4", ueventBind)
	}
	if ueventUnbind != 5 {
		t.Errorf("ueventUnbind = %d, want 5", ueventUnbind)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkParseUEvent_Small(b *testing.B) {
	data := []byte(
		"add@/devices/usb1/1-1\x00" +
			"ACTION=add\x00" +
			"SUBSYSTEM=usb\x00" +
			"DEVTYPE=usb_device\x00",
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseUEvent(data)
	}
}

func BenchmarkParseUEvent_Large(b *testing.B) {
	data := []byte(
		"add@/devices/pci0000:00/0000:00:14.0/usb1/1-1\x00" +
			"ACTION=add\x00" +
			"DEVPATH=/devices/pci0000:00/0000:00:14.0/usb1/1-1\x00" +
			"SUBSYSTEM=usb\x00" +
			"DEVTYPE=usb_device\x00" +
			"BUSNUM=001\x00" +
			"DEVNUM=002\x00" +
			"ID_VENDOR_ID=046d\x00" +
			"ID_MODEL_ID=c52b\x00" +
			"INTERFACE=3/1/1\x00" +
			"SEQNUM=12345\x00",
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseUEvent(data)
	}
}
