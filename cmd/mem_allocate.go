package main

// This script is meant to test memory allocation and freeing
// Expect Resident Set Size(RSS) to go up due to manual memory allocation(outside the GC)
// while other memory (Sys) remains relatively the same.

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	manual "github.com/oryankibandi/clock_replacement/internal/manual"
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
	key uint32
}

type entry struct {
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

	// size of an entry. Default is ENTRY_SIZE
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

func (e *entry) markDirty() {
	e.meta.isDirty.Store(true)
}

func (e *entry) setData(key uint32, data [ENTRY_SIZE]byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.data = data
	e.meta.key = key

	e.isOccupied.Store(true)

	e.markDirty()

	return nil
}

// Returns a pointer to new entry
// To reduce pressure on the GC and improve performance, memory is allocated manually via calloc().
// This memory also needs to be freed after use to avoid memory leaks.
// In a storage engine's buffer manager, this memory will be initialized at startup and reused as blocks are paged-in and evicted.
func NewEntry() *entry {
	p := manual.Alloc(unsafe.Sizeof(entry{}))

	e := (*entry)(p)

	e.dataSize = ENTRY_SIZE
	e.cPtr = p

	return e
}

//func manual.CurrentRSSBytes() (float64, error) {
//	f, err := os.Open("/proc/self/statm")
//	if err != nil {
//		return 0, err
//	}
//	defer f.Close()
//	scanner := bufio.NewScanner(f)
//	if !scanner.Scan() {
//		return 0, scanner.Err()
//	}
//	fields := strings.Fields(scanner.Text())
//	if len(fields) < 2 {
//		return 0, fmt.Errorf("unexpected statm format")
//	}
//	rssPages, err := strconv.ParseUint(fields[1], 10, 64)
//	if err != nil {
//		return 0, err
//	}
//	pageSize := uint64(syscall.Getpagesize())
//	rssBytes := rssPages * pageSize
//	return float64(rssBytes) / 1024.0 / 1024.0, nil
//}

func main() {
	iterations := 500
	buffSizeMB := 24 * (1024 * 1024) // 24MB
	entrySize := unsafe.Sizeof(entry{})
	entryCount := buffSizeMB / int(entrySize)

	var entries []*entry
	var m runtime.MemStats
	var e *entry

	// entrySize := unsafe.Sizeof(entry{})
	// pSize := os.Getpagesize()

	var d [ENTRY_SIZE]byte
	for i := range ENTRY_SIZE {
		d[i] = 1
	}

	var currRss uint64

	// allocate and deallocate multiple times
	for range iterations {
		runtime.ReadMemStats(&m)
		currRss, err := manual.CurrentRSSBytes()
		if err != nil {
			panic(err)
		}
		fmt.Println("---------------------------------------")

		fmt.Printf("before: Alloc=%d KB TotalAlloc=%d KB Sys=%d KB RSS=%dMB\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, currRss/1024/1024)

		for range entryCount {
			e = NewEntry()

			// touch pages by adding data to trigger allocation
			// Since calloc zeros out allocated memory, the OS may
			// not actually assign the physical memory until data is
			// added.
			e.setData(25, d)

			entries = append(entries, e)
		}

		runtime.ReadMemStats(&m)
		currRss, err = manual.CurrentRSSBytes()

		if err != nil {
			panic(err)
		}

		fmt.Printf("After allocation: Alloc=%d KB TotalAlloc=%d KB Sys=%d KB RSS=%dMB\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, currRss/1024/1024)

		runtime.GC()

		runtime.ReadMemStats(&m)

		currRss, err = manual.CurrentRSSBytes()

		if err != nil {
			panic(err)
		}

		fmt.Printf("After GC: Alloc=%d KB TotalAlloc=%d KB Sys=%d KB RSS=%dMB\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, currRss/1024/1024)

		// Free
		for i, en := range entries {
			manual.FreeMem(unsafe.Pointer(en))
			en = nil
			entries[i] = nil
		}

		entries = nil

		runtime.ReadMemStats(&m)

		currRss, err = manual.CurrentRSSBytes()

		if err != nil {
			panic(err)
		}
		fmt.Printf("After manual Free: Alloc=%d KB TotalAlloc=%d KB Sys=%d KB RSS=%dMB\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, currRss/1024/1024)

		fmt.Println("---------------------------------------")
	}

	currRss, err := manual.CurrentRSSBytes()

	if err != nil {
		panic(fmt.Errorf("Could not get current RSS: %v", err))
	}

	fmt.Printf("Final RSS: %dMB\n", currRss/1024/1024)

}
