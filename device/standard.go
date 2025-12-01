package device

import (
	"encoding/binary"

	"github.com/ardnew/softusb/pkg"
)

// MaxDescriptorResponseSize is the maximum size for descriptor responses.
// This covers the largest typical configuration descriptor.
const MaxDescriptorResponseSize = 512

// StandardRequestHandler handles standard USB device requests.
type StandardRequestHandler struct {
	device *Device

	// Pre-allocated response buffer to avoid allocations on hot path.
	// The returned slice from HandleSetup references this buffer.
	responseBuf [MaxDescriptorResponseSize]byte
}

// NewStandardRequestHandler creates a new standard request handler.
func NewStandardRequestHandler(dev *Device) *StandardRequestHandler {
	return &StandardRequestHandler{device: dev}
}

// HandleSetup processes a standard SETUP request.
// Returns the response data (may be nil) and an error.
func (h *StandardRequestHandler) HandleSetup(setup *SetupPacket, data []byte) ([]byte, error) {
	if !setup.IsStandard() {
		return nil, pkg.ErrInvalidRequest
	}

	switch setup.Recipient() {
	case RequestRecipientDevice:
		return h.handleDeviceRequest(setup, data)
	case RequestRecipientInterface:
		return h.handleInterfaceRequest(setup, data)
	case RequestRecipientEndpoint:
		return h.handleEndpointRequest(setup, data)
	default:
		return nil, pkg.ErrInvalidRequest
	}
}

// handleDeviceRequest handles device-level standard requests.
func (h *StandardRequestHandler) handleDeviceRequest(setup *SetupPacket, data []byte) ([]byte, error) {
	switch setup.Request {
	case RequestGetStatus:
		return h.getDeviceStatus(setup)
	case RequestClearFeature:
		return h.clearDeviceFeature(setup)
	case RequestSetFeature:
		return h.setDeviceFeature(setup)
	case RequestSetAddress:
		return h.setAddress(setup)
	case RequestGetDescriptor:
		return h.getDescriptor(setup)
	case RequestSetDescriptor:
		return nil, pkg.ErrNotSupported // Optional, not implemented
	case RequestGetConfiguration:
		return h.getConfiguration(setup)
	case RequestSetConfiguration:
		return h.setConfiguration(setup)
	default:
		return nil, pkg.ErrInvalidRequest
	}
}

// handleInterfaceRequest handles interface-level standard requests.
func (h *StandardRequestHandler) handleInterfaceRequest(setup *SetupPacket, data []byte) ([]byte, error) {
	switch setup.Request {
	case RequestGetStatus:
		return h.getInterfaceStatus(setup)
	case RequestClearFeature:
		return nil, nil // No standard interface features
	case RequestSetFeature:
		return nil, nil // No standard interface features
	case RequestGetInterface:
		return h.getInterface(setup)
	case RequestSetInterface:
		return h.setInterface(setup)
	default:
		return nil, pkg.ErrInvalidRequest
	}
}

// handleEndpointRequest handles endpoint-level standard requests.
func (h *StandardRequestHandler) handleEndpointRequest(setup *SetupPacket, data []byte) ([]byte, error) {
	switch setup.Request {
	case RequestGetStatus:
		return h.getEndpointStatus(setup)
	case RequestClearFeature:
		return h.clearEndpointFeature(setup)
	case RequestSetFeature:
		return h.setEndpointFeature(setup)
	case RequestSynchFrame:
		return h.synchFrame(setup)
	default:
		return nil, pkg.ErrInvalidRequest
	}
}

// getDeviceStatus returns device status (2 bytes).
func (h *StandardRequestHandler) getDeviceStatus(setup *SetupPacket) ([]byte, error) {
	if setup.Length < 2 {
		return nil, pkg.ErrInvalidRequest
	}

	status := h.device.GetStatus()
	binary.LittleEndian.PutUint16(h.responseBuf[:2], uint16(status))
	return h.responseBuf[:2], nil
}

