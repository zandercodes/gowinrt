package resolve

import (
	"errors"
	"fmt"
	"strings"

	winmd "github.com/microsoft/go-winmd/winmd"

	"github.com/zandercodes/gowinrt/internal/ir"
	"github.com/zandercodes/gowinrt/internal/logger"
	"github.com/zandercodes/gowinrt/internal/metadata"
	"github.com/zandercodes/gowinrt/winrt"
)

const invokeMethodName = "Invoke"

// tdWindowsRuntime indicates that a type is a WinRT type (0x4000 flag)
const tdWindowsRuntime = 0x4000

// errReceiveArray is returned when a method uses the WinRT "receive array"
// pattern (BYREF SZARRAY out-parameter) which the generator does not yet support.
var errReceiveArray = errors.New("receive-array out-parameter pattern not supported")

// Resolver transforms metadata TypeDefs into IR types.
type Resolver struct {
	store       *metadata.Store
	filter      Filter
	inheritance bool
	logger      logger.Log
}

// NewResolver creates a new Resolver.
func NewResolver(store *metadata.Store, filter Filter, inheritance bool, log logger.Log) *Resolver {
	return &Resolver{
		store:       store,
		filter:      filter,
		inheritance: inheritance,
		logger:      log,
	}
}

// ResolveType resolves a metadata TypeDef into all IR types needed for code generation.
// It returns a list of DataFiles (typically one) for the given TypeDef.
func (r *Resolver) ResolveType(td *metadata.TypeDef) ([]*ir.DataFile, error) {
	if td.Flags&tdWindowsRuntime == 0 {
		return nil, fmt.Errorf("%s.%s is not a WinRT type", td.Namespace.String(), td.Name.String())
	}

	f := r.newDataFile(td)

	switch {
	case td.IsInterface():
		r.logger.Info().Str("interface", td.Namespace.String()+"."+td.Name.String()).Msg("resolving interface")
		if err := r.validateInterface(td); err != nil {
			return nil, err
		}
		iface, err := r.createInterface(td, false)
		if err != nil {
			return nil, err
		}
		f.Data.Interfaces = append(f.Data.Interfaces, iface)

	case td.IsEnum():
		r.logger.Info().Str("enum", td.Namespace.String()+"."+td.Name.String()).Msg("resolving enum")
		enum, err := r.createEnum(td)
		if err != nil {
			return nil, err
		}
		f.Data.Enums = append(f.Data.Enums, enum)

	case td.IsStruct():
		r.logger.Info().Str("struct", td.Namespace.String()+"."+td.Name.String()).Msg("resolving struct")
		s, err := r.createStruct(td)
		if err != nil {
			return nil, err
		}
		f.Data.Structs = append(f.Data.Structs, s)

	case td.IsDelegate():
		r.logger.Info().Str("delegate", td.Namespace.String()+"."+td.Name.String()).Msg("resolving delegate")
		d, err := r.createDelegate(td)
		if err != nil {
			return nil, err
		}
		f.Data.Delegates = append(f.Data.Delegates, d)

	default:
		r.logger.Info().Str("class", td.Namespace.String()+"."+td.Name.String()).Msg("resolving class")
		cls, err := r.createClass(td)
		if err != nil {
			return nil, err
		}
		f.Data.Classes = append(f.Data.Classes, cls)
	}

	return []*ir.DataFile{f}, nil
}

func (r *Resolver) newDataFile(td *metadata.TypeDef) *ir.DataFile {
	folder := TypeToFolder(td.Namespace.String())
	filename := folder + "/" + TypeFilename(td.Name.String()) + ".go"
	return &ir.DataFile{
		Filename: filename,
		Data: ir.Data{
			Package: TypePackage(td.Namespace.String()),
		},
	}
}

func (r *Resolver) validateInterface(td *metadata.TypeDef) error {
	if td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_NotPublic {
		return fmt.Errorf("interface %s.%s is not public", td.Namespace.String(), td.Name.String())
	}
	return nil
}

// ---- type creators ----

