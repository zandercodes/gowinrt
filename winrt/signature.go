package winrt

import (
	// #nosec — not used for security, required by WinRT spec
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strings"
)

// Primitive type signatures used by the WinRT type system.
const (
	SignatureUInt8   = "u1"
	SignatureUInt16  = "u2"
	SignatureUInt32  = "u4"
	SignatureUInt64  = "u8"
	SignatureInt8    = "i1"
	SignatureInt16   = "i2"
	SignatureInt32   = "i4"
	SignatureInt64   = "i8"
	SignatureFloat32 = "f4"
	SignatureFloat64 = "f8"
	SignatureBool    = "b1"
	SignatureChar    = "c2"
	SignatureString  = "string"
	SignatureGUID    = "g16"
)

// GUID namespace for parameterized (generic) WinRT interfaces and delegates.
var guidNamespace = NewGUID("11f47ad5-7b73-42c0-abae-878b1e16adee")

// ParameterizedInstanceGUID computes the GUID for a parameterized (generic)
// WinRT interface or delegate, following the algorithm described in:
// https://docs.microsoft.com/en-us/uwp/winrt-cref/winrt-type-system#guid-generation-for-parameterized-types
func ParameterizedInstanceGUID(baseGUID string, signatures ...string) string {
	sig := fmt.Sprintf("pinterface({%s};%s)", baseGUID, strings.Join(signatures, ";"))
	return guidFromSignature(sig)
}

func guidFromSignature(signature string) string {
	nsBytes := guidToBytes(guidNamespace)

	h := sha1.Sum(append(nsBytes, []byte(signature)...)) // #nosec

	// Setting UUID v5 Bits as per RFC 4122
	h[6] = (h[6] & 0x0f) | (5 << 4) // Version 5
	h[8] = (h[8] & 0x3f) | 0x80     // Variant bits

	return bytesToGUID(h[:16]).String()
}

func guidToBytes(guid *GUID) []byte {
	b := make([]byte, 16)

	binary.BigEndian.PutUint32(b[0:4], guid.Data1)
	binary.BigEndian.PutUint16(b[4:6], guid.Data2)
	binary.BigEndian.PutUint16(b[6:8], guid.Data3)
	copy(b[8:16], guid.Data4[:])

	return b
}

func bytesToGUID(b []byte) *GUID {
	return &GUID{
		Data1: binary.BigEndian.Uint32(b[0:4]),
		Data2: binary.BigEndian.Uint16(b[4:6]),
		Data3: binary.BigEndian.Uint16(b[6:8]),
		Data4: [8]byte{b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15]},
	}
}
