package device

import (
	"context"
	"testing"
	"time"

	"github.com/ardnew/softusb/pkg"
)

func init() {
}

func TestNewDevice(t *testing.T) {
	desc := &DeviceDescriptor{
		Length:            18,
		DescriptorType:    DescriptorTypeDevice,
		USBVersion:        0x0200,
		DeviceClass:       ClassPerInterface,
		MaxPacketSize0:    64,
		VendorID:          0x1234,
		ProductID:         0x5678,
		NumConfigurations: 1,
	}

	dev := NewDevice(desc)

	if dev.Descriptor != desc {
		t.Error("Descriptor not set")
	}
	if dev.State() != StateAttached {
		t.Errorf("State() = %v, want %v", dev.State(), StateAttached)
	}
	if dev.Speed() != SpeedFull {
		t.Errorf("Speed() = %v, want %v", dev.Speed(), SpeedFull)
	}
	if dev.ep0 == nil {
		t.Error("EP0 not initialized")
	}
	if dev.ep0.MaxPacketSize != 64 {
		t.Errorf("EP0 MaxPacketSize = %d, want 64", dev.ep0.MaxPacketSize)
	}
}

func TestDeviceConfiguration(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)

	err := dev.AddConfiguration(config)
	if err != nil {
		t.Fatalf("AddConfiguration() error = %v", err)
	}

	// Adding same config again should fail
	err = dev.AddConfiguration(config)
	if err == nil {
		t.Error("AddConfiguration() should fail for duplicate")
	}

	if got := dev.GetConfiguration(1); got != config {
		t.Error("GetConfiguration(1) returned wrong config")
	}
	if got := dev.GetConfiguration(2); got != nil {
		t.Error("GetConfiguration(2) should return nil")
	}
}

func TestDeviceStrings(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})

	// Create language descriptor
	var langBuf [4]byte
	langLen := LanguageDescriptorTo(langBuf[:], LangIDUSEnglish)
	dev.SetLanguages(langBuf[:langLen])

	// Create string descriptors
	var mfrBuf [256]byte
	mfrLen := StringDescriptorTo(mfrBuf[:], "Test Manufacturer")
	dev.SetString(1, mfrBuf[:mfrLen])

	var prodBuf [256]byte
	prodLen := StringDescriptorTo(prodBuf[:], "Test Product")
	dev.SetString(2, prodBuf[:prodLen])

	lang := dev.GetString(0)
	if lang == nil {
		t.Fatal("GetString(0) returned nil")
	}
	if lang[0] != 4 { // 2 bytes header + 2 bytes lang ID
		t.Errorf("language descriptor length = %d, want 4", lang[0])
	}

	mfr := dev.GetString(1)
	if mfr == nil {
		t.Fatal("GetString(1) returned nil")
	}
	if mfr[1] != DescriptorTypeString {
		t.Errorf("string descriptor type = 0x%02X, want 0x%02X", mfr[1], DescriptorTypeString)
	}
}

func TestDeviceStateTransitions(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)

	// Initial state
	if dev.State() != StateAttached {
		t.Errorf("initial state = %v, want %v", dev.State(), StateAttached)
	}

	// Reset transitions to Default
	dev.Reset()
	if dev.State() != StateDefault {
		t.Errorf("after reset, state = %v, want %v", dev.State(), StateDefault)
	}

	// SetAddress transitions to Address
	err := dev.SetAddress(5)
	if err != nil {
		t.Fatalf("SetAddress() error = %v", err)
	}
	if dev.State() != StateAddress {
		t.Errorf("after set address, state = %v, want %v", dev.State(), StateAddress)
	}
	if dev.Address() != 5 {
		t.Errorf("Address() = %d, want 5", dev.Address())
	}

	// SetConfiguration transitions to Configured
	err = dev.SetConfiguration(1)
	if err != nil {
		t.Fatalf("SetConfiguration() error = %v", err)
	}
	if dev.State() != StateConfigured {
		t.Errorf("after configure, state = %v, want %v", dev.State(), StateConfigured)
	}
	if !dev.IsConfigured() {
		t.Error("IsConfigured() should return true")
	}

	// SetConfiguration(0) unconfigures
	err = dev.SetConfiguration(0)
	if err != nil {
		t.Fatalf("SetConfiguration(0) error = %v", err)
	}
	if dev.State() != StateAddress {
		t.Errorf("after unconfigure, state = %v, want %v", dev.State(), StateAddress)
	}
}

