package device

import (
	"context"
	"sync"

	"github.com/ardnew/softusb/pkg"
)

// Device represents a USB device.
type Device struct {
	// Device descriptor
	Descriptor *DeviceDescriptor

	// Configurations - fixed-size array for zero allocation
	configurations     [MaxConfigurations]*Configuration
	configurationCount int
	activeConfig       *Configuration

	// String descriptors - fixed-size array, each entry is a slice reference
	strings [MaxStrings][]byte

	// Device state
	state         State
	previousState State // State before suspend
	address       uint8
	speed         Speed

	// Control endpoint
	ep0 *Endpoint

	// Remote wakeup enabled
	remoteWakeupEnabled bool

	// Synchronization
	mutex sync.RWMutex

	// Event callbacks
	onStateChange      func(old, new State)
	onSuspend          func()
	onResume           func()
	onReset            func()
	onSetAddress       func(address uint8)
	onSetConfiguration func(config uint8)
}

// NewDevice creates a new USB device.
func NewDevice(desc *DeviceDescriptor) *Device {
	return &Device{
		Descriptor: desc,
		state:      StateAttached,
		speed:      SpeedFull,
		ep0: &Endpoint{
			Address:       0x00,
			Attributes:    EndpointTypeControl,
			MaxPacketSize: uint16(desc.MaxPacketSize0),
		},
	}
}

// AddConfiguration adds a configuration to the device.
func (d *Device) AddConfiguration(config *Configuration) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.configurationCount >= MaxConfigurations {
		return pkg.ErrNoMemory
	}

	// Check for duplicate configuration value
	for idx := 0; idx < d.configurationCount; idx++ {
		if d.configurations[idx].Value == config.Value {
			return pkg.ErrBusy
		}
	}

	d.configurations[d.configurationCount] = config
	d.configurationCount++

	pkg.LogDebug(pkg.ComponentDevice, "configuration added",
		"value", config.Value)

	return nil
}

// GetConfiguration returns the configuration with the given value.
func (d *Device) GetConfiguration(value uint8) *Configuration {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for idx := 0; idx < d.configurationCount; idx++ {
		if d.configurations[idx].Value == value {
			return d.configurations[idx]
		}
	}
	return nil
}

// ActiveConfiguration returns the currently active configuration.
func (d *Device) ActiveConfiguration() *Configuration {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.activeConfig
}

// SetString sets a string descriptor from a pre-encoded descriptor.
// The data slice is stored by reference (not copied).
func (d *Device) SetString(index uint8, data []byte) {
	if index >= MaxStrings {
		return
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.strings[index] = data
}

// SetStringFrom encodes a string as a USB string descriptor into buf
// and stores the resulting slice at the given index.
// Returns the number of bytes written.
func (d *Device) SetStringFrom(index uint8, buf []byte, s string) int {
	if index >= MaxStrings {
		return 0
	}
	n := StringDescriptorTo(buf, s)
	if n > 0 {
		d.mutex.Lock()
		d.strings[index] = buf[:n]
		d.mutex.Unlock()
	}
	return n
}

// SetLanguages sets the supported language IDs (index 0).
// The data slice is stored by reference (not copied).
func (d *Device) SetLanguages(data []byte) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.strings[0] = data
}

// SetLanguagesFrom encodes language IDs as a USB string descriptor into buf
// and stores the resulting slice at index 0.
// Returns the number of bytes written.
func (d *Device) SetLanguagesFrom(buf []byte, langIDs ...uint16) int {
	n := LanguageDescriptorTo(buf, langIDs...)
	if n > 0 {
		d.mutex.Lock()
		d.strings[0] = buf[:n]
		d.mutex.Unlock()
	}
	return n
}

// GetString returns a string descriptor by index.
func (d *Device) GetString(index uint8) []byte {
	if index >= MaxStrings {
		return nil
	}
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.strings[index]
}

// State returns the current device state.
func (d *Device) State() State {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.state
}

// setState changes the device state and triggers callback.
func (d *Device) setState(newState State) {
	d.mutex.Lock()
	oldState := d.state
	d.state = newState
	callback := d.onStateChange
	d.mutex.Unlock()

	if oldState != newState {
		pkg.LogDebug(pkg.ComponentDevice, "device state changed",
			"from", oldState.String(),
			"to", newState.String())
		if callback != nil {
			callback(oldState, newState)
		}
	}
}

// Address returns the device address.
func (d *Device) Address() uint8 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.address
}

// Speed returns the device speed.
func (d *Device) Speed() Speed {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.speed
}

// SetSpeed sets the device speed.
func (d *Device) SetSpeed(speed Speed) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.speed = speed
}

// ControlEndpoint returns the control endpoint (EP0).
func (d *Device) ControlEndpoint() *Endpoint {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.ep0
}

// IsConfigured returns true if the device is configured.
func (d *Device) IsConfigured() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.state == StateConfigured
}

// IsSuspended returns true if the device is suspended.
func (d *Device) IsSuspended() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.state == StateSuspended
}

