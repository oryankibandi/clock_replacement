package cache

import (
	"fmt"
	"sync"
	"testing"

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

				err := c.Put(test.key, v)

				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))

				v2, err := c.Get(test.key)

				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))

				assert.Equal(t, v, *v2, "Wrong value received during get")
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

				err := c.Put(test.key, v)

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

	testsCount := 100
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

	var wg sync.WaitGroup

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

	// add concurrently
	startPut := make(chan struct{})
	for i, test := range tests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startPut
			t.Run(fmt.Sprintf("put_%d", i), func(t *testing.T) {
				var v [clock.ENTRY_SIZE]byte
				copy(v[:len(test.val)], test.val)

				err := c.Put(test.key, v)
				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
			})
		}()
	}
	close(startPut)
	wg.Wait()

	// retrieve data concurrently
	startGet := make(chan struct{})
	for i, test := range tests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startGet
			t.Run(fmt.Sprintf("concurr_test_%d", i), func(t *testing.T) {
				var v [clock.ENTRY_SIZE]byte
				copy(v[:len(test.val)], test.val)

				v2, err := c.Get(test.key)
				assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
				assert.Equal(t, v, *v2, "Wrong value received during get")
			})
		}()
	}

	close(startGet)
	wg.Wait()

	err = c.Close()

	if err != nil {
		t.Fatal(fmt.Errorf("Unable to close cache: %s", err.Error()))
	}
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
			var v [clock.ENTRY_SIZE]byte
			copy(v[:len(test.val)], test.val)

			err := c.Put(test.key, v)
			assert.Nil(t, err, fmt.Errorf("Expected no error, got: %v", err))
		})
	}

	pageFaults := 0
	for i, test := range tests {
		t.Run(fmt.Sprintf("testevict_get_%d", i), func(t *testing.T) {
			var v [clock.ENTRY_SIZE]byte
			copy(v[:len(test.val)], test.val)

			v2, err := c.Get(test.key)

			if err != nil {
				pageFaults++
			} else {
				assert.Equal(t, v, *v2, "Wrong value received during get")
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
