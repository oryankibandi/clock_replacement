package cache

import (
	"fmt"
	"log/slog"
	"math/rand"

	// "sync"
	"testing"
	"time"

	clock "github.com/oryankibandi/clock_replacement/internal/clockreplacement"
	"github.com/stretchr/testify/assert"
)

func TestNewCache(t *testing.T) {
	// t.Parallel()

	tests := []struct {
		cacheSize   uint64
		expectError bool
	}{
		{cacheSize: 100, expectError: false},
		{cacheSize: 2, expectError: true},
		{cacheSize: 3, expectError: false},
		{cacheSize: 10000, expectError: false},
		{cacheSize: 5235, expectError: false},
	}

	for n, test := range tests {
		t.Run(fmt.Sprintf("new_cache_%d", n), func(t *testing.T) {
			options := CacheOptions{
				Capacity: uint64(test.cacheSize),
			}

			c, err := NewCache(options)

			if test.expectError {
				if c != nil {
					t.Fatal(fmt.Errorf("Initialized cache with invalid size %d", test.cacheSize))
				}

				if err == nil {
					t.Fatal(fmt.Errorf("Initialized cache with invalid size %d", test.cacheSize))
				}
			} else {

				if c == nil {
					t.Fatal("Unable to initialize cache")
				}

				if err != nil {
					t.Fatal(fmt.Errorf("Cache failed with error: %s", err.Error()))
				}

				if len(c.bPool.bufferPool) <= 0 {
					t.Fatal("Unable to initialize buffer pool")
				}

				if c.cBuffer == nil {
					t.Fatal("Circular buffer not assigned")
				}

				if c.capacity != test.cacheSize {
					t.Fatal(fmt.Errorf("Expected capacity of %d, got %d", test.cacheSize, c.capacity))
				}

				// ensure entries are linked
				for i := range test.cacheSize {
					if c.cBuffer.Head.GetNextLink() == nil {
						t.Fatal(fmt.Errorf("%d item not linked in circular buffer", i+1))
					}

					c.cBuffer.Head = c.cBuffer.Head.GetNextLink()
				}

				err = c.Close()

				if err != nil {
					t.Fatal(fmt.Errorf("Unable to close cache: %s", err.Error()))
				}
			}
		})

	}
}

func TestPutGet(t *testing.T) {
	// t.Parallel()

	tests := []struct {
		name string
		key  uint32
		val  []byte
		add  bool
	}{
		{name: "item_1", key: 25, val: []byte("item_1"), add: true},
		{name: "item_2", key: 55, val: []byte("item_2"), add: true},
		{name: "item_3", key: 75, val: []byte("item_3"), add: true},
		{name: "item_4", key: 28, val: []byte("item_4"), add: false},
		{name: "item_5", key: 98, val: []byte("item_5"), add: true},
		{name: "item_6", key: 25, val: []byte("item_6"), add: true},
		{name: "item_7", key: 38, val: []byte("item_7"), add: false},
		{name: "item_8", key: 44, val: []byte("item_8"), add: true},
	}

	options := CacheOptions{
		Capacity: 100,
	}

	c, err := NewCache(options)

	if c == nil {
		t.Fatal("Unable to initialize cache")
	}

	if err != nil {
		t.Fatal(fmt.Errorf("Cache failed with error: %s", err.Error()))
	}

	if len(c.bPool.bufferPool) <= 0 {
		t.Fatal("Unable to initialize buffer pool")
	}

	if c.cBuffer == nil {
		t.Fatal("Circular buffer not assigned")
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.add {
				var v [clock.ENTRY_SIZE]byte
				copy(v[:len(test.val)], test.val)
				_, err := c.Put(test.key, test.val)

				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))

				e2, err := c.Get(test.key)
				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
				v2 := e2.GetData()
				assert.Equal(t, v, v2, "Wrong value received during get")
			} else {
				_, err := c.Get(test.key)
				assert.NotNil(t, err, fmt.Errorf("Expected err but got nil"))
			}
		})
	}

	err = c.Close()

	if err != nil {
		t.Fatal(fmt.Errorf("Unable to close cache: %s", err.Error()))
	}
}

