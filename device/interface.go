package device

import (
	"sync"

	"github.com/ardnew/softusb/pkg"
)

// Interface represents a USB interface within a configuration.
type Interface struct {
	// Descriptor data
	Number           uint8 // Interface number
	AlternateSetting uint8 // Current alternate setting
	Class            uint8 // Interface class
	SubClass         uint8 // Interface subclass
	Protocol         uint8 // Interface protocol

	// Endpoints (excluding EP0) - fixed-size array for zero allocation
	endpoints     [MaxEndpointsPerInterface]*Endpoint
	endpointCount int
	mutex         sync.RWMutex

	// Class driver
	classDriver ClassDriver

	// String descriptor index
	StringIndex uint8
}

// ClassDriver defines the interface for USB class-specific handling.
type ClassDriver interface {
	// Init initializes the class driver for the interface.
	Init(iface *Interface) error

	// HandleSetup processes class-specific SETUP requests.
	// Returns true if the request was handled, false otherwise.
	HandleSetup(iface *Interface, setup *SetupPacket, data []byte) (bool, error)

	// SetAlternate is called when the alternate setting changes.
	SetAlternate(iface *Interface, alt uint8) error

	// Close releases any resources held by the class driver.
	Close() error
}

// NewInterface creates a new interface from a descriptor.
func NewInterface(desc *InterfaceDescriptor) *Interface {
	return &Interface{
		Number:           desc.InterfaceNumber,
		AlternateSetting: desc.AlternateSetting,
		Class:            desc.InterfaceClass,
		SubClass:         desc.InterfaceSubClass,
		Protocol:         desc.InterfaceProtocol,
		StringIndex:      desc.InterfaceIndex,
	}
}

// AddEndpoint adds an endpoint to the interface.
func (i *Interface) AddEndpoint(ep *Endpoint) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.endpointCount >= MaxEndpointsPerInterface {
		return pkg.ErrNoMemory
	}

	// Check for duplicate address
	addr := ep.Address
	for idx := 0; idx < i.endpointCount; idx++ {
		if i.endpoints[idx].Address == addr {
			return pkg.ErrBusy
		}
	}

	i.endpoints[i.endpointCount] = ep
	i.endpointCount++

	pkg.LogDebug(pkg.ComponentDevice, "endpoint added to interface",
		"interface", i.Number,
		"endpoint", addr,
		"type", TransferTypeName(ep.TransferType()),
		"direction", DirectionName(ep.Direction()))

	return nil
}

// RemoveEndpoint removes an endpoint from the interface.
func (i *Interface) RemoveEndpoint(address uint8) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	for idx := 0; idx < i.endpointCount; idx++ {
		if i.endpoints[idx].Address == address {
			// Shift remaining endpoints down
			for j := idx; j < i.endpointCount-1; j++ {
				i.endpoints[j] = i.endpoints[j+1]
			}
			i.endpoints[i.endpointCount-1] = nil
			i.endpointCount--
			return
		}
	}
}

// GetEndpoint returns the endpoint with the given address.
func (i *Interface) GetEndpoint(address uint8) *Endpoint {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	for idx := 0; idx < i.endpointCount; idx++ {
		if i.endpoints[idx].Address == address {
			return i.endpoints[idx]
		}
	}
	return nil
}

// Endpoints returns all endpoints in the interface.
// The returned slice references internal storage; do not modify.
func (i *Interface) Endpoints() []*Endpoint {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.endpoints[:i.endpointCount]
}

// NumEndpoints returns the number of endpoints in the interface.
func (i *Interface) NumEndpoints() int {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.endpointCount
}

// GetInEndpoint returns an IN endpoint by number.
func (i *Interface) GetInEndpoint(num uint8) *Endpoint {
	return i.GetEndpoint(num | EndpointDirectionIn)
}

// GetOutEndpoint returns an OUT endpoint by number.
func (i *Interface) GetOutEndpoint(num uint8) *Endpoint {
	return i.GetEndpoint(num & 0x0F)
}

// SetClassDriver sets the class driver for this interface.
func (i *Interface) SetClassDriver(driver ClassDriver) error {
	i.mutex.Lock()
	oldDriver := i.classDriver
	i.classDriver = driver
	i.mutex.Unlock()

	// Close old driver outside the lock
	if oldDriver != nil {
		if err := oldDriver.Close(); err != nil {
			pkg.LogWarn(pkg.ComponentDevice, "error closing previous class driver",
				"error", err)
		}
	}

	// Initialize new driver outside the lock to avoid re-entrant locking
	// when driver callbacks access interface methods
	if driver != nil {
		return driver.Init(i)
	}
	return nil
}

