package device

import (
	"testing"
)

func TestNewInterface(t *testing.T) {
	desc := &InterfaceDescriptor{
		Length:            9,
		DescriptorType:    DescriptorTypeInterface,
		InterfaceNumber:   1,
		AlternateSetting:  0,
		NumEndpoints:      2,
		InterfaceClass:    ClassCDC,
		InterfaceSubClass: 0x02,
		InterfaceProtocol: 0x01,
		InterfaceIndex:    3,
	}

	iface := NewInterface(desc)

	if iface.Number != 1 {
		t.Errorf("Number = %d, want 1", iface.Number)
	}
	if iface.Class != ClassCDC {
		t.Errorf("Class = 0x%02X, want 0x%02X", iface.Class, ClassCDC)
	}
	if iface.StringIndex != 3 {
		t.Errorf("StringIndex = %d, want 3", iface.StringIndex)
	}
}

func TestInterfaceAddEndpoint(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})

	ep := &Endpoint{
		Address:       0x81,
		Attributes:    EndpointTypeBulk,
		MaxPacketSize: 512,
	}

	err := iface.AddEndpoint(ep)
	if err != nil {
		t.Fatalf("AddEndpoint() error = %v", err)
	}

	// Adding same endpoint again should fail
	err = iface.AddEndpoint(ep)
	if err == nil {
		t.Error("AddEndpoint() should fail for duplicate endpoint")
	}
}

func TestInterfaceGetEndpoint(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})

	epIn := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	epOut := &Endpoint{Address: 0x02, Attributes: EndpointTypeBulk}

	iface.AddEndpoint(epIn)
	iface.AddEndpoint(epOut)

	if got := iface.GetEndpoint(0x81); got != epIn {
		t.Error("GetEndpoint(0x81) returned wrong endpoint")
	}
	if got := iface.GetEndpoint(0x02); got != epOut {
		t.Error("GetEndpoint(0x02) returned wrong endpoint")
	}
	if got := iface.GetEndpoint(0x03); got != nil {
		t.Error("GetEndpoint(0x03) should return nil")
	}
}

func TestInterfaceGetInOutEndpoint(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})

	epIn := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	epOut := &Endpoint{Address: 0x01, Attributes: EndpointTypeBulk}

	iface.AddEndpoint(epIn)
	iface.AddEndpoint(epOut)

	if got := iface.GetInEndpoint(1); got != epIn {
		t.Error("GetInEndpoint(1) returned wrong endpoint")
	}
	if got := iface.GetOutEndpoint(1); got != epOut {
		t.Error("GetOutEndpoint(1) returned wrong endpoint")
	}
}

func TestInterfaceRemoveEndpoint(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})

	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	iface.AddEndpoint(ep)

	if iface.NumEndpoints() != 1 {
		t.Errorf("NumEndpoints() = %d, want 1", iface.NumEndpoints())
	}

	iface.RemoveEndpoint(0x81)

	if iface.NumEndpoints() != 0 {
		t.Errorf("NumEndpoints() = %d, want 0 after removal", iface.NumEndpoints())
	}
}

func TestInterfaceEndpoints(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})

	iface.AddEndpoint(&Endpoint{Address: 0x81})
	iface.AddEndpoint(&Endpoint{Address: 0x02})
	iface.AddEndpoint(&Endpoint{Address: 0x83})

	eps := iface.Endpoints()
	if len(eps) != 3 {
		t.Errorf("Endpoints() length = %d, want 3", len(eps))
	}
}

type mockClassDriver struct {
	initCalled      bool
	setupCalled     bool
	altCalled       bool
	closeCalled     bool
	handleSetupResp bool
}

func (m *mockClassDriver) Init(iface *Interface) error {
	m.initCalled = true
	return nil
}

func (m *mockClassDriver) HandleSetup(iface *Interface, setup *SetupPacket, data []byte) (bool, error) {
	m.setupCalled = true
	return m.handleSetupResp, nil
}

func (m *mockClassDriver) SetAlternate(iface *Interface, alt uint8) error {
	m.altCalled = true
	return nil
}

func (m *mockClassDriver) Close() error {
	m.closeCalled = true
	return nil
}

