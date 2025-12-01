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