func (r *Resolver) createInterface(td *metadata.TypeDef, requiresActivation bool) (*ir.Interface, error) {
	funcs, err := r.getGenFuncs(td, requiresActivation)
	if err != nil {
		return nil, err
	}

	allFuncs := append([]*ir.Func{}, funcs...)

	if r.inheritance {
		inherited, err := r.getInheritedMethods(td, requiresActivation)
		if err != nil {
			return nil, err
		}
		allFuncs = append(allFuncs, inherited...)
	}

	guid, err := td.GUID()
	if err != nil {
		return nil, err
	}

	sig, err := r.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	return &ir.Interface{
		Name:      ToGoName(td.Name.String(), isPublic),
		GUID:      guid,
		Signature: sig,
		Funcs:     allFuncs,
	}, nil
}

func (r *Resolver) createClass(td *metadata.TypeDef) (*ir.Class, error) {
	var reqImports []*ir.Import
	var exclusiveTypes []*metadata.TypeDef
	activatedMap := make(map[string]bool)

	interfaces, err := td.GetImplementedInterfaces()
	if err != nil {
		return nil, err
	}

	implIfaces := make([]*ir.Interface, 0, len(interfaces))
	for _, iface := range interfaces {
		reqImports = append(reqImports, &ir.Import{Namespace: iface.Namespace, Name: iface.Name})

		ifaceTD, err := r.store.TypeDefByName(iface.Namespace + "." + iface.Name)
		if err != nil {
			return nil, err
		}

		ifaceGen, err := r.createInterface(ifaceTD, false)
		if err != nil {
			return nil, err
		}

		pkg := ""
		if td.Namespace.String() != ifaceTD.Namespace.String() {
			pkg = TypePackage(iface.Namespace)
		}
		ifacePublic := ifaceTD.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
		for _, fn := range ifaceGen.Funcs {
			fn.InheritedFrom = ir.QualifiedID{
				Namespace: pkg,
				Name:      ToGoName(ifaceTD.Name.String(), ifacePublic),
			}
		}
		implIfaces = append(implIfaces, ifaceGen)

		if exTD, err := r.store.TypeDefByName(iface.Namespace + "." + iface.Name); err == nil {
			if _, ok := r.interfaceExclusiveTo(exTD); ok {
				exclusiveTypes = append(exclusiveTypes, exTD)
				activatedMap[exTD.Namespace.String()+"."+exTD.Name.String()] = false
			}
		}
	}

	// Static interfaces
	for _, blob := range td.GetTypeDefAttributesWithType(metadata.AttributeTypeStaticAttribute) {
		className := extractClassFromBlob(blob)
		r.logger.Debug().Str("class", className).Msg("found static interface")
		staticTD, err := r.store.TypeDefByName(className)
		if err != nil {
			return nil, err
		}
		exclusiveTypes = append(exclusiveTypes, staticTD)
		activatedMap[staticTD.Namespace.String()+"."+staticTD.Name.String()] = true
	}

	// Activatable interfaces
	hasEmptyCtor := false
	for _, blob := range td.GetTypeDefAttributesWithType(metadata.AttributeTypeActivatableAttribute) {
		if activatableAttrIsEmpty(blob) {
			hasEmptyCtor = true
			continue
		}
		className := extractClassFromBlob(blob)
		r.logger.Debug().Str("class", className).Msg("found activatable interface")
		actTD, err := r.store.TypeDefByName(className)
		if err != nil {
			r.logger.Warn().Err(err).Str("class", className).Msg("activatable class not found, skipping")
			continue
		}
		exclusiveTypes = append(exclusiveTypes, actTD)
		activatedMap[actTD.Namespace.String()+"."+actTD.Name.String()] = true
	}

	var exclusiveIfaces []*ir.Interface
	for _, exTD := range exclusiveTypes {
		key := exTD.Namespace.String() + "." + exTD.Name.String()
		requiresAct := activatedMap[key]
		isExtended := !requiresAct

		ifaceGen, err := r.createInterface(exTD, requiresAct)
		if err != nil {
			return nil, err
		}

		hasImpl := false
		for _, fn := range ifaceGen.Funcs {
			if fn.Implement {
				hasImpl = true
				break
			}
		}

		if isExtended || hasImpl {
			exclusiveIfaces = append(exclusiveIfaces, ifaceGen)
		}
	}

	sig, err := r.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	isAbstract := td.Flags&winmd.TypeAttributes_Abstract != 0
	return &ir.Class{
		Name:                ToGoName(td.Name.String(), isPublic),
		Signature:           sig,
		RequiredImports:     reqImports,
		FullyQualifiedName:  td.Namespace.String() + "." + td.Name.String(),
		ImplInterfaces:      implIfaces,
		ExclusiveInterfaces: exclusiveIfaces,
		HasEmptyConstructor: hasEmptyCtor,
		IsAbstract:          isAbstract,
	}, nil
}

