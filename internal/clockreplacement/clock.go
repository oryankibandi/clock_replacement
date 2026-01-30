package clock_replacement

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	// max number of times we loop the clock entries to find a suitable candidate
	MAX_LOOP = 25
)

type clock struct {
	// entry that the clock hand points to
	head *entry

	capacity uint64
	mu       sync.RWMutex
}

// advances the clock hand, finds a valid entry to evict and
// clears the entry. Returns evicted entry and it's  key.
// If no suitable entry is found after MAX_LOOP return nil entry
// and -1 as evictedKey
func (clk *clock) evict() (evicted *entry, evictedKey int) {
	start := time.Now()
	clk.mu.Lock()
	defer clk.mu.Unlock()

	for i := 0; i < int(clk.capacity)*MAX_LOOP; i++ {
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
			eKey := clk.head.meta.key
			clk.head.clear()
			e := clk.head

			// advance clock hand
			clk.head = clk.head.links.next

			end := time.Since(start)
			slog.Info(fmt.Sprintf("Evicted in  %v", end))
			return e, int(eKey)
		}
	}

	end := time.Since(start)
	slog.Info(fmt.Sprintf("Evict failed in %v", end))
	// unable to find suitable entry. All entries referenced
	return nil, -1
}

// Returns a pointer to a new circular buffer
func NewClock(head *entry, itemCount uint64) *clock {
	clk := &clock{
		head:     head,
		capacity: itemCount,
	}

	return clk
}
