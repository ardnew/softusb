package hid

import (
	"context"
	"sync"

	"github.com/ardnew/softusb/device"
	"github.com/ardnew/softusb/pkg"
)

// MaxReportSize is the maximum HID report size.
const MaxReportSize = 64

// HID implements a HID class driver.
type HID struct {
	// Interface
	iface *device.Interface

	// Endpoints
	inEP  *device.Endpoint // Interrupt IN for input reports
	outEP *device.Endpoint // Interrupt OUT for output reports (optional)

	// Stack reference for data transfer
	stack *device.Stack

	// Report descriptor (stored by reference)
	reportDescriptor []byte

	// HID descriptor
	hidDescriptor HIDDescriptor

	// State
	protocol uint8 // 0 = boot, 1 = report
	idleRate uint8 // Idle rate in 4ms units (0 = infinite)

	// Callbacks
	onOutputReport  func(data []byte)
	onFeatureReport func(reportID uint8, data []byte)
	onSetProtocol   func(protocol uint8)
	onSetIdle       func(rate uint8, reportID uint8)

	// Buffers (zero-allocation)
	reportBuf   [MaxReportSize]byte
	responseBuf [MaxReportSize]byte

	// State
	mutex      sync.RWMutex
	configured bool
}

// New creates a new HID class driver with the given report descriptor.
// The report descriptor is stored by reference.
func New(reportDescriptor []byte) *HID {
	return &HID{
		reportDescriptor: reportDescriptor,
		hidDescriptor: HIDDescriptor{
			Length:         HIDDescriptorSize,
			DescriptorType: DescriptorTypeHID,
			HIDVersion:     0x0111, // HID 1.11
			CountryCode:    CountryNone,
			NumDescriptors: 1,
			ReportDescType: DescriptorTypeReport,
			ReportDescLen:  uint16(len(reportDescriptor)),
		},
		protocol: ProtocolReport,
	}
}

// SetStack sets the device stack reference for data transfer.
func (h *HID) SetStack(stack *device.Stack) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.stack = stack
}

// SetOnOutputReport sets the callback for output reports from the host.
func (h *HID) SetOnOutputReport(cb func(data []byte)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.onOutputReport = cb
}

// SetOnFeatureReport sets the callback for feature report requests.
func (h *HID) SetOnFeatureReport(cb func(reportID uint8, data []byte)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.onFeatureReport = cb
}

// SetOnSetProtocol sets the callback for protocol changes.
func (h *HID) SetOnSetProtocol(cb func(protocol uint8)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.onSetProtocol = cb
}

// SetOnSetIdle sets the callback for idle rate changes.
func (h *HID) SetOnSetIdle(cb func(rate uint8, reportID uint8)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.onSetIdle = cb
}

// Protocol returns the current protocol (boot or report).
func (h *HID) Protocol() uint8 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.protocol
}

// IdleRate returns the current idle rate.
func (h *HID) IdleRate() uint8 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.idleRate
}

// ReportDescriptor returns the report descriptor.
func (h *HID) ReportDescriptor() []byte {
	return h.reportDescriptor
}

// Init initializes the class driver for the given interface.
func (h *HID) Init(iface *device.Interface) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.iface = iface

	// Find endpoints
	for _, ep := range iface.Endpoints() {
		if ep.IsInterrupt() {
			if ep.IsIn() {
				h.inEP = ep
			} else {
				h.outEP = ep
			}
		}
	}

	if h.inEP == nil {
		return pkg.ErrInvalidEndpoint
	}

	h.configured = true
	pkg.LogDebug(pkg.ComponentDevice, "HID configured",
		"inEP", h.inEP.Address,
		"reportDescLen", len(h.reportDescriptor))

	return nil
}