// Reset handles a bus reset.
func (d *Device) Reset() {
	d.mutex.Lock()
	d.address = 0
	d.activeConfig = nil
	d.remoteWakeupEnabled = false
	callback := d.onReset
	d.mutex.Unlock()

	d.setState(StateDefault)

	if callback != nil {
		callback()
	}

	pkg.LogDebug(pkg.ComponentDevice, "device reset")
}

// SetAddress handles SET_ADDRESS request.
func (d *Device) SetAddress(address uint8) error {
	d.mutex.Lock()
	if d.state != StateDefault && d.state != StateAddress {
		d.mutex.Unlock()
		return pkg.ErrInvalidState
	}
	d.address = address
	callback := d.onSetAddress
	d.mutex.Unlock()

	if address == 0 {
		d.setState(StateDefault)
	} else {
		d.setState(StateAddress)
	}

	if callback != nil {
		callback(address)
	}

	pkg.LogDebug(pkg.ComponentDevice, "device address set",
		"address", address)

	return nil
}

// SetConfiguration handles SET_CONFIGURATION request.
func (d *Device) SetConfiguration(value uint8) error {
	d.mutex.Lock()
	if d.state != StateAddress && d.state != StateConfigured {
		d.mutex.Unlock()
		return pkg.ErrInvalidState
	}

	if value == 0 {
		// Unconfigure device
		d.activeConfig = nil
		d.mutex.Unlock()
		d.setState(StateAddress)
		return nil
	}

	// Find configuration by value
	var config *Configuration
	for idx := 0; idx < d.configurationCount; idx++ {
		if d.configurations[idx].Value == value {
			config = d.configurations[idx]
			break
		}
	}
	if config == nil {
		d.mutex.Unlock()
		return pkg.ErrInvalidRequest
	}

	d.activeConfig = config
	callback := d.onSetConfiguration
	d.mutex.Unlock()

	d.setState(StateConfigured)

	if callback != nil {
		callback(value)
	}

	pkg.LogDebug(pkg.ComponentDevice, "device configured",
		"configuration", value)

	return nil
}

// Suspend handles USB suspend.
func (d *Device) Suspend() {
	d.mutex.Lock()
	d.previousState = d.state
	callback := d.onSuspend
	d.mutex.Unlock()

	d.setState(StateSuspended)

	if callback != nil {
		callback()
	}

	pkg.LogDebug(pkg.ComponentDevice, "device suspended")
}

// Resume handles USB resume.
func (d *Device) Resume() {
	d.mutex.Lock()
	previousState := d.previousState
	callback := d.onResume
	d.mutex.Unlock()

	if previousState != StateAttached && previousState != StatePowered {
		d.setState(previousState)
	} else {
		d.setState(StateDefault)
	}

	if callback != nil {
		callback()
	}

	pkg.LogDebug(pkg.ComponentDevice, "device resumed")
}

// EnableRemoteWakeup enables remote wakeup capability.
func (d *Device) EnableRemoteWakeup(enabled bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.remoteWakeupEnabled = enabled
}

// IsRemoteWakeupEnabled returns true if remote wakeup is enabled.
func (d *Device) IsRemoteWakeupEnabled() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.remoteWakeupEnabled
}

// GetInterface returns an interface from the active configuration.
func (d *Device) GetInterface(number uint8) *Interface {
	d.mutex.RLock()
	config := d.activeConfig
	d.mutex.RUnlock()

	if config == nil {
		return nil
	}
	return config.GetInterface(number)
}

// GetEndpoint returns an endpoint from the active configuration.
func (d *Device) GetEndpoint(address uint8) *Endpoint {
	if address == 0 || address == 0x80 {
		return d.ControlEndpoint()
	}

	d.mutex.RLock()
	config := d.activeConfig
	d.mutex.RUnlock()

	if config == nil {
		return nil
	}

	for _, iface := range config.Interfaces() {
		if ep := iface.GetEndpoint(address); ep != nil {
			return ep
		}
	}
	return nil
}

// SetEndpointStall sets or clears the stall condition on an endpoint.
func (d *Device) SetEndpointStall(address uint8, stalled bool) error {
	ep := d.GetEndpoint(address)
	if ep == nil {
		return pkg.ErrInvalidEndpoint
	}
	ep.SetStall(stalled)
	return nil
}

// SetOnStateChange sets the state change callback.
func (d *Device) SetOnStateChange(cb func(old, new State)) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.onStateChange = cb
}

// SetOnSuspend sets the suspend callback.
func (d *Device) SetOnSuspend(cb func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.onSuspend = cb
}

// SetOnResume sets the resume callback.
func (d *Device) SetOnResume(cb func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.onResume = cb
}

// SetOnReset sets the reset callback.
func (d *Device) SetOnReset(cb func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.onReset = cb
}

// SetOnSetAddress sets the set address callback.
func (d *Device) SetOnSetAddress(cb func(address uint8)) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.onSetAddress = cb
}

// SetOnSetConfiguration sets the set configuration callback.
func (d *Device) SetOnSetConfiguration(cb func(config uint8)) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.onSetConfiguration = cb
}

