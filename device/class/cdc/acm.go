package cdc

import (
	"context"
	"sync"

	"github.com/ardnew/softusb/device"
	"github.com/ardnew/softusb/pkg"
)

// MaxRxBufferSize is the maximum receive buffer size.
const MaxRxBufferSize = 4096

// MaxTxBufferSize is the maximum transmit buffer size.
const MaxTxBufferSize = 4096

// ACM implements a CDC-ACM (Abstract Control Model) class driver.
// It provides USB serial port functionality.
type ACM struct {
	// Interfaces
	controlIface *device.Interface
	dataIface    *device.Interface

	// Endpoints
	notifyEP  *device.Endpoint // Interrupt IN for notifications
	dataInEP  *device.Endpoint // Bulk IN for data to host
	dataOutEP *device.Endpoint // Bulk OUT for data from host

	// Stack reference for data transfer
	stack *device.Stack

	// Configuration
	lineCoding   LineCoding
	controlState uint16
	serialState  uint16

	// Callbacks
	onLineCodingChange   func(*LineCoding)
	onControlStateChange func(dtr, rts bool)
	onBreak              func(millis uint16)

	// Buffers (zero-allocation)
	rxBuf       [MaxRxBufferSize]byte
	txBuf       [MaxTxBufferSize]byte
	responseBuf [LineCodingSize]byte

	// State
	mutex      sync.RWMutex
	configured bool
}

// NewACM creates a new CDC-ACM class driver.
func NewACM() *ACM {
	return &ACM{
		lineCoding: DefaultLineCoding,
	}
}

// SetStack sets the device stack reference for data transfer.
func (a *ACM) SetStack(stack *device.Stack) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.stack = stack
}

// SetOnLineCodingChange sets the callback for line coding changes.
func (a *ACM) SetOnLineCodingChange(cb func(*LineCoding)) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.onLineCodingChange = cb
}

// SetOnControlStateChange sets the callback for control line state changes.
func (a *ACM) SetOnControlStateChange(cb func(dtr, rts bool)) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.onControlStateChange = cb
}

// SetOnBreak sets the callback for break signaling.
func (a *ACM) SetOnBreak(cb func(millis uint16)) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.onBreak = cb
}

// LineCoding returns the current line coding configuration.
func (a *ACM) LineCoding() LineCoding {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.lineCoding
}

// DTR returns the current DTR (Data Terminal Ready) state.
func (a *ACM) DTR() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.controlState&ControlLineDTR != 0
}

// RTS returns the current RTS (Request To Send) state.
func (a *ACM) RTS() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.controlState&ControlLineRTS != 0
}

// Init initializes the class driver for the given interface.
// This is called by the device stack when the class driver is attached.
func (a *ACM) Init(iface *device.Interface) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Determine which interface this is based on class
	if iface.Class == ClassCDC {
		a.controlIface = iface
		// Find the notification endpoint
		for _, ep := range iface.Endpoints() {
			if ep.IsIn() && ep.IsInterrupt() {
				a.notifyEP = ep
				break
			}
		}
	} else if iface.Class == ClassCDCData {
		a.dataIface = iface
		// Find data endpoints
		for _, ep := range iface.Endpoints() {
			if ep.IsIn() && ep.IsBulk() {
				a.dataInEP = ep
			} else if ep.IsOut() && ep.IsBulk() {
				a.dataOutEP = ep
			}
		}
	}

	// Check if fully configured
	if a.controlIface != nil && a.dataIface != nil &&
		a.dataInEP != nil && a.dataOutEP != nil {
		a.configured = true
		pkg.LogDebug(pkg.ComponentDevice, "CDC-ACM configured",
			"dataIn", a.dataInEP.Address,
			"dataOut", a.dataOutEP.Address)
	}

	return nil
}

