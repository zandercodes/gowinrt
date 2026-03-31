package winrt

// HRESULT constants.
const (
	S_OK          = 0x00000000
	E_NOTIMPL     = 0x80004001
	E_NOINTERFACE = 0x80004002
	E_POINTER     = 0x80004003
	E_ABORT       = 0x80004004
	E_FAIL        = 0x80004005
)

// HResultError represents a COM HRESULT error code.
type HResultError struct {
	hr uintptr
}

// NewError creates an error from an HRESULT code.
func NewError(hr uintptr) *HResultError {
	return &HResultError{hr: hr}
}

// Code returns the raw HRESULT code.
func (e *HResultError) Code() uintptr {
	return e.hr
}

// Error implements the error interface.
func (e *HResultError) Error() string {
	return errstr(int(e.hr))
}
