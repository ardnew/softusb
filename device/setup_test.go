package device

import (
	"sync"
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

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestParseSetupPacket_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    SetupPacket
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "exactly 7 bytes (one short)",
			data:    []byte{0x80, 0x06, 0x00, 0x01, 0x00, 0x00, 0x12},
			wantErr: true,
		},
		{
			name: "exactly 8 bytes",
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
			name: "more than 8 bytes (extra ignored)",
			data: []byte{0x80, 0x06, 0x00, 0x01, 0x00, 0x00, 0x12, 0x00, 0xFF, 0xFF},
			want: SetupPacket{
				RequestType: 0x80,
				Request:     0x06,
				Value:       0x0100,
				Index:       0x0000,
				Length:      18,
			},
		},
		{
			name: "max wValue (0xFFFF)",
			data: []byte{0x80, 0x06, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00},
			want: SetupPacket{
				RequestType: 0x80,
				Request:     0x06,
				Value:       0xFFFF,
				Index:       0x0000,
				Length:      0,
			},
		},
		{
			name: "max wIndex (0xFFFF)",
			data: []byte{0x80, 0x06, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00},
			want: SetupPacket{
				RequestType: 0x80,
				Request:     0x06,
				Value:       0x0000,
				Index:       0xFFFF,
				Length:      0,
			},
		},
		{
			name: "max wLength (0xFFFF)",
			data: []byte{0x80, 0x06, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF},
			want: SetupPacket{
				RequestType: 0x80,
				Request:     0x06,
				Value:       0x0000,
				Index:       0x0000,
				Length:      0xFFFF,
			},
		},
		{
			name: "all max values",
			data: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			want: SetupPacket{
				RequestType: 0xFF,
				Request:     0xFF,
				Value:       0xFFFF,
				Index:       0xFFFF,
				Length:      0xFFFF,
			},
		},
		{
			name: "all zeros",
			data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want: SetupPacket{
				RequestType: 0x00,
				Request:     0x00,
				Value:       0x0000,
				Index:       0x0000,
				Length:      0,
			},
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
			if got != tt.want {
				t.Errorf("ParseSetupPacket() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSetupPacketMarshalTo_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pkt     SetupPacket
		bufSize int
		wantN   int
	}{
		{
			name:    "buffer too small (0 bytes)",
			pkt:     SetupPacket{RequestType: 0x80},
			bufSize: 0,
			wantN:   0,
		},
		{
			name:    "buffer too small (7 bytes)",
			pkt:     SetupPacket{RequestType: 0x80},
			bufSize: 7,
			wantN:   0,
		},
		{
			name:    "buffer exactly 8 bytes",
			pkt:     SetupPacket{RequestType: 0x80},
			bufSize: 8,
			wantN:   8,
		},
		{
			name:    "buffer larger than needed",
			pkt:     SetupPacket{RequestType: 0x80},
			bufSize: 64,
			wantN:   8,
		},
		{
			name: "max values",
			pkt: SetupPacket{
				RequestType: 0xFF,
				Request:     0xFF,
				Value:       0xFFFF,
				Index:       0xFFFF,
				Length:      0xFFFF,
			},
			bufSize: 8,
			wantN:   8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufSize)
			n := tt.pkt.MarshalTo(buf)
			if n != tt.wantN {
				t.Errorf("MarshalTo() = %d, want %d", n, tt.wantN)
			}

			// Verify round-trip for successful marshals
			if n == SetupPacketSize {
				var parsed SetupPacket
				if err := ParseSetupPacket(buf, &parsed); err != nil {
					t.Errorf("round-trip ParseSetupPacket() error = %v", err)
				} else if parsed != tt.pkt {
					t.Errorf("round-trip failed: got %+v, want %+v", parsed, tt.pkt)
				}
			}
		})
	}
}

