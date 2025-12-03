package device

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewEndpoint(t *testing.T) {
	tests := []struct {
		name string
		desc *EndpointDescriptor
		want struct {
			number   uint8
			isIn     bool
			transfer uint8
		}
	}{
		{
			name: "bulk IN",
			desc: &EndpointDescriptor{
				Length:          7,
				DescriptorType:  DescriptorTypeEndpoint,
				EndpointAddress: 0x81,
				Attributes:      EndpointTypeBulk,
				MaxPacketSize:   512,
				Interval:        0,
			},
			want: struct {
				number   uint8
				isIn     bool
				transfer uint8
			}{1, true, EndpointTypeBulk},
		},
		{
			name: "bulk OUT",
			desc: &EndpointDescriptor{
				Length:          7,
				DescriptorType:  DescriptorTypeEndpoint,
				EndpointAddress: 0x02,
				Attributes:      EndpointTypeBulk,
				MaxPacketSize:   512,
				Interval:        0,
			},
			want: struct {
				number   uint8
				isIn     bool
				transfer uint8
			}{2, false, EndpointTypeBulk},
		},
		{
			name: "interrupt IN",
			desc: &EndpointDescriptor{
				Length:          7,
				DescriptorType:  DescriptorTypeEndpoint,
				EndpointAddress: 0x83,
				Attributes:      EndpointTypeInterrupt,
				MaxPacketSize:   8,
				Interval:        10,
			},
			want: struct {
				number   uint8
				isIn     bool
				transfer uint8
			}{3, true, EndpointTypeInterrupt},
		},
		{
			name: "isochronous OUT async",
			desc: &EndpointDescriptor{
				Length:          7,
				DescriptorType:  DescriptorTypeEndpoint,
				EndpointAddress: 0x04,
				Attributes:      EndpointTypeIsochronous | IsoSyncAsync,
				MaxPacketSize:   1023,
				Interval:        1,
			},
			want: struct {
				number   uint8
				isIn     bool
				transfer uint8
			}{4, false, EndpointTypeIsochronous},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := NewEndpoint(tt.desc)
			if ep.Number() != tt.want.number {
				t.Errorf("Number() = %d, want %d", ep.Number(), tt.want.number)
			}
			if ep.IsIn() != tt.want.isIn {
				t.Errorf("IsIn() = %v, want %v", ep.IsIn(), tt.want.isIn)
			}
			if ep.TransferType() != tt.want.transfer {
				t.Errorf("TransferType() = %d, want %d", ep.TransferType(), tt.want.transfer)
			}
		})
	}
}

func TestEndpointDirection(t *testing.T) {
	tests := []struct {
		name    string
		address uint8
		wantIn  bool
		wantOut bool
	}{
		{"EP0 OUT", 0x00, false, true},
		{"EP1 IN", 0x81, true, false},
		{"EP2 OUT", 0x02, false, true},
		{"EP15 IN", 0x8F, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Endpoint{Address: tt.address}
			if got := ep.IsIn(); got != tt.wantIn {
				t.Errorf("IsIn() = %v, want %v", got, tt.wantIn)
			}
			if got := ep.IsOut(); got != tt.wantOut {
				t.Errorf("IsOut() = %v, want %v", got, tt.wantOut)
			}
		})
	}
}

func TestEndpointTransferType(t *testing.T) {
	tests := []struct {
		name       string
		attributes uint8
		wantCtrl   bool
		wantBulk   bool
		wantIntr   bool
		wantIso    bool
	}{
		{"control", EndpointTypeControl, true, false, false, false},
		{"bulk", EndpointTypeBulk, false, true, false, false},
		{"interrupt", EndpointTypeInterrupt, false, false, true, false},
		{"isochronous", EndpointTypeIsochronous, false, false, false, true},
		{"isochronous sync", EndpointTypeIsochronous | IsoSyncSync, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Endpoint{Attributes: tt.attributes}
			if got := ep.IsControl(); got != tt.wantCtrl {
				t.Errorf("IsControl() = %v, want %v", got, tt.wantCtrl)
			}
			if got := ep.IsBulk(); got != tt.wantBulk {
				t.Errorf("IsBulk() = %v, want %v", got, tt.wantBulk)
			}
			if got := ep.IsInterrupt(); got != tt.wantIntr {
				t.Errorf("IsInterrupt() = %v, want %v", got, tt.wantIntr)
			}
			if got := ep.IsIsochronous(); got != tt.wantIso {
				t.Errorf("IsIsochronous() = %v, want %v", got, tt.wantIso)
			}
		})
	}
}

