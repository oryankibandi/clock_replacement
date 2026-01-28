package clock_replacement

import "sync"

type clock struct {
	// entry that the clock hand points to
	head *entry

	capacity uint64
	mu       sync.RWMutex
}

// advances the clock hand, finds a valid entry to evict and
// clears the entry
func (clk *clock) evict() *entry {
	// TODO: Add time limit(context cancelation) to avoid infinite loops
	clk.mu.Lock()
	defer clk.mu.Unlock()

	for {
		if clk.head.acc.Load() {
			// access bit set, advance clock hand
			clk.head = clk.head.links.next
			continue
		}

		if clk.head.ref.Load() {
			// ref bit set, unset it
			clk.head.unsetRef()

			clk.head = clk.head.links.next
		} else {
			// both access bit and reference bit unset, clear and evict
			clk.head.clear()
			e := clk.head

			// advance clock hand
			clk.head = clk.head.links.next

			return e
		}
	}
}

// Returns a pointer to a new circular buffer
func NewClock(head *entry, itemCount uint64) *clock {
	clk := &clock{
		head:     head,
		capacity: itemCount,
	}

	return clk
}