func TestDeviceSetAddressInvalidState(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	// Device is in Attached state, SetAddress should fail
	err := dev.SetAddress(5)
	if err != pkg.ErrInvalidState {
		t.Errorf("SetAddress() error = %v, want %v", err, pkg.ErrInvalidState)
	}
}

func TestDeviceSetConfigurationInvalidState(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	dev.Reset()
	// Device is in Default state, SetConfiguration should fail
	err := dev.SetConfiguration(1)
	if err != pkg.ErrInvalidState {
		t.Errorf("SetConfiguration() error = %v, want %v", err, pkg.ErrInvalidState)
	}
}

func TestDeviceSetConfigurationInvalid(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	dev.Reset()
	dev.SetAddress(5)

	// No configuration 2 exists
	err := dev.SetConfiguration(2)
	if err != pkg.ErrInvalidRequest {
		t.Errorf("SetConfiguration() error = %v, want %v", err, pkg.ErrInvalidRequest)
	}
}

func TestDeviceSuspendResume(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(5)
	dev.SetConfiguration(1)

	// Suspend
	dev.Suspend()
	if !dev.IsSuspended() {
		t.Error("IsSuspended() should return true after suspend")
	}
	if dev.State() != StateSuspended {
		t.Errorf("state = %v, want %v", dev.State(), StateSuspended)
	}

	// Resume should restore previous state
	dev.Resume()
	if dev.IsSuspended() {
		t.Error("IsSuspended() should return false after resume")
	}
	if dev.State() != StateConfigured {
		t.Errorf("after resume, state = %v, want %v", dev.State(), StateConfigured)
	}
}

func TestDeviceCallbacks(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)

	var stateChanges []string
	var suspendCalled, resumeCalled, resetCalled bool
	var addressSet uint8
	var configSet uint8

	dev.SetOnStateChange(func(old, new State) {
		stateChanges = append(stateChanges, old.String()+"->"+new.String())
	})
	dev.SetOnSuspend(func() { suspendCalled = true })
	dev.SetOnResume(func() { resumeCalled = true })
	dev.SetOnReset(func() { resetCalled = true })
	dev.SetOnSetAddress(func(addr uint8) { addressSet = addr })
	dev.SetOnSetConfiguration(func(cfg uint8) { configSet = cfg })

	dev.Reset()
	if !resetCalled {
		t.Error("reset callback not called")
	}

	dev.SetAddress(5)
	if addressSet != 5 {
		t.Errorf("address callback got %d, want 5", addressSet)
	}

	dev.SetConfiguration(1)
	if configSet != 1 {
		t.Errorf("configuration callback got %d, want 1", configSet)
	}

	dev.Suspend()
	if !suspendCalled {
		t.Error("suspend callback not called")
	}

	dev.Resume()
	if !resumeCalled {
		t.Error("resume callback not called")
	}

	if len(stateChanges) == 0 {
		t.Error("state change callback not called")
	}
}

func TestDeviceRemoteWakeup(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})

	if dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be disabled by default")
	}

	dev.EnableRemoteWakeup(true)
	if !dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be enabled")
	}

	dev.EnableRemoteWakeup(false)
	if dev.IsRemoteWakeupEnabled() {
		t.Error("remote wakeup should be disabled")
	}
}