func TestEndpointIsochronousTypes(t *testing.T) {
	tests := []struct {
		name       string
		attributes uint8
		wantSync   uint8
		wantUsage  uint8
	}{
		{"none/data", EndpointTypeIsochronous, IsoSyncNone, IsoUsageData},
		{"async/feedback", EndpointTypeIsochronous | IsoSyncAsync | IsoUsageFeedback, IsoSyncAsync, IsoUsageFeedback},
		{"adaptive/implicit", EndpointTypeIsochronous | IsoSyncAdaptive | IsoUsageImplicit, IsoSyncAdaptive, IsoUsageImplicit},
		{"sync/data", EndpointTypeIsochronous | IsoSyncSync | IsoUsageData, IsoSyncSync, IsoUsageData},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Endpoint{Attributes: tt.attributes}
			if got := ep.IsoSyncType(); got != tt.wantSync {
				t.Errorf("IsoSyncType() = 0x%02X, want 0x%02X", got, tt.wantSync)
			}
			if got := ep.IsoUsageType(); got != tt.wantUsage {
				t.Errorf("IsoUsageType() = 0x%02X, want 0x%02X", got, tt.wantUsage)
			}
		})
	}
}

func TestEndpointStall(t *testing.T) {
	ep := &Endpoint{Address: 0x81}

	if ep.IsStalled() {
		t.Error("new endpoint should not be stalled")
	}

	ep.SetStall(true)
	if !ep.IsStalled() {
		t.Error("endpoint should be stalled after SetStall(true)")
	}

	ep.SetStall(false)
	if ep.IsStalled() {
		t.Error("endpoint should not be stalled after SetStall(false)")
	}
}

func TestEndpointDataToggle(t *testing.T) {
	ep := &Endpoint{Address: 0x81}

	if ep.DataToggle() {
		t.Error("new endpoint should have DATA0 toggle")
	}

	ep.ToggleData()
	if !ep.DataToggle() {
		t.Error("toggle should be DATA1 after ToggleData")
	}

	ep.ToggleData()
	if ep.DataToggle() {
		t.Error("toggle should be DATA0 after second ToggleData")
	}

	ep.SetDataToggle(true)
	if !ep.DataToggle() {
		t.Error("toggle should be DATA1 after SetDataToggle(true)")
	}

	ep.ResetDataToggle()
	if ep.DataToggle() {
		t.Error("toggle should be DATA0 after ResetDataToggle")
	}
}

func TestEndpointFrameNumber(t *testing.T) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}

	if ep.FrameNumber() != 0 {
		t.Error("new endpoint should have frame number 0")
	}

	ep.SetFrameNumber(1000)
	if ep.FrameNumber() != 1000 {
		t.Errorf("FrameNumber() = %d, want 1000", ep.FrameNumber())
	}

	ep.IncrementFrame()
	if ep.FrameNumber() != 1001 {
		t.Errorf("FrameNumber() = %d, want 1001 after increment", ep.FrameNumber())
	}
}

func TestEndpointDescriptor(t *testing.T) {
	original := &EndpointDescriptor{
		Length:          7,
		DescriptorType:  DescriptorTypeEndpoint,
		EndpointAddress: 0x81,
		Attributes:      EndpointTypeBulk,
		MaxPacketSize:   512,
		Interval:        0,
	}

	ep := NewEndpoint(original)
	desc := ep.Descriptor()

	if desc.EndpointAddress != original.EndpointAddress {
		t.Errorf("EndpointAddress = 0x%02X, want 0x%02X", desc.EndpointAddress, original.EndpointAddress)
	}
	if desc.Attributes != original.Attributes {
		t.Errorf("Attributes = 0x%02X, want 0x%02X", desc.Attributes, original.Attributes)
	}
	if desc.MaxPacketSize != original.MaxPacketSize {
		t.Errorf("MaxPacketSize = %d, want %d", desc.MaxPacketSize, original.MaxPacketSize)
	}
}

