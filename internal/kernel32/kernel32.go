//go:build windows

package kernel32

import (
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type (
	heapHandle = uintptr
	heapFlags  uint32
)

const (
	heapNone       heapFlags = 0
	heapZeroMemory heapFlags = 8
)

var (
	libKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	pHeapFree       uintptr
	pHeapAlloc      uintptr
	pGetProcessHeap uintptr

	hHeap heapHandle
)

func init() {
	hHeap, _ = getProcessHeap()
}

// Malloc allocates the given amount of zero-initialized bytes on the process heap.
func Malloc(size uintptr) unsafe.Pointer {
	return heapAlloc(hHeap, heapZeroMemory, size)
}

// Free releases the given pointer that was previously allocated with Malloc.
func Free(ptr unsafe.Pointer) {
	heapFree(hHeap, heapNone, ptr)
}

func heapAlloc(hHeap heapHandle, flags heapFlags, size uintptr) unsafe.Pointer {
	addr := loadProc(&pHeapAlloc, "HeapAlloc")
	ret, _, _ := syscall.SyscallN(addr, hHeap, uintptr(flags), size)
	// The pointer is allocated on the Windows process heap and is not managed by Go's GC.
	// This uintptr→unsafe.Pointer conversion is safe because the pointer's lifetime is
	// managed by explicit Free calls.
	ptr := ret
	return *(*unsafe.Pointer)(unsafe.Pointer(&ptr))
}

func heapFree(hHeap heapHandle, flags heapFlags, ptr unsafe.Pointer) {
	addr := loadProc(&pHeapFree, "HeapFree")
	syscall.SyscallN(addr, hHeap, uintptr(flags), uintptr(ptr))
}

func getProcessHeap() (heapHandle, error) {
	addr := loadProc(&pGetProcessHeap, "GetProcessHeap")
	ret, _, err := syscall.SyscallN(addr)
	if ret == 0 {
		return 0, err
	}
	return ret, nil
}

func loadProc(cached *uintptr, name string) uintptr {
	addr := atomic.LoadUintptr(cached)
	if addr == 0 {
		addr = libKernel32.NewProc(name).Addr()
		atomic.StoreUintptr(cached, addr)
	}
	return addr
}