func TestDelete(t *testing.T) {
	// t.Parallel()

	tests := []struct {
		name string
		key  uint32
		val  []byte
		add  bool
	}{
		{name: "item_1", key: 25, val: []byte("item_1"), add: true},
		{name: "item_2", key: 55, val: []byte("item_2"), add: true},
		{name: "item_3", key: 75, val: []byte("item_3"), add: true},
		{name: "item_4", key: 28, val: []byte("item_4"), add: false},
		{name: "item_5", key: 98, val: []byte("item_5"), add: true},
		{name: "item_6", key: 25, val: []byte("item_6"), add: true},
		{name: "item_7", key: 38, val: []byte("item_7"), add: false},
		{name: "item_8", key: 44, val: []byte("item_8"), add: true},
	}

	options := CacheOptions{
		Capacity: 100,
	}

	c, err := NewCache(options)

	if c == nil {
		t.Fatal("Unable to initialize cache")
	}

	if err != nil {
		t.Fatal(fmt.Errorf("Cache failed with error: %s", err.Error()))
	}

	if len(c.bPool.bufferPool) <= 0 {
		t.Fatal("Unable to initialize buffer pool")
	}

	if c.cBuffer == nil {
		t.Fatal("Circular buffer not assigned")
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.add {
				var v [clock.ENTRY_SIZE]byte

				copy(v[:len(test.val)], test.val)

				_, err := c.Put(test.key, test.val)

				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))

				// Delete
				err = c.Delete(test.key)

				assert.Nil(t, err, fmt.Errorf("Expeced no error while deleting, got %v", err))

				_, err = c.Get(test.key)

				assert.NotNil(t, err, fmt.Errorf("Expected error, got nil"))

			} else {
				_, err := c.Get(test.key)

				assert.NotNil(t, err, fmt.Errorf("Expected err but got nil"))

			}
		})
	}

	err = c.Close()

	if err != nil {
		t.Fatal(fmt.Errorf("Unable to close cache: %s", err.Error()))
	}
}

