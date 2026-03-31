package winrt

import "unsafe"

// IUnknown is the base COM interface.
type IUnknown struct {
	RawVTable *interface{}
}

// IUnknownVtbl is the virtual method table for IUnknown.
type IUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

// VTable returns the IUnknown virtual method table.
func (v *IUnknown) VTable() *IUnknownVtbl {
	return (*IUnknownVtbl)(unsafe.Pointer(v.RawVTable))
}

// IInspectable is the base WinRT interface.
type IInspectable struct {
	IUnknown
}

// IInspectableVtbl is the virtual method table for IInspectable.
type IInspectableVtbl struct {
	IUnknownVtbl
	GetIIds             uintptr
	GetRuntimeClassName uintptr
	GetTrustLevel       uintptr
}

// VTable returns the IInspectable virtual method table.
func (v *IInspectable) VTable() *IInspectableVtbl {
	return (*IInspectableVtbl)(unsafe.Pointer(v.RawVTable))
}

// HString is a WinRT string handle.
type HString uintptr