// HandleSetup processes class-specific SETUP requests.
func (a *ACM) HandleSetup(iface *device.Interface, setup *device.SetupPacket, data []byte) (bool, error) {
	if !setup.IsClass() {
		return false, nil
	}

	switch setup.Request {
	case RequestSetLineCoding:
		return a.handleSetLineCoding(setup, data)

	case RequestGetLineCoding:
		return a.handleGetLineCoding(setup)

	case RequestSetControlLineState:
		return a.handleSetControlLineState(setup)

	case RequestSendBreak:
		return a.handleSendBreak(setup)

	default:
		return false, nil
	}
}

// handleSetLineCoding handles the SET_LINE_CODING request.
func (a *ACM) handleSetLineCoding(setup *device.SetupPacket, data []byte) (bool, error) {
	if len(data) < LineCodingSize {
		return true, pkg.ErrBufferTooSmall
	}

	a.mutex.Lock()
	if !ParseLineCoding(data, &a.lineCoding) {
		a.mutex.Unlock()
		return true, pkg.ErrBufferTooSmall
	}
	cb := a.onLineCodingChange
	lc := a.lineCoding
	a.mutex.Unlock()

	pkg.LogDebug(pkg.ComponentDevice, "line coding set",
		"baud", lc.DTERate,
		"dataBits", lc.DataBits,
		"parity", lc.ParityType,
		"stopBits", lc.CharFormat)

	if cb != nil {
		cb(&lc)
	}

	return true, nil
}

// handleGetLineCoding handles the GET_LINE_CODING request.
func (a *ACM) handleGetLineCoding(setup *device.SetupPacket) (bool, error) {
	a.mutex.RLock()
	n := a.lineCoding.MarshalTo(a.responseBuf[:])
	a.mutex.RUnlock()

	if n == 0 {
		return true, pkg.ErrBufferTooSmall
	}

	// The standard request handler will send this data
	// For now, we just indicate we handled it
	return true, nil
}

// handleSetControlLineState handles the SET_CONTROL_LINE_STATE request.
func (a *ACM) handleSetControlLineState(setup *device.SetupPacket) (bool, error) {
	a.mutex.Lock()
	a.controlState = setup.Value
	cb := a.onControlStateChange
	dtr := a.controlState&ControlLineDTR != 0
	rts := a.controlState&ControlLineRTS != 0
	a.mutex.Unlock()

	pkg.LogDebug(pkg.ComponentDevice, "control line state set",
		"dtr", dtr,
		"rts", rts)

	if cb != nil {
		cb(dtr, rts)
	}

	return true, nil
}

// handleSendBreak handles the SEND_BREAK request.
func (a *ACM) handleSendBreak(setup *device.SetupPacket) (bool, error) {
	millis := setup.Value

	a.mutex.RLock()
	cb := a.onBreak
	a.mutex.RUnlock()

	pkg.LogDebug(pkg.ComponentDevice, "break signaled",
		"duration_ms", millis)

	if cb != nil {
		cb(millis)
	}

	return true, nil
}

// SetAlternate handles alternate setting changes.
func (a *ACM) SetAlternate(iface *device.Interface, alt uint8) error {
	pkg.LogDebug(pkg.ComponentDevice, "CDC alternate setting",
		"interface", iface.Number,
		"alt", alt)
	return nil
}

// Close releases resources held by the class driver.
func (a *ACM) Close() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.controlIface = nil
	a.dataIface = nil
	a.notifyEP = nil
	a.dataInEP = nil
	a.dataOutEP = nil
	a.stack = nil
	a.configured = false

	return nil
}

// Read reads data from the host (blocking).
// Returns the number of bytes read into buf.
func (a *ACM) Read(ctx context.Context, buf []byte) (int, error) {
	a.mutex.RLock()
	stack := a.stack
	ep := a.dataOutEP
	configured := a.configured
	a.mutex.RUnlock()

	if !configured || stack == nil || ep == nil {
		return 0, pkg.ErrNotConfigured
	}

	return stack.Read(ctx, ep, buf)
}

