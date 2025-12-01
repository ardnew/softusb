package hid

// HID class codes.
const (
	ClassHID = 0x03 // Human Interface Device Class
)

// HID subclass codes.
const (
	SubclassNone = 0x00 // No subclass
	SubclassBoot = 0x01 // Boot Interface Subclass
)

// HID protocol codes (for boot interface).
const (
	ProtocolNone     = 0x00 // No protocol
	ProtocolKeyboard = 0x01 // Keyboard boot protocol
	ProtocolMouse    = 0x02 // Mouse boot protocol
)

// HID descriptor types.
const (
	DescriptorTypeHID      = 0x21 // HID descriptor
	DescriptorTypeReport   = 0x22 // Report descriptor
	DescriptorTypePhysical = 0x23 // Physical descriptor
)

// HID request codes.
const (
	RequestGetReport   = 0x01
	RequestGetIdle     = 0x02
	RequestGetProtocol = 0x03
	RequestSetReport   = 0x09
	RequestSetIdle     = 0x0A
	RequestSetProtocol = 0x0B
)

// Report types (high byte of wValue in GET_REPORT/SET_REPORT).
const (
	ReportTypeInput   = 0x01
	ReportTypeOutput  = 0x02
	ReportTypeFeature = 0x03
)

// Protocol values for GET_PROTOCOL/SET_PROTOCOL.
const (
	ProtocolBoot   = 0x00 // Boot protocol
	ProtocolReport = 0x01 // Report protocol
)

// HIDDescriptor is the HID class descriptor.
type HIDDescriptor struct {
	Length         uint8  // Size of this descriptor (9)
	DescriptorType uint8  // HID (0x21)
	HIDVersion     uint16 // HID specification release number (0x0111 for 1.11)
	CountryCode    uint8  // Country code
	NumDescriptors uint8  // Number of class descriptors (at least 1)
	ReportDescType uint8  // Report descriptor type (0x22)
	ReportDescLen  uint16 // Total size of report descriptor
}

// HIDDescriptorSize is the size of the HID descriptor.
const HIDDescriptorSize = 9

// MarshalTo writes the HID descriptor to buf.
// Returns the number of bytes written, or 0 if buf is too small.
func (d *HIDDescriptor) MarshalTo(buf []byte) int {
	if len(buf) < HIDDescriptorSize {
		return 0
	}
	buf[0] = HIDDescriptorSize
	buf[1] = DescriptorTypeHID
	buf[2] = byte(d.HIDVersion)
	buf[3] = byte(d.HIDVersion >> 8)
	buf[4] = d.CountryCode
	buf[5] = d.NumDescriptors
	buf[6] = DescriptorTypeReport
	buf[7] = byte(d.ReportDescLen)
	buf[8] = byte(d.ReportDescLen >> 8)
	return HIDDescriptorSize
}

// Common country codes.
const (
	CountryNone       = 0x00
	CountryArabic     = 0x01
	CountryBelgian    = 0x02
	CountryCanadian   = 0x03
	CountryCzech      = 0x04
	CountryDanish     = 0x05
	CountryFinnish    = 0x06
	CountryFrench     = 0x07
	CountryGerman     = 0x08
	CountryGreek      = 0x09
	CountryHebrew     = 0x0A
	CountryHungarian  = 0x0B
	CountryISO        = 0x0C
	CountryItalian    = 0x0D
	CountryJapanese   = 0x0E
	CountryKorean     = 0x0F
	CountryLatin      = 0x10
	CountryDutch      = 0x11
	CountryNorwegian  = 0x12
	CountryPersian    = 0x13
	CountryPolish     = 0x14
	CountryPortuguese = 0x15
	CountryRussian    = 0x16
	CountrySlovak     = 0x17
	CountrySpanish    = 0x18
	CountrySwedish    = 0x19
	CountrySwissF     = 0x1A
	CountrySwissG     = 0x1B
	CountrySwiss      = 0x1C
	CountryTaiwan     = 0x1D
	CountryTurkishQ   = 0x1E
	CountryUK         = 0x1F
	CountryUS         = 0x20
	CountryYugoslav   = 0x21
	CountryTurkishF   = 0x22
)

// Keyboard modifier bits.
const (
	ModLeftCtrl   = 1 << 0
	ModLeftShift  = 1 << 1
	ModLeftAlt    = 1 << 2
	ModLeftGUI    = 1 << 3
	ModRightCtrl  = 1 << 4
	ModRightShift = 1 << 5
	ModRightAlt   = 1 << 6
	ModRightGUI   = 1 << 7
)

// Keyboard LED bits (for output report).
const (
	LEDNumLock    = 1 << 0
	LEDCapsLock   = 1 << 1
	LEDScrollLock = 1 << 2
	LEDCompose    = 1 << 3
	LEDKana       = 1 << 4
)

