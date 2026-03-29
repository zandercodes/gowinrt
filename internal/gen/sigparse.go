package gen

import (
	"fmt"

	winmd "github.com/microsoft/go-winmd/winmd"
)

// parseMethodSig parses a raw ECMA-335 MethodDefSig blob (§II.23.2.1) into a
// winmd.SigMethodDef. It extends go-winmd's built-in parser by supporting
// ElementType_GENERICINST, ElementType_VAR, ElementType_MVAR and
// ElementType_SZARRAY which go-winmd does not yet decode.
func parseMethodSig(data winmd.SigMethodDefBlob) (winmd.SigMethodDef, error) {
	p := sigParser{data: []byte(data)}
	sig := p.methodDefSig()
	return sig, p.err
}

// parseFieldSig parses a raw ECMA-335 FieldSig blob (§II.23.2.4).
func parseFieldSig(data winmd.SigFieldBlob) (winmd.SigField, error) {
	p := sigParser{data: []byte(data)}
	sig := p.fieldSig()
	return sig, p.err
}

// sigParser is a minimal ECMA-335 signature blob reader.
type sigParser struct {
	data []byte
	err  error
}

func (p *sigParser) fieldSig() (v winmd.SigField) {
	if p.err != nil {
		return
	}
	firstByte := p.readByte()
	if p.err != nil {
		return
	}
	kind := firstByte & 0x0F
	if kind != 0x06 { // sigKind_FIELD
		p.err = fmt.Errorf("signature kind is not a field signature: %v", kind)
		return
	}
	v.Type = p.decodeType()
	return
}

func (p *sigParser) methodDefSig() (v winmd.SigMethodDef) {
	if p.err != nil {
		return
	}
	firstByte := p.readByte()
	if p.err != nil {
		return
	}

	kind := firstByte & 0x0F
	if kind > 0x05 { // sigKind_VARARG = 5
		p.err = fmt.Errorf("signature kind is not a method def signature: %v", kind)
		return
	}

	thisiness := firstByte & 0xF0
	v.HasThis = thisiness&0x20 != 0      // sigAbbrev_HASTHIS
	v.ExplicitThis = thisiness&0x40 != 0 // sigAbbrev_EXPLICITTHIS
	if thisiness&0x10 != 0 {             // sigAbbrev_GENERIC
		v.Generic = p.compressedUint32()
		if p.err != nil {
			return
		}
	}

	paramCount := p.compressedUint32()
	if p.err != nil {
		return
	}

	v.RetType = p.retType()
	if p.err != nil {
		return
	}
	for i := uint32(0); i < paramCount; i++ {
		v.Param = append(v.Param, p.param())
		if p.err != nil {
			return
		}
	}
	return
}

func (p *sigParser) param() (v winmd.SigParam) {
	if p.err != nil {
		return
	}
	v.Type = p.decodeType()
	switch v.Type.Kind {
	case winmd.ElementType_BYREF:
		v.Kind = winmd.SigParamKind_ByRef
	case winmd.ElementType_TYPEDBYREF:
		v.Kind = winmd.SigParamKind_TypedByRef
	default:
		v.Kind = winmd.SigParamKind_ByValue
	}
	return
}

func (p *sigParser) retType() (v winmd.SigRetType) {
	if p.err != nil {
		return
	}
	v.Type = p.decodeType()
	switch v.Type.Kind {
	case winmd.ElementType_BYREF:
		v.Kind = winmd.SigRetTypeKind_ByRef
	case winmd.ElementType_TYPEDBYREF:
		v.Kind = winmd.SigRetTypeKind_TypedByRef
	case winmd.ElementType_VOID:
		v.Kind = winmd.SigRetTypeKind_Void
	default:
		v.Kind = winmd.SigRetTypeKind_ByValue
	}
	return
}