// clearDeviceFeature clears a device feature.
func (h *StandardRequestHandler) clearDeviceFeature(setup *SetupPacket) ([]byte, error) {
	feature := setup.Value
	switch feature {
	case FeatureDeviceRemoteWakeup:
		h.device.EnableRemoteWakeup(false)
		return nil, nil
	default:
		return nil, pkg.ErrInvalidRequest
	}
}

// setDeviceFeature sets a device feature.
func (h *StandardRequestHandler) setDeviceFeature(setup *SetupPacket) ([]byte, error) {
	feature := setup.Value
	switch feature {
	case FeatureDeviceRemoteWakeup:
		h.device.EnableRemoteWakeup(true)
		return nil, nil
	case FeatureTestMode:
		// Test mode is not typically implemented
		return nil, pkg.ErrNotSupported
	default:
		return nil, pkg.ErrInvalidRequest
	}
}

// setAddress handles SET_ADDRESS request.
func (h *StandardRequestHandler) setAddress(setup *SetupPacket) ([]byte, error) {
	address := uint8(setup.Value & 0x7F)
	if err := h.device.SetAddress(address); err != nil {
		return nil, err
	}
	return nil, nil
}

// getDescriptor handles GET_DESCRIPTOR request.
func (h *StandardRequestHandler) getDescriptor(setup *SetupPacket) ([]byte, error) {
	descType := setup.DescriptorType()
	descIndex := setup.DescriptorIndex()
	maxLen := int(setup.Length)

	var n int

	switch descType {
	case DescriptorTypeDevice:
		n = h.device.Descriptor.MarshalTo(h.responseBuf[:])

	case DescriptorTypeConfiguration:
		config := h.device.GetConfiguration(descIndex + 1)
		if config == nil {
			return nil, pkg.ErrInvalidRequest
		}
		n = config.MarshalTo(h.responseBuf[:])

	case DescriptorTypeString:
		data := h.device.GetString(descIndex)
		if data == nil {
			return nil, pkg.ErrInvalidRequest
		}
		// String descriptors are pre-encoded, copy to response buffer
		n = copy(h.responseBuf[:], data)

	case DescriptorTypeDeviceQualifier:
		// Device qualifier for high-speed capable devices
		n = h.getDeviceQualifier()
		if n == 0 {
			return nil, pkg.ErrNotSupported
		}

	case DescriptorTypeOtherSpeedConfig:
		// Other speed configuration
		return nil, pkg.ErrNotSupported

	default:
		return nil, pkg.ErrInvalidRequest
	}

	if n == 0 {
		return nil, pkg.ErrBufferTooSmall
	}

	if n > maxLen {
		n = maxLen
	}
	return h.responseBuf[:n], nil
}

// getDeviceQualifier writes the device qualifier descriptor to responseBuf.
// Returns the number of bytes written, or 0 if not supported.
func (h *StandardRequestHandler) getDeviceQualifier() int {
	// Only required for high-speed capable devices
	if h.device.Speed() != SpeedHigh {
		return 0
	}

	desc := h.device.Descriptor
	h.responseBuf[0] = 10 // Length
	h.responseBuf[1] = DescriptorTypeDeviceQualifier
	binary.LittleEndian.PutUint16(h.responseBuf[2:4], desc.USBVersion)
	h.responseBuf[4] = desc.DeviceClass
	h.responseBuf[5] = desc.DeviceSubClass
	h.responseBuf[6] = desc.DeviceProtocol
	h.responseBuf[7] = desc.MaxPacketSize0
	h.responseBuf[8] = desc.NumConfigurations
	h.responseBuf[9] = 0 // Reserved
	return 10
}

// getConfiguration handles GET_CONFIGURATION request.
func (h *StandardRequestHandler) getConfiguration(setup *SetupPacket) ([]byte, error) {
	config := h.device.ActiveConfiguration()
	if config == nil {
		return []byte{0}, nil
	}
	return []byte{config.Value}, nil
}