func TestInterfaceClassDriver(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	driver := &mockClassDriver{handleSetupResp: true}

	err := iface.SetClassDriver(driver)
	if err != nil {
		t.Fatalf("SetClassDriver() error = %v", err)
	}

	if !driver.initCalled {
		t.Error("Init() should be called on driver")
	}

	// Test HandleSetup
	handled, err := iface.HandleSetup(&SetupPacket{}, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if !handled {
		t.Error("HandleSetup() should return true")
	}
	if !driver.setupCalled {
		t.Error("driver HandleSetup() should be called")
	}

	// Test SetAlternate
	err = iface.SetAlternate(1)
	if err != nil {
		t.Fatalf("SetAlternate() error = %v", err)
	}
	if !driver.altCalled {
		t.Error("driver SetAlternate() should be called")
	}

	// Test Close
	err = iface.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !driver.closeCalled {
		t.Error("driver Close() should be called")
	}
}

func TestInterfaceHandleSetupNoDriver(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})

	handled, err := iface.HandleSetup(&SetupPacket{}, nil)
	if err != nil {
		t.Fatalf("HandleSetup() error = %v", err)
	}
	if handled {
		t.Error("HandleSetup() should return false without driver")
	}
}

func TestInterfaceDescriptor(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{
		InterfaceNumber:   1,
		AlternateSetting:  0,
		InterfaceClass:    ClassHID,
		InterfaceSubClass: 1,
		InterfaceProtocol: 1,
		InterfaceIndex:    4,
	})

	iface.AddEndpoint(&Endpoint{Address: 0x81})
	iface.AddEndpoint(&Endpoint{Address: 0x01})

	desc := iface.Descriptor()

	if desc.InterfaceNumber != 1 {
		t.Errorf("InterfaceNumber = %d, want 1", desc.InterfaceNumber)
	}
	if desc.NumEndpoints != 2 {
		t.Errorf("NumEndpoints = %d, want 2", desc.NumEndpoints)
	}
	if desc.InterfaceClass != ClassHID {
		t.Errorf("InterfaceClass = 0x%02X, want 0x%02X", desc.InterfaceClass, ClassHID)
	}
}

func TestNewConfiguration(t *testing.T) {
	config := NewConfiguration(1)

	if config.Value != 1 {
		t.Errorf("Value = %d, want 1", config.Value)
	}
	if config.Attributes&ConfigAttrBusPowered == 0 {
		t.Error("should be bus-powered by default")
	}
}

func TestConfigurationAddInterface(t *testing.T) {
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})

	err := config.AddInterface(iface)
	if err != nil {
		t.Fatalf("AddInterface() error = %v", err)
	}

	// Adding same interface again should fail
	err = config.AddInterface(iface)
	if err == nil {
		t.Error("AddInterface() should fail for duplicate interface")
	}
}

func TestConfigurationGetInterface(t *testing.T) {
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	config.AddInterface(iface)

	if got := config.GetInterface(0); got != iface {
		t.Error("GetInterface(0) returned wrong interface")
	}
	if got := config.GetInterface(1); got != nil {
		t.Error("GetInterface(1) should return nil")
	}
}

func TestConfigurationRemoveInterface(t *testing.T) {
	config := NewConfiguration(1)
	driver := &mockClassDriver{}
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	iface.SetClassDriver(driver)
	config.AddInterface(iface)

	config.RemoveInterface(0)

	if config.GetInterface(0) != nil {
		t.Error("interface should be removed")
	}
	if !driver.closeCalled {
		t.Error("driver Close() should be called on removal")
	}
}

func TestConfigurationInterfaces(t *testing.T) {
	config := NewConfiguration(1)
	config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: 0}))
	config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: 1}))

	ifaces := config.Interfaces()
	if len(ifaces) != 2 {
		t.Errorf("Interfaces() length = %d, want 2", len(ifaces))
	}
}

func TestConfigurationAssociation(t *testing.T) {
	config := NewConfiguration(1)
	assoc := InterfaceAssociation{
		FirstInterface:   0,
		InterfaceCount:   2,
		FunctionClass:    ClassCDC,
		FunctionSubClass: 0x02,
		FunctionProtocol: 0x01,
	}

	config.AddAssociation(&assoc)

	assocs := config.Associations()
	if len(assocs) != 1 {
		t.Errorf("Associations() length = %d, want 1", len(assocs))
	}
	if assocs[0] != assoc {
		t.Error("wrong association returned")
	}
}

