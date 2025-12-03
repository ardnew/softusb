//go:build linux

package linux

import (
	"testing"

	"github.com/ardnew/softusb/host/hal"
)

// =============================================================================
// endpointState Tests
// =============================================================================

func TestEndpointState_Init(t *testing.T) {
	var ep endpointState
	ep.init()

	// Check free list is set up correctly
	if ep.freeHead != 0 {
		t.Errorf("freeHead = %d, want 0", ep.freeHead)
	}
	if ep.pending != 0 {
		t.Errorf("pending = %d, want 0", ep.pending)
	}

	// Check all slots are initialized
	for i := 0; i < MaxURBsPerEndpoint-1; i++ {
		if ep.slots[i].inUse {
			t.Errorf("slots[%d].inUse = true, want false", i)
		}
		if ep.slots[i].next != int8(i+1) {
			t.Errorf("slots[%d].next = %d, want %d", i, ep.slots[i].next, i+1)
		}
	}

	// Last slot should point to -1
	last := MaxURBsPerEndpoint - 1
	if ep.slots[last].next != -1 {
		t.Errorf("slots[%d].next = %d, want -1", last, ep.slots[last].next)
	}
}

func TestEndpointState_AllocFree(t *testing.T) {
	var ep endpointState
	ep.init()

	// Allocate all slots
	allocated := make([]int, MaxURBsPerEndpoint)
	for i := 0; i < MaxURBsPerEndpoint; i++ {
		idx := ep.allocSlot()
		if idx < 0 {
			t.Fatalf("allocSlot failed on iteration %d", i)
		}
		allocated[i] = idx
		if ep.pending != i+1 {
			t.Errorf("pending = %d after alloc %d, want %d", ep.pending, i, i+1)
		}
	}

	// Verify no more slots available
	idx := ep.allocSlot()
	if idx != -1 {
		t.Errorf("allocSlot should return -1 when full, got %d", idx)
	}

	// Free all slots
	for i, idx := range allocated {
		ep.freeSlot(idx)
		if ep.pending != MaxURBsPerEndpoint-i-1 {
			t.Errorf("pending = %d after free %d, want %d",
				ep.pending, i, MaxURBsPerEndpoint-i-1)
		}
	}

	// Should be able to allocate again
	idx = ep.allocSlot()
	if idx < 0 {
		t.Error("allocSlot should succeed after freeing")
	}
}

func TestEndpointState_AllocSlot_InUse(t *testing.T) {
	var ep endpointState
	ep.init()

	idx := ep.allocSlot()
	slot := ep.getSlot(idx)

	if !slot.inUse {
		t.Error("allocated slot should be marked inUse")
	}
}

func TestEndpointState_FreeSlot_Invalid(t *testing.T) {
	var ep endpointState
	ep.init()

	// Free invalid indices should not panic
	ep.freeSlot(-1)
	ep.freeSlot(MaxURBsPerEndpoint)
	ep.freeSlot(MaxURBsPerEndpoint + 100)
}

func TestEndpointState_FreeSlot_NotInUse(t *testing.T) {
	var ep endpointState
	ep.init()

	// Free a slot that's not in use should be idempotent
	initialPending := ep.pending
	ep.freeSlot(0)
	if ep.pending != initialPending {
		t.Errorf("pending changed when freeing unused slot")
	}
}

func TestEndpointState_GetSlot(t *testing.T) {
	var ep endpointState
	ep.init()

	// Valid index
	slot := ep.getSlot(0)
	if slot == nil {
		t.Error("getSlot(0) returned nil")
	}

	// Invalid indices
	if ep.getSlot(-1) != nil {
		t.Error("getSlot(-1) should return nil")
	}
	if ep.getSlot(MaxURBsPerEndpoint) != nil {
		t.Errorf("getSlot(%d) should return nil", MaxURBsPerEndpoint)
	}
}

// =============================================================================
// devicePool Tests
// =============================================================================