func (r *Resolver) createEnum(td *metadata.TypeDef) (*ir.Enum, error) {
	fields, err := r.resolveFields(td)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("enum %s has no fields", td.Name.String())
	}

	first := fields[0]
	firstFlags := first.Flags
	isPrivate := firstFlags&winmd.FieldAttributes_FieldAccessMask == winmd.FieldAttributes_Private
	isSpecialName := firstFlags&winmd.FieldAttributes_SpecialName != 0
	isRTSpecialName := firstFlags&winmd.FieldAttributes_RTSpecialName != 0
	if !(isPrivate && isSpecialName && isRTSpecialName) {
		return nil, fmt.Errorf("enum %s.%s first field does not match spec", td.Namespace.String(), td.Name.String())
	}

	fieldSig, err := metadata.ParseFieldSig(first.Signature)
	if err != nil {
		return nil, err
	}
	elType, err := r.elementType(td.Ctx(), fieldSig.Type, false)
	if err != nil {
		return nil, err
	}
	enumType := elType.Name

	var values []*ir.EnumValue
	for i, f := range fields[1:] {
		fFlags := f.Flags
		isPublicField := fFlags&winmd.FieldAttributes_FieldAccessMask == winmd.FieldAttributes_Public
		isStatic := fFlags&winmd.FieldAttributes_Static != 0
		isLiteral := fFlags&winmd.FieldAttributes_Literal != 0
		hasDefault := fFlags&winmd.FieldAttributes_HasDefault != 0
		if !(isPublicField && isStatic && isLiteral && hasDefault) {
			return nil, fmt.Errorf("enum %s.%s field does not comply with spec", td.Namespace.String(), td.Name.String())
		}

		fieldIndex := uint32(td.FieldList.Start) + 1 + uint32(i)
		rawValue, err := td.GetValueForEnumField(fieldIndex)
		if err != nil {
			return nil, err
		}
		values = append(values, &ir.EnumValue{
			Name:  EnumValueName(td.Name.String(), f.Name.String()),
			Value: rawValue,
		})
	}

	sig, err := r.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	return &ir.Enum{
		Name:      ToGoName(td.Name.String(), isPublic),
		Type:      enumType,
		Signature: sig,
		Values:    values,
	}, nil
}

func (r *Resolver) createStruct(td *metadata.TypeDef) (*ir.Struct, error) {
	fields, err := r.resolveFields(td)
	if err != nil {
		return nil, err
	}

	curPkg := TypePackage(td.Namespace.String())
	var genFields []*ir.Param
	for _, f := range fields {
		fSig, err := metadata.ParseFieldSig(f.Signature)
		if err != nil {
			return nil, err
		}

		fType, err := r.elementType(td.Ctx(), fSig.Type, false)
		if err != nil {
			return nil, err
		}

		genFields = append(genFields, &ir.Param{
			CallerPackage: curPkg,
			VarName:       CleanReservedWords(f.Name.String()),
			IsOut:         false,
			Type:          fType,
		})
	}

	sig, err := r.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	return &ir.Struct{
		Name:      ToGoName(td.Name.String(), isPublic),
		Signature: sig,
		Fields:    genFields,
	}, nil
}

