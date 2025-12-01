package host

import (
	"context"
	"sync"

	"github.com/ardnew/softusb/host/hal"
)

// Device represents a connected USB device from the host's perspective.
type Device struct {
	host    *Host
	address uint8
	port    int
	speed   hal.Speed

	// Device descriptor
	descriptor DeviceDescriptor

	// Configuration descriptor (current)
	config ConfigurationDescriptor

	// Interface descriptors (current configuration)
	interfaces []InterfaceDescriptor

	// Endpoint descriptors (current configuration)
	endpoints []EndpointDescriptor

	// Current configuration value
	configurationValue uint8

	// State
	state DeviceState
	mutex sync.RWMutex

	// String descriptors cache (indexed by string index)
	strings [MaxStringsPerDevice]string

	// Class-specific descriptors per interface
	classDescriptors [MaxInterfacesPerConfiguration][][]byte
}

// newDevice creates a new device instance.
func newDevice(host *Host, port int, address uint8, speed hal.Speed) *Device {
	return &Device{
		host:    host,
		address: address,
		port:    port,
		speed:   speed,
		state:   DeviceStateDefault,
	}
}

// Address returns the device address.
func (d *Device) Address() uint8 {
	return d.address
}

// Port returns the port number the device is connected to.
func (d *Device) Port() int {
	return d.port
}

// Speed returns the device speed.
func (d *Device) Speed() hal.Speed {
	return d.speed
}

// VendorID returns the device vendor ID.
func (d *Device) VendorID() uint16 {
	return d.descriptor.VendorID
}

// ProductID returns the device product ID.
func (d *Device) ProductID() uint16 {
	return d.descriptor.ProductID
}

// DeviceClass returns the device class.
func (d *Device) DeviceClass() uint8 {
	return d.descriptor.DeviceClass
}

// DeviceSubClass returns the device subclass.
func (d *Device) DeviceSubClass() uint8 {
	return d.descriptor.DeviceSubClass
}

// DeviceProtocol returns the device protocol.
func (d *Device) DeviceProtocol() uint8 {
	return d.descriptor.DeviceProtocol
}

// Descriptor returns the device descriptor.
func (d *Device) Descriptor() DeviceDescriptor {
	return d.descriptor
}

// Configuration returns the current configuration descriptor.
func (d *Device) Configuration() ConfigurationDescriptor {
	return d.config
}

// Interfaces returns the interface descriptors for the current configuration.
// The returned slice references internal storage; do not modify.
func (d *Device) Interfaces() []InterfaceDescriptor {
	return d.interfaces
}

// Endpoints returns the endpoint descriptors for the current configuration.
// The returned slice references internal storage; do not modify.
func (d *Device) Endpoints() []EndpointDescriptor {
	return d.endpoints
}

// GetInterface returns the interface descriptor for the given interface number.
func (d *Device) GetInterface(num uint8) *InterfaceDescriptor {
	for i := range d.interfaces {
		if d.interfaces[i].InterfaceNumber == num {
			return &d.interfaces[i]
		}
	}
	return nil
}

// GetEndpoint returns the endpoint descriptor for the given address.
func (d *Device) GetEndpoint(address uint8) *EndpointDescriptor {
	for i := range d.endpoints {
		if d.endpoints[i].EndpointAddress == address {
			return &d.endpoints[i]
		}
	}
	return nil
}

// GetString returns a cached string descriptor.
func (d *Device) GetString(index uint8) string {
	if index == 0 || int(index) >= len(d.strings) {
		return ""
	}
	return d.strings[index]
}

// Manufacturer returns the manufacturer string.
func (d *Device) Manufacturer() string {
	return d.GetString(d.descriptor.ManufacturerIndex)
}

// Product returns the product string.
func (d *Device) Product() string {
	return d.GetString(d.descriptor.ProductIndex)
}

// SerialNumber returns the serial number string.
func (d *Device) SerialNumber() string {
	return d.GetString(d.descriptor.SerialNumberIndex)
}

// State returns the current device state.
func (d *Device) State() DeviceState {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.state
}

// SetConfiguration sets the device configuration.
func (d *Device) SetConfiguration(ctx context.Context, value uint8) error {
	setup := hal.SetupPacket{
		RequestType: RequestTypeOut | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestSetConfiguration,
		Value:       uint16(value),
		Index:       0,
		Length:      0,
	}

	_, err := d.host.hal.ControlTransfer(ctx, hal.DeviceAddress(d.address), &setup, nil)
	if err != nil {
		return err
	}

	d.mutex.Lock()
	d.configurationValue = value
	if value > 0 {
		d.state = DeviceStateConfigured
	} else {
		d.state = DeviceStateAddress
	}
	d.mutex.Unlock()

	return nil
}

// GetConfiguration returns the current configuration value.
func (d *Device) GetConfiguration() uint8 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.configurationValue
}

// ControlTransfer performs a control transfer to the device.
func (d *Device) ControlTransfer(ctx context.Context, setup *hal.SetupPacket, data []byte) (int, error) {
	return d.host.hal.ControlTransfer(ctx, hal.DeviceAddress(d.address), setup, data)
}

