//go:build windows

package delegate

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/zandercodes/gowinrt/winrt"
)

// Syscall callbacks are limited in number per process and never freed.
var (
	cbQueryInterface = syscall.NewCallback(queryInterface)
	cbAddRef         = syscall.NewCallback(addRef)
	cbRelease        = syscall.NewCallback(release)
	cbInvoke         = syscall.NewCallback(invoke)
)

// Delegate represents a WinRT delegate that can receive callbacks.
type Delegate interface {
	GetIID() *winrt.GUID
	Invoke(instancePtr, rawArgs0, rawArgs1, rawArgs2, rawArgs3, rawArgs4, rawArgs5, rawArgs6, rawArgs7, rawArgs8 unsafe.Pointer) uintptr
	AddRef() uintptr
	Release() uintptr
}

// Callbacks holds the pre-registered syscall callback addresses for a delegate's vtable.
type Callbacks struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	Invoke         uintptr
}

var (
	mu        sync.RWMutex
	instances = make(map[uintptr]Delegate)
)

// Register associates an instance pointer with a Delegate and returns the vtable callbacks.
func Register(ptr unsafe.Pointer, inst Delegate) *Callbacks {
	mu.Lock()
	defer mu.Unlock()
	instances[uintptr(ptr)] = inst
	return &Callbacks{
		QueryInterface: cbQueryInterface,
		AddRef:         cbAddRef,
		Release:        cbRelease,
		Invoke:         cbInvoke,
	}
}

func get(ptr unsafe.Pointer) (Delegate, bool) {
	mu.RLock()
	defer mu.RUnlock()
	d, ok := instances[uintptr(ptr)]
	return d, ok
}

func remove(ptr unsafe.Pointer) {
	mu.Lock()
	defer mu.Unlock()
	delete(instances, uintptr(ptr))
}

func queryInterface(instancePtr, iidPtr unsafe.Pointer, ppv *unsafe.Pointer) uintptr {
	inst, ok := get(instancePtr)
	if !ok || ppv == nil {
		return winrt.E_POINTER
	}

	iid := (*winrt.GUID)(iidPtr)
	if winrt.IsEqualGUID(iid, inst.GetIID()) || winrt.IsEqualGUID(iid, winrt.IID_IUnknown) || winrt.IsEqualGUID(iid, winrt.IID_IInspectable) {
		*ppv = instancePtr
	} else {
		*ppv = nil
		return winrt.E_NOINTERFACE
	}

	(*winrt.IUnknown)(*ppv).AddRef()
	return winrt.S_OK
}

func invoke(instancePtr, a0, a1, a2, a3, a4, a5, a6, a7, a8 unsafe.Pointer) uintptr {
	inst, ok := get(instancePtr)
	if !ok {
		return winrt.E_FAIL
	}
	return inst.Invoke(instancePtr, a0, a1, a2, a3, a4, a5, a6, a7, a8)
}

func addRef(instancePtr unsafe.Pointer) uintptr {
	inst, ok := get(instancePtr)
	if !ok {
		return winrt.E_FAIL
	}
	return inst.AddRef()
}

func release(instancePtr unsafe.Pointer) uintptr {
	inst, ok := get(instancePtr)
	if !ok {
		return winrt.E_FAIL
	}
	rem := inst.Release()
	if rem == 0 {
		remove(instancePtr)
	}
	return rem
}