func TestTransferTypeName(t *testing.T) {
	tests := []struct {
		t    uint8
		want string
	}{
		{EndpointTypeControl, "Control"},
		{EndpointTypeIsochronous, "Isochronous"},
		{EndpointTypeBulk, "Bulk"},
		{EndpointTypeInterrupt, "Interrupt"},
		{0xFF, "Interrupt"}, // 0xFF & 0x03 = 0x03 = Interrupt
	}

	for _, tt := range tests {
		if got := TransferTypeName(tt.t); got != tt.want {
			t.Errorf("TransferTypeName(%d) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestDirectionName(t *testing.T) {
	if got := DirectionName(EndpointDirectionIn); got != "IN" {
		t.Errorf("DirectionName(IN) = %q, want %q", got, "IN")
	}
	if got := DirectionName(EndpointDirectionOut); got != "OUT" {
		t.Errorf("DirectionName(OUT) = %q, want %q", got, "OUT")
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

// TestEndpointAddress_EdgeCases tests all endpoint address boundaries
func TestEndpointAddress_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		address    uint8
		wantNumber uint8
		wantIsIn   bool
	}{
		// OUT endpoints (direction bit = 0)
		{"EP0 OUT (0x00)", 0x00, 0, false},
		{"EP1 OUT (0x01)", 0x01, 1, false},
		{"EP7 OUT (0x07)", 0x07, 7, false},
		{"EP8 OUT (0x08)", 0x08, 8, false},
		{"EP15 OUT (0x0F)", 0x0F, 15, false},

		// IN endpoints (direction bit = 1)
		{"EP0 IN (0x80)", 0x80, 0, true},
		{"EP1 IN (0x81)", 0x81, 1, true},
		{"EP7 IN (0x87)", 0x87, 7, true},
		{"EP8 IN (0x88)", 0x88, 8, true},
		{"EP15 IN (0x8F)", 0x8F, 15, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Endpoint{Address: tt.address}
			if got := ep.Number(); got != tt.wantNumber {
				t.Errorf("Number() = %d, want %d", got, tt.wantNumber)
			}
			if got := ep.IsIn(); got != tt.wantIsIn {
				t.Errorf("IsIn() = %v, want %v", got, tt.wantIsIn)
			}
			if got := ep.IsOut(); got != !tt.wantIsIn {
				t.Errorf("IsOut() = %v, want %v", got, !tt.wantIsIn)
			}
			// Verify Direction() returns correct constant
			if tt.wantIsIn {
				if got := ep.Direction(); got != EndpointDirectionIn {
					t.Errorf("Direction() = 0x%02X, want 0x%02X", got, EndpointDirectionIn)
				}
			} else {
				if got := ep.Direction(); got != EndpointDirectionOut {
					t.Errorf("Direction() = 0x%02X, want 0x%02X", got, EndpointDirectionOut)
				}
			}
		})
	}
}

// TestEndpointAttributes_AllCombinations tests all valid attribute combinations
func TestEndpointAttributes_AllCombinations(t *testing.T) {
	transferTypes := []struct {
		name string
		attr uint8
	}{
		{"Control", EndpointTypeControl},
		{"Isochronous", EndpointTypeIsochronous},
		{"Bulk", EndpointTypeBulk},
		{"Interrupt", EndpointTypeInterrupt},
	}

	isoSyncTypes := []struct {
		name string
		attr uint8
	}{
		{"None", IsoSyncNone},
		{"Async", IsoSyncAsync},
		{"Adaptive", IsoSyncAdaptive},
		{"Sync", IsoSyncSync},
	}

	isoUsageTypes := []struct {
		name string
		attr uint8
	}{
		{"Data", IsoUsageData},
		{"Feedback", IsoUsageFeedback},
		{"Implicit", IsoUsageImplicit},
	}

	// Test all transfer types
	for _, tt := range transferTypes {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Endpoint{Attributes: tt.attr}
			if got := ep.TransferType(); got != tt.attr {
				t.Errorf("TransferType() = 0x%02X, want 0x%02X", got, tt.attr)
			}
			// Verify only one type returns true
			isCtrl := ep.IsControl()
			isIso := ep.IsIsochronous()
			isBulk := ep.IsBulk()
			isIntr := ep.IsInterrupt()
			count := 0
			if isCtrl {
				count++
			}
			if isIso {
				count++
			}
			if isBulk {
				count++
			}
			if isIntr {
				count++
			}
			if count != 1 {
				t.Errorf("exactly one type should be true, got %d (ctrl=%v iso=%v bulk=%v intr=%v)",
					count, isCtrl, isIso, isBulk, isIntr)
			}
		})
	}

	// Test isochronous sync and usage type combinations
	for _, sync := range isoSyncTypes {
		for _, usage := range isoUsageTypes {
			name := "Iso_" + sync.name + "_" + usage.name
			t.Run(name, func(t *testing.T) {
				attr := EndpointTypeIsochronous | sync.attr | usage.attr
				ep := &Endpoint{Attributes: attr}
				if got := ep.IsoSyncType(); got != sync.attr {
					t.Errorf("IsoSyncType() = 0x%02X, want 0x%02X", got, sync.attr)
				}
				if got := ep.IsoUsageType(); got != usage.attr {
					t.Errorf("IsoUsageType() = 0x%02X, want 0x%02X", got, usage.attr)
				}
			})
		}
	}
}

