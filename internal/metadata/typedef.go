package metadata

import (
	"fmt"
	"strconv"

	winmd "github.com/microsoft/go-winmd/winmd"
	"github.com/zandercodes/gowinrt/internal/logger"
)

// TypeDef is a helper struct that wraps winmd.TypeDef and stores the original context
// of the typeDef.
type TypeDef struct {
	winmd.TypeDef
	HasContext

	index  winmd.Index
	logger logger.Log
}

// QualifiedID holds the namespace and the name of a qualified element.
// This may be a type, a static function or a field.
type QualifiedID struct {
	Namespace string
	Name      string
}

// ResolveTypeDefOrRefName resolves a TypeDefOrRef coded index to its namespace and name.
func ResolveTypeDefOrRefName(m *winmd.Metadata, ci winmd.CodedIndex[winmd.TypeDefOrRef]) (string, string, error) {
	switch {
	case ci.Tag == winmd.TypeDefOrRef_TypeDef:
		td, err := m.Tables.TypeDef.At(ci.Index)
		if err != nil {
			return "", "", err
		}
		return td.Namespace.String(), td.Name.String(), nil
	case ci.Tag == winmd.TypeDefOrRef_TypeRef:
		tr, err := m.Tables.TypeRef.At(ci.Index)
		if err != nil {
			return "", "", err
		}
		return tr.Namespace.String(), tr.Name.String(), nil
	default:
		return "", "", fmt.Errorf("unsupported TypeDefOrRef tag")
	}
}

// GetValueForEnumField returns the value of the requested enum field.
func (typeDef *TypeDef) GetValueForEnumField(fieldIndex uint32) (string, error) {
	for i := range typeDef.Ctx().Tables.Constant.Indices() {
		constant, err := typeDef.Ctx().Tables.Constant.At(i)
		if err != nil {
			return "", err
		}

		if constant.Parent.Tag != winmd.HasConstant_Field {
			continue
		}

		if constant.Parent.Index != winmd.Index(fieldIndex) {
			continue
		}

		var blobIndex uint32
		for j, b := range constant.Value {
			blobIndex += uint32(b) << (j * 8)
		}
		return strconv.Itoa(int(blobIndex)), nil
	}

	return "", fmt.Errorf("no value found for field %d", fieldIndex)
}

// GetAttributeWithType returns the value of the given attribute type and fails if not found.
func (typeDef *TypeDef) GetAttributeWithType(lookupAttrTypeClass string) ([]byte, error) {
	result := typeDef.GetTypeDefAttributesWithType(lookupAttrTypeClass)
	if len(result) == 0 {
		return nil, fmt.Errorf("type %s has no custom attribute %s",
			typeDef.Namespace.String()+"."+typeDef.Name.String(), lookupAttrTypeClass)
	} else if len(result) > 1 {
		typeDef.logger.Warn().
			Str("type", typeDef.Namespace.String()+"."+typeDef.Name.String()).
			Str("attr", lookupAttrTypeClass).
			Msg("type has multiple custom attributes, returning the first one")
	}

	return result[0], nil
}

// GetTypeDefAttributesWithType returns the values of all the attributes that match the given type.
func (typeDef *TypeDef) GetTypeDefAttributesWithType(lookupAttrTypeClass string) [][]byte {
	result := make([][]byte, 0)

	for i := range typeDef.Ctx().Tables.CustomAttribute.Indices() {
		cAttr, err := typeDef.Ctx().Tables.CustomAttribute.At(i)
		if err != nil {
			continue
		}

		if cAttr.Parent.Tag != winmd.HasCustomAttribute_TypeDef {
			continue
		}
		if cAttr.Parent.Index != typeDef.index {
			continue
		}

		if cAttr.Type.Tag != winmd.CustomAttributeType_MemberRef {
			continue
		}

		attrTypeMemberRef, err := typeDef.Ctx().Tables.MemberRef.At(cAttr.Type.Index)
		if err != nil {
			continue
		}

		if attrTypeMemberRef.Class.Tag != winmd.MemberRefParent_TypeRef {
			continue
		}

		attrTypeRef, err := typeDef.Ctx().Tables.TypeRef.At(attrTypeMemberRef.Class.Index)
		if err != nil {
			continue
		}

		if attrTypeRef.Namespace.String()+"."+attrTypeRef.Name.String() == lookupAttrTypeClass {
			result = append(result, cAttr.Value)
		}
	}

	return result
}