// Write writes data to the host (blocking).
// Returns the number of bytes written.
func (a *ACM) Write(ctx context.Context, data []byte) (int, error) {
	a.mutex.RLock()
	stack := a.stack
	ep := a.dataInEP
	configured := a.configured
	a.mutex.RUnlock()

	if !configured || stack == nil || ep == nil {
		return 0, pkg.ErrNotConfigured
	}

	return stack.Write(ctx, ep, data)
}

// SendSerialState sends a SERIAL_STATE notification to the host.
func (a *ACM) SendSerialState(state uint16) error {
	a.mutex.Lock()
	a.serialState = state
	stack := a.stack
	ep := a.notifyEP
	a.mutex.Unlock()

	if stack == nil || ep == nil {
		return pkg.ErrNotConfigured
	}

	// Build notification packet (10 bytes)
	// bmRequestType: 0xA1 (device-to-host, class, interface)
	// bNotification: SERIAL_STATE (0x20)
	// wValue: 0
	// wIndex: interface number
	// wLength: 2
	// data: 2 bytes of serial state
	var buf [10]byte
	buf[0] = 0xA1 // bmRequestType
	buf[1] = NotificationSerialState
	buf[2] = 0 // wValue low
	buf[3] = 0 // wValue high
	buf[4] = 0 // wIndex low (control interface number)
	buf[5] = 0 // wIndex high
	buf[6] = 2 // wLength low
	buf[7] = 0 // wLength high
	buf[8] = byte(state)
	buf[9] = byte(state >> 8)

	_, err := stack.Write(context.Background(), ep, buf[:])
	return err
}

// ConfigureDevice adds CDC-ACM interfaces to a device builder.
// Call this after AddConfiguration to add the CDC interfaces.
func (a *ACM) ConfigureDevice(builder *device.DeviceBuilder, notifyEPAddr, dataInEPAddr, dataOutEPAddr uint8) *device.DeviceBuilder {
	// Add Interface Association Descriptor (IAD) for composite device
	// This groups the control and data interfaces together

	// Control Interface (Communications Class)
	builder.AddInterface(ClassCDC, SubclassACM, ProtocolAT)
	// Add notification endpoint (interrupt IN)
	builder.AddEndpoint(notifyEPAddr|device.EndpointDirectionIn, device.EndpointTypeInterrupt, 8)

	// Data Interface (Data Class)
	builder.AddInterface(ClassCDCData, 0, 0)
	// Add bulk endpoints
	builder.AddEndpoint(dataInEPAddr|device.EndpointDirectionIn, device.EndpointTypeBulk, 64)
	builder.AddEndpoint(dataOutEPAddr&0x0F, device.EndpointTypeBulk, 64) // OUT has direction bit = 0

	return builder
}

// AttachToInterfaces attaches this class driver to the CDC interfaces.
// configValue is the configuration value (e.g., 1), controlIfaceNum and dataIfaceNum
// are the interface numbers within that configuration.
func (a *ACM) AttachToInterfaces(dev *device.Device, configValue, controlIfaceNum, dataIfaceNum uint8) error {
	config := dev.GetConfiguration(configValue)
	if config == nil {
		return pkg.ErrInvalidRequest
	}

	controlIface := config.GetInterface(controlIfaceNum)
	if controlIface == nil {
		return pkg.ErrInvalidRequest
	}

	dataIface := config.GetInterface(dataIfaceNum)
	if dataIface == nil {
		return pkg.ErrInvalidRequest
	}

	// Set this driver as the class driver for both interfaces
	if err := controlIface.SetClassDriver(a); err != nil {
		return err
	}

	// Note: We use a wrapper for the data interface to reuse the same ACM instance
	return dataIface.SetClassDriver(a)
}

// Compile-time interface check
var _ device.ClassDriver = (*ACM)(nil)
