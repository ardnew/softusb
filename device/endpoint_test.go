package device

import "testing"

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