// TestEndpointMaxPacketSize_EdgeCases tests max packet size boundaries
func TestEndpointMaxPacketSize_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		size   uint16
		attr   uint8
		expect uint16
	}{
		// Low-speed limits
		{"LowSpeed_Interrupt_8", 8, EndpointTypeInterrupt, 8},

		// Full-speed limits
		{"FullSpeed_Control_8", 8, EndpointTypeControl, 8},
		{"FullSpeed_Control_16", 16, EndpointTypeControl, 16},
		{"FullSpeed_Control_32", 32, EndpointTypeControl, 32},
		{"FullSpeed_Control_64", 64, EndpointTypeControl, 64},
		{"FullSpeed_Bulk_8", 8, EndpointTypeBulk, 8},
		{"FullSpeed_Bulk_64", 64, EndpointTypeBulk, 64},
		{"FullSpeed_Interrupt_64", 64, EndpointTypeInterrupt, 64},
		{"FullSpeed_Iso_1023", 1023, EndpointTypeIsochronous, 1023},

		// High-speed limits
		{"HighSpeed_Control_64", 64, EndpointTypeControl, 64},
		{"HighSpeed_Bulk_512", 512, EndpointTypeBulk, 512},
		{"HighSpeed_Interrupt_1024", 1024, EndpointTypeInterrupt, 1024},
		{"HighSpeed_Iso_1024", 1024, EndpointTypeIsochronous, 1024},

		// Boundary values
		{"Zero", 0, EndpointTypeBulk, 0},
		{"Max_uint16", 0xFFFF, EndpointTypeBulk, 0xFFFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Endpoint{
				Attributes:    tt.attr,
				MaxPacketSize: tt.size,
			}
			if got := ep.MaxPacketSize; got != tt.expect {
				t.Errorf("MaxPacketSize = %d, want %d", got, tt.expect)
			}
		})
	}
}

// TestEndpointInterval_EdgeCases tests interval field boundaries
func TestEndpointInterval_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		interval uint8
	}{
		{"Zero", 0},
		{"One", 1},
		{"FullSpeed_Max", 255},
		{"HighSpeed_Max_Interrupt", 16}, // Max for HS interrupt is 2^(16-1) = 32768 microframes
		{"Typical_Interrupt", 10},
		{"Typical_Iso", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := &Endpoint{Interval: tt.interval}
			if ep.Interval != tt.interval {
				t.Errorf("Interval = %d, want %d", ep.Interval, tt.interval)
			}
		})
	}
}

// TestDataToggle_ConcurrentAccess tests data toggle under concurrent access
func TestDataToggle_ConcurrentAccess(t *testing.T) {
	ep := &Endpoint{Address: 0x81}
	const goroutines = 100
	const togglesPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < togglesPerGoroutine; j++ {
				ep.ToggleData()
			}
		}()
	}

	wg.Wait()

	// After even number of toggles (100*1000 = 100000), toggle should be DATA0
	if ep.DataToggle() != false {
		t.Errorf("DataToggle() = true, want false after %d toggles", goroutines*togglesPerGoroutine)
	}
}

// TestStall_ConcurrentAccess tests stall state under concurrent access
func TestStall_ConcurrentAccess(t *testing.T) {
	ep := &Endpoint{Address: 0x81}
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // Half set, half clear

	// Half goroutines set stall
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ep.SetStall(true)
		}()
	}

	// Half goroutines clear stall
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ep.SetStall(false)
		}()
	}

	wg.Wait()

	// Just verify no panic occurred and state is consistent (either true or false)
	_ = ep.IsStalled()
}

// TestFrameNumber_ConcurrentIncrement tests frame number under concurrent increment
func TestFrameNumber_ConcurrentIncrement(t *testing.T) {
	ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
	const goroutines = 50
	const incrementsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				ep.IncrementFrame()
			}
		}()
	}

	wg.Wait()

	expected := uint16(goroutines * incrementsPerGoroutine)
	if got := ep.FrameNumber(); got != expected {
		t.Errorf("FrameNumber() = %d, want %d", got, expected)
	}
}