func TestDeviceGetInterface(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	config.AddInterface(iface)
	dev.AddConfiguration(config)

	// Not configured yet
	if got := dev.GetInterface(0); got != nil {
		t.Error("GetInterface should return nil when not configured")
	}

	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	if got := dev.GetInterface(0); got != iface {
		t.Error("GetInterface(0) returned wrong interface")
	}
	if got := dev.GetInterface(1); got != nil {
		t.Error("GetInterface(1) should return nil")
	}
}

func TestDeviceGetEndpoint(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk, MaxPacketSize: 512}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	// EP0
	if got := dev.GetEndpoint(0); got != dev.ControlEndpoint() {
		t.Error("GetEndpoint(0) should return EP0")
	}
	if got := dev.GetEndpoint(0x80); got != dev.ControlEndpoint() {
		t.Error("GetEndpoint(0x80) should return EP0")
	}

	// Non-control endpoint
	if got := dev.GetEndpoint(0x81); got != ep {
		t.Error("GetEndpoint(0x81) returned wrong endpoint")
	}
	if got := dev.GetEndpoint(0x82); got != nil {
		t.Error("GetEndpoint(0x82) should return nil")
	}
}

func TestDeviceSetEndpointStall(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	err := dev.SetEndpointStall(0x81, true)
	if err != nil {
		t.Fatalf("SetEndpointStall() error = %v", err)
	}
	if !ep.IsStalled() {
		t.Error("endpoint should be stalled")
	}

	err = dev.SetEndpointStall(0x81, false)
	if err != nil {
		t.Fatalf("SetEndpointStall() error = %v", err)
	}
	if ep.IsStalled() {
		t.Error("endpoint should not be stalled")
	}

	// Invalid endpoint
	err = dev.SetEndpointStall(0x82, true)
	if err != pkg.ErrInvalidEndpoint {
		t.Errorf("SetEndpointStall() error = %v, want %v", err, pkg.ErrInvalidEndpoint)
	}
}

func TestDeviceGetStatus(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	config.SetSelfPowered(true)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	status := dev.GetStatus()
	if status&DeviceStatusSelfPowered == 0 {
		t.Error("status should indicate self-powered")
	}
	if status&DeviceStatusRemoteWakeup != 0 {
		t.Error("status should not indicate remote wakeup")
	}

	dev.EnableRemoteWakeup(true)
	status = dev.GetStatus()
	if status&DeviceStatusRemoteWakeup == 0 {
		t.Error("status should indicate remote wakeup")
	}
}

func TestDeviceClose(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	driver := &mockClassDriver{}
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	iface.SetClassDriver(driver)
	config := NewConfiguration(1)
	config.AddInterface(iface)
	dev.AddConfiguration(config)

	err := dev.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !driver.closeCalled {
		t.Error("class driver Close() should be called")
	}
}

func TestDeviceBuilder(t *testing.T) {
	dev, err := NewDeviceBuilder().
		WithVendorProduct(0x1234, 0x5678).
		WithStrings("Test Mfr", "Test Prod", "12345").
		AddConfiguration(1).
		AddInterface(ClassCDC, 0x02, 0x01).
		AddEndpoint(0x81, EndpointTypeBulk, 512).
		AddEndpoint(0x02, EndpointTypeBulk, 512).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if dev.Descriptor.VendorID != 0x1234 {
		t.Errorf("VendorID = 0x%04X, want 0x1234", dev.Descriptor.VendorID)
	}
	if dev.Descriptor.ProductID != 0x5678 {
		t.Errorf("ProductID = 0x%04X, want 0x5678", dev.Descriptor.ProductID)
	}

	config := dev.GetConfiguration(1)
	if config == nil {
		t.Fatal("configuration 1 not found")
	}

	iface := config.GetInterface(0)
	if iface == nil {
		t.Fatal("interface 0 not found")
	}

	if iface.Class != ClassCDC {
		t.Errorf("interface class = 0x%02X, want 0x%02X", iface.Class, ClassCDC)
	}

	if iface.NumEndpoints() != 2 {
		t.Errorf("interface has %d endpoints, want 2", iface.NumEndpoints())
	}
}

