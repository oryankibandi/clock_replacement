package clock_replacement

import (
	"clock_replacement_algorithm/internal/manual"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

type cache struct {
	bPool struct {
		bufferPool []*entry
		mu         sync.RWMutex
	}

	// hash table mapping item ids to entries. The swiss hash table offers
	// better performance that  Go's standard map
	hashTable map[uint32]*entry
	capacity  uint64

	// circular buffer
	cBuffer *clock
}

type CacheOptions struct {
	// amount of items. At least thrree items required to create a circular buffer
	Capacity uint64
}

func NewCache(options CacheOptions) (*cache, error) {
	if options.Capacity < 3 {
		return nil, errors.New("Minimum capacity is 3")
	}

	slog.Info(fmt.Sprintf("Initializing cache with %d items", options.Capacity))

	var wg sync.WaitGroup
	c := cache{}

	slog.Info(fmt.Sprintf("creating %d entries", options.Capacity))
	for i := uint64(0); i < options.Capacity; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := NewEntry()

			if e == nil {
				panic("Unable to create entry")
			}

			c.bPool.mu.Lock()
			c.bPool.bufferPool = append(c.bPool.bufferPool, e)
			c.bPool.mu.Unlock()

		}()
	}

	for i, ent := range c.bPool.bufferPool {
		if i == 0 {
			// first item
			ent.links.next = c.bPool.bufferPool[i+1]

			ent.links.prev = c.bPool.bufferPool[options.Capacity-1]
		} else if i == int(options.Capacity)-1 {
			// last item
			ent.links.prev = c.bPool.bufferPool[i-1]

			ent.links.next = c.bPool.bufferPool[0]
		} else {
			ent.links.next = c.bPool.bufferPool[i+1]

			ent.links.prev = c.bPool.bufferPool[i-1]
		}
	}

	clk := NewClock(c.bPool.bufferPool[options.Capacity-1], options.Capacity)

	c.cBuffer = clk

	slog.Info("successfully created the circular buffer.")

	return &c, nil
}

// Clears cache and frees allocated memory. Ensure data in buffers that
// should be persisted is flushed to disk before calling Close()
func (c *cache) Close() error {
	var wg sync.WaitGroup

	slog.Info("Freeing buffer memory")
	for _, e := range c.bPool.bufferPool {
		go func() {
			defer wg.Done()
			manual.FreeMem(e.cPtr)
		}()
	}

	for _, v := range c.hashTable {
		// free memory
		manual.FreeMem(v.cPtr)
	}

	for k := range c.hashTable {
		delete(c.hashTable, k)
	}

	return nil
}