func TestSetupPacket_AllDirectionTypeCombinations(t *testing.T) {
	// Test all 16 direction/type/recipient combinations
	directions := []struct {
		name  string
		value uint8
		isD2H bool
	}{
		{"H2D", RequestDirectionHostToDevice, false},
		{"D2H", RequestDirectionDeviceToHost, true},
	}

	types := []struct {
		name     string
		value    uint8
		isStd    bool
		isClass  bool
		isVendor bool
	}{
		{"Standard", RequestTypeStandard, true, false, false},
		{"Class", RequestTypeClass, false, true, false},
		{"Vendor", RequestTypeVendor, false, false, true},
		{"Reserved", 0x60, false, false, false}, // Reserved type
	}

	recipients := []struct {
		name    string
		value   uint8
		isDev   bool
		isIface bool
		isEP    bool
	}{
		{"Device", RequestRecipientDevice, true, false, false},
		{"Interface", RequestRecipientInterface, false, true, false},
		{"Endpoint", RequestRecipientEndpoint, false, false, true},
		{"Other", RequestRecipientOther, false, false, false},
	}

	for _, dir := range directions {
		for _, typ := range types {
			for _, recip := range recipients {
				name := dir.name + "_" + typ.name + "_" + recip.name
				requestType := dir.value | typ.value | recip.value

				t.Run(name, func(t *testing.T) {
					pkt := &SetupPacket{RequestType: requestType}

					if got := pkt.IsDeviceToHost(); got != dir.isD2H {
						t.Errorf("IsDeviceToHost() = %v, want %v", got, dir.isD2H)
					}
					if got := pkt.IsHostToDevice(); got != !dir.isD2H {
						t.Errorf("IsHostToDevice() = %v, want %v", got, !dir.isD2H)
					}
					if got := pkt.IsStandard(); got != typ.isStd {
						t.Errorf("IsStandard() = %v, want %v", got, typ.isStd)
					}
					if got := pkt.IsClass(); got != typ.isClass {
						t.Errorf("IsClass() = %v, want %v", got, typ.isClass)
					}
					if got := pkt.IsVendor(); got != typ.isVendor {
						t.Errorf("IsVendor() = %v, want %v", got, typ.isVendor)
					}
					if got := pkt.IsDeviceRecipient(); got != recip.isDev {
						t.Errorf("IsDeviceRecipient() = %v, want %v", got, recip.isDev)
					}
					if got := pkt.IsInterfaceRecipient(); got != recip.isIface {
						t.Errorf("IsInterfaceRecipient() = %v, want %v", got, recip.isIface)
					}
					if got := pkt.IsEndpointRecipient(); got != recip.isEP {
						t.Errorf("IsEndpointRecipient() = %v, want %v", got, recip.isEP)
					}
				})
			}
		}
	}
}

func TestSetupPacket_DescriptorFieldsBoundary(t *testing.T) {
	tests := []struct {
		name      string
		value     uint16
		wantType  uint8
		wantIndex uint8
	}{
		{"type=0,index=0", 0x0000, 0x00, 0x00},
		{"type=FF,index=0", 0xFF00, 0xFF, 0x00},
		{"type=0,index=FF", 0x00FF, 0x00, 0xFF},
		{"type=FF,index=FF", 0xFFFF, 0xFF, 0xFF},
		{"type=01,index=05", 0x0105, 0x01, 0x05},
		{"type=03,index=01", 0x0301, 0x03, 0x01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := &SetupPacket{Value: tt.value}
			if got := pkt.DescriptorType(); got != tt.wantType {
				t.Errorf("DescriptorType() = 0x%02X, want 0x%02X", got, tt.wantType)
			}
			if got := pkt.DescriptorIndex(); got != tt.wantIndex {
				t.Errorf("DescriptorIndex() = 0x%02X, want 0x%02X", got, tt.wantIndex)
			}
		})
	}
}