// ClassDriver returns the current class driver.
func (i *Interface) ClassDriver() ClassDriver {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.classDriver
}

// HandleSetup processes a class-specific SETUP request.
func (i *Interface) HandleSetup(setup *SetupPacket, data []byte) (bool, error) {
	i.mutex.RLock()
	driver := i.classDriver
	i.mutex.RUnlock()

	if driver == nil {
		return false, nil
	}
	return driver.HandleSetup(i, setup, data)
}

// SetAlternate changes the alternate setting.
func (i *Interface) SetAlternate(alt uint8) error {
	i.mutex.Lock()
	i.AlternateSetting = alt
	driver := i.classDriver
	i.mutex.Unlock()

	if driver != nil {
		return driver.SetAlternate(i, alt)
	}
	return nil
}

// Descriptor returns the interface descriptor.
func (i *Interface) Descriptor() *InterfaceDescriptor {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return &InterfaceDescriptor{
		Length:            InterfaceDescriptorSize,
		DescriptorType:    DescriptorTypeInterface,
		InterfaceNumber:   i.Number,
		AlternateSetting:  i.AlternateSetting,
		NumEndpoints:      uint8(i.endpointCount),
		InterfaceClass:    i.Class,
		InterfaceSubClass: i.SubClass,
		InterfaceProtocol: i.Protocol,
		InterfaceIndex:    i.StringIndex,
	}
}

// Close releases resources held by the interface.
func (i *Interface) Close() error {
	i.mutex.Lock()
	driver := i.classDriver
	i.classDriver = nil
	i.mutex.Unlock()

	if driver != nil {
		return driver.Close()
	}
	return nil
}

// MaxAssociationsPerConfiguration is the maximum number of IADs per configuration.
const MaxAssociationsPerConfiguration = 4

// Configuration represents a USB device configuration.
type Configuration struct {
	// Descriptor data
	Value       uint8 // Configuration value for SET_CONFIGURATION
	Attributes  uint8 // Configuration attributes (bus/self powered, remote wakeup)
	MaxPower    uint8 // Maximum power consumption (2mA units)
	StringIndex uint8 // String descriptor index

	// Interfaces - fixed-size array for zero allocation
	interfaces     [MaxInterfacesPerConfiguration]*Interface
	interfaceCount int
	mutex          sync.RWMutex

	// Interface associations (for composite devices) - fixed-size array
	associations     [MaxAssociationsPerConfiguration]InterfaceAssociation
	associationCount int
}

// InterfaceAssociation groups related interfaces (e.g., CDC control + data).
type InterfaceAssociation struct {
	FirstInterface   uint8
	InterfaceCount   uint8
	FunctionClass    uint8
	FunctionSubClass uint8
	FunctionProtocol uint8
	StringIndex      uint8
}

// NewConfiguration creates a new configuration.
func NewConfiguration(value uint8) *Configuration {
	return &Configuration{
		Value:      value,
		Attributes: ConfigAttrBusPowered,
		MaxPower:   50, // 100mA default
	}
}

// AddInterface adds an interface to the configuration.
func (c *Configuration) AddInterface(iface *Interface) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.interfaceCount >= MaxInterfacesPerConfiguration {
		return pkg.ErrNoMemory
	}

	// Check for duplicate interface number
	for idx := 0; idx < c.interfaceCount; idx++ {
		if c.interfaces[idx].Number == iface.Number {
			return pkg.ErrBusy
		}
	}

	c.interfaces[c.interfaceCount] = iface
	c.interfaceCount++

	pkg.LogDebug(pkg.ComponentDevice, "interface added to configuration",
		"config", c.Value,
		"interface", iface.Number)

	return nil
}

// RemoveInterface removes an interface from the configuration.
func (c *Configuration) RemoveInterface(number uint8) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for idx := 0; idx < c.interfaceCount; idx++ {
		if c.interfaces[idx].Number == number {
			c.interfaces[idx].Close()
			// Shift remaining interfaces down
			for j := idx; j < c.interfaceCount-1; j++ {
				c.interfaces[j] = c.interfaces[j+1]
			}
			c.interfaces[c.interfaceCount-1] = nil
			c.interfaceCount--
			return
		}
	}
}

// GetInterface returns the interface with the given number.
func (c *Configuration) GetInterface(number uint8) *Interface {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for idx := 0; idx < c.interfaceCount; idx++ {
		if c.interfaces[idx].Number == number {
			return c.interfaces[idx]
		}
	}
	return nil
}

// Interfaces returns all interfaces in the configuration.
// The returned slice references internal storage; do not modify.
func (c *Configuration) Interfaces() []*Interface {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.interfaces[:c.interfaceCount]
}

