//go:build linux

package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ardnew/softusb/host/hal"
	"github.com/ardnew/softusb/host/hal/linux"
	"github.com/ardnew/softusb/pkg"
)

// Component identifier for hid-monitor logging.
const componentMonitor pkg.Component = "monitor"

var (
	verbose   = flag.Bool("v", false, "Enable verbose logging")
	jsonOut   = flag.Bool("json", false, "Output logs as JSON")
	vendorID  = flag.String("vid", "", "Filter by Vendor ID (hex)")
	productID = flag.String("pid", "", "Filter by Product ID (hex)")
)

// =============================================================================
// Output Events
// =============================================================================

// outputEvent represents an event to be logged by the output goroutine.
type outputEvent interface {
	log()
}

// deviceConnectedEvent is sent when a device is first detected.
type deviceConnectedEvent struct {
	port  int
	speed hal.Speed
}

func (e deviceConnectedEvent) log() {
	pkg.LogDebug(componentMonitor, "device connected",
		"port", e.port,
		"speed", e.speed.String())
}

// deviceEnumeratedEvent is sent after a device has been fully enumerated.
type deviceEnumeratedEvent struct {
	info          *deviceInfo
	matchesFilter bool
	enumError     error
}

func (e deviceEnumeratedEvent) log() {
	if e.enumError != nil {
		pkg.LogError(componentMonitor, "enumeration failed",
			"port", e.info.port,
			"error", e.enumError)
		return
	}
	if !e.matchesFilter {
		return
	}

	// Build HID interface attributes
	hidIfaces := make([]any, 0, len(e.info.hidInterfaces)*4)
	for _, iface := range e.info.hidInterfaces {
		hidIfaces = append(hidIfaces,
			slog.Group("hid_interface",
				"number", iface.number,
				"subclass", iface.subclass,
				"protocol", iface.protocol,
				"endpoint", iface.epAddr,
				"max_packet", iface.maxPacket))
	}

	attrs := []any{
		"port", e.info.port,
		"vid", e.info.vid,
		"pid", e.info.pid,
		"speed", e.info.speed.String(),
	}
	if e.info.manufacturer != "" {
		attrs = append(attrs, "manufacturer", e.info.manufacturer)
	}
	if e.info.product != "" {
		attrs = append(attrs, "product", e.info.product)
	}
	if e.info.serialNumber != "" {
		attrs = append(attrs, "serial", e.info.serialNumber)
	}
	if len(e.info.hidInterfaces) > 0 {
		attrs = append(attrs, "hid_interfaces", len(e.info.hidInterfaces))
	}
	attrs = append(attrs, hidIfaces...)

	pkg.LogInfo(componentMonitor, "device enumerated", attrs...)
}

// deviceDisconnectedEvent is sent when a device is disconnected.
type deviceDisconnectedEvent struct {
	port int
}

func (e deviceDisconnectedEvent) log() {
	pkg.LogInfo(componentMonitor, "device disconnected",
		"port", e.port)
}

// hidReportEvent is sent when an HID report is received.
type hidReportEvent struct {
	port      int
	ifaceNum  uint8
	reportLen int
	data      []byte
}

func (e hidReportEvent) log() {
	pkg.LogInfo(componentMonitor, "hid report",
		"port", e.port,
		"interface", e.ifaceNum,
		"length", e.reportLen,
		"data", hex.EncodeToString(e.data))
}

// errorEvent is sent when an error occurs.
type errorEvent struct {
	message string
	err     error
}

func (e errorEvent) log() {
	if e.err != nil {
		pkg.LogError(componentMonitor, e.message, "error", e.err)
	} else {
		pkg.LogError(componentMonitor, e.message)
	}
}

// interfaceClaimErrorEvent is sent when claiming an interface fails.
type interfaceClaimErrorEvent struct {
	port     int
	ifaceNum uint8
	err      error
}

func (e interfaceClaimErrorEvent) log() {
	pkg.LogError(componentMonitor, "failed to claim interface",
		"port", e.port,
		"interface", e.ifaceNum,
		"error", e.err)
}

// =============================================================================
// Device Registry
// =============================================================================

// deviceInfo holds information about a connected USB device.
type deviceInfo struct {
	port          int
	vid           uint16
	pid           uint16
	speed         hal.Speed
	manufacturer  string
	product       string
	serialNumber  string
	hidInterfaces []hidInterface
	connected     bool
}

// deviceRegistry tracks all connected devices.
type deviceRegistry struct {
	devices map[int]*deviceInfo
	mu      sync.RWMutex
}

func newDeviceRegistry() *deviceRegistry {
	return &deviceRegistry{
		devices: make(map[int]*deviceInfo),
	}
}

