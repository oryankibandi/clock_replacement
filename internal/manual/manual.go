package manual

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"
)

// allocates memory of size s using calloc
func Alloc(size uintptr) unsafe.Pointer {
	ptr := C.calloc(1, C.size_t(size))

	if ptr == nil {
		panic("Out of memory.")
	}

	return ptr
}

// Frees memory pointed to by d
func FreeMem(d unsafe.Pointer) {
	if d == nil {
		return
	}

	C.free(d)
}