// Close releases resources held by the device.
func (d *Device) Close() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	var lastErr error
	for idx := 0; idx < d.configurationCount; idx++ {
		if err := d.configurations[idx].Close(); err != nil {
			lastErr = err
		}
		d.configurations[idx] = nil
	}
	d.configurationCount = 0
	d.activeConfig = nil
	return lastErr
}

// DeviceStatus represents the device status bits.
type DeviceStatus uint16

// Device status bits.
const (
	DeviceStatusSelfPowered  DeviceStatus = 1 << 0 // Device is self-powered
	DeviceStatusRemoteWakeup DeviceStatus = 1 << 1 // Remote wakeup enabled
)

// GetStatus returns the device status.
func (d *Device) GetStatus() DeviceStatus {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var status DeviceStatus
	if d.activeConfig != nil && d.activeConfig.IsSelfPowered() {
		status |= DeviceStatusSelfPowered
	}
	if d.remoteWakeupEnabled {
		status |= DeviceStatusRemoteWakeup
	}
	return status
}

// DeviceBuilder provides a fluent API for building devices.
type DeviceBuilder struct {
	device *Device
	config *Configuration
	iface  *Interface
	errors []error

	// Pre-allocated string buffers
	stringBufs [MaxStrings][256]byte
}

// NewDeviceBuilder creates a new device builder.
func NewDeviceBuilder() *DeviceBuilder {
	return &DeviceBuilder{}
}

// WithDescriptor sets the device descriptor.
func (b *DeviceBuilder) WithDescriptor(desc *DeviceDescriptor) *DeviceBuilder {
	b.device = NewDevice(desc)
	return b
}

// WithVendorProduct sets vendor and product IDs.
func (b *DeviceBuilder) WithVendorProduct(vendorID, productID uint16) *DeviceBuilder {
	if b.device == nil {
		b.device = NewDevice(&DeviceDescriptor{
			Length:         DeviceDescriptorSize,
			DescriptorType: DescriptorTypeDevice,
			USBVersion:     0x0200,
			MaxPacketSize0: 64,
		})
	}
	b.device.Descriptor.VendorID = vendorID
	b.device.Descriptor.ProductID = productID
	return b
}

// WithStrings sets the manufacturer, product, and serial strings.
func (b *DeviceBuilder) WithStrings(manufacturer, product, serial string) *DeviceBuilder {
	if b.device == nil {
		b.errors = append(b.errors, pkg.ErrInvalidState)
		return b
	}
	b.device.SetLanguagesFrom(b.stringBufs[0][:], LangIDUSEnglish)
	if manufacturer != "" {
		b.device.Descriptor.ManufacturerIndex = 1
		b.device.SetStringFrom(1, b.stringBufs[1][:], manufacturer)
	}
	if product != "" {
		b.device.Descriptor.ProductIndex = 2
		b.device.SetStringFrom(2, b.stringBufs[2][:], product)
	}
	if serial != "" {
		b.device.Descriptor.SerialNumberIndex = 3
		b.device.SetStringFrom(3, b.stringBufs[3][:], serial)
	}
	return b
}

// AddConfiguration adds a new configuration.
func (b *DeviceBuilder) AddConfiguration(value uint8) *DeviceBuilder {
	if b.device == nil {
		b.errors = append(b.errors, pkg.ErrInvalidState)
		return b
	}
	b.config = NewConfiguration(value)
	if err := b.device.AddConfiguration(b.config); err != nil {
		b.errors = append(b.errors, err)
	}
	b.device.Descriptor.NumConfigurations++
	return b
}

// AddInterface adds a new interface to the current configuration.
func (b *DeviceBuilder) AddInterface(class, subClass, protocol uint8) *DeviceBuilder {
	if b.config == nil {
		b.errors = append(b.errors, pkg.ErrInvalidState)
		return b
	}
	num := uint8(b.config.NumInterfaces())
	b.iface = NewInterface(&InterfaceDescriptor{
		Length:            InterfaceDescriptorSize,
		DescriptorType:    DescriptorTypeInterface,
		InterfaceNumber:   num,
		InterfaceClass:    class,
		InterfaceSubClass: subClass,
		InterfaceProtocol: protocol,
	})
	if err := b.config.AddInterface(b.iface); err != nil {
		b.errors = append(b.errors, err)
	}
	return b
}

// AddEndpoint adds an endpoint to the current interface.
func (b *DeviceBuilder) AddEndpoint(address uint8, transferType uint8, maxPacketSize uint16) *DeviceBuilder {
	if b.iface == nil {
		b.errors = append(b.errors, pkg.ErrInvalidState)
		return b
	}
	ep := &Endpoint{
		Address:       address,
		Attributes:    transferType,
		MaxPacketSize: maxPacketSize,
	}
	if err := b.iface.AddEndpoint(ep); err != nil {
		b.errors = append(b.errors, err)
	}
	return b
}

// Build returns the constructed device.
func (b *DeviceBuilder) Build(ctx context.Context) (*Device, error) {
	if len(b.errors) > 0 {
		return nil, b.errors[0]
	}
	if b.device == nil {
		return nil, pkg.ErrInvalidState
	}
	return b.device, nil
}