func TestPutGetConcurrent(t *testing.T) {
	type testStruct struct {
		name string
		key  uint32
		val  []byte
	}

	testsCount := 10000
	// generate test data
	tests := make([]testStruct, testsCount)

	for i := range testsCount {
		name := fmt.Sprintf("item_%d", i+1)
		tests[i] = testStruct{
			name: name,
			key:  25 + uint32(i), // sequential keys
			val:  []byte(name),
		}
	}

	// var wg sync.WaitGroup

	options := CacheOptions{
		Capacity: 5000,
	}

	c, err := NewCache(options)
	if c == nil {
		t.Fatal("Unable to initialize cache")
	}

	if err != nil {
		t.Fatal(fmt.Errorf("Cache failed with error: %s", err.Error()))
	}
	if len(c.bPool.bufferPool) <= 0 {
		t.Fatal("Unable to initialize buffer pool")
	}
	if c.cBuffer == nil {
		t.Fatal("Circular buffer not assigned")
	}

	// add concurrently
	t.Run("Concurrent Put", func(t *testing.T) {
		startPut := make(chan struct{})
		for i, test := range tests {
			t.Run(fmt.Sprintf("concurr_put_%d", i), func(t *testing.T) {
				t.Parallel()
				<-startPut
				_, err := c.Put(test.key, test.val)
				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
			})
		}
		close(startPut)

	})
	// wg.Wait()

	// retrieve data concurrently
	t.Run("Concurrent Get", func(t *testing.T) {
		startGet := make(chan struct{})
		for i, test := range tests {
			t.Run(fmt.Sprintf("concurr_get_%d", i), func(t *testing.T) {
				t.Parallel()
				<-startGet
				var v [clock.ENTRY_SIZE]byte
				copy(v[:len(test.val)], test.val)

				e2, err := c.Get(test.key)

				if err != nil {
					//  item evicted
					slog.Info(fmt.Sprintf("concurr_get_%d: Item evicted", i))
				} else {
					v2 := e2.GetData()

					assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
					assert.Equal(t, v, v2, "Wrong value received during get")
				}
			})
		}

		close(startGet)

	})
	// wg.Wait()

	// run get and put concurrently, with random time intervals
	t.Run("Concurrent Put and Get", func(t *testing.T) {
		startConcurr := make(chan struct{})
		for i, test := range tests {
			t.Run(fmt.Sprintf("concurr_putget_%d", i), func(t *testing.T) {
				t.Parallel()
				<-startConcurr
				time.Sleep(time.Duration(rand.Intn(15)+5) * time.Millisecond)

				_, err := c.Put(test.key, test.val)
				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
			})

			t.Run(fmt.Sprintf("concurr_putget_test_%d", i), func(t *testing.T) {
				t.Parallel()
				<-startConcurr
				time.Sleep(time.Duration(rand.Intn(15)+5) * time.Millisecond)
				var v [clock.ENTRY_SIZE]byte
				copy(v[:len(test.val)], test.val)

				e2, err := c.Get(test.key)

				if err != nil {
					// item already evicted
					slog.Info(fmt.Sprintf("concurr_putget_test_%d: Item evicted", i))
				} else {
					assert.NotNil(t, e2, fmt.Errorf("Expected retrieved entry to not be nil while error is: %v", err))
					v2 := e2.GetData()

					assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
					assert.Equal(t, v, v2, "Wrong value received during get")

				}
			})
		}

		close(startConcurr)

	})
	// wg.Wait()

	t.Cleanup(func() {
		slog.Info("CLOSING DOWN CACHE....")
		err = c.Close()

		if err != nil {
			t.Fatal(fmt.Errorf("Unable to close cache: %s", err.Error()))
		}
	})
}

func TestEvict(t *testing.T) {
	type testStruct struct {
		name string
		key  uint32
		val  []byte
	}

	testsCount := 5
	cacheCapacity := 4
	// generate test data
	tests := make([]testStruct, testsCount)

	for i := range testsCount {
		name := fmt.Sprintf("item_%d", i+1)
		tests[i] = testStruct{
			name: name,
			key:  25 + uint32(i), // sequential keys
			val:  []byte(name),
		}
	}

	options := CacheOptions{
		Capacity: uint64(cacheCapacity),
	}
	c, err := NewCache(options)
	if c == nil {
		t.Fatal("Unable to initialize cache")
	}

	if err != nil {
		t.Fatal(fmt.Errorf("Cache failed with error: %s", err.Error()))
	}
	if len(c.bPool.bufferPool) <= 0 {
		t.Fatal("Unable to initialize buffer pool")
	}
	if c.cBuffer == nil {
		t.Fatal("Circular buffer not assigned")
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("testevict_put_%d", i), func(t *testing.T) {
			_, err := c.Put(test.key, test.val)
			assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
		})
	}

	pageFaults := 0
	for i, test := range tests {
		t.Run(fmt.Sprintf("testevict_get_%d", i), func(t *testing.T) {
			var v [clock.ENTRY_SIZE]byte
			copy(v[:len(test.val)], test.val)

			e2, err := c.Get(test.key)

			if err != nil {
				pageFaults++
			} else {
				v2 := e2.GetData()

				assert.Equal(t, v, v2, "Wrong value received during get")
			}
		})
	}

	t.Run("Evict", func(t *testing.T) {
		expectedPageFaults := testsCount - cacheCapacity
		assert.Equal(t, expectedPageFaults, pageFaults, fmt.Errorf("Expected %d page faults, got %d", expectedPageFaults, pageFaults))
	})

	err = c.Close()

	if err != nil {
		t.Fatal(fmt.Errorf("Unable to close cache: %s", err.Error()))
	}
}