// GetImplementedInterfaces returns the interfaces implemented by the type.
func (typeDef *TypeDef) GetImplementedInterfaces() ([]QualifiedID, error) {
	interfaces := make([]QualifiedID, 0)

	for i := range typeDef.Ctx().Tables.InterfaceImpl.Indices() {
		interfaceImpl, err := typeDef.Ctx().Tables.InterfaceImpl.At(i)
		if err != nil {
			return nil, err
		}

		if interfaceImpl.Class != typeDef.index {
			continue
		}

		if interfaceImpl.Interface.Tag == winmd.TypeDefOrRef_TypeSpec {
			continue
		}

		ifaceNS, ifaceName, err := ResolveTypeDefOrRefName(typeDef.Ctx(), interfaceImpl.Interface)
		if err != nil {
			return nil, err
		}

		interfaces = append(interfaces, QualifiedID{Namespace: ifaceNS, Name: ifaceName})
	}

	return interfaces, nil
}

// Extends returns true if the type extends the given class.
func (typeDef *TypeDef) Extends(class string) (bool, error) {
	ns, name, err := ResolveTypeDefOrRefName(typeDef.Ctx(), typeDef.TypeDef.Extends)
	if err != nil {
		return false, err
	}
	return ns+"."+name == class, nil
}

// IsInterface returns true if the type is an interface.
func (typeDef *TypeDef) IsInterface() bool {
	return typeDef.Flags&winmd.TypeAttributes_ClassSemanticsMask == winmd.TypeAttributes_Interface
}

// IsEnum returns true if the type is an enum.
func (typeDef *TypeDef) IsEnum() bool {
	ok, err := typeDef.Extends("System.Enum")
	if err != nil {
		typeDef.logger.Error().Err(err).Msg("error resolving type extends, all classes should extend at least System.Object")
		return false
	}
	return ok
}

// IsDelegate returns true if the type is a delegate.
func (typeDef *TypeDef) IsDelegate() bool {
	isPublic := typeDef.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	isSealed := typeDef.Flags&winmd.TypeAttributes_Sealed != 0
	if !(isPublic && isSealed) {
		return false
	}

	ok, err := typeDef.Extends("System.MulticastDelegate")
	if err != nil {
		typeDef.logger.Error().Err(err).Msg("error resolving type extends, all classes should extend at least System.Object")
		return false
	}
	return ok
}

// IsStruct returns true if the type is a struct.
func (typeDef *TypeDef) IsStruct() bool {
	ok, err := typeDef.Extends("System.ValueType")
	if err != nil {
		typeDef.logger.Error().Err(err).Msg("error resolving type extends, all classes should extend at least System.Object")
		return false
	}
	return ok
}

// IsRuntimeClass returns true if the type is a runtime class.
func (typeDef *TypeDef) IsRuntimeClass() bool {
	isPublic := typeDef.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	isAutoLayout := typeDef.Flags&winmd.TypeAttributes_LayoutMask == winmd.TypeAttributes_AutoLayout
	isClass := typeDef.Flags&winmd.TypeAttributes_ClassSemanticsMask == winmd.TypeAttributes_Class
	isWindowsRuntime := typeDef.Flags&0x4000 != 0
	return isPublic && isAutoLayout && isClass && isWindowsRuntime
}

// GUID returns the GUID of the type.
func (typeDef *TypeDef) GUID() (string, error) {
	blob, err := typeDef.GetAttributeWithType(AttributeTypeGUID)
	if err != nil {
		return "", err
	}
	return guidBlobToString(blob)
}

// guidBlobToString converts an array into the textual representation of a GUID.
func guidBlobToString(b []byte) (string, error) {
	if len(b) != 20 {
		return "", fmt.Errorf("invalid GUID blob length: %d", len(b))
	}

	if b[0] != 0x01 || b[1] != 0x00 {
		return "", fmt.Errorf("invalid GUID blob header, expected '0x01 0x00' but found '0x%02x 0x%02x'", b[0], b[1])
	}

	if b[18] != 0x00 || b[19] != 0x00 {
		return "", fmt.Errorf("invalid GUID blob footer, expected '0x00 0x00' but found '0x%02x 0x%02x'", b[18], b[19])
	}

	guid := b[2 : len(b)-2]
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%04x%08x",
		uint32(guid[0])|uint32(guid[1])<<8|uint32(guid[2])<<16|uint32(guid[3])<<24,
		uint16(guid[4])|uint16(guid[5])<<8,
		uint16(guid[6])|uint16(guid[7])<<8,
		uint16(guid[8])<<8|uint16(guid[9]),
		uint16(guid[10])<<8|uint16(guid[11]),
		uint32(guid[12])<<24|uint32(guid[13])<<16|uint32(guid[14])<<8|uint32(guid[15])), nil
}