func TestDevicePool_Init(t *testing.T) {
	var pool devicePool
	pool.init()

	if pool.freeHead != 0 {
		t.Errorf("freeHead = %d, want 0", pool.freeHead)
	}
	if pool.count != 0 {
		t.Errorf("count = %d, want 0", pool.count)
	}

	// All slots should be free
	for i := 0; i < MaxDevices; i++ {
		if pool.slots[i].conn != nil {
			t.Errorf("slots[%d].conn should be nil", i)
		}
	}
}

func TestDevicePool_AllocFree(t *testing.T) {
	var pool devicePool
	pool.init()

	// Allocate all slots
	for i := 0; i < MaxDevices; i++ {
		idx := pool.alloc(i + 1)
		if idx < 0 {
			t.Fatalf("alloc failed on iteration %d", i)
		}
		if pool.count != i+1 {
			t.Errorf("count = %d after alloc %d, want %d", pool.count, i, i+1)
		}
		if pool.slots[idx].port != i+1 {
			t.Errorf("slots[%d].port = %d, want %d", idx, pool.slots[idx].port, i+1)
		}
	}

	// Verify no more slots available
	idx := pool.alloc(999)
	if idx != -1 {
		t.Errorf("alloc should return -1 when full, got %d", idx)
	}

	// Free a slot
	pool.free(0)
	if pool.count != MaxDevices-1 {
		t.Errorf("count = %d after free, want %d", pool.count, MaxDevices-1)
	}

	// Should be able to allocate again
	idx = pool.alloc(100)
	if idx < 0 {
		t.Error("alloc should succeed after freeing")
	}
}

func TestDevicePool_Free_Invalid(t *testing.T) {
	var pool devicePool
	pool.init()

	// Free invalid indices should not panic
	pool.free(-1)
	pool.free(MaxDevices)
	pool.free(MaxDevices + 100)
}

func TestDevicePool_Get(t *testing.T) {
	var pool devicePool
	pool.init()

	// Initially all nil
	for i := 0; i < MaxDevices; i++ {
		if pool.get(i) != nil {
			t.Errorf("get(%d) should return nil initially", i)
		}
	}

	// Invalid indices
	if pool.get(-1) != nil {
		t.Error("get(-1) should return nil")
	}
	if pool.get(MaxDevices) != nil {
		t.Error("get(MaxDevices) should return nil")
	}
}

func TestDevicePool_Set(t *testing.T) {
	var pool devicePool
	pool.init()

	// Create a mock connection
	conn := &deviceConn{
		address: 1,
	}

	idx := pool.alloc(1)
	pool.set(idx, conn)

	if got := pool.get(idx); got != conn {
		t.Error("get after set returned wrong connection")
	}

	// Set on invalid index should not panic
	pool.set(-1, conn)
	pool.set(MaxDevices, conn)
}

func TestDevicePool_FindByAddress(t *testing.T) {
	var pool devicePool
	pool.init()

	// Add some connections
	conn1 := &deviceConn{address: 1}
	conn2 := &deviceConn{address: 2}

	idx1 := pool.alloc(1)
	pool.set(idx1, conn1)

	idx2 := pool.alloc(2)
	pool.set(idx2, conn2)

	// Find by address
	if got := pool.findByAddress(1); got != conn1 {
		t.Error("findByAddress(1) returned wrong connection")
	}
	if got := pool.findByAddress(2); got != conn2 {
		t.Error("findByAddress(2) returned wrong connection")
	}
	if got := pool.findByAddress(3); got != nil {
		t.Error("findByAddress(3) should return nil")
	}
}

func TestDevicePool_FindByPort(t *testing.T) {
	var pool devicePool
	pool.init()

	conn1 := &deviceConn{address: 1}
	conn2 := &deviceConn{address: 2}

	idx1 := pool.alloc(10)
	pool.set(idx1, conn1)

	idx2 := pool.alloc(20)
	pool.set(idx2, conn2)

	// Find by port
	if got := pool.findByPort(10); got != conn1 {
		t.Error("findByPort(10) returned wrong connection")
	}
	if got := pool.findByPort(20); got != conn2 {
		t.Error("findByPort(20) returned wrong connection")
	}
	if got := pool.findByPort(30); got != nil {
		t.Error("findByPort(30) should return nil")
	}
}

