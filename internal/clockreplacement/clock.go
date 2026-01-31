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

type Clock struct {
	// entry that the clock hand points to
	Head *Entry

	capacity uint64
	mu       sync.RWMutex
}

// advances the clock hand, finds a valid entry to evict and
// clears the entry. Returns evicted entry and it's  key.
// If no suitable entry is found after MAX_LOOP return nil entry
// and -1 as evictedKey
func (clk *Clock) Evict() (evicted *Entry, evictedKey int) {
	start := time.Now()
	clk.mu.Lock()
	defer clk.mu.Unlock()

	for i := 0; i < int(clk.capacity)*MAX_LOOP; i++ {
		if clk.Head.acc.Load() {
			// access bit set, advance clock hand
			clk.Head = clk.Head.links.next
			continue
		}

		if clk.Head.ref.Load() {
			// ref bit set, unset it
			clk.Head.unsetRef()

			clk.Head = clk.Head.links.next
		} else {
			// both access bit and reference bit unset, clear and evict
			eKey := clk.Head.meta.key
			// clk.Head.clear()
			e := clk.Head

			// advance clock hand
			clk.Head = clk.Head.links.next

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
func NewClock(Head *Entry, itemCount uint64) *Clock {
	clk := &Clock{
		Head:     Head,
		capacity: itemCount,
	}

	return clk
}
