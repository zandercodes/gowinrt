package gen

import (
	"fmt"

	winmd "github.com/microsoft/go-winmd/winmd"

	mdStore "github.com/zandercodes/gowinrt/internal/winmd"
)

// ---- resolution helpers ----

func (g *generator) resolveFields(td *mdStore.TypeDef) ([]winmd.Field, error) {
	var out []winmd.Field
	for idx := range td.FieldList.All() {
		f, err := td.Ctx().Tables.Field.At(idx)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (g *generator) resolveMethods(td *mdStore.TypeDef) ([]winmd.MethodDef, error) {
	var out []winmd.MethodDef
	for idx := range td.MethodList.All() {
		m, err := td.Ctx().Tables.MethodDef.At(idx)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (g *generator) resolveParams(td *mdStore.TypeDef, md *winmd.MethodDef) ([]winmd.Param, error) {
	var out []winmd.Param
	for idx := range md.ParamList.All() {
		p, err := td.Ctx().Tables.Param.At(idx)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func paramBySequence(params []winmd.Param, seq uint16) *winmd.Param {
	for i := range params {
		if params[i].Sequence == seq {
			return &params[i]
		}
	}
	return nil
}

// ---- element type mapping ----

func resolveTypeDefOrRefOrSpec(m *winmd.Metadata, ci winmd.CodedIndex[winmd.TypeDefOrRefOrSpec]) (string, string, error) {
	switch ci.Tag {
	case winmd.TypeDefOrRefOrSpec_TypeDef:
		td, err := m.Tables.TypeDef.At(ci.Index)
		if err != nil {
			return "", "", err
		}
		return td.Namespace.String(), td.Name.String(), nil
	case winmd.TypeDefOrRefOrSpec_TypeRef:
		tr, err := m.Tables.TypeRef.At(ci.Index)
		if err != nil {
			return "", "", err
		}
		return tr.Namespace.String(), tr.Name.String(), nil
	default:
		return "", "", fmt.Errorf("unsupported TypeDefOrRefOrSpec tag %d", ci.Tag)
	}
}

func (g *generator) elementType(ctx *winmd.Metadata, st winmd.SigType, isByRef bool) (*genParamType, error) {
	switch st.Kind {
	case winmd.ElementType_BOOLEAN:
		return &genParamType{name: "bool", IsPrimitive: true, defaultValue: genDefaultValue{"false", true}}, nil
	case winmd.ElementType_CHAR:
		return &genParamType{name: "byte", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_I1:
		return &genParamType{name: "int8", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_U1:
		return &genParamType{name: "uint8", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_I2:
		return &genParamType{name: "int16", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_U2:
		return &genParamType{name: "uint16", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_I4:
		return &genParamType{name: "int32", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_U4:
		return &genParamType{name: "uint32", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_I8:
		return &genParamType{name: "int64", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_U8:
		return &genParamType{name: "uint64", IsPrimitive: true, defaultValue: genDefaultValue{"0", true}}, nil
	case winmd.ElementType_R4:
		return &genParamType{name: "float32", IsPrimitive: true, defaultValue: genDefaultValue{"0.0", true}}, nil
	case winmd.ElementType_R8:
		return &genParamType{name: "float64", IsPrimitive: true, defaultValue: genDefaultValue{"0.0", true}}, nil
	case winmd.ElementType_STRING:
		return &genParamType{name: "string", IsPrimitive: true, defaultValue: genDefaultValue{`""`, true}}, nil

	case winmd.ElementType_GENERICINST, winmd.ElementType_CLASS:
		ci := st.Value.(winmd.CodedIndex[winmd.TypeDefOrRefOrSpec])
		ns, name, err := resolveTypeDefOrRefOrSpec(ctx, ci)
		if err != nil {
			return nil, err
		}
		return &genParamType{
			namespace: ns, name: name,
			IsPointer:    true,
			defaultValue: genDefaultValue{"nil", true},
		}, nil

	case winmd.ElementType_VALUETYPE:
		ci := st.Value.(winmd.CodedIndex[winmd.TypeDefOrRefOrSpec])
		ns, name, err := resolveTypeDefOrRefOrSpec(ctx, ci)
		if err != nil {
			return nil, err
		}
		if t, ok := isSystemType(ns, name); ok {
			return t, nil
		}

		elemTD, err := g.store.TypeDefByName(ns + "." + name)
		if err != nil {
			return nil, err
		}

		isEnum := false
		enumType := ""
		if elemTD.IsEnum() {
			enumData, err := g.createGenEnum(elemTD)
			if err != nil {
				return nil, err
			}
			isEnum = true
			enumType = enumData.Type
		}

		return &genParamType{
			namespace: ns, name: name,
			IsEnum: isEnum, UnderlyingEnumType: enumType,
			defaultValue: g.valueTypeDefault(ctx, ns, name),
		}, nil

	case winmd.ElementType_VAR:
		return &genParamType{
			namespace: "unsafe", name: "Pointer",
			IsGeneric:    true,
			defaultValue: genDefaultValue{"nil", true},
		}, nil

	case winmd.ElementType_SZARRAY:
		inner := st.Value.(winmd.SigType)
		param, err := g.elementType(ctx, inner, false)
		if err != nil {
			return nil, err
		}
		param.IsArray = true
		param.defaultValue = genDefaultValue{"nil", true}
		return param, nil

	case winmd.ElementType_OBJECT:
		return &genParamType{
			namespace: "unsafe", name: "Pointer",
			defaultValue: genDefaultValue{"nil", true},
		}, nil

	case winmd.ElementType_BYREF:
		inner := st.Value.(winmd.SigType)
		return g.elementType(ctx, inner, true)

	default:
		return nil, fmt.Errorf("unsupported element type: 0x%02x", st.Kind)
	}
}

func isSystemType(ns, name string) (*genParamType, bool) {
	if ns != "System" {
		return nil, false
	}
	switch name {
	case "Guid":
		return &genParamType{
			namespace: "syscall", name: "GUID",
			defaultValue: genDefaultValue{"GUID{}", false},
		}, true
	}
	return nil, false
}

func (g *generator) valueTypeDefault(ctx *winmd.Metadata, ns, name string) genDefaultValue {
	td, err := g.store.TypeDefByName(ns + "." + name)
	if err != nil {
		return genDefaultValue{"nil", true}
	}
	if td.IsEnum() {
		fields, err := g.resolveFields(td)
		if err != nil || len(fields) < 2 {
			return genDefaultValue{"0", true}
		}
		return genDefaultValue{enumValueName(td.Name.String(), fields[1].Name.String()), false}
	}
	if td.IsStruct() {
		return genDefaultValue{td.Name.String() + "{}", false}
	}
	return genDefaultValue{"nil", true}
}

// ---- attribute helpers ----

func (g *generator) interfaceExclusiveTo(td *mdStore.TypeDef) (string, bool) {
	blob, err := td.GetAttributeWithType(mdStore.AttributeTypeExclusiveTo)
	if err != nil {
		return "", false
	}
	return extractClassFromBlob(blob), true
}

func activatableAttrIsEmpty(blob []byte) bool {
	return len(blob) >= 3 && blob[0] == 0x01 && blob[1] == 0x00 && blob[2] == 0x00
}

func extractClassFromBlob(blob []byte) string {
	if len(blob) < 4 {
		return ""
	}
	size := blob[2]
	return string(blob[3 : 3+size])
}