// =============================================================================
// endpointIndex Tests
// =============================================================================

func TestEndpointIndex(t *testing.T) {
	tests := []struct {
		addr     uint8
		expected int
	}{
		{0x00, 0},  // OUT EP0
		{0x01, 1},  // OUT EP1
		{0x0F, 15}, // OUT EP15
		{0x80, 16}, // IN EP0
		{0x81, 17}, // IN EP1
		{0x8F, 31}, // IN EP15
	}

	for _, tt := range tests {
		got := endpointIndex(tt.addr)
		if got != tt.expected {
			t.Errorf("endpointIndex(0x%02X) = %d, want %d", tt.addr, got, tt.expected)
		}
	}
}

// =============================================================================
// deviceConn Tests
// =============================================================================

func TestDeviceConn_IsDisconnected(t *testing.T) {
	conn := &deviceConn{}

	if conn.isDisconnected() {
		t.Error("new connection should not be disconnected")
	}

	conn.markDisconnected()

	if !conn.isDisconnected() {
		t.Error("connection should be disconnected after markDisconnected")
	}
}

func TestDeviceConn_EnsureInterfaceClaimed(t *testing.T) {
	// We can't actually test this without a real device, but we can test
	// the interface validation
	conn := &deviceConn{fd: -1}

	// Invalid interface number should fail
	err := conn.ensureInterfaceClaimed(MaxInterfacesPerDevice)
	if err == nil {
		t.Error("ensureInterfaceClaimed should fail for invalid interface")
	}
}

func TestDeviceConn_ReleaseInterfaceClaim_NotClaimed(t *testing.T) {
	conn := &deviceConn{fd: -1}

	// Releasing interface that's not claimed should succeed
	err := conn.releaseInterfaceClaim(0)
	if err != nil {
		t.Errorf("releaseInterfaceClaim on unclaimed interface failed: %v", err)
	}
}

func TestDeviceConn_ReleaseInterfaceClaim_InvalidInterface(t *testing.T) {
	conn := &deviceConn{fd: -1}

	err := conn.releaseInterfaceClaim(MaxInterfacesPerDevice)
	if err == nil {
		t.Error("releaseInterfaceClaim should fail for invalid interface")
	}
}

// =============================================================================
// urbSlot Tests
// =============================================================================

func TestURBSlot_Complete(t *testing.T) {
	var ep endpointState
	ep.init()

	slot := ep.getSlot(0)
	if slot.complete == nil {
		t.Error("complete channel should be initialized")
	}

	// Channel should have capacity 1
	select {
	case slot.complete <- nil:
		// OK
	default:
		t.Error("complete channel should have capacity")
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkEndpointState_AllocFree(b *testing.B) {
	var ep endpointState
	ep.init()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := ep.allocSlot()
		if idx >= 0 {
			ep.freeSlot(idx)
		}
	}
}

func BenchmarkDevicePool_AllocFree(b *testing.B) {
	var pool devicePool
	pool.init()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := pool.alloc(i % MaxDevices)
		if idx >= 0 {
			pool.free(idx)
		}
	}
}

func BenchmarkDevicePool_FindByAddress(b *testing.B) {
	var pool devicePool
	pool.init()

	// Set up some connections
	for i := 0; i < MaxDevices/2; i++ {
		idx := pool.alloc(i)
		pool.set(idx, &deviceConn{address: hal.DeviceAddress(i + 1)})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.findByAddress(hal.DeviceAddress((i % (MaxDevices / 2)) + 1))
	}
}

func BenchmarkEndpointIndex(b *testing.B) {
	addrs := []uint8{0x01, 0x81, 0x02, 0x82}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = endpointIndex(addrs[i%4])
	}
}