func TestConfigurationDescriptor(t *testing.T) {
	config := NewConfiguration(1)
	config.StringIndex = 5
	config.MaxPower = 100 // 200mA

	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	iface.AddEndpoint(&Endpoint{Address: 0x81})
	config.AddInterface(iface)

	desc := config.Descriptor()

	if desc.ConfigurationValue != 1 {
		t.Errorf("ConfigurationValue = %d, want 1", desc.ConfigurationValue)
	}
	if desc.NumInterfaces != 1 {
		t.Errorf("NumInterfaces = %d, want 1", desc.NumInterfaces)
	}
	// Total: 9 (config) + 9 (interface) + 7 (endpoint) = 25
	if desc.TotalLength != 25 {
		t.Errorf("TotalLength = %d, want 25", desc.TotalLength)
	}
}

func TestConfigurationMarshalTo(t *testing.T) {
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	iface.AddEndpoint(&Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 64})
	config.AddInterface(iface)

	var buf [512]byte
	n := config.MarshalTo(buf[:])

	// Total: 9 (config) + 9 (interface) + 7 (endpoint) = 25
	if n != 25 {
		t.Errorf("MarshalTo() length = %d, want 25", n)
	}

	// First byte should be config descriptor length
	if buf[0] != 9 {
		t.Errorf("buf[0] = %d, want 9 (config desc length)", buf[0])
	}

	// Second byte should be config descriptor type
	if buf[1] != DescriptorTypeConfiguration {
		t.Errorf("buf[1] = 0x%02X, want 0x%02X", buf[1], DescriptorTypeConfiguration)
	}

	// Interface descriptor starts at offset 9
	if buf[9] != 9 {
		t.Errorf("buf[9] = %d, want 9 (interface desc length)", buf[9])
	}
	if buf[10] != DescriptorTypeInterface {
		t.Errorf("buf[10] = 0x%02X, want 0x%02X", buf[10], DescriptorTypeInterface)
	}

	// Endpoint descriptor starts at offset 18
	if buf[18] != 7 {
		t.Errorf("buf[18] = %d, want 7 (endpoint desc length)", buf[18])
	}
	if buf[19] != DescriptorTypeEndpoint {
		t.Errorf("buf[19] = 0x%02X, want 0x%02X", buf[19], DescriptorTypeEndpoint)
	}
}

func TestConfigurationMarshalToWithAssociation(t *testing.T) {
	config := NewConfiguration(1)
	config.AddAssociation(&InterfaceAssociation{
		FirstInterface: 0,
		InterfaceCount: 2,
		FunctionClass:  ClassCDC,
	})
	config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: 0}))

	var buf [512]byte
	n := config.MarshalTo(buf[:])

	// Total: 9 (config) + 8 (IAD) + 9 (interface) = 26
	if n != 26 {
		t.Errorf("MarshalTo() length = %d, want 26", n)
	}

	// IAD should be after config descriptor
	if buf[9] != 8 {
		t.Errorf("buf[9] = %d, want 8 (IAD length)", buf[9])
	}
	if buf[10] != DescriptorTypeInterfaceAssociation {
		t.Errorf("buf[10] = 0x%02X, want 0x%02X", buf[10], DescriptorTypeInterfaceAssociation)
	}
}

func TestConfigurationPowerAttributes(t *testing.T) {
	config := NewConfiguration(1)

	// Default is bus-powered
	if config.IsSelfPowered() {
		t.Error("should not be self-powered by default")
	}

	config.SetSelfPowered(true)
	if !config.IsSelfPowered() {
		t.Error("should be self-powered after SetSelfPowered(true)")
	}

	config.SetSelfPowered(false)
	if config.IsSelfPowered() {
		t.Error("should not be self-powered after SetSelfPowered(false)")
	}
}

func TestConfigurationRemoteWakeup(t *testing.T) {
	config := NewConfiguration(1)

	if config.SupportsRemoteWakeup() {
		t.Error("should not support remote wakeup by default")
	}

	config.SetRemoteWakeup(true)
	if !config.SupportsRemoteWakeup() {
		t.Error("should support remote wakeup after SetRemoteWakeup(true)")
	}

	config.SetRemoteWakeup(false)
	if config.SupportsRemoteWakeup() {
		t.Error("should not support remote wakeup after SetRemoteWakeup(false)")
	}
}