func TestDeviceBuilderNoDevice(t *testing.T) {
	_, err := NewDeviceBuilder().
		AddConfiguration(1).
		Build(context.Background())

	if err == nil {
		t.Error("Build() should fail without device initialization")
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestDeviceAddConfigurationEdgeCases(t *testing.T) {
	t.Run("MaxConfigurations", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		for i := uint8(1); i <= MaxConfigurations; i++ {
			config := NewConfiguration(i)
			err := dev.AddConfiguration(config)
			if err != nil {
				t.Fatalf("AddConfiguration(%d) error = %v", i, err)
			}
		}
		// Adding one more should fail
		config := NewConfiguration(MaxConfigurations + 1)
		err := dev.AddConfiguration(config)
		if err != pkg.ErrNoMemory {
			t.Errorf("AddConfiguration() error = %v, want %v", err, pkg.ErrNoMemory)
		}
	})

	t.Run("DuplicateValue", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		config1 := NewConfiguration(1)
		dev.AddConfiguration(config1)
		config2 := NewConfiguration(1) // Same value
		err := dev.AddConfiguration(config2)
		if err != pkg.ErrBusy {
			t.Errorf("AddConfiguration() error = %v, want %v", err, pkg.ErrBusy)
		}
	})
}

func TestDeviceSetAddressEdgeCases(t *testing.T) {
	t.Run("AddressZero", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		dev.Reset()
		dev.SetAddress(5)
		if dev.State() != StateAddress {
			t.Fatalf("State() = %v, want %v", dev.State(), StateAddress)
		}
		// Setting address to 0 should transition to Default
		err := dev.SetAddress(0)
		if err != nil {
			t.Fatalf("SetAddress(0) error = %v", err)
		}
		if dev.State() != StateDefault {
			t.Errorf("State() = %v, want %v", dev.State(), StateDefault)
		}
	})

	t.Run("MaxAddress", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		dev.Reset()
		// Max USB address is 127
		err := dev.SetAddress(127)
		if err != nil {
			t.Fatalf("SetAddress(127) error = %v", err)
		}
		if dev.Address() != 127 {
			t.Errorf("Address() = %d, want 127", dev.Address())
		}
	})

	t.Run("AllInvalidStates", func(t *testing.T) {
		for _, state := range []State{StateAttached, StatePowered, StateConfigured, StateSuspended} {
			t.Run(state.String(), func(t *testing.T) {
				dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
				config := NewConfiguration(1)
				dev.AddConfiguration(config)
				// Manually set state for testing
				dev.mutex.Lock()
				dev.state = state
				dev.mutex.Unlock()

				err := dev.SetAddress(5)
				if state != StateAddress && state != StateDefault {
					if err != pkg.ErrInvalidState {
						t.Errorf("SetAddress() error = %v, want %v", err, pkg.ErrInvalidState)
					}
				}
			})
		}
	})
}

func TestDeviceSetConfigurationEdgeCases(t *testing.T) {
	t.Run("ReconfigureDifferentValue", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		config1 := NewConfiguration(1)
		config2 := NewConfiguration(2)
		dev.AddConfiguration(config1)
		dev.AddConfiguration(config2)
		dev.Reset()
		dev.SetAddress(5)

		// Configure with first config
		if err := dev.SetConfiguration(1); err != nil {
			t.Fatalf("SetConfiguration(1) error = %v", err)
		}
		if dev.ActiveConfiguration() != config1 {
			t.Error("ActiveConfiguration() should be config1")
		}

		// Reconfigure with second config
		if err := dev.SetConfiguration(2); err != nil {
			t.Fatalf("SetConfiguration(2) error = %v", err)
		}
		if dev.ActiveConfiguration() != config2 {
			t.Error("ActiveConfiguration() should be config2")
		}
	})

	t.Run("AllInvalidStates", func(t *testing.T) {
		for _, state := range []State{StateAttached, StatePowered, StateDefault, StateSuspended} {
			t.Run(state.String(), func(t *testing.T) {
				dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
				config := NewConfiguration(1)
				dev.AddConfiguration(config)
				// Manually set state for testing
				dev.mutex.Lock()
				dev.state = state
				dev.mutex.Unlock()

				err := dev.SetConfiguration(1)
				if state != StateAddress && state != StateConfigured {
					if err != pkg.ErrInvalidState {
						t.Errorf("SetConfiguration() error = %v, want %v", err, pkg.ErrInvalidState)
					}
				}
			})
		}
	})
}

