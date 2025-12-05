//go:build linux

package usbid

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNew verifies that New() creates a Database with default paths.
func TestNew(t *testing.T) {
	db := New()
	if db == nil {
		t.Fatal("New() returned nil")
	}
	if len(db.paths) != len(DefaultPaths) {
		t.Errorf("Expected %d paths, got %d", len(DefaultPaths), len(db.paths))
	}
	if db.vendors == nil || db.products == nil {
		t.Error("Database maps not initialized")
	}
}

// TestNewWithPaths verifies that NewWithPaths() creates a Database with custom paths.
func TestNewWithPaths(t *testing.T) {
	customPaths := []string{"/custom/path1", "/custom/path2"}
	db := NewWithPaths(customPaths)
	if db == nil {
		t.Fatal("NewWithPaths() returned nil")
	}
	if len(db.paths) != len(customPaths) {
		t.Errorf("Expected %d paths, got %d", len(customPaths), len(db.paths))
	}
	for i, path := range db.paths {
		if path != customPaths[i] {
			t.Errorf("Path %d: expected %q, got %q", i, customPaths[i], path)
		}
	}
}

// TestLoad_FileNotFound verifies that Load() handles missing files gracefully.
func TestLoad_FileNotFound(t *testing.T) {
	db := NewWithPaths([]string{"/nonexistent/path/usb.ids"})
	loaded := db.Load()
	if loaded {
		t.Error("Load() should return false when file not found")
	}
	if !db.IsLoaded() {
		t.Error("IsLoaded() should return true after Load() attempt")
	}
}

// TestLoad_Idempotent verifies that Load() is idempotent.
func TestLoad_Idempotent(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "usb.ids")
	content := `# Test USB IDs
1234  Test Vendor
	5678  Test Product
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db := NewWithPaths([]string{testFile})

	// First load
	if !db.Load() {
		t.Error("First Load() failed")
	}
	vendorCount1 := db.VendorCount()
	productCount1 := db.ProductCount()

	// Second load should be no-op
	if !db.Load() {
		t.Error("Second Load() failed")
	}
	vendorCount2 := db.VendorCount()
	productCount2 := db.ProductCount()

	if vendorCount1 != vendorCount2 || productCount1 != productCount2 {
		t.Error("Second Load() modified the database")
	}
}

// TestParsing verifies basic database parsing.
func TestParsing(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "usb.ids")
	content := `# USB ID Database
# Comment line

1234  Test Vendor One
	5678  Test Product One
	9abc  Test Product Two
abcd  Test Vendor Two
	def0  Test Product Three

# Another comment
0001  Another Vendor
	0002  Another Product
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db := NewWithPaths([]string{testFile})
	if !db.Load() {
		t.Fatal("Load() failed")
	}

	tests := []struct {
		name        string
		vid         uint16
		pid         uint16
		wantVendor  string
		wantProduct string
	}{
		{
			name:        "First vendor and product",
			vid:         0x1234,
			pid:         0x5678,
			wantVendor:  "Test Vendor One",
			wantProduct: "Test Product One",
		},
		{
			name:        "Second product of first vendor",
			vid:         0x1234,
			pid:         0x9abc,
			wantVendor:  "Test Vendor One",
			wantProduct: "Test Product Two",
		},
		{
			name:        "Second vendor",
			vid:         0xabcd,
			pid:         0xdef0,
			wantVendor:  "Test Vendor Two",
			wantProduct: "Test Product Three",
		},
		{
			name:        "Third vendor",
			vid:         0x0001,
			pid:         0x0002,
			wantVendor:  "Another Vendor",
			wantProduct: "Another Product",
		},
		{
			name:        "Unknown vendor",
			vid:         0xFFFF,
			pid:         0x0000,
			wantVendor:  "",
			wantProduct: "",
		},
		{
			name:        "Known vendor, unknown product",
			vid:         0x1234,
			pid:         0xFFFF,
			wantVendor:  "Test Vendor One",
			wantProduct: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVendor := db.LookupVendor(tt.vid)
			if gotVendor != tt.wantVendor {
				t.Errorf("LookupVendor(0x%04x) = %q, want %q",
					tt.vid, gotVendor, tt.wantVendor)
			}

			gotProduct := db.LookupProduct(tt.vid, tt.pid)
			if gotProduct != tt.wantProduct {
				t.Errorf("LookupProduct(0x%04x, 0x%04x) = %q, want %q",
					tt.vid, tt.pid, gotProduct, tt.wantProduct)
			}
		})
	}
}

// TestCounts verifies VendorCount and ProductCount.
func TestCounts(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "usb.ids")
	content := `1234  Vendor One
	5678  Product One
	abcd  Product Two
5678  Vendor Two
	0001  Product Three
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db := NewWithPaths([]string{testFile})
	if !db.Load() {
		t.Fatal("Load() failed")
	}

	if got := db.VendorCount(); got != 2 {
		t.Errorf("VendorCount() = %d, want 2", got)
	}
	if got := db.ProductCount(); got != 3 {
		t.Errorf("ProductCount() = %d, want 3", got)
	}
}

// TestEmptyDatabase verifies behavior with an empty database.
func TestEmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "usb.ids")
	content := `# Only comments
# No actual data
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db := NewWithPaths([]string{testFile})
	if !db.Load() {
		t.Fatal("Load() failed")
	}

	if got := db.VendorCount(); got != 0 {
		t.Errorf("VendorCount() = %d, want 0", got)
	}
	if got := db.ProductCount(); got != 0 {
		t.Errorf("ProductCount() = %d, want 0", got)
	}
	if got := db.LookupVendor(0x1234); got != "" {
		t.Errorf("LookupVendor() = %q, want empty string", got)
	}
	if got := db.LookupProduct(0x1234, 0x5678); got != "" {
		t.Errorf("LookupProduct() = %q, want empty string", got)
	}
}

// TestMalformedLines verifies that malformed lines are skipped gracefully.
func TestMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "usb.ids")
	content := `# Test malformed lines
1234  Valid Vendor
	5678  Valid Product
ZZZZ  Invalid VID (non-hex)
	YYYY  Invalid PID (non-hex)
12    Too short
	34    Too short
1234Valid Vendor No Space
	5678Valid Product No Space
9abc  Another Valid Vendor
	def0  Another Valid Product
`
	if err := os.WriteFile(testFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	db := NewWithPaths([]string{testFile})
	if !db.Load() {
		t.Fatal("Load() failed")
	}

	// Should have parsed the valid entries
	if got := db.VendorCount(); got != 2 {
		t.Errorf("VendorCount() = %d, want 2", got)
	}
	if got := db.ProductCount(); got != 2 {
		t.Errorf("ProductCount() = %d, want 2", got)
	}

	// Verify the valid entries
	if got := db.LookupVendor(0x1234); got != "Valid Vendor" {
		t.Errorf("LookupVendor(0x1234) = %q, want %q", got, "Valid Vendor")
	}
	if got := db.LookupProduct(0x1234, 0x5678); got != "Valid Product" {
		t.Errorf("LookupProduct(0x1234, 0x5678) = %q, want %q", got, "Valid Product")
	}
	if got := db.LookupVendor(0x9abc); got != "Another Valid Vendor" {
		t.Errorf("LookupVendor(0x9abc) = %q, want %q", got, "Another Valid Vendor")
	}
	if got := db.LookupProduct(0x9abc, 0xdef0); got != "Another Valid Product" {
		t.Errorf("LookupProduct(0x9abc, 0xdef0) = %q, want %q", got, "Another Valid Product")
	}
}