func TestConfigurationClose(t *testing.T) {
	config := NewConfiguration(1)
	driver := &mockClassDriver{}
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	iface.SetClassDriver(driver)
	config.AddInterface(iface)

	err := config.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !driver.closeCalled {
		t.Error("driver Close() should be called")
	}

	if config.NumInterfaces() != 0 {
		t.Error("interfaces should be cleared after Close()")
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestInterfaceAddEndpointEdgeCases(t *testing.T) {
	t.Run("MaxEndpoints", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		// Add maximum endpoints (16 per direction typical)
		for i := uint8(1); i <= MaxEndpointsPerInterface; i++ {
			addr := i | 0x80 // IN endpoints
			err := iface.AddEndpoint(&Endpoint{Address: addr, Attributes: EndpointTypeBulk})
			if err != nil {
				t.Fatalf("AddEndpoint(0x%02X) error = %v", addr, err)
			}
		}
		// Adding one more should fail
		err := iface.AddEndpoint(&Endpoint{Address: 0x01, Attributes: EndpointTypeBulk})
		if err == nil {
			t.Error("AddEndpoint() should fail when at capacity")
		}
	})

	t.Run("AllTransferTypes", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		types := []uint8{EndpointTypeControl, EndpointTypeIsochronous, EndpointTypeBulk, EndpointTypeInterrupt}
		for i, epType := range types {
			err := iface.AddEndpoint(&Endpoint{Address: uint8(0x81 + i), Attributes: epType})
			if err != nil {
				t.Fatalf("AddEndpoint() with type %d error = %v", epType, err)
			}
		}
		if iface.NumEndpoints() != len(types) {
			t.Errorf("NumEndpoints() = %d, want %d", iface.NumEndpoints(), len(types))
		}
	})
}

func TestInterfaceRemoveEndpointEdgeCases(t *testing.T) {
	t.Run("RemoveNonexistent", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		iface.AddEndpoint(&Endpoint{Address: 0x81})
		// Should not panic or error
		iface.RemoveEndpoint(0x82)
		if iface.NumEndpoints() != 1 {
			t.Error("should not remove existing endpoint")
		}
	})

	t.Run("RemoveFromEmpty", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		// Should not panic
		iface.RemoveEndpoint(0x81)
		if iface.NumEndpoints() != 0 {
			t.Error("should remain empty")
		}
	})

	t.Run("RemoveMiddle", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		iface.AddEndpoint(&Endpoint{Address: 0x81})
		iface.AddEndpoint(&Endpoint{Address: 0x82})
		iface.AddEndpoint(&Endpoint{Address: 0x83})

		iface.RemoveEndpoint(0x82)

		if iface.NumEndpoints() != 2 {
			t.Errorf("NumEndpoints() = %d, want 2", iface.NumEndpoints())
		}
		if iface.GetEndpoint(0x81) == nil {
			t.Error("0x81 should still exist")
		}
		if iface.GetEndpoint(0x83) == nil {
			t.Error("0x83 should still exist")
		}
	})
}

func TestInterfaceGetInOutEndpointEdgeCases(t *testing.T) {
	t.Run("NoMatchingEndpoint", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		iface.AddEndpoint(&Endpoint{Address: 0x81}) // Only IN

		if got := iface.GetOutEndpoint(1); got != nil {
			t.Error("GetOutEndpoint(1) should return nil")
		}
		if got := iface.GetInEndpoint(2); got != nil {
			t.Error("GetInEndpoint(2) should return nil")
		}
	})

	t.Run("SameNumberDifferentDirection", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		epIn := &Endpoint{Address: 0x81}  // EP1 IN
		epOut := &Endpoint{Address: 0x01} // EP1 OUT
		iface.AddEndpoint(epIn)
		iface.AddEndpoint(epOut)

		if got := iface.GetInEndpoint(1); got != epIn {
			t.Error("GetInEndpoint(1) returned wrong endpoint")
		}
		if got := iface.GetOutEndpoint(1); got != epOut {
			t.Error("GetOutEndpoint(1) returned wrong endpoint")
		}
	})
}

