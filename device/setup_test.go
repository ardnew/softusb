package device

import (
	"testing"
)

func TestParseSetupPacket(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    SetupPacket
		wantErr bool
	}{
		{
			name: "GET_DESCRIPTOR device",
			data: []byte{0x80, 0x06, 0x00, 0x01, 0x00, 0x00, 0x12, 0x00},
			want: SetupPacket{
				RequestType: 0x80,
				Request:     0x06,
				Value:       0x0100,
				Index:       0x0000,
				Length:      18,
			},
		},
		{
			name: "SET_ADDRESS",
			data: []byte{0x00, 0x05, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: SetupPacket{
				RequestType: 0x00,
				Request:     0x05,
				Value:       5,
				Index:       0,
				Length:      0,
			},
		},
		{
			name: "SET_CONFIGURATION",
			data: []byte{0x00, 0x09, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: SetupPacket{
				RequestType: 0x00,
				Request:     0x09,
				Value:       1,
				Index:       0,
				Length:      0,
			},
		},
		{
			name:    "too short",
			data:    []byte{0x80, 0x06, 0x00},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got SetupPacket
			err := ParseSetupPacket(tt.data, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSetupPacket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.RequestType != tt.want.RequestType {
				t.Errorf("RequestType = 0x%02X, want 0x%02X", got.RequestType, tt.want.RequestType)
			}
			if got.Request != tt.want.Request {
				t.Errorf("Request = 0x%02X, want 0x%02X", got.Request, tt.want.Request)
			}
			if got.Value != tt.want.Value {
				t.Errorf("Value = 0x%04X, want 0x%04X", got.Value, tt.want.Value)
			}
			if got.Index != tt.want.Index {
				t.Errorf("Index = 0x%04X, want 0x%04X", got.Index, tt.want.Index)
			}
			if got.Length != tt.want.Length {
				t.Errorf("Length = %d, want %d", got.Length, tt.want.Length)
			}
		})
	}
}

func TestSetupPacketMarshalTo(t *testing.T) {
	pkt := SetupPacket{
		RequestType: 0x80,
		Request:     0x06,
		Value:       0x0100,
		Index:       0x0000,
		Length:      18,
	}

	var buf [SetupPacketSize]byte
	n := pkt.MarshalTo(buf[:])
	if n != SetupPacketSize {
		t.Errorf("MarshalTo() length = %d, want %d", n, SetupPacketSize)
	}

	// Parse it back
	var parsed SetupPacket
	err := ParseSetupPacket(buf[:], &parsed)
	if err != nil {
		t.Fatalf("ParseSetupPacket() error = %v", err)
	}
	if parsed != pkt {
		t.Errorf("round-trip failed: got %+v, want %+v", parsed, pkt)
	}
}

func TestSetupPacketDirection(t *testing.T) {
	tests := []struct {
		name          string
		requestType   uint8
		wantDirection uint8
		wantD2H       bool
		wantH2D       bool
	}{
		{"device-to-host", 0x80, RequestDirectionDeviceToHost, true, false},
		{"host-to-device", 0x00, RequestDirectionHostToDevice, false, true},
		{"class IN", 0xA1, RequestDirectionDeviceToHost, true, false},
		{"vendor OUT", 0x40, RequestDirectionHostToDevice, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := &SetupPacket{RequestType: tt.requestType}
			if got := pkt.Direction(); got != tt.wantDirection {
				t.Errorf("Direction() = 0x%02X, want 0x%02X", got, tt.wantDirection)
			}
			if got := pkt.IsDeviceToHost(); got != tt.wantD2H {
				t.Errorf("IsDeviceToHost() = %v, want %v", got, tt.wantD2H)
			}
			if got := pkt.IsHostToDevice(); got != tt.wantH2D {
				t.Errorf("IsHostToDevice() = %v, want %v", got, tt.wantH2D)
			}
		})
	}
}

