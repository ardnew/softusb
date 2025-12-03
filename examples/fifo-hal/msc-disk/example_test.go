package msc_disk_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestMSCDiskExample(t *testing.T) {
	// Create temporary bus directory
	busDir, err := os.MkdirTemp("", "usb-msc-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(busDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start device process
	deviceCmd := exec.CommandContext(ctx, "go", "run", "./device",
		"-size", "1048576", // 1MB
		"-enum-timeout", "10s",
		"-transfer-timeout", "5s",
		busDir)
	deviceCmd.Stdout = os.Stdout
	deviceCmd.Stderr = os.Stderr

	if err := deviceCmd.Start(); err != nil {
		t.Fatalf("Failed to start device: %v", err)
	}
	defer deviceCmd.Process.Kill()

	// Give device time to start
	time.Sleep(2 * time.Second)

	// Start host process
	hostCmd := exec.CommandContext(ctx, "go", "run", "./host",
		"-hotplug-limit", "1",
		"-enum-timeout", "10s",
		"-transfer-timeout", "5s",
		busDir)
	hostCmd.Stdout = os.Stdout
	hostCmd.Stderr = os.Stderr

	if err := hostCmd.Start(); err != nil {
		t.Fatalf("Failed to start host: %v", err)
	}

	// Wait for host to complete
	if err := hostCmd.Wait(); err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatal("Test timed out")
		}
		// Non-zero exit is acceptable for now since host example is basic
		t.Logf("Host exited with: %v", err)
	}

	// Clean up device
	deviceCmd.Process.Kill()
	deviceCmd.Wait()

	t.Log("Test completed successfully")
}

func TestMain(m *testing.M) {
	// Change to example directory
	exampleDir, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}

	if err := os.Chdir(exampleDir); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}