func (r *deviceRegistry) add(info *deviceInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.devices[info.port] = info
}

func (r *deviceRegistry) remove(port int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if dev, ok := r.devices[port]; ok {
		dev.connected = false
	}
	delete(r.devices, port)
}

func (r *deviceRegistry) get(port int) *deviceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.devices[port]
}

func (r *deviceRegistry) logSummary() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.devices) == 0 {
		pkg.LogInfo(componentMonitor, "device summary", "count", 0)
		return
	}

	connectedCount := 0
	for _, dev := range r.devices {
		if !dev.connected {
			continue
		}
		connectedCount++

		attrs := []any{
			"port", dev.port,
			"vid", dev.vid,
			"pid", dev.pid,
			"speed", dev.speed.String(),
		}
		if dev.manufacturer != "" {
			attrs = append(attrs, "manufacturer", dev.manufacturer)
		}
		if dev.product != "" {
			attrs = append(attrs, "product", dev.product)
		}
		if dev.serialNumber != "" {
			attrs = append(attrs, "serial", dev.serialNumber)
		}
		if len(dev.hidInterfaces) > 0 {
			attrs = append(attrs, "hid_interfaces", len(dev.hidInterfaces))
		}

		pkg.LogInfo(componentMonitor, "device summary", attrs...)
	}

	pkg.LogInfo(componentMonitor, "device summary total", "count", connectedCount)
}

func speedName(s hal.Speed) string {
	switch s {
	case hal.SpeedLow:
		return "Low"
	case hal.SpeedFull:
		return "Full"
	case hal.SpeedHigh:
		return "High"
	default:
		return "Unknown"
	}
}

// Global device registry
var registry = newDeviceRegistry()

// outputCh is the channel for serialized output events
var outputCh = make(chan outputEvent, 100)

func main() {
	flag.Parse()

	// Set up logging based on flags
	if *verbose {
		pkg.SetLogLevel(slog.LevelDebug)
	} else {
		pkg.SetLogLevel(slog.LevelInfo)
	}

	// Configure JSON output if requested
	if *jsonOut {
		pkg.SetLogger(pkg.NewJSONLogger(os.Stderr, &slog.HandlerOptions{
			Level: pkg.GetLogLevel(),
		}))
	}

	// Create HAL
	halImpl := linux.NewHostHAL()

	// Initialize
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := halImpl.Init(ctx); err != nil {
		pkg.LogError(componentMonitor, "failed to initialize HAL", "error", err)
		os.Exit(1)
	}
	defer halImpl.Close()

	// Start HAL
	if err := halImpl.Start(); err != nil {
		pkg.LogError(componentMonitor, "failed to start HAL", "error", err)
		os.Exit(1)
	}

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	pkg.LogInfo(componentMonitor, "started",
		"message", "Waiting for HID devices... (Ctrl+T for device summary, Ctrl+C to exit)")

	// Start output logger goroutine
	go outputLogger(ctx)

	// Set up raw terminal mode to capture Ctrl+T
	go handleKeyboard(ctx, cancel)

	// Main loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Wait for device connection
			port, err := halImpl.WaitForConnection(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			// Get port status
			status, err := halImpl.GetPortStatus(port)
			if err != nil {
				outputCh <- errorEvent{message: "failed to get port status", err: err}
				continue
			}

			if !status.Connected {
				continue
			}

			outputCh <- deviceConnectedEvent{port: port, speed: status.Speed}

			// Start reading HID reports in a goroutine
			go handleDevice(ctx, halImpl, hal.DeviceAddress(port), port, status.Speed)
		}
	}()

	// Wait for disconnect events
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			port, err := halImpl.WaitForDisconnection(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			outputCh <- deviceDisconnectedEvent{port: port}
			registry.remove(port)
		}
	}()

	// Wait for signal
	<-sigCh
	pkg.LogInfo(componentMonitor, "shutting down")
	cancel()
}

// outputLogger reads from outputCh and logs events one at a time.
func outputLogger(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-outputCh:
			event.log()
		}
	}
}

// handleKeyboard reads keyboard input and handles Ctrl+T.
func handleKeyboard(ctx context.Context, cancel context.CancelFunc) {
	// Get the current terminal state
	fd := int(os.Stdin.Fd())

	// Read terminal attributes
	var oldTermios syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		syscall.TCGETS, uintptr(unsafe.Pointer(&oldTermios)), 0, 0, 0); err != 0 {
		return // Can't get terminal state, skip keyboard handling
	}

	// Set raw mode
	newTermios := oldTermios
	newTermios.Lflag &^= syscall.ICANON | syscall.ECHO
	newTermios.Cc[syscall.VMIN] = 1
	newTermios.Cc[syscall.VTIME] = 0

	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		syscall.TCSETS, uintptr(unsafe.Pointer(&newTermios)), 0, 0, 0); err != 0 {
		return
	}

	// Restore terminal on exit
	defer func() {
		syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
			syscall.TCSETS, uintptr(unsafe.Pointer(&oldTermios)), 0, 0, 0)
	}()

	buf := make([]byte, 1)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		switch buf[0] {
		case 0x14: // Ctrl+T
			registry.logSummary()
		case 0x03: // Ctrl+C
			cancel()
			return
		}
	}
}