func TestSetupPacketType(t *testing.T) {
	tests := []struct {
		name        string
		requestType uint8
		wantType    uint8
		wantStd     bool
		wantClass   bool
		wantVendor  bool
	}{
		{"standard", 0x00, RequestTypeStandard, true, false, false},
		{"class", 0x21, RequestTypeClass, false, true, false},
		{"vendor", 0x40, RequestTypeVendor, false, false, true},
		{"class IN", 0xA1, RequestTypeClass, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := &SetupPacket{RequestType: tt.requestType}
			if got := pkt.Type(); got != tt.wantType {
				t.Errorf("Type() = 0x%02X, want 0x%02X", got, tt.wantType)
			}
			if got := pkt.IsStandard(); got != tt.wantStd {
				t.Errorf("IsStandard() = %v, want %v", got, tt.wantStd)
			}
			if got := pkt.IsClass(); got != tt.wantClass {
				t.Errorf("IsClass() = %v, want %v", got, tt.wantClass)
			}
			if got := pkt.IsVendor(); got != tt.wantVendor {
				t.Errorf("IsVendor() = %v, want %v", got, tt.wantVendor)
			}
		})
	}
}

func TestSetupPacketRecipient(t *testing.T) {
	tests := []struct {
		name        string
		requestType uint8
		wantRecip   uint8
		wantDevice  bool
		wantIface   bool
		wantEP      bool
	}{
		{"device", 0x00, RequestRecipientDevice, true, false, false},
		{"interface", 0x01, RequestRecipientInterface, false, true, false},
		{"endpoint", 0x02, RequestRecipientEndpoint, false, false, true},
		{"class interface", 0x21, RequestRecipientInterface, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := &SetupPacket{RequestType: tt.requestType}
			if got := pkt.Recipient(); got != tt.wantRecip {
				t.Errorf("Recipient() = 0x%02X, want 0x%02X", got, tt.wantRecip)
			}
			if got := pkt.IsDeviceRecipient(); got != tt.wantDevice {
				t.Errorf("IsDeviceRecipient() = %v, want %v", got, tt.wantDevice)
			}
			if got := pkt.IsInterfaceRecipient(); got != tt.wantIface {
				t.Errorf("IsInterfaceRecipient() = %v, want %v", got, tt.wantIface)
			}
			if got := pkt.IsEndpointRecipient(); got != tt.wantEP {
				t.Errorf("IsEndpointRecipient() = %v, want %v", got, tt.wantEP)
			}
		})
	}
}

func TestSetupPacketDescriptorFields(t *testing.T) {
	pkt := &SetupPacket{
		Value: 0x0301, // String descriptor, index 1
	}

	if got := pkt.DescriptorType(); got != 0x03 {
		t.Errorf("DescriptorType() = 0x%02X, want 0x03", got)
	}
	if got := pkt.DescriptorIndex(); got != 0x01 {
		t.Errorf("DescriptorIndex() = 0x%02X, want 0x01", got)
	}
}

func TestSetupPacketIndexFields(t *testing.T) {
	pkt := &SetupPacket{
		Index: 0x0081, // Endpoint 1 IN
	}

	if got := pkt.InterfaceNumber(); got != 0x81 {
		t.Errorf("InterfaceNumber() = 0x%02X, want 0x81", got)
	}
	if got := pkt.EndpointAddress(); got != 0x81 {
		t.Errorf("EndpointAddress() = 0x%02X, want 0x81", got)
	}
}

func TestSetupPacketString(t *testing.T) {
	pkt := &SetupPacket{
		RequestType: 0x80,
		Request:     0x06,
		Value:       0x0100,
		Index:       0x0000,
		Length:      18,
	}

	s := pkt.String()
	if s == "" {
		t.Error("String() returned empty string")
	}
}

func TestGetDescriptorSetup(t *testing.T) {
	var pkt SetupPacket
	GetDescriptorSetup(&pkt, DescriptorTypeDevice, 0, 18)

	if !pkt.IsDeviceToHost() {
		t.Error("should be device-to-host")
	}
	if !pkt.IsStandard() {
		t.Error("should be standard request")
	}
	if pkt.Request != RequestGetDescriptor {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestGetDescriptor)
	}
	if pkt.DescriptorType() != DescriptorTypeDevice {
		t.Errorf("DescriptorType() = 0x%02X, want 0x%02X", pkt.DescriptorType(), DescriptorTypeDevice)
	}
	if pkt.Length != 18 {
		t.Errorf("Length = %d, want 18", pkt.Length)
	}
}