func TestInterfaceClassDriverEdgeCases(t *testing.T) {
	t.Run("ReplaceDriver", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		driver1 := &mockClassDriver{}
		driver2 := &mockClassDriver{}

		iface.SetClassDriver(driver1)
		if !driver1.initCalled {
			t.Error("driver1.Init() should be called")
		}

		iface.SetClassDriver(driver2)
		if !driver1.closeCalled {
			t.Error("driver1.Close() should be called when replaced")
		}
		if !driver2.initCalled {
			t.Error("driver2.Init() should be called")
		}
	})

	t.Run("SetNilDriver", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		driver := &mockClassDriver{}
		iface.SetClassDriver(driver)

		err := iface.SetClassDriver(nil)
		if err != nil {
			t.Fatalf("SetClassDriver(nil) error = %v", err)
		}
		if !driver.closeCalled {
			t.Error("driver.Close() should be called")
		}
		if iface.ClassDriver() != nil {
			t.Error("ClassDriver() should return nil")
		}
	})

	t.Run("SetAlternateNoDriver", func(t *testing.T) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		err := iface.SetAlternate(1)
		if err != nil {
			t.Fatalf("SetAlternate() error = %v", err)
		}
		if iface.AlternateSetting != 1 {
			t.Errorf("AlternateSetting = %d, want 1", iface.AlternateSetting)
		}
	})
}

func TestConfigurationAddInterfaceEdgeCases(t *testing.T) {
	t.Run("MaxInterfaces", func(t *testing.T) {
		config := NewConfiguration(1)
		for i := uint8(0); i < MaxInterfacesPerConfiguration; i++ {
			err := config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: i}))
			if err != nil {
				t.Fatalf("AddInterface(%d) error = %v", i, err)
			}
		}
		// Adding one more should fail
		err := config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: MaxInterfacesPerConfiguration}))
		if err == nil {
			t.Error("AddInterface() should fail when at capacity")
		}
	})
}

func TestConfigurationAddAssociationEdgeCases(t *testing.T) {
	t.Run("MaxAssociations", func(t *testing.T) {
		config := NewConfiguration(1)
		for i := 0; i < MaxAssociationsPerConfiguration; i++ {
			err := config.AddAssociation(&InterfaceAssociation{FirstInterface: uint8(i * 2)})
			if err != nil {
				t.Fatalf("AddAssociation(%d) error = %v", i, err)
			}
		}
		// Adding one more should fail
		err := config.AddAssociation(&InterfaceAssociation{FirstInterface: 100})
		if err == nil {
			t.Error("AddAssociation() should fail when at capacity")
		}
	})
}

func TestConfigurationRemoveInterfaceEdgeCases(t *testing.T) {
	t.Run("RemoveNonexistent", func(t *testing.T) {
		config := NewConfiguration(1)
		config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: 0}))
		// Should not panic
		config.RemoveInterface(5)
		if config.NumInterfaces() != 1 {
			t.Error("should not affect existing interface")
		}
	})

	t.Run("RemoveFromEmpty", func(t *testing.T) {
		config := NewConfiguration(1)
		// Should not panic
		config.RemoveInterface(0)
	})
}

func TestConfigurationMarshalToEdgeCases(t *testing.T) {
	t.Run("EmptyConfiguration", func(t *testing.T) {
		config := NewConfiguration(1)
		var buf [512]byte
		n := config.MarshalTo(buf[:])
		// Just config descriptor = 9 bytes
		if n != ConfigurationDescriptorSize {
			t.Errorf("MarshalTo() = %d, want %d", n, ConfigurationDescriptorSize)
		}
	})

	t.Run("BufferTooSmall", func(t *testing.T) {
		config := NewConfiguration(1)
		var buf [5]byte // Too small for config descriptor
		n := config.MarshalTo(buf[:])
		if n != 0 {
			t.Errorf("MarshalTo() = %d, want 0", n)
		}
	})

	t.Run("ComplexConfiguration", func(t *testing.T) {
		config := NewConfiguration(1)
		config.AddAssociation(&InterfaceAssociation{
			FirstInterface: 0,
			InterfaceCount: 2,
			FunctionClass:  ClassCDC,
		})

		iface0 := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0, InterfaceClass: ClassCDC})
		iface0.AddEndpoint(&Endpoint{Address: 0x83, Attributes: EndpointTypeInterrupt, MaxPacketSize: 8})
		config.AddInterface(iface0)

		iface1 := NewInterface(&InterfaceDescriptor{InterfaceNumber: 1, InterfaceClass: ClassCDCData})
		iface1.AddEndpoint(&Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 512})
		iface1.AddEndpoint(&Endpoint{Address: 0x02, Attributes: EndpointTypeBulk, MaxPacketSize: 512})
		config.AddInterface(iface1)

		var buf [1024]byte
		n := config.MarshalTo(buf[:])
		// 9 (config) + 8 (IAD) + 9 (iface0) + 7 (ep) + 9 (iface1) + 7 (ep) + 7 (ep) = 56
		expected := 9 + 8 + 9 + 7 + 9 + 7 + 7
		if n != expected {
			t.Errorf("MarshalTo() = %d, want %d", n, expected)
		}

		// Verify TotalLength in header matches
		totalLen := uint16(buf[2]) | (uint16(buf[3]) << 8)
		if int(totalLen) != expected {
			t.Errorf("TotalLength = %d, want %d", totalLen, expected)
		}
	})
}

