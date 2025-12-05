# usbid Package

The `usbid` package provides access to the USB ID database for looking up vendor and product names on Linux systems.

## Overview

The USB ID database is a standard file maintained by the USB Implementers Forum and distributed with most Linux systems. It maps USB vendor IDs (VID) and product IDs (PID) to human-readable names.

This package automatically searches common database locations and provides efficient cached lookups with thread-safe concurrent access.

## Build Constraints

This package is Linux-only and includes the build constraint `//go:build linux` in all files.

## Installation

```bash
go get github.com/ardnew/softusb/pkg/linux/usbid
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "github.com/ardnew/softusb/pkg/linux/usbid"
)

func main() {
    // Create and load the database
    db := usbid.New()
    if !db.Load() {
        fmt.Println("Warning: USB ID database not found")
    }

    // Look up vendor name
    vendorName := db.LookupVendor(0x046d) // Logitech
    fmt.Printf("Vendor: %s\n", vendorName)

    // Look up product name
    productName := db.LookupProduct(0x046d, 0xc52b) // Logitech Unifying Receiver
    fmt.Printf("Product: %s\n", productName)
}
```

### Custom Database Paths

```go
db := usbid.NewWithPaths([]string{
    "/custom/path/usb.ids",
    "/another/path/usb.ids",
})
db.Load()
```

### Check Database Status

```go
db := usbid.New()
db.Load()

fmt.Printf("Loaded: %v\n", db.IsLoaded())
fmt.Printf("Vendors: %d\n", db.VendorCount())
fmt.Printf("Products: %d\n", db.ProductCount())
```

## API Reference

### Types

#### `Database`

The main database type that caches vendor and product names.

### Functions

#### `New() *Database`

Creates a new USB ID database that searches the default paths:

- `/usr/share/hwdata/usb.ids`
- `/var/lib/usbutils/usb.ids`
- `/usr/share/misc/usb.ids`

#### `NewWithPaths(paths []string) *Database`

Creates a new USB ID database that searches the specified paths.

### Methods

#### `(*Database) Load() bool`

Parses the USB ID database file. This method is idempotent - subsequent calls do nothing if the database is already loaded. Returns `true` if the database was loaded (or already loaded), `false` if no database file could be found.

#### `(*Database) LookupVendor(vid uint16) string`

Returns the vendor name for the given VID. Returns an empty string if the vendor is not found or if the database has not been loaded.

#### `(*Database) LookupProduct(vid, pid uint16) string`

Returns the product name for the given VID/PID combination. Returns an empty string if the product is not found or if the database has not been loaded.

#### `(*Database) IsLoaded() bool`

Returns `true` if the database has been loaded (or load was attempted).

#### `(*Database) VendorCount() int`

Returns the number of vendors in the database.

#### `(*Database) ProductCount() int`

Returns the number of products in the database.

## Thread Safety

All methods are safe for concurrent use. The database uses read-write locks to allow concurrent lookups while protecting against concurrent loads.

## Database Format

The USB ID database uses a simple text format:

```text
# Comment lines start with #
#  (Product lines are indented with a single tab '\t' — shown below as 8 spaces)
1234  Vendor Name
        5678  Product Name
        9abc  Another Product
abcd  Another Vendor
        def0  Product Name
```

- Vendor lines start at column 0 with a 4-digit hexadecimal VID
- Product lines start with a tab and contain a 4-digit hexadecimal PID
- Names follow two spaces after the ID

## Error Handling

The package is designed to be resilient:

- If the database file is not found, lookups return empty strings
- Malformed lines are silently skipped
- Thread-safe concurrent access is guaranteed

## Testing

Run the tests with:

```bash
go test -v ./pkg/linux/usbid/
```

The test suite includes:

- Basic parsing and lookup tests
- Idempotent load verification
- Malformed input handling
- Empty database handling
- Concurrent access tests

## Example: USB Device Monitor

See `examples/linux-hal/hid-monitor/main.go` for a complete example of using this package to provide human-readable names for USB devices detected by the Linux host HAL.