func (r *Resolver) createDelegate(td *metadata.TypeDef) (*ir.Delegate, error) {
	guid, err := td.GUID()
	if err != nil {
		return nil, err
	}

	methods, err := r.resolveMethods(td)
	if err != nil {
		return nil, err
	}
	if len(methods) != 2 {
		return nil, fmt.Errorf("delegate %s.%s should have exactly 2 methods, found %d",
			td.Namespace.String(), td.Name.String(), len(methods))
	}

	invokeMethod := methods[1]
	if invokeMethod.Name.String() != invokeMethodName {
		return nil, fmt.Errorf("expected method '%s' on delegate %s.%s, found '%s'",
			invokeMethodName, td.Namespace.String(), td.Name.String(), invokeMethod.Name.String())
	}

	fn, err := r.genFuncFromMethod(td, &invokeMethod, "", false)
	if err != nil {
		return nil, fmt.Errorf("parsing delegate %s invoke: %w", td.Name.String(), err)
	}

	sig, err := r.typeSignature(td)
	if err != nil {
		return nil, err
	}

	return &ir.Delegate{
		Name:      ToGoName(td.Name.String(), true),
		GUID:      guid,
		Signature: sig,
		InParams:  fn.InParams,
	}, nil
}

// ---- resolution helpers ----

