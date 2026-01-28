package clock_replacement

import (
	"clock_replacement_algorithm/internal/manual"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemAlloc(t *testing.T) {
	t.Parallel()
	var en *entry

	t.Run("allocate_memory", func(t *testing.T) {
		en = NewEntry()

		if en == nil {
			t.Fatal("Memory not allocated.")
		}

		if en.dataSize == 0 {
			t.Fatal("Invalid data size set.")
		}

		if en.cPtr == nil {
			t.Fatal("Unsafe pointer not set")
		}
	})

	t.Run("free memory", func(t *testing.T) {
		manual.FreeMem(en.cPtr)

		en = nil

		if en != nil {
			t.Fatal(fmt.Errorf("Memory not freed: %v\n", en))
		}
	})
}

func TestRefAndUnref(t *testing.T) {
	t.Parallel()
	tests := []struct {
		operation          string // reference | unreference
		expectedPinCount   uint64
		expectedUnpinCount uint64
		expectedAccBit     bool
		expectedRefBit     bool
	}{
		{operation: "reference", expectedPinCount: 1, expectedUnpinCount: 0, expectedAccBit: true, expectedRefBit: true},
		{operation: "reference", expectedPinCount: 2, expectedUnpinCount: 0, expectedAccBit: true, expectedRefBit: true},
		{operation: "reference", expectedPinCount: 3, expectedUnpinCount: 0, expectedAccBit: true, expectedRefBit: true},
		{operation: "unreference", expectedPinCount: 3, expectedUnpinCount: 1, expectedAccBit: true, expectedRefBit: true},
		{operation: "unreference", expectedPinCount: 3, expectedUnpinCount: 2, expectedAccBit: true, expectedRefBit: true},
		{operation: "unreference", expectedPinCount: 3, expectedUnpinCount: 3, expectedAccBit: false, expectedRefBit: true},
	}

	var en *entry

	t.Run("allocate_memory", func(t *testing.T) {
		en = NewEntry()

		if en == nil {
			t.Fatal("Memory not allocated.")
		}

		if en.dataSize == 0 {
			t.Fatal("Invalid data size set.")
		}

		if en.cPtr == nil {
			t.Fatal("Unsafe pointer not set")
		}
	})

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_reference_%d", i), func(t *testing.T) {
			if test.operation == "reference" {
				en.reference()

				// check ref bit, acess bit and pin count
				assert.Equal(t, test.expectedAccBit, en.acc.Load(), "access bit not set")
				assert.Equal(t, test.expectedRefBit, en.ref.Load(), "reference bit not set")
				assert.Equal(t, test.expectedPinCount, en.counters.pinCount.Load(), "Invalid pin count")
				assert.Equal(t, test.expectedUnpinCount, en.counters.unpinCount.Load(), "Invalid unpin cunt")
			}
		})
	}

	manual.FreeMem(en.cPtr)
	en = nil
}

func TestRefAndUnrefConcurrent(t *testing.T) {
	t.Parallel()
	testCount := 1000
	var wg sync.WaitGroup
	type testItem struct {
		operation string // reference | unreference
	}

	tests := make([]testItem, 0)
	for i := range testCount {
		if i%4 == 0 {
			tests = append(tests, testItem{operation: "unreference"})
		} else {
			tests = append(tests, testItem{operation: "reference"})
		}
	}

	var en *entry

	t.Run("allocate_memory_concurrent", func(t *testing.T) {
		en = NewEntry()

		if en == nil {
			t.Fatal("Memory not allocated.")
		}

		if en.dataSize == 0 {
			t.Fatal("Invalid data size set.")
		}

		if en.cPtr == nil {
			t.Fatal("Unsafe pointer not set")
		}
	})

	start := make(chan struct{})
	for i, test := range tests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			t.Run(fmt.Sprintf("test_concurrent_%d", i), func(t *testing.T) {
				if test.operation == "reference" {
					en.reference()

					// check ref bit
					assert.Equal(t, true, en.ref.Load(), "reference bit not set")
				}
			})

		}()
	}

	// start gorutines simultaneously
	close(start)

	wg.Wait()
	manual.FreeMem(en.cPtr)
	en = nil
}