// BulkTransfer performs a bulk transfer.
func (d *Device) BulkTransfer(ctx context.Context, endpoint uint8, data []byte) (int, error) {
	return d.host.hal.BulkTransfer(ctx, hal.DeviceAddress(d.address), endpoint, data)
}

// InterruptTransfer performs an interrupt transfer.
func (d *Device) InterruptTransfer(ctx context.Context, endpoint uint8, data []byte) (int, error) {
	return d.host.hal.InterruptTransfer(ctx, hal.DeviceAddress(d.address), endpoint, data)
}

// Close closes the device.
func (d *Device) Close() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.state = DeviceStateDetached
	return nil
}

// parseDeviceDescriptor parses a device descriptor from raw bytes.
// Returns true if successful.
func (d *Device) parseDeviceDescriptor(data []byte) bool {
	return ParseDeviceDescriptor(data, &d.descriptor)
}

// parseConfigurationTree parses the full configuration descriptor tree.
func (d *Device) parseConfigurationTree(data []byte) {
	if len(data) < ConfigurationDescriptorSize {
		return
	}

	// Parse configuration descriptor header
	if !ParseConfigurationDescriptor(data, &d.config) {
		return
	}

	// Allocate space for interfaces and endpoints
	d.interfaces = make([]InterfaceDescriptor, 0, d.config.NumInterfaces)
	d.endpoints = make([]EndpointDescriptor, 0, MaxEndpointsPerInterface)

	// Parse child descriptors
	offset := ConfigurationDescriptorSize
	currentIfaceIdx := -1

	for offset < len(data) && offset < int(d.config.TotalLength) {
		if offset+2 > len(data) {
			break
		}

		length := int(data[offset])
		descType := data[offset+1]

		if length < 2 || offset+length > len(data) {
			break
		}

		switch descType {
		case DescriptorTypeInterface:
			var iface InterfaceDescriptor
			if ParseInterfaceDescriptor(data[offset:], &iface) {
				d.interfaces = append(d.interfaces, iface)
				currentIfaceIdx = len(d.interfaces) - 1
			}

		case DescriptorTypeEndpoint:
			var ep EndpointDescriptor
			if ParseEndpointDescriptor(data[offset:], &ep) {
				d.endpoints = append(d.endpoints, ep)
			}

		default:
			// Class-specific or other descriptor
			if currentIfaceIdx >= 0 && currentIfaceIdx < MaxInterfacesPerConfiguration {
				// Copy descriptor data
				descData := make([]byte, length)
				copy(descData, data[offset:offset+length])
				d.classDescriptors[currentIfaceIdx] = append(
					d.classDescriptors[currentIfaceIdx], descData)
			}
		}

		offset += length
	}
}

// GetDescriptor performs a GET_DESCRIPTOR request.
func (d *Device) GetDescriptor(ctx context.Context, descType, descIndex uint8, langID uint16, data []byte) (int, error) {
	setup := hal.SetupPacket{
		RequestType: RequestTypeIn | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestGetDescriptor,
		Value:       uint16(descType)<<8 | uint16(descIndex),
		Index:       langID,
		Length:      uint16(len(data)),
	}

	return d.ControlTransfer(ctx, &setup, data)
}

// GetStatus performs a GET_STATUS request.
func (d *Device) GetStatus(ctx context.Context) (uint16, error) {
	var buf [2]byte
	setup := hal.SetupPacket{
		RequestType: RequestTypeIn | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestGetStatus,
		Value:       0,
		Index:       0,
		Length:      2,
	}

	_, err := d.ControlTransfer(ctx, &setup, buf[:])
	if err != nil {
		return 0, err
	}

	return uint16(buf[0]) | uint16(buf[1])<<8, nil
}

// ClearFeature performs a CLEAR_FEATURE request.
func (d *Device) ClearFeature(ctx context.Context, feature uint16) error {
	setup := hal.SetupPacket{
		RequestType: RequestTypeOut | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestClearFeature,
		Value:       feature,
		Index:       0,
		Length:      0,
	}

	_, err := d.ControlTransfer(ctx, &setup, nil)
	return err
}

// SetFeature performs a SET_FEATURE request.
func (d *Device) SetFeature(ctx context.Context, feature uint16) error {
	setup := hal.SetupPacket{
		RequestType: RequestTypeOut | RequestTypeStandard | RequestTypeDevice,
		Request:     RequestSetFeature,
		Value:       feature,
		Index:       0,
		Length:      0,
	}

	_, err := d.ControlTransfer(ctx, &setup, nil)
	return err
}

// ClearEndpointHalt clears the halt condition on an endpoint.
func (d *Device) ClearEndpointHalt(ctx context.Context, endpoint uint8) error {
	setup := hal.SetupPacket{
		RequestType: RequestTypeOut | RequestTypeStandard | RequestTypeEndpoint,
		Request:     RequestClearFeature,
		Value:       0, // ENDPOINT_HALT feature
		Index:       uint16(endpoint),
		Length:      0,
	}

	_, err := d.ControlTransfer(ctx, &setup, nil)
	return err
}