func (p *sigParser) decodeType() (v winmd.SigType) {
	if p.err != nil {
		return
	}
	b := winmd.ElementType(p.compressedUint32())
	if p.err != nil {
		return
	}

	switch b {
	case winmd.ElementType_CMOD_OPT, winmd.ElementType_CMOD_REQD:
		_ = p.typeDefOrRefOrSpec() // skip modifier type
		return p.decodeType()

	case winmd.ElementType_BYREF:
		v.Kind = b
		v.Value = p.decodeType()

	case winmd.ElementType_TYPEDBYREF:
		v.Kind = winmd.ElementType_TYPEDBYREF
	case winmd.ElementType_VOID:
		v.Kind = winmd.ElementType_VOID

	case winmd.ElementType_GENERICINST:
		// §II.23.2.12: GENERICINST (CLASS|VALUETYPE) TypeDefOrRefOrSpecEncoded GenArgCount Type*
		_ = p.compressedUint32() // CLASS (0x12) or VALUETYPE (0x11) – consumed but not used
		ci := p.typeDefOrRefOrSpec()
		genArgCount := p.compressedUint32()
		for i := uint32(0); i < genArgCount && p.err == nil; i++ {
			p.decodeType() // skip each type argument
		}
		v.Kind = winmd.ElementType_GENERICINST
		v.Value = ci

	case winmd.ElementType_CLASS, winmd.ElementType_VALUETYPE:
		v.Kind = b
		v.Value = p.typeDefOrRefOrSpec()

	case winmd.ElementType_VAR, winmd.ElementType_MVAR:
		_ = p.compressedUint32() // generic parameter index – not used
		v.Kind = winmd.ElementType_VAR

	case winmd.ElementType_SZARRAY:
		inner := p.decodeType()
		v.Kind = winmd.ElementType_SZARRAY
		v.Value = inner

	case winmd.ElementType_PTR:
		v.Kind = b
		v.Value = p.decodeType()

	case winmd.ElementType_ARRAY:
		// §II.23.2.13: Type Rank NumSizes Size* NumLoBounds LoBound*
		inner := p.decodeType()
		rank := p.compressedUint32()
		numSizes := p.compressedUint32()
		for i := uint32(0); i < numSizes && p.err == nil; i++ {
			p.compressedUint32()
		}
		numLoBounds := p.compressedUint32()
		for i := uint32(0); i < numLoBounds && p.err == nil; i++ {
			p.compressedUint32() // signed, but byte count is the same
		}
		v.Kind = b
		v.Value = winmd.SigArray{Type: inner, Rank: rank}

	case winmd.ElementType_BOOLEAN, winmd.ElementType_CHAR,
		winmd.ElementType_I1, winmd.ElementType_U1,
		winmd.ElementType_I2, winmd.ElementType_U2,
		winmd.ElementType_I4, winmd.ElementType_U4,
		winmd.ElementType_I8, winmd.ElementType_U8,
		winmd.ElementType_R4, winmd.ElementType_R8,
		winmd.ElementType_I, winmd.ElementType_U,
		winmd.ElementType_OBJECT, winmd.ElementType_STRING:
		v.Kind = b

	default:
		p.err = fmt.Errorf("unsupported element type: 0x%02x", b)
	}
	return
}

// typeDefOrRefOrSpec decodes a TypeDefOrRefOrSpecEncoded value (§II.23.2.8).
// 3 possible tables → 2 tag bits.
func (p *sigParser) typeDefOrRefOrSpec() winmd.CodedIndex[winmd.TypeDefOrRefOrSpec] {
	if p.err != nil {
		return winmd.CodedIndex[winmd.TypeDefOrRefOrSpec]{}
	}
	code := p.compressedUint32()
	if p.err != nil {
		return winmd.CodedIndex[winmd.TypeDefOrRefOrSpec]{}
	}

	tag := code & 0x3
	row := code >> 2
	if row == 0 {
		return winmd.CodedIndex[winmd.TypeDefOrRefOrSpec]{Tag: winmd.TypeDefOrRefOrSpec_Null}
	}
	row-- // 1-based → 0-based

	var t winmd.TypeDefOrRefOrSpec
	switch tag {
	case 0:
		t = winmd.TypeDefOrRefOrSpec_TypeDef
	case 1:
		t = winmd.TypeDefOrRefOrSpec_TypeRef
	case 2:
		t = winmd.TypeDefOrRefOrSpec_TypeSpec
	default:
		p.err = fmt.Errorf("unknown TypeDefOrRefOrSpec tag: %d", tag)
		return winmd.CodedIndex[winmd.TypeDefOrRefOrSpec]{}
	}
	return winmd.CodedIndex[winmd.TypeDefOrRefOrSpec]{Index: winmd.Index(row), Tag: t}
}

// ---- low-level byte reading ----

func (p *sigParser) readByte() byte {
	if p.err != nil {
		return 0
	}
	if len(p.data) == 0 {
		p.err = fmt.Errorf("unexpected end of signature blob")
		return 0
	}
	v := p.data[0]
	p.data = p.data[1:]
	return v
}

// compressedUint32 reads an ECMA-335 §II.23.2 compressed unsigned integer.
func (p *sigParser) compressedUint32() uint32 {
	if p.err != nil {
		return 0
	}
	if len(p.data) == 0 {
		p.err = fmt.Errorf("unexpected end of signature blob")
		return 0
	}

	v := p.data[0]
	if v&0x80 == 0 {
		// 1-byte: 0bbb_bbbb
		p.data = p.data[1:]
		return uint32(v)
	}
	if v&0xC0 == 0x80 {
		// 2-byte: 10bb_bbbb xxxx_xxxx
		if len(p.data) < 2 {
			p.err = fmt.Errorf("unexpected end of signature blob")
			return 0
		}
		result := uint32(v&0x3F)<<8 | uint32(p.data[1])
		p.data = p.data[2:]
		return result
	}
	if v&0xE0 == 0xC0 {
		// 4-byte: 110b_bbbb xxxx_xxxx xxxx_xxxx xxxx_xxxx
		if len(p.data) < 4 {
			p.err = fmt.Errorf("unexpected end of signature blob")
			return 0
		}
		result := uint32(v&0x1F)<<24 | uint32(p.data[1])<<16 | uint32(p.data[2])<<8 | uint32(p.data[3])
		p.data = p.data[4:]
		return result
	}

	p.err = fmt.Errorf("invalid compressed uint32: 0x%02x", v)
	return 0
}