// HandleSetup processes class-specific SETUP requests.
func (h *HID) HandleSetup(iface *device.Interface, setup *device.SetupPacket, data []byte) (bool, error) {
	// Handle standard requests for HID descriptors
	if setup.IsStandard() && setup.Request == device.RequestGetDescriptor {
		return h.handleGetDescriptor(setup)
	}

	if !setup.IsClass() {
		return false, nil
	}

	switch setup.Request {
	case RequestGetReport:
		return h.handleGetReport(setup)

	case RequestSetReport:
		return h.handleSetReport(setup, data)

	case RequestGetIdle:
		return h.handleGetIdle(setup)

	case RequestSetIdle:
		return h.handleSetIdle(setup)

	case RequestGetProtocol:
		return h.handleGetProtocol(setup)

	case RequestSetProtocol:
		return h.handleSetProtocol(setup)

	default:
		return false, nil
	}
}

// handleGetDescriptor handles GET_DESCRIPTOR for HID and Report descriptors.
func (h *HID) handleGetDescriptor(setup *device.SetupPacket) (bool, error) {
	descType := setup.DescriptorType()

	switch descType {
	case DescriptorTypeHID:
		// Return HID descriptor
		h.mutex.RLock()
		n := h.hidDescriptor.MarshalTo(h.responseBuf[:])
		h.mutex.RUnlock()

		if n == 0 {
			return true, pkg.ErrBufferTooSmall
		}
		// The stack should send this data
		return true, nil

	case DescriptorTypeReport:
		// Return Report descriptor
		// The stack should send h.reportDescriptor
		return true, nil

	default:
		return false, nil
	}
}

// handleGetReport handles GET_REPORT request.
func (h *HID) handleGetReport(setup *device.SetupPacket) (bool, error) {
	reportType := uint8(setup.Value >> 8)
	reportID := uint8(setup.Value & 0xFF)

	pkg.LogDebug(pkg.ComponentDevice, "GET_REPORT",
		"type", reportType,
		"id", reportID)

	// For now, return zeros
	// A real implementation would get the current report state
	return true, nil
}

// handleSetReport handles SET_REPORT request.
func (h *HID) handleSetReport(setup *device.SetupPacket, data []byte) (bool, error) {
	reportType := uint8(setup.Value >> 8)
	reportID := uint8(setup.Value & 0xFF)

	pkg.LogDebug(pkg.ComponentDevice, "SET_REPORT",
		"type", reportType,
		"id", reportID,
		"len", len(data))

	h.mutex.RLock()
	outputCb := h.onOutputReport
	featureCb := h.onFeatureReport
	h.mutex.RUnlock()

	switch reportType {
	case ReportTypeOutput:
		if outputCb != nil {
			outputCb(data)
		}
	case ReportTypeFeature:
		if featureCb != nil {
			featureCb(reportID, data)
		}
	}

	return true, nil
}

// handleGetIdle handles GET_IDLE request.
func (h *HID) handleGetIdle(setup *device.SetupPacket) (bool, error) {
	h.mutex.RLock()
	h.responseBuf[0] = h.idleRate
	h.mutex.RUnlock()

	return true, nil
}

// handleSetIdle handles SET_IDLE request.
func (h *HID) handleSetIdle(setup *device.SetupPacket) (bool, error) {
	rate := uint8(setup.Value >> 8)
	reportID := uint8(setup.Value & 0xFF)

	h.mutex.Lock()
	h.idleRate = rate
	cb := h.onSetIdle
	h.mutex.Unlock()

	pkg.LogDebug(pkg.ComponentDevice, "SET_IDLE",
		"rate", rate,
		"reportID", reportID)

	if cb != nil {
		cb(rate, reportID)
	}

	return true, nil
}

// handleGetProtocol handles GET_PROTOCOL request.
func (h *HID) handleGetProtocol(setup *device.SetupPacket) (bool, error) {
	h.mutex.RLock()
	h.responseBuf[0] = h.protocol
	h.mutex.RUnlock()

	return true, nil
}

// handleSetProtocol handles SET_PROTOCOL request.
func (h *HID) handleSetProtocol(setup *device.SetupPacket) (bool, error) {
	protocol := uint8(setup.Value & 0xFF)

	h.mutex.Lock()
	h.protocol = protocol
	cb := h.onSetProtocol
	h.mutex.Unlock()

	pkg.LogDebug(pkg.ComponentDevice, "SET_PROTOCOL",
		"protocol", protocol)

	if cb != nil {
		cb(protocol)
	}

	return true, nil
}

