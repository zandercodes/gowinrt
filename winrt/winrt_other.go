//go:build !windows

package winrt

import "fmt"

// RoInitialize is a stub for non-Windows platforms.
func RoInitialize(threadType uint32) error {
	return NewError(E_NOTIMPL)
}

// RoActivateInstance is a stub for non-Windows platforms.
func RoActivateInstance(clsid string) (*IInspectable, error) {
	return nil, NewError(E_NOTIMPL)
}

// RoGetActivationFactory is a stub for non-Windows platforms.
func RoGetActivationFactory(clsid string, iid *GUID) (*IInspectable, error) {
	return nil, NewError(E_NOTIMPL)
}

// NewHString is a stub for non-Windows platforms.
func NewHString(s string) (HString, error) {
	return 0, NewError(E_NOTIMPL)
}

// DeleteHString is a stub for non-Windows platforms.
func DeleteHString(hstring HString) error {
	return NewError(E_NOTIMPL)
}

// String is a stub for non-Windows platforms.
func (h HString) String() string {
	return ""
}

// QueryInterface is a stub for non-Windows platforms.
func (v *IUnknown) QueryInterface(iid *GUID) (*IUnknown, error) {
	return nil, NewError(E_NOTIMPL)
}

// MustQueryInterface is a stub for non-Windows platforms.
func (v *IUnknown) MustQueryInterface(iid *GUID) *IUnknown {
	panic(NewError(E_NOTIMPL))
}

// AddRef is a stub for non-Windows platforms.
func (v *IUnknown) AddRef() int32 {
	return 0
}

// Release is a stub for non-Windows platforms.
func (v *IUnknown) Release() int32 {
	return 0
}

func errstr(errno int) string {
	return fmt.Sprintf("HRESULT: 0x%08X", uint32(errno))
}