// NumInterfaces returns the number of interfaces.
func (c *Configuration) NumInterfaces() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.interfaceCount
}

// AddAssociation adds an interface association (for composite devices).
func (c *Configuration) AddAssociation(assoc *InterfaceAssociation) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.associationCount >= MaxAssociationsPerConfiguration {
		return pkg.ErrNoMemory
	}

	c.associations[c.associationCount] = *assoc
	c.associationCount++
	return nil
}

// Associations returns all interface associations.
// The returned slice references internal storage; do not modify.
func (c *Configuration) Associations() []InterfaceAssociation {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.associations[:c.associationCount]
}

// Descriptor returns the configuration descriptor.
func (c *Configuration) Descriptor() *ConfigurationDescriptor {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return &ConfigurationDescriptor{
		Length:             ConfigurationDescriptorSize,
		DescriptorType:     DescriptorTypeConfiguration,
		TotalLength:        c.calculateTotalLength(),
		NumInterfaces:      uint8(c.interfaceCount),
		ConfigurationValue: c.Value,
		ConfigurationIndex: c.StringIndex,
		Attributes:         c.Attributes,
		MaxPower:           c.MaxPower,
	}
}

// calculateTotalLength calculates the total configuration descriptor length.
func (c *Configuration) calculateTotalLength() uint16 {
	length := uint16(ConfigurationDescriptorSize) // Configuration descriptor

	// Add IAD lengths
	length += uint16(c.associationCount) * IADSize

	// Add interface and endpoint lengths
	for idx := 0; idx < c.interfaceCount; idx++ {
		iface := c.interfaces[idx]
		length += InterfaceDescriptorSize                               // Interface descriptor
		length += uint16(iface.NumEndpoints()) * EndpointDescriptorSize // Endpoint descriptors
	}

	return length
}

// MarshalTo writes the full configuration descriptor including all sub-descriptors to buf.
// Returns the number of bytes written.
func (c *Configuration) MarshalTo(buf []byte) int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	offset := 0

	// Configuration descriptor
	n := c.Descriptor().MarshalTo(buf[offset:])
	if n == 0 {
		return 0
	}
	offset += n

	// Interface associations (must come before interfaces)
	for idx := 0; idx < c.associationCount; idx++ {
		assoc := &c.associations[idx]
		iad := InterfaceAssociationDescriptor{
			Length:           IADSize,
			DescriptorType:   DescriptorTypeInterfaceAssociation,
			FirstInterface:   assoc.FirstInterface,
			InterfaceCount:   assoc.InterfaceCount,
			FunctionClass:    assoc.FunctionClass,
			FunctionSubClass: assoc.FunctionSubClass,
			FunctionProtocol: assoc.FunctionProtocol,
			FunctionIndex:    assoc.StringIndex,
		}
		n = iad.MarshalTo(buf[offset:])
		if n == 0 {
			return 0
		}
		offset += n
	}

	// Interfaces and their endpoints
	for idx := 0; idx < c.interfaceCount; idx++ {
		iface := c.interfaces[idx]
		n = iface.Descriptor().MarshalTo(buf[offset:])
		if n == 0 {
			return 0
		}
		offset += n

		for _, ep := range iface.Endpoints() {
			n = ep.Descriptor().MarshalTo(buf[offset:])
			if n == 0 {
				return 0
			}
			offset += n
		}
	}

	return offset
}

// SetSelfPowered sets or clears the self-powered attribute.
func (c *Configuration) SetSelfPowered(selfPowered bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if selfPowered {
		c.Attributes |= ConfigAttrSelfPowered
	} else {
		c.Attributes &^= ConfigAttrSelfPowered
	}
}

// IsSelfPowered returns true if the configuration is self-powered.
func (c *Configuration) IsSelfPowered() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Attributes&ConfigAttrSelfPowered != 0
}

// SetRemoteWakeup sets or clears the remote wakeup capability.
func (c *Configuration) SetRemoteWakeup(enabled bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if enabled {
		c.Attributes |= ConfigAttrRemoteWakeup
	} else {
		c.Attributes &^= ConfigAttrRemoteWakeup
	}
}

// SupportsRemoteWakeup returns true if remote wakeup is supported.
func (c *Configuration) SupportsRemoteWakeup() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Attributes&ConfigAttrRemoteWakeup != 0
}

// Close releases resources held by the configuration.
func (c *Configuration) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var lastErr error
	for idx := 0; idx < c.interfaceCount; idx++ {
		if err := c.interfaces[idx].Close(); err != nil {
			lastErr = err
		}
		c.interfaces[idx] = nil
	}
	c.interfaceCount = 0
	return lastErr
}
