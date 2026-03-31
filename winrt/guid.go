package winrt

// GUID is a globally unique identifier compatible with the Windows API.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	IID_IUnknown     = NewGUID("00000000-0000-0000-C000-000000000046")
	IID_IInspectable = NewGUID("AF86E2E0-B12D-4C6A-9C5A-D7AA65101E90")
)

// NewGUID parses a GUID string into a GUID struct.
// Accepted formats: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX,
// XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX,
// {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}.
func NewGUID(guid string) *GUID {
	d := []byte(guid)
	var d1, d2, d3, d4a, d4b []byte

	switch len(d) {
	case 38:
		if d[0] != '{' || d[37] != '}' {
			return nil
		}
		d = d[1:37]
		fallthrough
	case 36:
		if d[8] != '-' || d[13] != '-' || d[18] != '-' || d[23] != '-' {
			return nil
		}
		d1 = d[0:8]
		d2 = d[9:13]
		d3 = d[14:18]
		d4a = d[19:23]
		d4b = d[24:36]
	case 32:
		d1 = d[0:8]
		d2 = d[8:12]
		d3 = d[12:16]
		d4a = d[16:20]
		d4b = d[20:32]
	default:
		return nil
	}

	var g GUID
	var ok1, ok2, ok3, ok4 bool
	g.Data1, ok1 = decodeHexUint32(d1)
	g.Data2, ok2 = decodeHexUint16(d2)
	g.Data3, ok3 = decodeHexUint16(d3)
	g.Data4, ok4 = decodeHexByte64(d4a, d4b)
	if ok1 && ok2 && ok3 && ok4 {
		return &g
	}
	return nil
}

const hextable = "0123456789ABCDEF"

// String returns the GUID in {XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX} format.
func (guid *GUID) String() string {
	if guid == nil {
		return "{00000000-0000-0000-0000-000000000000}"
	}
	var c [38]byte
	c[0] = '{'
	putUint32Hex(c[1:9], guid.Data1)
	c[9] = '-'
	putUint16Hex(c[10:14], guid.Data2)
	c[14] = '-'
	putUint16Hex(c[15:19], guid.Data3)
	c[19] = '-'
	putByteHex(c[20:24], guid.Data4[0:2])
	c[24] = '-'
	putByteHex(c[25:37], guid.Data4[2:8])
	c[37] = '}'
	return string(c[:])
}

// IsEqualGUID compares two GUIDs for equality.
func IsEqualGUID(guid1 *GUID, guid2 *GUID) bool {
	return guid1.Data1 == guid2.Data1 &&
		guid1.Data2 == guid2.Data2 &&
		guid1.Data3 == guid2.Data3 &&
		guid1.Data4 == guid2.Data4
}

func decodeHexUint32(src []byte) (uint32, bool) {
	b1, ok1 := decodeHexByte(src[0], src[1])
	b2, ok2 := decodeHexByte(src[2], src[3])
	b3, ok3 := decodeHexByte(src[4], src[5])
	b4, ok4 := decodeHexByte(src[6], src[7])
	return (uint32(b1) << 24) | (uint32(b2) << 16) | (uint32(b3) << 8) | uint32(b4),
		ok1 && ok2 && ok3 && ok4
}

func decodeHexUint16(src []byte) (uint16, bool) {
	b1, ok1 := decodeHexByte(src[0], src[1])
	b2, ok2 := decodeHexByte(src[2], src[3])
	return (uint16(b1) << 8) | uint16(b2), ok1 && ok2
}

func decodeHexByte64(s1 []byte, s2 []byte) ([8]byte, bool) {
	var v [8]byte
	var ok bool
	v[0], _ = decodeHexByte(s1[0], s1[1])
	v[1], _ = decodeHexByte(s1[2], s1[3])
	v[2], _ = decodeHexByte(s2[0], s2[1])
	v[3], _ = decodeHexByte(s2[2], s2[3])
	v[4], _ = decodeHexByte(s2[4], s2[5])
	v[5], _ = decodeHexByte(s2[6], s2[7])
	v[6], _ = decodeHexByte(s2[8], s2[9])
	v[7], ok = decodeHexByte(s2[10], s2[11])
	return v, ok
}

func decodeHexByte(c1, c2 byte) (byte, bool) {
	n1, ok1 := decodeHexChar(c1)
	n2, ok2 := decodeHexChar(c2)
	return (n1 << 4) | n2, ok1 && ok2
}

func decodeHexChar(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func putUint32Hex(b []byte, v uint32) {
	b[0] = hextable[byte(v>>24)>>4]
	b[1] = hextable[byte(v>>24)&0x0f]
	b[2] = hextable[byte(v>>16)>>4]
	b[3] = hextable[byte(v>>16)&0x0f]
	b[4] = hextable[byte(v>>8)>>4]
	b[5] = hextable[byte(v>>8)&0x0f]
	b[6] = hextable[byte(v)>>4]
	b[7] = hextable[byte(v)&0x0f]
}

func putUint16Hex(b []byte, v uint16) {
	b[0] = hextable[byte(v>>8)>>4]
	b[1] = hextable[byte(v>>8)&0x0f]
	b[2] = hextable[byte(v)>>4]
	b[3] = hextable[byte(v)&0x0f]
}

func putByteHex(dst, src []byte) {
	for i := 0; i < len(src); i++ {
		dst[i*2] = hextable[src[i]>>4]
		dst[i*2+1] = hextable[src[i]&0x0f]
	}
}