func TestInterfaceConcurrentAccess(t *testing.T) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	for i := uint8(1); i <= 4; i++ {
		iface.AddEndpoint(&Endpoint{Address: 0x80 | i})
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				_ = iface.NumEndpoints()
				_ = iface.GetEndpoint(0x81)
				_ = iface.GetInEndpoint(1)
				_ = iface.GetOutEndpoint(1)
				_ = iface.Endpoints()
				_ = iface.Descriptor()
				_ = iface.ClassDriver()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConfigurationConcurrentAccess(t *testing.T) {
	config := NewConfiguration(1)
	for i := uint8(0); i < 4; i++ {
		config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: i}))
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				_ = config.NumInterfaces()
				_ = config.GetInterface(0)
				_ = config.Interfaces()
				_ = config.Associations()
				_ = config.Descriptor()
				_ = config.IsSelfPowered()
				_ = config.SupportsRemoteWakeup()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkNewInterface(b *testing.B) {
	desc := &InterfaceDescriptor{
		InterfaceNumber:   0,
		InterfaceClass:    ClassCDC,
		InterfaceSubClass: 0x02,
		InterfaceProtocol: 0x01,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewInterface(desc)
	}
}

func BenchmarkInterfaceAddEndpoint(b *testing.B) {
	desc := &InterfaceDescriptor{InterfaceNumber: 0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iface := NewInterface(desc)
		_ = iface.AddEndpoint(&Endpoint{Address: 0x81})
	}
}

func BenchmarkInterfaceGetEndpoint(b *testing.B) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	for i := uint8(1); i <= 8; i++ {
		iface.AddEndpoint(&Endpoint{Address: 0x80 | i})
	}

	b.Run("First", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = iface.GetEndpoint(0x81)
		}
	})

	b.Run("Last", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = iface.GetEndpoint(0x88)
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = iface.GetEndpoint(0x8F)
		}
	})
}

func BenchmarkInterfaceEndpoints(b *testing.B) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	for i := uint8(1); i <= 8; i++ {
		iface.AddEndpoint(&Endpoint{Address: 0x80 | i})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iface.Endpoints()
	}
}

func BenchmarkInterfaceNumEndpoints(b *testing.B) {
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	for i := uint8(1); i <= 4; i++ {
		iface.AddEndpoint(&Endpoint{Address: 0x80 | i})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iface.NumEndpoints()
	}
}

func BenchmarkInterfaceRemoveEndpoint(b *testing.B) {
	desc := &InterfaceDescriptor{InterfaceNumber: 0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		iface := NewInterface(desc)
		iface.AddEndpoint(&Endpoint{Address: 0x81})
		b.StartTimer()
		iface.RemoveEndpoint(0x81)
	}
}

func BenchmarkInterfaceDescriptor(b *testing.B) {
	iface := NewInterface(&InterfaceDescriptor{
		InterfaceNumber:   0,
		InterfaceClass:    ClassCDC,
		InterfaceSubClass: 0x02,
		InterfaceProtocol: 0x01,
	})
	for i := uint8(1); i <= 4; i++ {
		iface.AddEndpoint(&Endpoint{Address: 0x80 | i})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = iface.Descriptor()
	}
}