// setConfiguration handles SET_CONFIGURATION request.
func (h *StandardRequestHandler) setConfiguration(setup *SetupPacket) ([]byte, error) {
	configValue := uint8(setup.Value & 0xFF)
	if err := h.device.SetConfiguration(configValue); err != nil {
		return nil, err
	}
	return nil, nil
}

// getInterfaceStatus returns interface status (2 bytes, always zero).
func (h *StandardRequestHandler) getInterfaceStatus(setup *SetupPacket) ([]byte, error) {
	if setup.Length < 2 {
		return nil, pkg.ErrInvalidRequest
	}

	ifaceNum := setup.InterfaceNumber()
	if h.device.GetInterface(ifaceNum) == nil {
		return nil, pkg.ErrInvalidRequest
	}

	// Interface status is reserved (zero)
	return []byte{0, 0}, nil
}

// getInterface handles GET_INTERFACE request.
func (h *StandardRequestHandler) getInterface(setup *SetupPacket) ([]byte, error) {
	ifaceNum := setup.InterfaceNumber()
	iface := h.device.GetInterface(ifaceNum)
	if iface == nil {
		return nil, pkg.ErrInvalidRequest
	}
	return []byte{iface.AlternateSetting}, nil
}

// setInterface handles SET_INTERFACE request.
func (h *StandardRequestHandler) setInterface(setup *SetupPacket) ([]byte, error) {
	ifaceNum := setup.InterfaceNumber()
	altSetting := uint8(setup.Value & 0xFF)

	iface := h.device.GetInterface(ifaceNum)
	if iface == nil {
		return nil, pkg.ErrInvalidRequest
	}

	if err := iface.SetAlternate(altSetting); err != nil {
		return nil, err
	}
	return nil, nil
}

// getEndpointStatus returns endpoint status (2 bytes).
func (h *StandardRequestHandler) getEndpointStatus(setup *SetupPacket) ([]byte, error) {
	if setup.Length < 2 {
		return nil, pkg.ErrInvalidRequest
	}

	epAddr := setup.EndpointAddress()
	ep := h.device.GetEndpoint(epAddr)
	if ep == nil {
		return nil, pkg.ErrInvalidEndpoint
	}

	var status uint16
	if ep.IsStalled() {
		status = 1 // Halt bit
	}
	binary.LittleEndian.PutUint16(h.responseBuf[:2], status)
	return h.responseBuf[:2], nil
}

// clearEndpointFeature clears an endpoint feature.
func (h *StandardRequestHandler) clearEndpointFeature(setup *SetupPacket) ([]byte, error) {
	if setup.Value != FeatureEndpointHalt {
		return nil, pkg.ErrInvalidRequest
	}

	epAddr := setup.EndpointAddress()
	ep := h.device.GetEndpoint(epAddr)
	if ep == nil {
		return nil, pkg.ErrInvalidEndpoint
	}

	ep.SetStall(false)
	ep.ResetDataToggle()
	return nil, nil
}

// setEndpointFeature sets an endpoint feature.
func (h *StandardRequestHandler) setEndpointFeature(setup *SetupPacket) ([]byte, error) {
	if setup.Value != FeatureEndpointHalt {
		return nil, pkg.ErrInvalidRequest
	}

	epAddr := setup.EndpointAddress()
	ep := h.device.GetEndpoint(epAddr)
	if ep == nil {
		return nil, pkg.ErrInvalidEndpoint
	}

	ep.SetStall(true)
	return nil, nil
}

// synchFrame handles SYNCH_FRAME request for isochronous endpoints.
func (h *StandardRequestHandler) synchFrame(setup *SetupPacket) ([]byte, error) {
	if setup.Length < 2 {
		return nil, pkg.ErrInvalidRequest
	}

	epAddr := setup.EndpointAddress()
	ep := h.device.GetEndpoint(epAddr)
	if ep == nil {
		return nil, pkg.ErrInvalidEndpoint
	}

	if !ep.IsIsochronous() {
		return nil, pkg.ErrInvalidRequest
	}

	frame := ep.FrameNumber()
	binary.LittleEndian.PutUint16(h.responseBuf[:2], frame)
	return h.responseBuf[:2], nil
}