func (r *Resolver) resolveFields(td *metadata.TypeDef) ([]winmd.Field, error) {
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

func (r *Resolver) resolveMethods(td *metadata.TypeDef) ([]winmd.MethodDef, error) {
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

func (r *Resolver) resolveParams(td *metadata.TypeDef, md *winmd.MethodDef) ([]winmd.Param, error) {
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

func (r *Resolver) elementType(ctx *winmd.Metadata, st winmd.SigType, isByRef bool) (*ir.ParamType, error) {
	switch st.Kind {
	case winmd.ElementType_BOOLEAN:
		return &ir.ParamType{Name: "bool", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "false", IsPrimitive: true}}, nil
	case winmd.ElementType_CHAR:
		return &ir.ParamType{Name: "byte", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_I1:
		return &ir.ParamType{Name: "int8", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_U1:
		return &ir.ParamType{Name: "uint8", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_I2:
		return &ir.ParamType{Name: "int16", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_U2:
		return &ir.ParamType{Name: "uint16", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_I4:
		return &ir.ParamType{Name: "int32", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_U4:
		return &ir.ParamType{Name: "uint32", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_I8:
		return &ir.ParamType{Name: "int64", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_U8:
		return &ir.ParamType{Name: "uint64", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true}}, nil
	case winmd.ElementType_R4:
		return &ir.ParamType{Name: "float32", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0.0", IsPrimitive: true}}, nil
	case winmd.ElementType_R8:
		return &ir.ParamType{Name: "float64", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: "0.0", IsPrimitive: true}}, nil
	case winmd.ElementType_STRING:
		return &ir.ParamType{Name: "string", IsPrimitive: true, DefaultValue: ir.DefaultValue{Value: `""`, IsPrimitive: true}}, nil

	case winmd.ElementType_GENERICINST, winmd.ElementType_CLASS:
		ci := st.Value.(winmd.CodedIndex[winmd.TypeDefOrRefOrSpec])
		ns, name, err := resolveTypeDefOrRefOrSpec(ctx, ci)
		if err != nil {
			return nil, err
		}
		return &ir.ParamType{
			Namespace:    ns,
			Name:         name,
			IsPointer:    true,
			DefaultValue: ir.DefaultValue{Value: "nil", IsPrimitive: true},
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

		elemTD, err := r.store.TypeDefByName(ns + "." + name)
		if err != nil {
			return nil, err
		}

		isEnum := false
		enumType := ""
		if elemTD.IsEnum() {
			enumData, err := r.createEnum(elemTD)
			if err != nil {
				return nil, err
			}
			isEnum = true
			enumType = enumData.Type
		}

		return &ir.ParamType{
			Namespace:          ns,
			Name:               name,
			IsEnum:             isEnum,
			UnderlyingEnumType: enumType,
			DefaultValue:       r.valueTypeDefault(ctx, ns, name),
		}, nil

	case winmd.ElementType_VAR:
		return &ir.ParamType{
			Namespace:    "unsafe",
			Name:         "Pointer",
			IsGeneric:    true,
			DefaultValue: ir.DefaultValue{Value: "nil", IsPrimitive: true},
		}, nil

	case winmd.ElementType_SZARRAY:
		inner := st.Value.(winmd.SigType)
		param, err := r.elementType(ctx, inner, false)
		if err != nil {
			return nil, err
		}
		param.IsArray = true
		param.DefaultValue = ir.DefaultValue{Value: "nil", IsPrimitive: true}
		return param, nil

	case winmd.ElementType_OBJECT:
		return &ir.ParamType{
			Namespace:    "unsafe",
			Name:         "Pointer",
			DefaultValue: ir.DefaultValue{Value: "nil", IsPrimitive: true},
		}, nil

	case winmd.ElementType_BYREF:
		inner := st.Value.(winmd.SigType)
		return r.elementType(ctx, inner, true)

	default:
		return nil, fmt.Errorf("unsupported element type: 0x%02x", st.Kind)
	}
}

func isSystemType(ns, name string) (*ir.ParamType, bool) {
	if ns != "System" {
		return nil, false
	}
	switch name {
	case "Guid":
		return &ir.ParamType{
			Namespace:    "syscall",
			Name:         "GUID",
			DefaultValue: ir.DefaultValue{Value: "GUID{}", IsPrimitive: false},
		}, true
	}
	return nil, false
}

func (r *Resolver) valueTypeDefault(ctx *winmd.Metadata, ns, name string) ir.DefaultValue {
	td, err := r.store.TypeDefByName(ns + "." + name)
	if err != nil {
		return ir.DefaultValue{Value: "nil", IsPrimitive: true}
	}
	if td.IsEnum() {
		fields, err := r.resolveFields(td)
		if err != nil || len(fields) < 2 {
			return ir.DefaultValue{Value: "0", IsPrimitive: true}
		}
		return ir.DefaultValue{Value: EnumValueName(td.Name.String(), fields[1].Name.String()), IsPrimitive: false}
	}
	if td.IsStruct() {
		return ir.DefaultValue{Value: td.Name.String() + "{}", IsPrimitive: false}
	}
	return ir.DefaultValue{Value: "nil", IsPrimitive: true}
}

// ---- attribute helpers ----

func (r *Resolver) interfaceExclusiveTo(td *metadata.TypeDef) (string, bool) {
	blob, err := td.GetAttributeWithType(metadata.AttributeTypeExclusiveTo)
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

// ---- type signature ----

func (r *Resolver) typeSignature(td *metadata.TypeDef) (string, error) {
	switch {
	case td.IsInterface():
		guid, err := td.GUID()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("{%s}", guid), nil

	case td.IsEnum():
		fields, err := r.resolveFields(td)
		if err != nil {
			return "", err
		}
		fSig, err := metadata.ParseFieldSig(fields[0].Signature)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("enum(%s;%s)", td.Namespace.String()+"."+td.Name.String(), PrimitiveTypeSignature(fSig.Type.Kind)), nil

	case td.IsStruct():
		fields, err := r.resolveFields(td)
		if err != nil {
			return "", err
		}
		var args []string
		for _, f := range fields {
			fSig, err := metadata.ParseFieldSig(f.Signature)
			if err != nil {
				return "", err
			}
			if fSig.Type.Kind == winmd.ElementType_VALUETYPE {
				fType, err := r.elementType(td.Ctx(), fSig.Type, false)
				if err != nil {
					return "", err
				}
				innerTD, err := r.store.TypeDefByName(fType.Namespace + "." + fType.Name)
				if err != nil {
					return "", err
				}
				innerSig, err := r.typeSignature(innerTD)
				if err != nil {
					return "", err
				}
				args = append(args, innerSig)
			} else {
				args = append(args, PrimitiveTypeSignature(fSig.Type.Kind))
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

		defaultIface, err := td.GetAttributeWithType(metadata.AttributeTypeDefaultAttribute)
		if err != nil {
			ifs, ifsErr := td.GetImplementedInterfaces()
			if ifsErr != nil || len(ifs) == 0 {
				return "", err
			}
			defaultIface = []byte(ifs[0].Namespace + "." + ifs[0].Name)
		}

		ifaceTD, err := r.store.TypeDefByName(string(defaultIface))
		if err != nil {
			return "", err
		}

		ifaceSig, err := r.typeSignature(ifaceTD)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("rc(%s;%s)", td.Namespace.String()+"."+td.Name.String(), ifaceSig), nil

	default:
		return "", fmt.Errorf("unsupported type for signature: %s", td.Name.String())
	}
}

// PrimitiveTypeSignature returns the WinRT type signature string for a primitive ElementType.
func PrimitiveTypeSignature(kind winmd.ElementType) string {
	switch kind {
	case winmd.ElementType_U1:
		return winrt.SignatureUInt8
	case winmd.ElementType_U2:
		return winrt.SignatureUInt16
	case winmd.ElementType_U4:
		return winrt.SignatureUInt32
	case winmd.ElementType_U8:
		return winrt.SignatureUInt64
	case winmd.ElementType_I1:
		return winrt.SignatureInt8
	case winmd.ElementType_I2:
		return winrt.SignatureInt16
	case winmd.ElementType_I4:
		return winrt.SignatureInt32
	case winmd.ElementType_I8:
		return winrt.SignatureInt64
	case winmd.ElementType_R4:
		return winrt.SignatureFloat32
	case winmd.ElementType_R8:
		return winrt.SignatureFloat64
	case winmd.ElementType_BOOLEAN:
		return winrt.SignatureBool
	case winmd.ElementType_CHAR:
		return winrt.SignatureChar
	case winmd.ElementType_STRING:
		return winrt.SignatureString
	default:
		return ""
	}
}

// ---- method helpers ----

func (r *Resolver) getGenFuncs(td *metadata.TypeDef, requiresActivation bool) ([]*ir.Func, error) {
	methods, err := r.resolveMethods(td)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve method list: %w", err)
	}

	exclusiveTo := ""
	if ex, ok := r.interfaceExclusiveTo(td); ok {
		exclusiveTo = ex
	}

	out := make([]*ir.Func, 0, len(methods))
	for _, m := range methods {
		md := m
		fn, err := r.genFuncFromMethod(td, &md, exclusiveTo, requiresActivation)
		if err != nil {
			return nil, fmt.Errorf("failed to generate function from method %s: %w", md.Name.String(), err)
		}
		out = append(out, fn)
	}
	return out, nil
}

func (r *Resolver) getInheritedMethods(td *metadata.TypeDef, requiresActivation bool) ([]*ir.Func, error) {
	var out []*ir.Func
	seen := map[string]bool{}

	interfaces, err := td.GetImplementedInterfaces()
	if err != nil {
		return out, nil
	}

	for _, iface := range interfaces {
		baseTD, err := r.store.TypeDefByName(iface.Namespace + "." + iface.Name)
		if err != nil {
			r.logger.Warn().Err(err).Str("interface", iface.Namespace+"."+iface.Name).Msg("base interface not found")
			continue
		}

		baseFuncs, err := r.getGenFuncs(baseTD, requiresActivation)
		if err != nil {
			return nil, err
		}

		for _, fn := range baseFuncs {
			if seen[fn.Name] {
				continue
			}
			w := *fn
			isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
			w.FuncOwner = ToGoName(td.Name.String(), isPublic)

			pkg := ""
			if td.Namespace.String() != baseTD.Namespace.String() {
				pkg = TypePackage(iface.Namespace)
			}
			basePublic := baseTD.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
			w.InheritedFrom = ir.QualifiedID{
				Namespace: pkg,
				Name:      ToGoName(baseTD.Name.String(), basePublic),
			}
			out = append(out, &w)
			seen[fn.Name] = true
		}
	}
	return out, nil
}

func (r *Resolver) genFuncFromMethod(td *metadata.TypeDef, md *winmd.MethodDef, exclusiveTo string, requiresActivation bool) (*ir.Func, error) {
	overloadName := metadata.GetMethodOverloadName(td.Ctx(), md)
	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	owner := ToGoName(td.Name.String(), isPublic)
	implement := r.filter.Matches(overloadName)

	stub := &ir.Func{
		Name:               overloadName,
		Implement:          false,
		FuncOwner:          owner,
		ExclusiveTo:        exclusiveTo,
		RequiresActivation: requiresActivation,
	}

	if !implement {
		return stub, nil
	}

	curPkg := TypePackage(td.Namespace.String())

	inParams, err := r.getInParams(curPkg, td, md)
	if err != nil {
		if errors.Is(err, errReceiveArray) {
			r.logger.Warn().Str("method", overloadName).Msg("skipping implementation: receive-array pattern not yet supported")
			return stub, nil
		}
		return nil, fmt.Errorf("parsing params for %s: %w", overloadName, err)
	}

	retParams, err := r.getReturnParams(curPkg, td, md)
	if err != nil {
		return nil, fmt.Errorf("parsing return for %s: %w", overloadName, err)
	}

	var allParams []*ir.Param
	allParams = append(allParams, inParams...)
	allParams = append(allParams, retParams...)

	var reqImports []*ir.Import
	for _, p := range allParams {
		p.CallerPackage = curPkg
		if !p.Type.IsPrimitive {
			reqImports = append(reqImports, &ir.Import{Namespace: p.Type.Namespace, Name: p.Type.Name})
		}
	}

	return &ir.Func{
		Name:               overloadName,
		RequiredImports:    reqImports,
		Implement:          true,
		InParams:           inParams,
		ReturnParams:       retParams,
		FuncOwner:          owner,
		ExclusiveTo:        exclusiveTo,
		RequiresActivation: requiresActivation,
	}, nil
}

func (r *Resolver) getInParams(curPkg string, td *metadata.TypeDef, md *winmd.MethodDef) ([]*ir.Param, error) {
	params, err := r.resolveParams(td, md)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve parameter list: %w", err)
	}

	sig, err := metadata.ParseMethodSig(md.Signature)
	if err != nil {
		return nil, err
	}

	var out []*ir.Param
	for i, sp := range sig.Param {
		param := paramBySequence(params, uint16(i+1))
		if param == nil {
			r.logger.Error().Int("index", i+1).Msg("parameter not found")
			continue
		}

		if sp.Type.Kind == winmd.ElementType_BYREF {
			if inner, ok := sp.Type.Value.(winmd.SigType); ok {
				if (inner.Kind == winmd.ElementType_SZARRAY || inner.Kind == winmd.ElementType_ARRAY) &&
					param.Flags&winmd.ParamAttributes_Out != 0 {
					return nil, errReceiveArray
				}
			}
		}

		if sp.Type.Kind == winmd.ElementType_SZARRAY || sp.Type.Kind == winmd.ElementType_ARRAY {
			isOutSize := param.Flags&winmd.ParamAttributes_Out != 0 && sp.Kind == winmd.SigParamKind_ByRef
			out = append(out, &ir.Param{
				CallerPackage: curPkg,
				VarName:       CleanReservedWords(param.Name.String() + "Size"),
				IsOut:         isOutSize,
				Type: &ir.ParamType{
					Name:         "uint32",
					IsPrimitive:  true,
					DefaultValue: ir.DefaultValue{Value: "0", IsPrimitive: true},
				},
			})
		}

		isByRef := sp.Kind == winmd.SigParamKind_ByRef
		elType, err := r.elementType(td.Ctx(), sp.Type, isByRef)
		if err != nil {
			return nil, err
		}

		isOut := param.Flags&winmd.ParamAttributes_Out != 0
		out = append(out, &ir.Param{
			CallerPackage: curPkg,
			VarName:       CleanReservedWords(param.Name.String()),
			IsOut:         isOut,
			Type:          elType,
		})
	}
	return out, nil
}

func (r *Resolver) getReturnParams(curPkg string, td *metadata.TypeDef, md *winmd.MethodDef) ([]*ir.Param, error) {
	sig, err := metadata.ParseMethodSig(md.Signature)
	if err != nil {
		return nil, err
	}

	if sig.RetType.Kind == winmd.SigRetTypeKind_Void {
		return nil, nil
	}

	elType, err := r.elementType(td.Ctx(), sig.RetType.Type, false)
	if err != nil {
		return nil, err
	}

	return []*ir.Param{{
		CallerPackage: curPkg,
		VarName:       "out",
		IsOut:         true,
		Type:          elType,
	}}, nil
}