// handleDevice handles a connected HID device.
func handleDevice(ctx context.Context, halImpl hal.HostHAL, addr hal.DeviceAddress, port int, speed hal.Speed) {
	// Create device info for registry
	devInfo := &deviceInfo{
		port:      port,
		speed:     speed,
		connected: true,
	}

	// Read device descriptor
	setup := &hal.SetupPacket{
		RequestType: 0x80,   // Device to host, standard, device
		Request:     0x06,   // GET_DESCRIPTOR
		Value:       0x0100, // Device descriptor
		Index:       0,
		Length:      18,
	}

	var descBuf [18]byte
	n, err := halImpl.ControlTransfer(ctx, addr, setup, descBuf[:])
	if err != nil {
		registry.add(devInfo)
		outputCh <- deviceEnumeratedEvent{
			info:      devInfo,
			enumError: fmt.Errorf("failed to get device descriptor: %w", err),
		}
		return
	}

	if n < 18 {
		registry.add(devInfo)
		outputCh <- deviceEnumeratedEvent{
			info:      devInfo,
			enumError: fmt.Errorf("short device descriptor: %d bytes", n),
		}
		return
	}

	vid := uint16(descBuf[8]) | uint16(descBuf[9])<<8
	pid := uint16(descBuf[10]) | uint16(descBuf[11])<<8
	manufacturerIdx := descBuf[14]
	productIdx := descBuf[15]
	serialIdx := descBuf[16]

	devInfo.vid = vid
	devInfo.pid = pid

	// Check vendor/product ID filter
	matchesFilter := true
	if *vendorID != "" {
		if filterVID, err := strconv.ParseUint(*vendorID, 16, 16); err == nil {
			if vid != uint16(filterVID) {
				matchesFilter = false
			}
		}
	}
	if *productID != "" {
		if filterPID, err := strconv.ParseUint(*productID, 16, 16); err == nil {
			if pid != uint16(filterPID) {
				matchesFilter = false
			}
		}
	}

	// Read string descriptors
	if manufacturerIdx != 0 {
		devInfo.manufacturer = readStringDescriptor(ctx, halImpl, addr, manufacturerIdx)
	}
	if productIdx != 0 {
		devInfo.product = readStringDescriptor(ctx, halImpl, addr, productIdx)
	}
	if serialIdx != 0 {
		devInfo.serialNumber = readStringDescriptor(ctx, halImpl, addr, serialIdx)
	}

	// Get configuration descriptor
	setup = &hal.SetupPacket{
		RequestType: 0x80,
		Request:     0x06,   // GET_DESCRIPTOR
		Value:       0x0200, // Configuration descriptor
		Index:       0,
		Length:      9, // First get header to find total length
	}

	var configBuf [256]byte
	n, err = halImpl.ControlTransfer(ctx, addr, setup, configBuf[:9])
	if err != nil {
		registry.add(devInfo)
		outputCh <- deviceEnumeratedEvent{
			info:          devInfo,
			matchesFilter: matchesFilter,
			enumError:     fmt.Errorf("failed to get config descriptor: %w", err),
		}
		return
	}

	if n < 9 {
		registry.add(devInfo)
		outputCh <- deviceEnumeratedEvent{
			info:          devInfo,
			matchesFilter: matchesFilter,
			enumError:     fmt.Errorf("short config descriptor: %d bytes", n),
		}
		return
	}

	totalLen := uint16(configBuf[2]) | uint16(configBuf[3])<<8
	if totalLen > 256 {
		totalLen = 256
	}

	// Get full configuration descriptor
	setup.Length = totalLen
	n, err = halImpl.ControlTransfer(ctx, addr, setup, configBuf[:totalLen])
	if err != nil {
		registry.add(devInfo)
		outputCh <- deviceEnumeratedEvent{
			info:          devInfo,
			matchesFilter: matchesFilter,
			enumError:     fmt.Errorf("failed to get full config descriptor: %w", err),
		}
		return
	}

	// Parse to find HID interfaces and their interrupt endpoints
	hidInterfaces := parseHIDInterfaces(configBuf[:n])
	devInfo.hidInterfaces = hidInterfaces

	// Add to registry and send enumeration event
	registry.add(devInfo)
	outputCh <- deviceEnumeratedEvent{
		info:          devInfo,
		matchesFilter: matchesFilter,
	}

	// If filter doesn't match, stop processing
	if !matchesFilter {
		return
	}

	// Claim interfaces and start reading HID reports
	for _, iface := range hidInterfaces {
		if err := halImpl.ClaimInterface(addr, iface.number); err != nil {
			outputCh <- interfaceClaimErrorEvent{port: port, ifaceNum: iface.number, err: err}
			continue
		}
		go readHIDReports(ctx, halImpl, addr, port, iface)
	}
}

