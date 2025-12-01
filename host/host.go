package host

import (
	"context"
	"sync"

	"github.com/ardnew/softusb/host/hal"
	"github.com/ardnew/softusb/pkg"
)

// Host manages the USB host controller and connected devices.
type Host struct {
	hal hal.HostHAL

	// Connected devices (indexed by address - 1)
	devices     [MaxDevices]*Device
	deviceCount int

	// Next available address
	nextAddress uint8

	// State
	running bool
	mutex   sync.RWMutex

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Event channels
	deviceConnected    chan *Device
	deviceDisconnected chan *Device

	// Callbacks
	onDeviceConnect    func(*Device)
	onDeviceDisconnect func(*Device)
}

// New creates a new USB host.
func New(h hal.HostHAL) *Host {
	return &Host{
		hal:                h,
		nextAddress:        1,
		deviceConnected:    make(chan *Device, MaxDevices),
		deviceDisconnected: make(chan *Device, MaxDevices),
	}
}

// Start starts the host controller.
func (h *Host) Start(ctx context.Context) error {
	h.mutex.Lock()
	if h.running {
		h.mutex.Unlock()
		return pkg.ErrAlreadyRunning
	}

	h.ctx, h.cancel = context.WithCancel(ctx)
	h.mutex.Unlock()

	if err := h.hal.Init(h.ctx); err != nil {
		return err
	}

	if err := h.hal.Start(); err != nil {
		return err
	}

	h.mutex.Lock()
	h.running = true
	h.mutex.Unlock()

	pkg.LogInfo(pkg.ComponentHost, "host started")

	// Start device monitoring
	go h.monitorDevices()

	return nil
}

// Stop stops the host controller.
func (h *Host) Stop() error {
	h.mutex.Lock()
	if !h.running {
		h.mutex.Unlock()
		return nil
	}

	h.running = false
	if h.cancel != nil {
		h.cancel()
	}
	h.mutex.Unlock()

	// Close all devices
	for i := 0; i < MaxDevices; i++ {
		if h.devices[i] != nil {
			h.devices[i].Close()
			h.devices[i] = nil
		}
	}
	h.deviceCount = 0

	if err := h.hal.Stop(); err != nil {
		return err
	}

	pkg.LogInfo(pkg.ComponentHost, "host stopped")
	return nil
}

// IsRunning returns true if the host is running.
func (h *Host) IsRunning() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.running
}

// Devices returns all connected devices.
// The returned slice references internal storage; do not modify.
func (h *Host) Devices() []*Device {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	result := make([]*Device, 0, h.deviceCount)
	for i := 0; i < MaxDevices; i++ {
		if h.devices[i] != nil {
			result = append(result, h.devices[i])
		}
	}
	return result
}

// GetDevice returns the device at the given address.
func (h *Host) GetDevice(address uint8) *Device {
	if address == 0 || address > MaxDevices {
		return nil
	}
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.devices[address-1]
}

// WaitDevice blocks until a device connects and is enumerated.
func (h *Host) WaitDevice(ctx context.Context) (*Device, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.ctx.Done():
		return nil, pkg.ErrCancelled
	case dev := <-h.deviceConnected:
		return dev, nil
	}
}

// SetOnDeviceConnect sets the callback for device connection.
func (h *Host) SetOnDeviceConnect(cb func(*Device)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.onDeviceConnect = cb
}

// SetOnDeviceDisconnect sets the callback for device disconnection.
func (h *Host) SetOnDeviceDisconnect(cb func(*Device)) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.onDeviceDisconnect = cb
}

// monitorDevices monitors for device connections and disconnections.
func (h *Host) monitorDevices() {
	for {
		select {
		case <-h.ctx.Done():
			return
		default:
		}

		// Wait for a device connection
		port, err := h.hal.WaitForConnection(h.ctx)
		if err != nil {
			if h.ctx.Err() != nil {
				return
			}
			pkg.LogWarn(pkg.ComponentHost, "error waiting for connection",
				"error", err)
			continue
		}

		pkg.LogInfo(pkg.ComponentHost, "device connected", "port", port)

		// Enumerate the device
		dev, err := h.enumerateDevice(port)
		if err != nil {
			pkg.LogWarn(pkg.ComponentHost, "enumeration failed",
				"port", port,
				"error", err)
			continue
		}

		// Add device to list
		h.mutex.Lock()
		if h.deviceCount < MaxDevices {
			h.devices[dev.address-1] = dev
			h.deviceCount++
			cb := h.onDeviceConnect
			h.mutex.Unlock()

			// Notify
			select {
			case h.deviceConnected <- dev:
			default:
			}

			if cb != nil {
				cb(dev)
			}

			pkg.LogInfo(pkg.ComponentHost, "device enumerated",
				"address", dev.address,
				"vendor", dev.descriptor.VendorID,
				"product", dev.descriptor.ProductID)
		} else {
			h.mutex.Unlock()
			pkg.LogWarn(pkg.ComponentHost, "max devices reached")
		}

		// Start monitoring for disconnection in a separate goroutine
		go h.monitorDisconnection(port, dev)
	}
}

// monitorDisconnection monitors for device disconnection.
func (h *Host) monitorDisconnection(port int, dev *Device) {
	// Wait for disconnection
	_, err := h.hal.WaitForDisconnection(h.ctx)
	if err != nil {
		if h.ctx.Err() != nil {
			return
		}
		return
	}

	pkg.LogInfo(pkg.ComponentHost, "device disconnected",
		"port", port,
		"address", dev.address)

	// Remove device from list
	h.mutex.Lock()
	if dev.address > 0 && dev.address <= MaxDevices {
		h.devices[dev.address-1] = nil
		h.deviceCount--
	}
	cb := h.onDeviceDisconnect
	h.mutex.Unlock()

	dev.Close()

	// Notify
	select {
	case h.deviceDisconnected <- dev:
	default:
	}

	if cb != nil {
		cb(dev)
	}
}

// allocateAddress allocates a new device address.
func (h *Host) allocateAddress() uint8 {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Find next available address
	for i := 0; i < MaxDevices; i++ {
		addr := h.nextAddress
		h.nextAddress++
		if h.nextAddress > MaxDevices {
			h.nextAddress = 1
		}

		if h.devices[addr-1] == nil {
			return addr
		}
	}
	return 0 // No address available
}

// NumPorts returns the number of root hub ports.
func (h *Host) NumPorts() int {
	return h.hal.NumPorts()
}

// GetPortStatus returns the status of a port.
func (h *Host) GetPortStatus(port int) (hal.PortStatus, error) {
	return h.hal.GetPortStatus(port)
}

// ControlTransfer performs a control transfer to a device at address 0.
// This is used during enumeration before the device has an assigned address.
func (h *Host) ControlTransfer(ctx context.Context, setup *hal.SetupPacket, data []byte) (int, error) {
	return h.hal.ControlTransfer(ctx, hal.DeviceAddress(0), setup, data)
}