func TestDeviceStringEdgeCases(t *testing.T) {
	t.Run("MaxStringIndex", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		// Index >= MaxStrings should be ignored
		var buf [256]byte
		n := StringDescriptorTo(buf[:], "test")
		dev.SetString(MaxStrings, buf[:n]) // Should be silently ignored
		if dev.GetString(MaxStrings) != nil {
			t.Error("GetString(MaxStrings) should return nil")
		}
	})

	t.Run("SetStringFromMaxIndex", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		var buf [256]byte
		n := dev.SetStringFrom(MaxStrings, buf[:], "test")
		if n != 0 {
			t.Errorf("SetStringFrom(MaxStrings) returned %d, want 0", n)
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		var buf [256]byte
		n := dev.SetStringFrom(1, buf[:], "")
		// Empty string should produce minimal descriptor (just header)
		if n > 0 {
			dev.SetString(1, buf[:n])
		}
		str := dev.GetString(1)
		if n > 0 && str == nil {
			t.Error("GetString(1) should not be nil")
		}
	})

	t.Run("LongString", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		// USB string descriptor max is 255 bytes total
		longStr := ""
		for i := 0; i < 200; i++ {
			longStr += "A"
		}
		var buf [512]byte
		n := dev.SetStringFrom(1, buf[:], longStr)
		if n <= 0 {
			t.Fatal("SetStringFrom() returned 0")
		}
		str := dev.GetString(1)
		if str == nil {
			t.Error("GetString(1) should not be nil")
		}
	})
}

func TestDeviceSuspendResumeEdgeCases(t *testing.T) {
	t.Run("SuspendFromDefault", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		dev.Reset()
		if dev.State() != StateDefault {
			t.Fatalf("State() = %v, want %v", dev.State(), StateDefault)
		}

		dev.Suspend()
		if dev.State() != StateSuspended {
			t.Errorf("State() = %v, want %v", dev.State(), StateSuspended)
		}

		dev.Resume()
		if dev.State() != StateDefault {
			t.Errorf("State() = %v, want %v", dev.State(), StateDefault)
		}
	})

	t.Run("SuspendFromAddress", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		dev.Reset()
		dev.SetAddress(5)

		dev.Suspend()
		dev.Resume()
		if dev.State() != StateAddress {
			t.Errorf("State() = %v, want %v", dev.State(), StateAddress)
		}
	})

	t.Run("DoubleSuspend", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		config := NewConfiguration(1)
		dev.AddConfiguration(config)
		dev.Reset()
		dev.SetAddress(5)
		dev.SetConfiguration(1)

		dev.Suspend()
		// Second suspend overwrites previousState with Suspended
		dev.Suspend()
		if dev.State() != StateSuspended {
			t.Errorf("State() = %v, want %v", dev.State(), StateSuspended)
		}

		// Resume will restore previousState which is now Suspended
		// But since Suspended is the current state and Resume checks
		// for Attached/Powered to go to Default, we get Default
		dev.Resume()
		// previousState was set to Suspended by second Suspend call
		// and Suspended is not Attached or Powered, so it restores to Suspended
		// Actually, looking at the code: Resume restores previousState unless
		// it's Attached or Powered, in which case it goes to Default.
		// So double-suspend means previousState=Suspended, which gets restored.
		if dev.State() != StateSuspended {
			t.Errorf("State() = %v, want %v", dev.State(), StateSuspended)
		}
	})

	t.Run("ResumeFromAttached", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		// Suspend from attached
		dev.Suspend()
		dev.Resume()
		// Should go to Default since Attached/Powered are invalid resume states
		if dev.State() != StateDefault {
			t.Errorf("State() = %v, want %v", dev.State(), StateDefault)
		}
	})
}