func BenchmarkInterfaceHandleSetup(b *testing.B) {
	b.Run("NoDriver", func(b *testing.B) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		setup := &SetupPacket{}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = iface.HandleSetup(setup, nil)
		}
	})

	b.Run("WithDriver", func(b *testing.B) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		iface.SetClassDriver(&mockClassDriver{handleSetupResp: true})
		setup := &SetupPacket{}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = iface.HandleSetup(setup, nil)
		}
	})
}

func BenchmarkInterfaceSetAlternate(b *testing.B) {
	b.Run("NoDriver", func(b *testing.B) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = iface.SetAlternate(uint8(i & 0xFF))
		}
	})

	b.Run("WithDriver", func(b *testing.B) {
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		iface.SetClassDriver(&mockClassDriver{})
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = iface.SetAlternate(uint8(i & 0xFF))
		}
	})
}

func BenchmarkInterfaceSetClassDriver(b *testing.B) {
	desc := &InterfaceDescriptor{InterfaceNumber: 0}
	driver := &mockClassDriver{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iface := NewInterface(desc)
		_ = iface.SetClassDriver(driver)
	}
}

func BenchmarkInterfaceClose(b *testing.B) {
	desc := &InterfaceDescriptor{InterfaceNumber: 0}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		iface := NewInterface(desc)
		iface.SetClassDriver(&mockClassDriver{})
		b.StartTimer()
		_ = iface.Close()
	}
}

func BenchmarkNewConfiguration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewConfiguration(1)
	}
}

func BenchmarkConfigurationAddInterface(b *testing.B) {
	desc := &InterfaceDescriptor{InterfaceNumber: 0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := NewConfiguration(1)
		_ = config.AddInterface(NewInterface(desc))
	}
}

func BenchmarkConfigurationGetInterface(b *testing.B) {
	config := NewConfiguration(1)
	for i := uint8(0); i < 8; i++ {
		config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: i}))
	}

	b.Run("First", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = config.GetInterface(0)
		}
	})

	b.Run("Last", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = config.GetInterface(7)
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = config.GetInterface(20)
		}
	})
}

func BenchmarkConfigurationInterfaces(b *testing.B) {
	config := NewConfiguration(1)
	for i := uint8(0); i < 4; i++ {
		config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: i}))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Interfaces()
	}
}

func BenchmarkConfigurationNumInterfaces(b *testing.B) {
	config := NewConfiguration(1)
	for i := uint8(0); i < 4; i++ {
		config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: i}))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.NumInterfaces()
	}
}

func BenchmarkConfigurationAddAssociation(b *testing.B) {
	assoc := &InterfaceAssociation{FirstInterface: 0, InterfaceCount: 2}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := NewConfiguration(1)
		_ = config.AddAssociation(assoc)
	}
}

func BenchmarkConfigurationAssociations(b *testing.B) {
	config := NewConfiguration(1)
	config.AddAssociation(&InterfaceAssociation{FirstInterface: 0, InterfaceCount: 2})
	config.AddAssociation(&InterfaceAssociation{FirstInterface: 2, InterfaceCount: 2})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Associations()
	}
}

func BenchmarkConfigurationDescriptor(b *testing.B) {
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	for i := uint8(1); i <= 4; i++ {
		iface.AddEndpoint(&Endpoint{Address: 0x80 | i})
	}
	config.AddInterface(iface)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Descriptor()
	}
}

func BenchmarkConfigurationMarshalTo(b *testing.B) {
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	for i := uint8(1); i <= 4; i++ {
		iface.AddEndpoint(&Endpoint{Address: 0x80 | i, Attributes: EndpointTypeBulk, MaxPacketSize: 512})
	}
	config.AddInterface(iface)

	var buf [512]byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.MarshalTo(buf[:])
	}
}

func BenchmarkConfigurationPowerAttributes(b *testing.B) {
	config := NewConfiguration(1)

	b.Run("SetSelfPowered", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			config.SetSelfPowered(true)
		}
	})

	b.Run("IsSelfPowered", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = config.IsSelfPowered()
		}
	})

	b.Run("SetRemoteWakeup", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			config.SetRemoteWakeup(true)
		}
	})

	b.Run("SupportsRemoteWakeup", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = config.SupportsRemoteWakeup()
		}
	})
}

func BenchmarkConfigurationClose(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		config := NewConfiguration(1)
		for j := uint8(0); j < 4; j++ {
			iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: j})
			config.AddInterface(iface)
		}
		b.StartTimer()
		_ = config.Close()
	}
}
