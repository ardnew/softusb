// Package linux provides a USB host HAL implementation for Linux using usbfs.
//
// This HAL uses the Linux usbfs interface (/dev/bus/usb/) for USB device access,
// sysfs (/sys/bus/usb/devices/) for device discovery, and netlink for hotplug
// event monitoring. It is designed for pure Go with no cgo dependencies.
//
// # Requirements
//
// To use this HAL, the user running the application must have read/write access
// to the USB device nodes in /dev/bus/usb/. This typically requires either:
//   - Running as root
//   - Appropriate udev rules granting access to the user/group
//
// See the softusb-udev-rules command for generating appropriate udev rules.
//
// # Architecture
//
// The HAL uses asynchronous I/O via epoll for efficient USB transfer handling:
//   - URBs (USB Request Blocks) are submitted via USBDEVFS_SUBMITURB
//   - Completion is polled via epoll on the device file descriptor
//   - Completed URBs are reaped via USBDEVFS_REAPURBNDELAY
//
// Device tracking uses pre-allocated fixed-size arrays with free-list management
// for zero-allocation operation in the hot path.
//
// # Supported Features
//
//   - Control, bulk, and interrupt transfers
//   - Device hotplug detection via netlink
//   - Interface claiming with kernel driver detachment
//   - USB 1.1 and USB 2.0 speeds (Low, Full, High)
package linux
