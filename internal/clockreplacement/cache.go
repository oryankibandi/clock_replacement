package clock_replacement

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/oryankibandi/clock_replacement/internal/manual"
)

type DiskManager interface {
	FlushToDisk(key uint32, data []byte) error
	ReadFromDisk(key uint32) (data []byte, err error)
}

type cache struct {
	bPool pool
	// hash table mapping item ids to entries. The swiss hash table offers
	// better performance that  Go's standard map
	hashTable map[uint32]*entry
	capacity  uint64

	// circular buffer
	cBuffer *clock

	// disk manager
	dManager DiskManager

	mu sync.RWMutex
}

type CacheOptions struct {
	// amount of items. At least thrree items required to create a circular buffer
	Capacity uint64
	// Disk manager implementation. This will be used by the cache to retrieve pages and
	// flush to disk
	DManager DiskManager
}

type pool struct {
	bufferPool []*entry
	mu         sync.RWMutex
}

// returns an entry from the pool
func (p *pool) pop() *entry {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.bufferPool) <= 0 {
		return nil
	}

	e := p.bufferPool[len(p.bufferPool)-1]
	p.bufferPool = append([]*entry{}, p.bufferPool[:len(p.bufferPool)-1]...)

	return e
}

// Retrieves an entry from cache, If entry doesn't exist
// return page fault error
func (c *cache) Get(key uint32) (data *[ENTRY_SIZE]byte, err error) {
	c.mu.RLock()

	d, ok := c.hashTable[key]

	if ok {
		c.mu.RUnlock()
		return &d.data, nil
	}

	if c.dManager != nil {
		// get page from disk
		pData, err := c.dManager.ReadFromDisk(key)

		if err != nil {
			c.mu.RUnlock()
			return nil, err
		}

		// add to cache
		c.mu.RUnlock()
		err = c.Put(key, [8192]byte(pData))

		if err != nil {
			return nil, err
		}

		return (*[8192]byte)(pData), nil
	}

	c.mu.RUnlock()
	return nil, errors.New("Page Fault")
}

// Add an item to cache. If all slots are occupied, evict an item.
func (c *cache) Put(key uint32, data [ENTRY_SIZE]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	existing, ok := c.hashTable[key]

	if ok {
		// in place update
		existing.setData(key, data)

		return nil
	}

	if len(c.hashTable) < int(c.capacity) {
		// cache not full, add item
		// get entry from buffer pool
		e := c.bPool.pop()

		if e == nil {
			panic("Invalid state. Cache not full but buffer pool is empty")
		}

		err := e.setData(key, data)

		if err != nil {
			return err
		}

		// add to hash table
		c.hashTable[key] = e
	} else {
		// cache full, find item to evict
		e, evictedKey := c.cBuffer.evict()

		if evictedKey == -1 {
			// could  not evict
			slog.Info("Unable to evict key, no suitable entry found")
			return errors.New("Could not evict key")
		}

		// flush to disk before clearing
		if c.dManager != nil {
			err := c.dManager.FlushToDisk(uint32(evictedKey), e.data[:])

			if err != nil {
				return nil
			}
		}

		// clear data in the entry/frame
		e.clear()

		// delete evicted  entry from hashtable
		delete(c.hashTable, uint32(evictedKey))

		err := e.setData(key, data)

		if err != nil {
			return err
		}

		c.hashTable[key] = e
	}

	return nil
}

// removes an item from cache
func (c *cache) Delete(key uint32) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ent, ok := c.hashTable[key]

	if !ok {
		msg := fmt.Sprintf("Entry with key: %d not in set", key)
		slog.Info(msg)
		return errors.New(msg)
	}

	// flush to disk before clearing
	if c.dManager != nil {
		err := c.dManager.FlushToDisk(uint32(key), ent.data[:])

		if err != nil {
			return nil
		}
	}

	ent.clear()
	delete(c.hashTable, key)

	// readd entry to buffer pool
	c.bPool.mu.Lock()
	c.bPool.bufferPool = append(c.bPool.bufferPool, ent)
	c.bPool.mu.Unlock()

	return nil
}

// Clears cache and frees allocated memory. Ensure data in buffers that
// should be persisted is flushed to disk before calling Close()
func (c *cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var wg sync.WaitGroup

	slog.Info("Freeing buffer memory")
	for _, e := range c.bPool.bufferPool {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manual.FreeMem(e.cPtr)
		}()
	}

	wg.Wait()
	slog.Info("Freed unused buffer pool memory")

	for _, v := range c.hashTable {
		// clear and free memory
		v.clear()
		manual.FreeMem(v.cPtr)
	}

	for k := range c.hashTable {
		delete(c.hashTable, k)
	}

	slog.Info("cleared hash table")

	return nil
}

// creates a new cache of size CacheOptions.Capacity and allocates memory
// to the buffer pool.
func NewCache(options CacheOptions) (*cache, error) {
	if options.Capacity < 3 {
		return nil, errors.New("Minimum capacity is 3")
	}

	slog.Info(fmt.Sprintf("Initializing cache with %d items", options.Capacity))

	var wg sync.WaitGroup
	c := cache{
		capacity:  options.Capacity,
		hashTable: make(map[uint32]*entry),
	}

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

	wg.Wait()

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
