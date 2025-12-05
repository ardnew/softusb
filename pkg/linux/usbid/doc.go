//go:build linux

// Package usbid provides access to the USB ID database for looking up vendor
// and product names.
//
// The USB ID database is a standard file maintained by the USB Implementers
// Forum and distributed with most Linux systems. It maps USB vendor IDs (VID)
// and product IDs (PID) to human-readable names.
//
// This package automatically searches common database locations and provides
// efficient cached lookups.
//
// # Usage
//
// Load the database once at startup:
//
//	db := usbid.New()
//	db.Load()
//
// Then look up vendor and product names:
//
//	vendorName := db.LookupVendor(0x1234)
//	productName := db.LookupProduct(0x1234, 0x5678)
//
// # Database Locations
//
// The package searches for the USB ID database in these locations:
//
//   - /usr/share/hwdata/usb.ids
//   - /var/lib/usbutils/usb.ids
//   - /usr/share/misc/usb.ids
//
// If the database file is not found, lookup methods return empty strings.
//
// # Thread Safety
//
// All methods are safe for concurrent use. The database uses read-write locks
// to allow concurrent lookups while protecting against concurrent loads.
package usbid