func TestDeviceCallbacksNil(t *testing.T) {
	// Verify device operations work without callbacks set
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)

	// These should not panic
	dev.Reset()
	dev.SetAddress(5)
	dev.SetConfiguration(1)
	dev.Suspend()
	dev.Resume()
}

func TestDeviceGetEndpointEdgeCases(t *testing.T) {
	t.Run("EP0InVariants", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})

		// Both 0x00 and 0x80 should return EP0
		ep0Out := dev.GetEndpoint(0x00)
		ep0In := dev.GetEndpoint(0x80)
		ctrl := dev.ControlEndpoint()

		if ep0Out != ctrl {
			t.Error("GetEndpoint(0x00) should return EP0")
		}
		if ep0In != ctrl {
			t.Error("GetEndpoint(0x80) should return EP0")
		}
	})

	t.Run("NotConfigured", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		config := NewConfiguration(1)
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
		iface.AddEndpoint(ep)
		config.AddInterface(iface)
		dev.AddConfiguration(config)

		// Device not configured - should return nil for non-control endpoints
		if got := dev.GetEndpoint(0x81); got != nil {
			t.Error("GetEndpoint(0x81) should return nil when not configured")
		}
	})

	t.Run("MultipleInterfaces", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		config := NewConfiguration(1)
		iface0 := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		ep1 := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
		iface0.AddEndpoint(ep1)

		iface1 := NewInterface(&InterfaceDescriptor{InterfaceNumber: 1})
		ep2 := &Endpoint{Address: 0x82, Attributes: EndpointTypeBulk}
		iface1.AddEndpoint(ep2)

		config.AddInterface(iface0)
		config.AddInterface(iface1)
		dev.AddConfiguration(config)
		dev.Reset()
		dev.SetAddress(1)
		dev.SetConfiguration(1)

		// Should find endpoints from different interfaces
		if got := dev.GetEndpoint(0x81); got != ep1 {
			t.Error("GetEndpoint(0x81) returned wrong endpoint")
		}
		if got := dev.GetEndpoint(0x82); got != ep2 {
			t.Error("GetEndpoint(0x82) returned wrong endpoint")
		}
	})
}

func TestDeviceGetStatusEdgeCases(t *testing.T) {
	t.Run("NotConfigured", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		status := dev.GetStatus()
		// Should return 0 when not configured (no active config)
		if status&DeviceStatusSelfPowered != 0 {
			t.Error("status should not indicate self-powered when not configured")
		}
	})

	t.Run("BusPowered", func(t *testing.T) {
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		config := NewConfiguration(1)
		// Don't call SetSelfPowered - defaults to bus-powered
		dev.AddConfiguration(config)
		dev.Reset()
		dev.SetAddress(1)
		dev.SetConfiguration(1)

		status := dev.GetStatus()
		if status&DeviceStatusSelfPowered != 0 {
			t.Error("status should not indicate self-powered")
		}
	})
}

