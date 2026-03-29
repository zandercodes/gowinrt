package gen

import (
	"fmt"
	"strings"

	winmd "github.com/microsoft/go-winmd/winmd"

	mdStore "github.com/zandercodes/gowinrt/internal/winmd"
)

// ---- type signature ----

func (g *generator) typeSignature(td *mdStore.TypeDef) (string, error) {
	switch {
	case td.IsInterface():
		guid, err := td.GUID()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("{%s}", guid), nil

	case td.IsEnum():
		fields, err := g.resolveFields(td)
		if err != nil {
			return "", err
		}
		fSig, err := parseFieldSig(fields[0].Signature)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("enum(%s;%s)", td.Namespace.String()+"."+td.Name.String(), primitiveTypeSignature(fSig.Type.Kind)), nil

	case td.IsStruct():
		fields, err := g.resolveFields(td)
		if err != nil {
			return "", err
		}
		var args []string
		for _, f := range fields {
			fSig, err := parseFieldSig(f.Signature)
			if err != nil {
				return "", err
			}
			if fSig.Type.Kind == winmd.ElementType_VALUETYPE {
				fType, err := g.elementType(td.Ctx(), fSig.Type, false)
				if err != nil {
					return "", err
				}
				innerTD, err := g.store.TypeDefByName(fType.namespace + "." + fType.name)
				if err != nil {
					return "", err
				}
				innerSig, err := g.typeSignature(innerTD)
				if err != nil {
					return "", err
				}
				args = append(args, innerSig)
			} else {
				args = append(args, primitiveTypeSignature(fSig.Type.Kind))
			}
		}
		return fmt.Sprintf("struct(%s;%s)", td.Namespace.String()+"."+td.Name.String(), strings.Join(args, ";")), nil

	case td.IsDelegate():
		guid, err := td.GUID()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("delegate({%s})", guid), nil

	case td.IsRuntimeClass():
		if td.Flags&winmd.TypeAttributes_Abstract != 0 {
			return "", nil
		}

		defaultIface, err := td.GetAttributeWithType(mdStore.AttributeTypeDefaultAttribute)
		if err != nil {
			ifs, ifsErr := td.GetImplementedInterfaces()
			if ifsErr != nil || len(ifs) == 0 {
				return "", err
			}
			defaultIface = []byte(ifs[0].Namespace + "." + ifs[0].Name)
		}

		ifaceTD, err := g.store.TypeDefByName(string(defaultIface))
		if err != nil {
			return "", err
		}

		ifaceSig, err := g.typeSignature(ifaceTD)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rc(%s;%s)", td.Namespace.String()+"."+td.Name.String(), ifaceSig), nil

	default:
		return "", fmt.Errorf("unsupported type for signature: %s", td.Name.String())
	}
}

func primitiveTypeSignature(kind winmd.ElementType) string {
	switch kind {
	case winmd.ElementType_U1:
		return SignatureUInt8
	case winmd.ElementType_U2:
		return SignatureUInt16
	case winmd.ElementType_U4:
		return SignatureUInt32
	case winmd.ElementType_U8:
		return SignatureUInt64
	case winmd.ElementType_I1:
		return SignatureInt8
	case winmd.ElementType_I2:
		return SignatureInt16
	case winmd.ElementType_I4:
		return SignatureInt32
	case winmd.ElementType_I8:
		return SignatureInt64
	case winmd.ElementType_R4:
		return SignatureFloat32
	case winmd.ElementType_R8:
		return SignatureFloat64
	case winmd.ElementType_BOOLEAN:
		return SignatureBool
	case winmd.ElementType_CHAR:
		return SignatureChar
	case winmd.ElementType_STRING:
		return SignatureString
	default:
		return ""
	}
}