// readStringDescriptor reads a string descriptor from a USB device.
func readStringDescriptor(ctx context.Context, halImpl hal.HostHAL, addr hal.DeviceAddress, index uint8) string {
	if index == 0 {
		return ""
	}

	var buf [256]byte

	// Request string descriptor (using US English language ID 0x0409)
	setup := &hal.SetupPacket{
		RequestType: 0x80,                   // Device to host, standard, device
		Request:     0x06,                   // GET_DESCRIPTOR
		Value:       0x0300 | uint16(index), // String descriptor type (0x03) + index
		Index:       0x0409,                 // US English
		Length:      uint16(len(buf)),
	}

	n, err := halImpl.ControlTransfer(ctx, addr, setup, buf[:])
	if err != nil {
		return ""
	}

	if n < 2 {
		return ""
	}

	// Parse Unicode string (skip header, convert UTF-16LE to string)
	length := int(buf[0])
	if length > n {
		length = n
	}
	if length < 2 {
		return ""
	}

	// Simple UTF-16LE to ASCII conversion (ignoring non-ASCII characters)
	result := make([]byte, 0, (length-2)/2)
	for i := 2; i < length-1; i += 2 {
		if buf[i+1] == 0 && buf[i] >= 0x20 && buf[i] < 0x7F {
			result = append(result, buf[i])
		}
	}

	return string(result)
}

// hidInterface describes an HID interface with its interrupt endpoint.
type hidInterface struct {
	number    uint8
	subclass  uint8
	protocol  uint8
	epAddr    uint8
	maxPacket uint16
}

// parseHIDInterfaces parses configuration descriptor to find HID interfaces.
func parseHIDInterfaces(data []byte) []hidInterface {
	var interfaces []hidInterface
	var currentIface *hidInterface

	i := 0
	for i < len(data) {
		if i+1 >= len(data) {
			break
		}

		length := int(data[i])
		if length < 2 || i+length > len(data) {
			break
		}

		descType := data[i+1]

		switch descType {
		case 0x04: // Interface descriptor
			if length >= 9 {
				ifaceNum := data[i+2]
				ifaceClass := data[i+5]
				ifaceSubclass := data[i+6]
				ifaceProtocol := data[i+7]

				if ifaceClass == 0x03 { // HID class
					interfaces = append(interfaces, hidInterface{
						number:   ifaceNum,
						subclass: ifaceSubclass,
						protocol: ifaceProtocol,
					})
					currentIface = &interfaces[len(interfaces)-1]
				} else {
					currentIface = nil
				}
			}

		case 0x05: // Endpoint descriptor
			if length >= 7 && currentIface != nil {
				epAddr := data[i+2]
				epAttr := data[i+3]
				maxPacket := uint16(data[i+4]) | uint16(data[i+5])<<8

				// Check for interrupt IN endpoint
				if epAttr&0x03 == 0x03 && epAddr&0x80 != 0 {
					currentIface.epAddr = epAddr
					currentIface.maxPacket = maxPacket
				}
			}
		}

		i += length
	}

	return interfaces
}

// readHIDReports continuously reads HID reports from an interrupt endpoint.
func readHIDReports(ctx context.Context, halImpl hal.HostHAL, addr hal.DeviceAddress, port int, iface hidInterface) {
	reportBuf := make([]byte, iface.maxPacket)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := halImpl.InterruptTransfer(ctx, addr, iface.epAddr, reportBuf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// NAK is expected when no data is available
			if err == pkg.ErrNAK {
				continue
			}
			outputCh <- errorEvent{
				message: "HID read error",
				err:     fmt.Errorf("port %d interface %d: %w", port, iface.number, err),
			}
			return
		}

		if n > 0 {
			// Make a copy of the data for the event
			dataCopy := make([]byte, n)
			copy(dataCopy, reportBuf[:n])
			outputCh <- hidReportEvent{
				port:      port,
				ifaceNum:  iface.number,
				reportLen: n,
				data:      dataCopy,
			}
		}
	}
}