func TestGetSetAddressSetup(t *testing.T) {
	var pkt SetupPacket
	GetSetAddressSetup(&pkt, 5)

	if !pkt.IsHostToDevice() {
		t.Error("should be host-to-device")
	}
	if pkt.Request != RequestSetAddress {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestSetAddress)
	}
	if pkt.Value != 5 {
		t.Errorf("Value = %d, want 5", pkt.Value)
	}
}

func TestGetSetConfigurationSetup(t *testing.T) {
	var pkt SetupPacket
	GetSetConfigurationSetup(&pkt, 1)

	if !pkt.IsHostToDevice() {
		t.Error("should be host-to-device")
	}
	if pkt.Request != RequestSetConfiguration {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestSetConfiguration)
	}
	if pkt.Value != 1 {
		t.Errorf("Value = %d, want 1", pkt.Value)
	}
}

func TestGetConfigurationSetup(t *testing.T) {
	var pkt SetupPacket
	GetConfigurationSetup(&pkt)

	if !pkt.IsDeviceToHost() {
		t.Error("should be device-to-host")
	}
	if pkt.Request != RequestGetConfiguration {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestGetConfiguration)
	}
	if pkt.Length != 1 {
		t.Errorf("Length = %d, want 1", pkt.Length)
	}
}

func TestGetStatusSetup(t *testing.T) {
	var pkt SetupPacket
	GetStatusSetup(&pkt, RequestRecipientDevice, 0)

	if !pkt.IsDeviceToHost() {
		t.Error("should be device-to-host")
	}
	if pkt.Request != RequestGetStatus {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestGetStatus)
	}
	if pkt.Length != 2 {
		t.Errorf("Length = %d, want 2", pkt.Length)
	}
}

func TestGetSetFeatureSetup(t *testing.T) {
	var pkt SetupPacket
	GetSetFeatureSetup(&pkt, RequestRecipientEndpoint, FeatureEndpointHalt, 0x81)

	if !pkt.IsHostToDevice() {
		t.Error("should be host-to-device")
	}
	if pkt.Request != RequestSetFeature {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestSetFeature)
	}
	if pkt.Value != FeatureEndpointHalt {
		t.Errorf("Value = %d, want %d", pkt.Value, FeatureEndpointHalt)
	}
	if pkt.Index != 0x81 {
		t.Errorf("Index = 0x%04X, want 0x0081", pkt.Index)
	}
}

func TestGetClearFeatureSetup(t *testing.T) {
	var pkt SetupPacket
	GetClearFeatureSetup(&pkt, RequestRecipientEndpoint, FeatureEndpointHalt, 0x81)

	if pkt.Request != RequestClearFeature {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestClearFeature)
	}
}

func TestGetSetInterfaceSetup(t *testing.T) {
	var pkt SetupPacket
	GetSetInterfaceSetup(&pkt, 0, 1)

	if !pkt.IsInterfaceRecipient() {
		t.Error("should be interface recipient")
	}
	if pkt.Request != RequestSetInterface {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestSetInterface)
	}
	if pkt.Value != 1 {
		t.Errorf("Value = %d, want 1", pkt.Value)
	}
	if pkt.Index != 0 {
		t.Errorf("Index = %d, want 0", pkt.Index)
	}
}

func TestGetInterfaceSetup(t *testing.T) {
	var pkt SetupPacket
	GetInterfaceSetup(&pkt, 2)

	if !pkt.IsDeviceToHost() {
		t.Error("should be device-to-host")
	}
	if pkt.Request != RequestGetInterface {
		t.Errorf("Request = 0x%02X, want 0x%02X", pkt.Request, RequestGetInterface)
	}
	if pkt.Index != 2 {
		t.Errorf("Index = %d, want 2", pkt.Index)
	}
	if pkt.Length != 1 {
		t.Errorf("Length = %d, want 1", pkt.Length)
	}
}