func TestDeviceBuilderEdgeCases(t *testing.T) {
	t.Run("AddInterfaceWithoutConfig", func(t *testing.T) {
		_, err := NewDeviceBuilder().
			WithVendorProduct(0x1234, 0x5678).
			AddInterface(ClassCDC, 0x02, 0x01). // No config added
			Build(context.Background())
		if err == nil {
			t.Error("Build() should fail without configuration")
		}
	})

	t.Run("AddEndpointWithoutInterface", func(t *testing.T) {
		_, err := NewDeviceBuilder().
			WithVendorProduct(0x1234, 0x5678).
			AddConfiguration(1).
			AddEndpoint(0x81, EndpointTypeBulk, 512). // No interface added
			Build(context.Background())
		if err == nil {
			t.Error("Build() should fail without interface")
		}
	})

	t.Run("WithStringsWithoutDevice", func(t *testing.T) {
		_, err := NewDeviceBuilder().
			WithStrings("Mfr", "Prod", "123").
			Build(context.Background())
		if err == nil {
			t.Error("Build() should fail without device")
		}
	})

	t.Run("MultipleConfigurations", func(t *testing.T) {
		dev, err := NewDeviceBuilder().
			WithVendorProduct(0x1234, 0x5678).
			AddConfiguration(1).
			AddInterface(ClassCDC, 0x02, 0x01).
			AddEndpoint(0x81, EndpointTypeBulk, 512).
			AddConfiguration(2).
			AddInterface(ClassHID, 0x01, 0x01).
			AddEndpoint(0x82, EndpointTypeInterrupt, 8).
			Build(context.Background())
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		if dev.Descriptor.NumConfigurations != 2 {
			t.Errorf("NumConfigurations = %d, want 2", dev.Descriptor.NumConfigurations)
		}
		if dev.GetConfiguration(1) == nil {
			t.Error("configuration 1 not found")
		}
		if dev.GetConfiguration(2) == nil {
			t.Error("configuration 2 not found")
		}
	})
}

func TestDeviceConcurrentAccess(t *testing.T) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				_ = dev.State()
				_ = dev.Address()
				_ = dev.Speed()
				_ = dev.IsConfigured()
				_ = dev.IsSuspended()
				_ = dev.IsRemoteWakeupEnabled()
				_ = dev.GetStatus()
				_ = dev.GetEndpoint(0x81)
				_ = dev.GetInterface(0)
				_ = dev.ControlEndpoint()
				_ = dev.ActiveConfiguration()
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

func BenchmarkNewDevice(b *testing.B) {
	desc := &DeviceDescriptor{
		Length:            DeviceDescriptorSize,
		DescriptorType:    DescriptorTypeDevice,
		USBVersion:        0x0200,
		MaxPacketSize0:    64,
		VendorID:          0x1234,
		ProductID:         0x5678,
		NumConfigurations: 1,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDevice(desc)
	}
}

func BenchmarkDeviceAddConfiguration(b *testing.B) {
	desc := &DeviceDescriptor{MaxPacketSize0: 64}
	dev := NewDevice(desc)
	config := NewConfiguration(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.AddConfiguration(config)
	}
}

func BenchmarkDeviceGetConfiguration(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	for i := uint8(1); i <= 5; i++ {
		dev.AddConfiguration(NewConfiguration(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.GetConfiguration(3)
	}
}

func BenchmarkDeviceState(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.State()
	}
}

func BenchmarkDeviceAddress(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	dev.Reset()
	dev.SetAddress(5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.Address()
	}
}

func BenchmarkDeviceSpeed(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.Speed()
	}
}

func BenchmarkDeviceSetAddress(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	dev.Reset()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.SetAddress(uint8(i & 0x7F))
	}
}

func BenchmarkDeviceSetConfiguration(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.SetConfiguration(1)
	}
}

func BenchmarkDeviceReset(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dev.Reset()
	}
}

func BenchmarkDeviceSuspendResume(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dev.Suspend()
		dev.Resume()
	}
}

func BenchmarkDeviceGetEndpoint(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	for addr := uint8(0x81); addr <= 0x84; addr++ {
		iface.AddEndpoint(&Endpoint{Address: addr, Attributes: EndpointTypeBulk})
	}
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	b.Run("EP0", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.GetEndpoint(0)
		}
	})

	b.Run("EP0_IN", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.GetEndpoint(0x80)
		}
	})

	b.Run("BulkIN", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.GetEndpoint(0x81)
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.GetEndpoint(0x8F)
		}
	})
}

