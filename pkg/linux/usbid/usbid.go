//go:build linux

package usbid

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
)

// DefaultPaths lists the standard locations for the USB ID database.
var DefaultPaths = []string{
	"/usr/share/hwdata/usb.ids",
	"/var/lib/usbutils/usb.ids",
	"/usr/share/misc/usb.ids",
}

// Database caches vendor and product names from the USB ID database.
type Database struct {
	vendors  map[uint16]string // VID -> vendor name
	products map[uint32]string // (VID<<16)|PID -> product name
	loaded   bool
	mu       sync.RWMutex
	paths    []string
}

// New creates a new USB ID database that searches the default paths.
func New() *Database {
	return &Database{
		vendors:  make(map[uint16]string),
		products: make(map[uint32]string),
		paths:    DefaultPaths,
	}
}

// NewWithPaths creates a new USB ID database that searches the specified paths.
func NewWithPaths(paths []string) *Database {
	return &Database{
		vendors:  make(map[uint16]string),
		products: make(map[uint32]string),
		paths:    paths,
	}
}

// Load parses the USB ID database file. This method is idempotent - subsequent
// calls do nothing if the database is already loaded.
//
// Returns true if the database was loaded (or already loaded), false if no
// database file could be found.
func (db *Database) Load() bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.loaded {
		return true
	}

	var file *os.File
	var err error

	for _, path := range db.paths {
		file, err = os.Open(path)
		if err == nil {
			defer file.Close()
			db.parseDatabase(file)
			db.loaded = true
			return true
		}
	}

	// Mark as loaded even if file not found to prevent repeated searches
	db.loaded = true
	return false
}

// parseDatabase parses the USB ID database format.
func (db *Database) parseDatabase(file *os.File) {
	scanner := bufio.NewScanner(file)
	var currentVID uint16

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Vendor lines start with no whitespace and have format: "xxxx  Vendor Name"
		// Product lines start with a tab and have format: "\txxxx  Product Name"
		if line[0] == '\t' {
			// Product line
			if currentVID == 0 {
				continue
			}
			line = line[1:] // Remove leading tab
			if len(line) < 6 {
				continue
			}
			pidStr := line[:4]
			pid, err := strconv.ParseUint(pidStr, 16, 16)
			if err != nil {
				continue
			}
			// Find the product name (after "xxxx  ")
			if len(line) > 6 && line[4] == ' ' {
				name := strings.TrimLeft(line[5:], " ")
				key := (uint32(currentVID) << 16) | uint32(pid)
				db.products[key] = name
			}
		} else if line[0] != '\t' && len(line) >= 6 {
			// Vendor line
			vidStr := line[:4]
			vid, err := strconv.ParseUint(vidStr, 16, 16)
			if err != nil {
				currentVID = 0
				continue
			}
			currentVID = uint16(vid)
			// Find the vendor name (after "xxxx  ")
			if len(line) > 6 && line[4] == ' ' {
				name := strings.TrimLeft(line[5:], " ")
				db.vendors[currentVID] = name
			}
		} else {
			// Some other line format (interface, class, etc.) - reset current vendor
			currentVID = 0
		}
	}
}

// LookupVendor returns the vendor name for the given VID.
// Returns an empty string if the vendor is not found or if the database
// has not been loaded.
func (db *Database) LookupVendor(vid uint16) string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.vendors[vid]
}

// LookupProduct returns the product name for the given VID/PID combination.
// Returns an empty string if the product is not found or if the database
// has not been loaded.
func (db *Database) LookupProduct(vid, pid uint16) string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	key := (uint32(vid) << 16) | uint32(pid)
	return db.products[key]
}

// IsLoaded returns true if the database has been loaded (or load was attempted).
func (db *Database) IsLoaded() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.loaded
}

// VendorCount returns the number of vendors in the database.
func (db *Database) VendorCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.vendors)
}

// ProductCount returns the number of products in the database.
func (db *Database) ProductCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.products)
}