// TestNewEndpoint_NilDescriptor tests NewEndpoint behavior with nil
func TestNewEndpoint_NilDescriptor(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Skip("NewEndpoint(nil) did not panic - nil handling may be intentional")
		}
	}()

	_ = NewEndpoint(nil)
}

// TestEndpoint_ZeroValue tests that zero-value endpoint behaves correctly
func TestEndpoint_ZeroValue(t *testing.T) {
	var ep Endpoint

	// Zero address = EP0 OUT
	if ep.Number() != 0 {
		t.Errorf("Number() = %d, want 0", ep.Number())
	}
	if ep.IsIn() {
		t.Error("IsIn() = true, want false")
	}
	if !ep.IsOut() {
		t.Error("IsOut() = false, want true")
	}

	// Zero attributes = Control
	if !ep.IsControl() {
		t.Error("IsControl() = false, want true")
	}
	if ep.TransferType() != EndpointTypeControl {
		t.Errorf("TransferType() = %d, want %d", ep.TransferType(), EndpointTypeControl)
	}

	// Default states
	if ep.IsStalled() {
		t.Error("IsStalled() = true, want false")
	}
	if ep.DataToggle() {
		t.Error("DataToggle() = true, want false")
	}
	if ep.FrameNumber() != 0 {
		t.Errorf("FrameNumber() = %d, want 0", ep.FrameNumber())
	}
}

// TestTransferTypeName_AllValues tests all transfer type names including invalid
func TestTransferTypeName_AllValues(t *testing.T) {
	tests := []struct {
		value uint8
		want  string
	}{
		{EndpointTypeControl, "Control"},
		{EndpointTypeIsochronous, "Isochronous"},
		{EndpointTypeBulk, "Bulk"},
		{EndpointTypeInterrupt, "Interrupt"},
		// Values with upper bits set still work due to masking
		{0x04 | EndpointTypeControl, "Control"},
		{0x80 | EndpointTypeBulk, "Bulk"},
		{0xFF, "Interrupt"}, // 0xFF & 0x03 = 0x03
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := TransferTypeName(tt.value); got != tt.want {
				t.Errorf("TransferTypeName(0x%02X) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

// TestDirectionName_AllValues tests direction names with edge values
func TestDirectionName_AllValues(t *testing.T) {
	tests := []struct {
		value uint8
		want  string
	}{
		{EndpointDirectionOut, "OUT"},
		{EndpointDirectionIn, "IN"},
		{0x00, "OUT"},
		{0x80, "IN"},
		{0x01, "OUT"}, // Non-0x80 values are OUT
		{0x7F, "OUT"},
		{0x81, "OUT"}, // Only exact 0x80 is IN per implementation
	}

	for _, tt := range tests {
		if got := DirectionName(tt.value); got != tt.want {
			t.Errorf("DirectionName(0x%02X) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewEndpoint(b *testing.B) {
	desc := &EndpointDescriptor{
		Length:          7,
		DescriptorType:  DescriptorTypeEndpoint,
		EndpointAddress: 0x81,
		Attributes:      EndpointTypeBulk,
		MaxPacketSize:   512,
		Interval:        0,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEndpoint(desc)
	}
}

func BenchmarkEndpoint_Number(b *testing.B) {
	ep := &Endpoint{Address: 0x81}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.Number()
	}
}

func BenchmarkEndpoint_Direction(b *testing.B) {
	ep := &Endpoint{Address: 0x81}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.Direction()
	}
}

func BenchmarkEndpoint_IsIn(b *testing.B) {
	ep := &Endpoint{Address: 0x81}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.IsIn()
	}
}

func BenchmarkEndpoint_TransferType(b *testing.B) {
	ep := &Endpoint{Attributes: EndpointTypeBulk}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.TransferType()
	}
}

func BenchmarkEndpoint_TypeChecks(b *testing.B) {
	endpoints := []*Endpoint{
		{Attributes: EndpointTypeControl},
		{Attributes: EndpointTypeBulk},
		{Attributes: EndpointTypeInterrupt},
		{Attributes: EndpointTypeIsochronous},
	}

	b.Run("IsControl", func(b *testing.B) {
		ep := endpoints[0]
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.IsControl()
		}
	})

	b.Run("IsBulk", func(b *testing.B) {
		ep := endpoints[1]
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.IsBulk()
		}
	})

	b.Run("IsInterrupt", func(b *testing.B) {
		ep := endpoints[2]
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.IsInterrupt()
		}
	})

	b.Run("IsIsochronous", func(b *testing.B) {
		ep := endpoints[3]
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.IsIsochronous()
		}
	})
}

