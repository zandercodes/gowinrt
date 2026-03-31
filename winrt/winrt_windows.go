//go:build windows

package winrt

import (
	"fmt"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modcombase = windows.NewLazySystemDLL("combase.dll")

	procRoInitialize              = modcombase.NewProc("RoInitialize")
	procRoActivateInstance        = modcombase.NewProc("RoActivateInstance")
	procRoGetActivationFactory    = modcombase.NewProc("RoGetActivationFactory")
	procWindowsCreateString       = modcombase.NewProc("WindowsCreateString")
	procWindowsDeleteString       = modcombase.NewProc("WindowsDeleteString")
	procWindowsGetStringRawBuffer = modcombase.NewProc("WindowsGetStringRawBuffer")
)

// RoInitialize initializes the Windows Runtime on the current thread.
func RoInitialize(threadType uint32) error {
	hr, _, _ := procRoInitialize.Call(uintptr(threadType))
	if hr != 0 {
		return NewError(hr)
	}
	return nil
}

// RoActivateInstance creates an instance of a WinRT class by name.
func RoActivateInstance(clsid string) (*IInspectable, error) {
	hClsid, err := NewHString(clsid)
	if err != nil {
		return nil, err
	}
	defer DeleteHString(hClsid)

	var ins *IInspectable
	hr, _, _ := procRoActivateInstance.Call(
		uintptr(hClsid),
		uintptr(unsafe.Pointer(&ins)))
	if hr != 0 {
		return nil, NewError(hr)
	}
	return ins, nil
}

// RoGetActivationFactory retrieves a factory interface for a WinRT class.
func RoGetActivationFactory(clsid string, iid *GUID) (*IInspectable, error) {
	hClsid, err := NewHString(clsid)
	if err != nil {
		return nil, err
	}
	defer DeleteHString(hClsid)

	var ins *IInspectable
	hr, _, _ := procRoGetActivationFactory.Call(
		uintptr(hClsid),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&ins)))
	if hr != 0 {
		return nil, NewError(hr)
	}
	return ins, nil
}

// NewHString creates a WinRT string handle from a Go string.
func NewHString(s string) (HString, error) {
	u16, err := syscall.UTF16FromString(s)
	if err != nil {
		return 0, err
	}
	var hstring HString
	hr, _, _ := procWindowsCreateString.Call(
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)-1),
		uintptr(unsafe.Pointer(&hstring)))
	if hr != 0 {
		return 0, NewError(hr)
	}
	return hstring, nil
}

// DeleteHString frees a WinRT string handle.
func DeleteHString(hstring HString) error {
	hr, _, _ := procWindowsDeleteString.Call(uintptr(hstring))
	if hr != 0 {
		return NewError(hr)
	}
	return nil
}

// String returns the Go string value of an HString.
func (h HString) String() string {
	var u16len uint32
	u16buf, _, _ := procWindowsGetStringRawBuffer.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&u16len)))
	if u16buf == 0 || u16len == 0 {
		return ""
	}
	// The uintptr returned by Call is safe to convert here because the HString
	// owns the buffer and stays alive for the duration of this function.
	u16 := unsafe.Slice(*(**uint16)(unsafe.Pointer(&u16buf)), u16len)
	return syscall.UTF16ToString(u16)
}

// QueryInterface queries a COM object for a specific interface.
func (v *IUnknown) QueryInterface(iid *GUID) (*IUnknown, error) {
	var out *IUnknown
	hr, _, _ := syscall.SyscallN(
		v.VTable().QueryInterface,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&out)))
	if hr != 0 {
		return nil, NewError(hr)
	}
	return out, nil
}

// MustQueryInterface queries a COM object for a specific interface and panics on failure.
func (v *IUnknown) MustQueryInterface(iid *GUID) *IUnknown {
	out, err := v.QueryInterface(iid)
	if err != nil {
		panic(err)
	}
	return out
}

// AddRef increments the reference count.
func (v *IUnknown) AddRef() int32 {
	ret, _, _ := syscall.SyscallN(v.VTable().AddRef, uintptr(unsafe.Pointer(v)))
	return int32(ret)
}

// Release decrements the reference count.
func (v *IUnknown) Release() int32 {
	ret, _, _ := syscall.SyscallN(v.VTable().Release, uintptr(unsafe.Pointer(v)))
	return int32(ret)
}

func errstr(errno int) string {
	var flags uint32 = syscall.FORMAT_MESSAGE_FROM_SYSTEM | syscall.FORMAT_MESSAGE_ARGUMENT_ARRAY | syscall.FORMAT_MESSAGE_IGNORE_INSERTS
	b := make([]uint16, 300)
	n, err := syscall.FormatMessage(flags, 0, uint32(errno), 0, b, nil)
	if err != nil {
		return fmt.Sprintf("HRESULT: 0x%08X", uint32(errno))
	}
	for ; n > 0 && (b[n-1] == '\n' || b[n-1] == '\r'); n-- {
	}
	return string(utf16.Decode(b[:n]))
}