func BenchmarkDeviceGetInterface(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	for i := uint8(0); i < 4; i++ {
		config.AddInterface(NewInterface(&InterfaceDescriptor{InterfaceNumber: i}))
	}
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)

	b.Run("First", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.GetInterface(0)
		}
	})

	b.Run("Last", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.GetInterface(3)
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.GetInterface(10)
		}
	})
}

func BenchmarkDeviceGetStatus(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	config.SetSelfPowered(true)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)
	dev.EnableRemoteWakeup(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.GetStatus()
	}
}

func BenchmarkDeviceSetEndpointStall(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
	ep := &Endpoint{Address: 0x81, Attributes: EndpointTypeBulk}
	iface.AddEndpoint(ep)
	config.AddInterface(iface)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.SetEndpointStall(0x81, true)
	}
}

func BenchmarkDeviceControlEndpoint(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.ControlEndpoint()
	}
}

func BenchmarkDeviceSetString(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	var buf [256]byte
	n := StringDescriptorTo(buf[:], "Test String")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dev.SetString(1, buf[:n])
	}
}

func BenchmarkDeviceGetString(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	var buf [256]byte
	n := StringDescriptorTo(buf[:], "Test String")
	dev.SetString(1, buf[:n])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.GetString(1)
	}
}

func BenchmarkDeviceSetStringFrom(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	var buf [256]byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.SetStringFrom(1, buf[:], "Test String")
	}
}

func BenchmarkDeviceRemoteWakeup(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	b.Run("Enable", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dev.EnableRemoteWakeup(true)
		}
	})
	b.Run("IsEnabled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = dev.IsRemoteWakeupEnabled()
		}
	})
}

func BenchmarkDeviceIsConfigured(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.IsConfigured()
	}
}

func BenchmarkDeviceIsSuspended(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	dev.Suspend()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.IsSuspended()
	}
}

func BenchmarkDeviceActiveConfiguration(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	config := NewConfiguration(1)
	dev.AddConfiguration(config)
	dev.Reset()
	dev.SetAddress(1)
	dev.SetConfiguration(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dev.ActiveConfiguration()
	}
}

func BenchmarkDeviceSetCallbacks(b *testing.B) {
	dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
	cb := func() {}
	cbState := func(old, new State) {}
	cbAddr := func(addr uint8) {}
	cbCfg := func(cfg uint8) {}

	b.Run("OnStateChange", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dev.SetOnStateChange(cbState)
		}
	})
	b.Run("OnSuspend", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dev.SetOnSuspend(cb)
		}
	})
	b.Run("OnResume", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dev.SetOnResume(cb)
		}
	})
	b.Run("OnReset", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dev.SetOnReset(cb)
		}
	})
	b.Run("OnSetAddress", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dev.SetOnSetAddress(cbAddr)
		}
	})
	b.Run("OnSetConfiguration", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			dev.SetOnSetConfiguration(cbCfg)
		}
	})
}

func BenchmarkDeviceBuilder(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewDeviceBuilder().
			WithVendorProduct(0x1234, 0x5678).
			WithStrings("Manufacturer", "Product", "Serial").
			AddConfiguration(1).
			AddInterface(ClassCDC, 0x02, 0x01).
			AddEndpoint(0x81, EndpointTypeBulk, 512).
			AddEndpoint(0x02, EndpointTypeBulk, 512).
			Build(ctx)
	}
}

func BenchmarkDeviceClose(b *testing.B) {
	time.Sleep(6 * 60 * time.Second)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		dev := NewDevice(&DeviceDescriptor{MaxPacketSize0: 64})
		config := NewConfiguration(1)
		iface := NewInterface(&InterfaceDescriptor{InterfaceNumber: 0})
		config.AddInterface(iface)
		dev.AddConfiguration(config)
		b.StartTimer()
		_ = dev.Close()
	}
}