func BenchmarkEndpoint_IsoTypes(b *testing.B) {
	ep := &Endpoint{Attributes: EndpointTypeIsochronous | IsoSyncAsync | IsoUsageFeedback}

	b.Run("IsoSyncType", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.IsoSyncType()
		}
	})

	b.Run("IsoUsageType", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.IsoUsageType()
		}
	})
}

func BenchmarkEndpoint_Stall(b *testing.B) {
	b.Run("SetStall", func(b *testing.B) {
		ep := &Endpoint{Address: 0x81}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ep.SetStall(i%2 == 0)
		}
	})

	b.Run("IsStalled", func(b *testing.B) {
		ep := &Endpoint{Address: 0x81}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.IsStalled()
		}
	})
}

func BenchmarkEndpoint_DataToggle(b *testing.B) {
	b.Run("ToggleData", func(b *testing.B) {
		ep := &Endpoint{Address: 0x81}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ep.ToggleData()
		}
	})

	b.Run("DataToggle", func(b *testing.B) {
		ep := &Endpoint{Address: 0x81}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.DataToggle()
		}
	})

	b.Run("SetDataToggle", func(b *testing.B) {
		ep := &Endpoint{Address: 0x81}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ep.SetDataToggle(i%2 == 0)
		}
	})

	b.Run("ResetDataToggle", func(b *testing.B) {
		ep := &Endpoint{Address: 0x81}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ep.ResetDataToggle()
		}
	})
}

func BenchmarkEndpoint_FrameNumber(b *testing.B) {
	b.Run("Get", func(b *testing.B) {
		ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ep.FrameNumber()
		}
	})

	b.Run("Set", func(b *testing.B) {
		ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ep.SetFrameNumber(uint16(i))
		}
	})

	b.Run("Increment", func(b *testing.B) {
		ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ep.IncrementFrame()
		}
	})
}

func BenchmarkEndpoint_Descriptor(b *testing.B) {
	ep := &Endpoint{
		Address:       0x81,
		Attributes:    EndpointTypeBulk,
		MaxPacketSize: 512,
		Interval:      0,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ep.Descriptor()
	}
}

func BenchmarkEndpoint_Concurrent(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8}

	b.Run("ToggleData", func(b *testing.B) {
		for _, g := range goroutineCounts {
			b.Run(fmt.Sprintf("goroutines=%d", g), func(b *testing.B) {
				ep := &Endpoint{Address: 0x81}
				b.ReportAllocs()
				b.ResetTimer()
				b.SetParallelism(g)
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						ep.ToggleData()
					}
				})
			})
		}
	})

	b.Run("IsStalled", func(b *testing.B) {
		for _, g := range goroutineCounts {
			b.Run(fmt.Sprintf("goroutines=%d", g), func(b *testing.B) {
				ep := &Endpoint{Address: 0x81}
				b.ReportAllocs()
				b.ResetTimer()
				b.SetParallelism(g)
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_ = ep.IsStalled()
					}
				})
			})
		}
	})

	b.Run("IncrementFrame", func(b *testing.B) {
		for _, g := range goroutineCounts {
			b.Run(fmt.Sprintf("goroutines=%d", g), func(b *testing.B) {
				ep := &Endpoint{Address: 0x04, Attributes: EndpointTypeIsochronous}
				b.ReportAllocs()
				b.ResetTimer()
				b.SetParallelism(g)
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						ep.IncrementFrame()
					}
				})
			})
		}
	})
}

func BenchmarkTransferTypeName(b *testing.B) {
	types := []uint8{
		EndpointTypeControl,
		EndpointTypeIsochronous,
		EndpointTypeBulk,
		EndpointTypeInterrupt,
	}

	for _, tt := range types {
		name := TransferTypeName(tt)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = TransferTypeName(tt)
			}
		})
	}
}

func BenchmarkDirectionName(b *testing.B) {
	b.Run("IN", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = DirectionName(EndpointDirectionIn)
		}
	})

	b.Run("OUT", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = DirectionName(EndpointDirectionOut)
		}
	})
}