// Common keyboard keycodes (USB HID Usage Tables).
const (
	KeyNone        = 0x00
	KeyA           = 0x04
	KeyB           = 0x05
	KeyC           = 0x06
	KeyD           = 0x07
	KeyE           = 0x08
	KeyF           = 0x09
	KeyG           = 0x0A
	KeyH           = 0x0B
	KeyI           = 0x0C
	KeyJ           = 0x0D
	KeyK           = 0x0E
	KeyL           = 0x0F
	KeyM           = 0x10
	KeyN           = 0x11
	KeyO           = 0x12
	KeyP           = 0x13
	KeyQ           = 0x14
	KeyR           = 0x15
	KeyS           = 0x16
	KeyT           = 0x17
	KeyU           = 0x18
	KeyV           = 0x19
	KeyW           = 0x1A
	KeyX           = 0x1B
	KeyY           = 0x1C
	KeyZ           = 0x1D
	Key1           = 0x1E
	Key2           = 0x1F
	Key3           = 0x20
	Key4           = 0x21
	Key5           = 0x22
	Key6           = 0x23
	Key7           = 0x24
	Key8           = 0x25
	Key9           = 0x26
	Key0           = 0x27
	KeyEnter       = 0x28
	KeyEscape      = 0x29
	KeyBackspace   = 0x2A
	KeyTab         = 0x2B
	KeySpace       = 0x2C
	KeyMinus       = 0x2D
	KeyEqual       = 0x2E
	KeyLeftBrace   = 0x2F
	KeyRightBrace  = 0x30
	KeyBackslash   = 0x31
	KeySemicolon   = 0x33
	KeyQuote       = 0x34
	KeyGrave       = 0x35
	KeyComma       = 0x36
	KeyDot         = 0x37
	KeySlash       = 0x38
	KeyCapsLock    = 0x39
	KeyF1          = 0x3A
	KeyF2          = 0x3B
	KeyF3          = 0x3C
	KeyF4          = 0x3D
	KeyF5          = 0x3E
	KeyF6          = 0x3F
	KeyF7          = 0x40
	KeyF8          = 0x41
	KeyF9          = 0x42
	KeyF10         = 0x43
	KeyF11         = 0x44
	KeyF12         = 0x45
	KeyPrintScreen = 0x46
	KeyScrollLock  = 0x47
	KeyPause       = 0x48
	KeyInsert      = 0x49
	KeyHome        = 0x4A
	KeyPageUp      = 0x4B
	KeyDelete      = 0x4C
	KeyEnd         = 0x4D
	KeyPageDown    = 0x4E
	KeyRight       = 0x4F
	KeyLeft        = 0x50
	KeyDown        = 0x51
	KeyUp          = 0x52
)

// Mouse button bits.
const (
	MouseButtonLeft   = 1 << 0
	MouseButtonRight  = 1 << 1
	MouseButtonMiddle = 1 << 2
)

// KeyboardReportDescriptor is a standard 8-byte keyboard report descriptor.
// Report format: [modifiers, reserved, key1, key2, key3, key4, key5, key6]
var KeyboardReportDescriptor = []byte{
	0x05, 0x01, // Usage Page (Generic Desktop)
	0x09, 0x06, // Usage (Keyboard)
	0xA1, 0x01, // Collection (Application)
	0x05, 0x07, //   Usage Page (Keyboard/Keypad)
	0x19, 0xE0, //   Usage Minimum (Left Control)
	0x29, 0xE7, //   Usage Maximum (Right GUI)
	0x15, 0x00, //   Logical Minimum (0)
	0x25, 0x01, //   Logical Maximum (1)
	0x75, 0x01, //   Report Size (1)
	0x95, 0x08, //   Report Count (8)
	0x81, 0x02, //   Input (Data, Variable, Absolute) - Modifier byte
	0x95, 0x01, //   Report Count (1)
	0x75, 0x08, //   Report Size (8)
	0x81, 0x01, //   Input (Constant) - Reserved byte
	0x95, 0x05, //   Report Count (5)
	0x75, 0x01, //   Report Size (1)
	0x05, 0x08, //   Usage Page (LEDs)
	0x19, 0x01, //   Usage Minimum (Num Lock)
	0x29, 0x05, //   Usage Maximum (Kana)
	0x91, 0x02, //   Output (Data, Variable, Absolute) - LED report
	0x95, 0x01, //   Report Count (1)
	0x75, 0x03, //   Report Size (3)
	0x91, 0x01, //   Output (Constant) - Padding
	0x95, 0x06, //   Report Count (6)
	0x75, 0x08, //   Report Size (8)
	0x15, 0x00, //   Logical Minimum (0)
	0x26, 0xFF, 0x00, // Logical Maximum (255)
	0x05, 0x07, //   Usage Page (Keyboard/Keypad)
	0x19, 0x00, //   Usage Minimum (0)
	0x2A, 0xFF, 0x00, // Usage Maximum (255)
	0x81, 0x00, //   Input (Data, Array) - Key array
	0xC0, // End Collection
}