func TestSetupPacket_ConcurrentAccess(t *testing.T) {
	// Verify that SetupPacket methods are safe for concurrent read access
	pkt := &SetupPacket{
		RequestType: 0xA1,
		Request:     0x21,
		Value:       0x0301,
		Index:       0x0002,
		Length:      64,
	}

	const goroutines = 10
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = pkt.Direction()
				_ = pkt.Type()
				_ = pkt.Recipient()
				_ = pkt.IsDeviceToHost()
				_ = pkt.IsClass()
				_ = pkt.DescriptorType()
				_ = pkt.DescriptorIndex()
				_ = pkt.String()
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkSetupPacket_Parse(b *testing.B) {
	benchmarks := []struct {
		name string
		data []byte
	}{
		{
			"GetDescriptor/Device",
			[]byte{0x80, 0x06, 0x00, 0x01, 0x00, 0x00, 0x12, 0x00},
		},
		{
			"GetDescriptor/Configuration",
			[]byte{0x80, 0x06, 0x00, 0x02, 0x00, 0x00, 0xFF, 0x00},
		},
		{
			"GetDescriptor/String",
			[]byte{0x80, 0x06, 0x01, 0x03, 0x09, 0x04, 0xFF, 0x00},
		},
		{
			"SetAddress",
			[]byte{0x00, 0x05, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"SetConfiguration",
			[]byte{0x00, 0x09, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			"ClassRequest",
			[]byte{0xA1, 0x21, 0x00, 0x00, 0x00, 0x00, 0x07, 0x00},
		},
		{
			"VendorRequest",
			[]byte{0xC0, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()
			var pkt SetupPacket
			for i := 0; i < b.N; i++ {
				_ = ParseSetupPacket(bb.data, &pkt)
			}
		})
	}
}

func BenchmarkSetupPacket_MarshalTo(b *testing.B) {
	benchmarks := []struct {
		name string
		pkt  SetupPacket
	}{
		{
			"GetDescriptor/Device",
			SetupPacket{
				RequestType: 0x80,
				Request:     0x06,
				Value:       0x0100,
				Index:       0x0000,
				Length:      18,
			},
		},
		{
			"SetAddress",
			SetupPacket{
				RequestType: 0x00,
				Request:     0x05,
				Value:       5,
				Index:       0,
				Length:      0,
			},
		},
		{
			"MaxValues",
			SetupPacket{
				RequestType: 0xFF,
				Request:     0xFF,
				Value:       0xFFFF,
				Index:       0xFFFF,
				Length:      0xFFFF,
			},
		},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()
			var buf [SetupPacketSize]byte
			for i := 0; i < b.N; i++ {
				_ = bb.pkt.MarshalTo(buf[:])
			}
		})
	}
}

func BenchmarkSetupPacket_Direction(b *testing.B) {
	b.ReportAllocs()
	pkt := &SetupPacket{RequestType: 0x80}
	for i := 0; i < b.N; i++ {
		_ = pkt.Direction()
	}
}

func BenchmarkSetupPacket_Type(b *testing.B) {
	b.ReportAllocs()
	pkt := &SetupPacket{RequestType: 0xA1}
	for i := 0; i < b.N; i++ {
		_ = pkt.Type()
	}
}

func BenchmarkSetupPacket_Recipient(b *testing.B) {
	b.ReportAllocs()
	pkt := &SetupPacket{RequestType: 0x02}
	for i := 0; i < b.N; i++ {
		_ = pkt.Recipient()
	}
}

func BenchmarkSetupPacket_DescriptorType(b *testing.B) {
	b.ReportAllocs()
	pkt := &SetupPacket{Value: 0x0301}
	for i := 0; i < b.N; i++ {
		_ = pkt.DescriptorType()
	}
}

func BenchmarkSetupPacket_String(b *testing.B) {
	benchmarks := []struct {
		name string
		pkt  SetupPacket
	}{
		{
			"Standard/Device/IN",
			SetupPacket{RequestType: 0x80, Request: 0x06, Value: 0x0100},
		},
		{
			"Class/Interface/OUT",
			SetupPacket{RequestType: 0x21, Request: 0x20, Value: 0x0000},
		},
		{
			"Vendor/Endpoint/IN",
			SetupPacket{RequestType: 0xC2, Request: 0x01, Value: 0xFFFF},
		},
	}

	for _, bb := range benchmarks {
		b.Run(bb.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = bb.pkt.String()
			}
		})
	}
}

func BenchmarkSetupPacket_RoundTrip(b *testing.B) {
	b.ReportAllocs()
	pkt := SetupPacket{
		RequestType: 0x80,
		Request:     0x06,
		Value:       0x0100,
		Index:       0x0000,
		Length:      18,
	}
	var buf [SetupPacketSize]byte
	var parsed SetupPacket

	for i := 0; i < b.N; i++ {
		pkt.MarshalTo(buf[:])
		_ = ParseSetupPacket(buf[:], &parsed)
	}
}

func BenchmarkGetDescriptorSetup(b *testing.B) {
	b.ReportAllocs()
	var pkt SetupPacket
	for i := 0; i < b.N; i++ {
		GetDescriptorSetup(&pkt, DescriptorTypeDevice, 0, 18)
	}
}

func BenchmarkGetSetAddressSetup(b *testing.B) {
	b.ReportAllocs()
	var pkt SetupPacket
	for i := 0; i < b.N; i++ {
		GetSetAddressSetup(&pkt, 5)
	}
}

func BenchmarkGetSetConfigurationSetup(b *testing.B) {
	b.ReportAllocs()
	var pkt SetupPacket
	for i := 0; i < b.N; i++ {
		GetSetConfigurationSetup(&pkt, 1)
	}
}