// SetAlternate handles alternate setting changes.
func (h *HID) SetAlternate(iface *device.Interface, alt uint8) error {
	pkg.LogDebug(pkg.ComponentDevice, "HID alternate setting",
		"interface", iface.Number,
		"alt", alt)
	return nil
}

// Close releases resources held by the class driver.
func (h *HID) Close() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.iface = nil
	h.inEP = nil
	h.outEP = nil
	h.stack = nil
	h.configured = false

	return nil
}

// SendReport sends an input report to the host.
func (h *HID) SendReport(ctx context.Context, data []byte) error {
	h.mutex.RLock()
	stack := h.stack
	ep := h.inEP
	configured := h.configured
	h.mutex.RUnlock()

	if !configured || stack == nil || ep == nil {
		return pkg.ErrNotConfigured
	}

	_, err := stack.Write(ctx, ep, data)
	return err
}

// SendKeyboardReport sends a keyboard report to the host.
func (h *HID) SendKeyboardReport(ctx context.Context, report *KeyboardReport) error {
	n := report.MarshalTo(h.reportBuf[:])
	if n == 0 {
		return pkg.ErrBufferTooSmall
	}
	return h.SendReport(ctx, h.reportBuf[:n])
}

// SendMouseReport sends a mouse report to the host.
func (h *HID) SendMouseReport(ctx context.Context, report *MouseReport) error {
	n := report.MarshalTo(h.reportBuf[:])
	if n == 0 {
		return pkg.ErrBufferTooSmall
	}
	return h.SendReport(ctx, h.reportBuf[:n])
}

// ReceiveReport receives an output report from the host (if OUT endpoint exists).
func (h *HID) ReceiveReport(ctx context.Context, buf []byte) (int, error) {
	h.mutex.RLock()
	stack := h.stack
	ep := h.outEP
	configured := h.configured
	h.mutex.RUnlock()

	if !configured || stack == nil {
		return 0, pkg.ErrNotConfigured
	}

	if ep == nil {
		return 0, pkg.ErrInvalidEndpoint
	}

	return stack.Read(ctx, ep, buf)
}

// ConfigureDevice adds the HID interface to a device builder.
func (h *HID) ConfigureDevice(builder *device.DeviceBuilder, inEPAddr uint8, subclass, protocol uint8) *device.DeviceBuilder {
	builder.AddInterface(ClassHID, subclass, protocol)
	builder.AddEndpoint(inEPAddr|device.EndpointDirectionIn, device.EndpointTypeInterrupt, 8)
	return builder
}

// ConfigureDeviceWithOutEP adds the HID interface with an OUT endpoint.
func (h *HID) ConfigureDeviceWithOutEP(builder *device.DeviceBuilder, inEPAddr, outEPAddr uint8, subclass, protocol uint8) *device.DeviceBuilder {
	builder.AddInterface(ClassHID, subclass, protocol)
	builder.AddEndpoint(inEPAddr|device.EndpointDirectionIn, device.EndpointTypeInterrupt, 8)
	builder.AddEndpoint(outEPAddr&0x0F, device.EndpointTypeInterrupt, 8)
	return builder
}

// AttachToInterface attaches this class driver to the HID interface.
// configValue is the configuration value (e.g., 1), ifaceNum is the interface number
// within that configuration.
func (h *HID) AttachToInterface(dev *device.Device, configValue, ifaceNum uint8) error {
	config := dev.GetConfiguration(configValue)
	if config == nil {
		return pkg.ErrInvalidRequest
	}

	iface := config.GetInterface(ifaceNum)
	if iface == nil {
		return pkg.ErrInvalidRequest
	}
	return iface.SetClassDriver(h)
}

// Compile-time interface check
var _ device.ClassDriver = (*HID)(nil)
