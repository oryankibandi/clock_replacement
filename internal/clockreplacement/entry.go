package clock_replacement

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/oryankibandi/clock_replacement/internal/manual"
)

const (
	ENTRY_SIZE uint64 = 8192 // 8K
)

type counter struct {
	pinCount   atomic.Uint64
	unpinCount atomic.Uint64
}

type metadata struct {
	isDirty atomic.Bool

	// Unique key of the entry. This can be the page block id
	key uint32
}

type Entry struct {
	// reference bit. Set when an item is accessed and unset by clock hand when
	// looking for an item to evict
	ref atomic.Bool

	// access bit. set when an entry is accessed(pinned) and unset during unpinning
	// When this item is set the reference bit cannot be unset. The clock hand will
	// advance past an entry with it's access bit set
	acc atomic.Bool

	// if its allocated
	isOccupied atomic.Bool
	counters   counter
	meta       metadata

	// page data
	Data [ENTRY_SIZE]byte

	// pointers to next aand prev values
	links struct {
		next *Entry
		prev *Entry
	}

	// unsafe pointer used to free memory
	CPtr unsafe.Pointer

	mu sync.RWMutex
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

// Sets the access bit and ref bit of an entry. Called when accessing an entry.
// The process that uses the entry data is required to call Unreference() when done
func (e *Entry) Reference() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ref.Store(true)
	e.acc.Store(true)

	// increment pin count
	e.counters.addPinCount()
}

func (e *Entry) SetNextLink(n *Entry) {
	if n == nil {
		panic("invalid nil entry provided.")
	}

	e.links.next = n
}

func (e *Entry) SetPrevLink(p *Entry) {
	if p == nil {
		panic("invalid nil entry provided.")
	}

	e.links.prev = p
}

func (e *Entry) GetNextLink() *Entry {
	return e.links.next
}

func (e *Entry) GetPrevLink() *Entry {
	return e.links.prev
}

// unreferences an entry. Reduces pin count and if no pins left, unsets access bit.
func (e *Entry) Unreference() {
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

// Returns true if access bit is set, else false
func (e *Entry) accessBitSet() bool {
	return e.acc.Load()
}

// Returns true if access bit is set, else false
func (e *Entry) refBitSet() bool {
	return e.ref.Load()
}

// Mark an entry/frame as dirty
func (e *Entry) MarkDirty() {
	e.meta.isDirty.Store(true)
}

// Mark an entry/frame as clean
func (e *Entry) MarkClean() {
	e.meta.isDirty.Store(false)
}

// Unsets the reference bit. This is exclusively called by the clock replacement algorithm.
func (e *Entry) unsetRef() {
	e.ref.Store(false)
}

// Sets an entry's byte data.
func (e *Entry) SetData(key uint32, data []byte) error {
	if len(data) > int(ENTRY_SIZE) {
		msg := fmt.Sprintf("Data of size %d exceeds set size %d", len(data), ENTRY_SIZE)
		slog.Error(msg)
		return fmt.Errorf("Data of size %d exceeds set size %d", len(data), ENTRY_SIZE)
	}

	var formatted [ENTRY_SIZE]byte
	copy(formatted[:], data)

	e.mu.Lock()
	defer e.mu.Unlock()

	e.Data = formatted
	e.meta.key = key
	e.isOccupied.Store(true)

	return nil
}

func (e *Entry) UpdateData(data []byte) error {
	if len(data) > int(ENTRY_SIZE) {
		return fmt.Errorf("Data of size %d exceeds set size %d", len(data), ENTRY_SIZE)
	}

	var formatted [ENTRY_SIZE]byte
	copy(formatted[:], data)

	e.mu.Lock()
	defer e.mu.Unlock()

	e.Data = formatted
	e.MarkDirty()

	return nil
}

func (e *Entry) getKey() uint32 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.meta.key
}

func (e *Entry) GetData() [ENTRY_SIZE]byte {
	var d [ENTRY_SIZE]byte

	e.mu.RLock()
	defer e.mu.RUnlock()

	copy(d[:], e.Data[:])

	return d
}

// zeros out the entry and resets all fields
func (e *Entry) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	s := make([]byte, ENTRY_SIZE)
	copy(e.Data[:], s)

	e.ref.Store(false)
	e.acc.Store(false)

	e.meta.isDirty.Store(false)
	e.meta.key = 0

	e.counters.reset()

	// mark as unallocated
	e.isOccupied.Store(false)
}

// Returns a pointer to new entry
// To reduce pressure on the GC and improve performance, memory is allocated manually via calloc().
// This memory also needs to be freed after use to avoid memory leaks.
// In a storage engine's buffer manager, this memory will be initialized at startup and reused as blocks are paged-in and evicted.
func NewEntry() *Entry {
	p := manual.Alloc(unsafe.Sizeof(Entry{}))

	e := (*Entry)(p)

	e.CPtr = p

	return e
}