// MouseReportDescriptor is a standard 4-byte mouse report descriptor.
// Report format: [buttons, X, Y, wheel]
var MouseReportDescriptor = []byte{
	0x05, 0x01, // Usage Page (Generic Desktop)
	0x09, 0x02, // Usage (Mouse)
	0xA1, 0x01, // Collection (Application)
	0x09, 0x01, //   Usage (Pointer)
	0xA1, 0x00, //   Collection (Physical)
	0x05, 0x09, //     Usage Page (Button)
	0x19, 0x01, //     Usage Minimum (Button 1)
	0x29, 0x03, //     Usage Maximum (Button 3)
	0x15, 0x00, //     Logical Minimum (0)
	0x25, 0x01, //     Logical Maximum (1)
	0x95, 0x03, //     Report Count (3)
	0x75, 0x01, //     Report Size (1)
	0x81, 0x02, //     Input (Data, Variable, Absolute) - Button bits
	0x95, 0x01, //     Report Count (1)
	0x75, 0x05, //     Report Size (5)
	0x81, 0x01, //     Input (Constant) - Padding
	0x05, 0x01, //     Usage Page (Generic Desktop)
	0x09, 0x30, //     Usage (X)
	0x09, 0x31, //     Usage (Y)
	0x09, 0x38, //     Usage (Wheel)
	0x15, 0x81, //     Logical Minimum (-127)
	0x25, 0x7F, //     Logical Maximum (127)
	0x75, 0x08, //     Report Size (8)
	0x95, 0x03, //     Report Count (3)
	0x81, 0x06, //     Input (Data, Variable, Relative) - X, Y, Wheel
	0xC0, //   End Collection
	0xC0, // End Collection
}

// KeyboardReport is an 8-byte keyboard input report.
type KeyboardReport struct {
	Modifiers uint8    // Modifier key state
	Reserved  uint8    // Reserved (always 0)
	Keys      [6]uint8 // Up to 6 simultaneous key codes
}

// KeyboardReportSize is the size of a keyboard report in bytes.
const KeyboardReportSize = 8

// MarshalTo writes the keyboard report to buf.
func (r *KeyboardReport) MarshalTo(buf []byte) int {
	if len(buf) < KeyboardReportSize {
		return 0
	}
	buf[0] = r.Modifiers
	buf[1] = r.Reserved
	buf[2] = r.Keys[0]
	buf[3] = r.Keys[1]
	buf[4] = r.Keys[2]
	buf[5] = r.Keys[3]
	buf[6] = r.Keys[4]
	buf[7] = r.Keys[5]
	return KeyboardReportSize
}

// Clear resets the keyboard report to all keys released.
func (r *KeyboardReport) Clear() {
	r.Modifiers = 0
	r.Reserved = 0
	r.Keys = [6]uint8{}
}

// SetKey sets a key in the key array.
// Returns false if no slot is available.
func (r *KeyboardReport) SetKey(key uint8) bool {
	for i := range r.Keys {
		if r.Keys[i] == 0 {
			r.Keys[i] = key
			return true
		}
		if r.Keys[i] == key {
			return true // Already set
		}
	}
	return false
}

// ClearKey removes a key from the key array.
func (r *KeyboardReport) ClearKey(key uint8) {
	for i := range r.Keys {
		if r.Keys[i] == key {
			// Shift remaining keys
			for j := i; j < len(r.Keys)-1; j++ {
				r.Keys[j] = r.Keys[j+1]
			}
			r.Keys[len(r.Keys)-1] = 0
			return
		}
	}
}

// MouseReport is a 4-byte mouse input report.
type MouseReport struct {
	Buttons uint8 // Button state
	X       int8  // X movement (-127 to 127)
	Y       int8  // Y movement (-127 to 127)
	Wheel   int8  // Wheel movement (-127 to 127)
}

// MouseReportSize is the size of a mouse report in bytes.
const MouseReportSize = 4

// MarshalTo writes the mouse report to buf.
func (r *MouseReport) MarshalTo(buf []byte) int {
	if len(buf) < MouseReportSize {
		return 0
	}
	buf[0] = r.Buttons
	buf[1] = byte(r.X)
	buf[2] = byte(r.Y)
	buf[3] = byte(r.Wheel)
	return MouseReportSize
}

// Clear resets the mouse report.
func (r *MouseReport) Clear() {
	r.Buttons = 0
	r.X = 0
	r.Y = 0
	r.Wheel = 0
}
