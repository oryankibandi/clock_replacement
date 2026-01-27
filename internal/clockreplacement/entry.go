package clock_replacement

import (
	"clock_replacement_algorithm/internal/manual"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	ENTRY_SIZE uint64 = 8192
)

type counter struct {
	pinCount   atomic.Uint64
	unpinCount atomic.Uint64
}

type metadata struct {
	isDirty atomic.Bool

	// Unique key of the entry. This can be the page block id
	key uint64
}

type entry struct {
	// reference bit. Set when an item is accessed and unset by clock hand when
	// looking for an item to evict
	ref atomic.Bool

	// access bit. set when an entry is accessed(pinned) and unset during unpinning
	// When this item is set the reference bit cannot be unset. The clock hand will
	// advance past an entry with it's access bit set
	acc      atomic.Bool
	counters counter

	// size of an entry. Default is 8K
	dataSize uint64
	meta     metadata

	// page data
	data [ENTRY_SIZE]byte

	// pointers to next aand prev values
	links struct {
		next *entry
		prev *entry
	}

	// unsafe pointer used to free memory
	cPtr unsafe.Pointer

	mu sync.Mutex
}

func (c *counter) addPinCount() {
	c.pinCount.Add(1)
}

func (c *counter) addUnpinCount() {
	c.unpinCount.Add(1)
}

func (c *counter) getTotalPins() uint64 {
	diff := c.pinCount.Load() - c.unpinCount.Load()

	if diff < 0 {
		panic("Invalid pin count")
	}

	return diff
}

func (c *counter) reset() {
	c.pinCount.Store(0)
	c.unpinCount.Store(0)
}

// Sets the access bit and ref bit of an entry. Called when accessing an entry
func (e *entry) reference() {
	e.ref.Store(true)
	e.acc.Store(true)

	// increment pin count
	e.counters.addPinCount()
}

// unreferences an entry. Reduces pin count and if no pins left, unset access bit
func (e *entry) unreference() {
	e.counters.addUnpinCount()

	e.mu.Lock()
	// check current count
	p := e.counters.getTotalPins()

	// If no pins, unset access bit
	if p == 0 {
		// unset access pin
		e.acc.Store(false)
	}

	e.mu.Unlock()
}

func (e *entry) markDirty() {
	e.meta.isDirty.Store(true)
}

func (e *entry) markClean() {
	e.meta.isDirty.Store(false)
}

func (e *entry) setData(data [ENTRY_SIZE]byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.data = data

	return nil
}

// zeros out the entry and resets al fields
func (e *entry) clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	s := make([]byte, ENTRY_SIZE)
	copy(e.data[:], s)

	e.ref.Store(false)
	e.acc.Store(false)

	e.meta.isDirty.Store(false)
	e.meta.key = 0

	e.counters.reset()
}

// Returns a pointer to new entry
// To reduce pressure on the GC and improve performance, memory is allocated manually via calloc().
// This memory also needs to be freed after use to avoid memory leaks.
// In a storage engine's buffer manager, this memory will be initialized at startup and reused as blocks are paged-in and evicted.
func New() *entry {
	p := manual.Alloc(unsafe.Sizeof(entry{}))

	e := (*entry)(p)

	e.dataSize = ENTRY_SIZE
	e.cPtr = p

	return e
}
