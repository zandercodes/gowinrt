package gen

import (
	"fmt"

	winmd "github.com/microsoft/go-winmd/winmd"

	mdStore "github.com/zandercodes/gowinrt/internal/winmd"
)

// ---- type creators ----

func (g *generator) createGenInterface(td *mdStore.TypeDef, requiresActivation bool) (*genInterface, error) {
	funcs, err := g.getGenFuncs(td, requiresActivation)
	if err != nil {
		return nil, err
	}

	allFuncs := append([]*genFunc{}, funcs...)

	if g.inheritance {
		inherited, err := g.getInheritedMethods(td, requiresActivation)
		if err != nil {
			return nil, err
		}
		allFuncs = append(allFuncs, inherited...)
	}

	guid, err := td.GUID()
	if err != nil {
		return nil, err
	}

	sig, err := g.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	return &genInterface{
		Name:      toGoName(td.Name.String(), isPublic),
		GUID:      guid,
		Signature: sig,
		Funcs:     allFuncs,
	}, nil
}

func (g *generator) createGenClass(td *mdStore.TypeDef) (*genClass, error) {
	var reqImports []*genImport
	var exclusiveTypes []*mdStore.TypeDef
	activatedMap := make(map[string]bool)

	interfaces, err := td.GetImplementedInterfaces()
	if err != nil {
		return nil, err
	}

	implIfaces := make([]*genInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		reqImports = append(reqImports, &genImport{iface.Namespace, iface.Name})

		ifaceTD, err := g.store.TypeDefByName(iface.Namespace + "." + iface.Name)
		if err != nil {
			return nil, err
		}

		ifaceGen, err := g.createGenInterface(ifaceTD, false)
		if err != nil {
			return nil, err
		}

		pkg := ""
		if td.Namespace.String() != ifaceTD.Namespace.String() {
			pkg = typePackage(iface.Namespace)
		}
		ifacePublic := ifaceTD.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
		for _, fn := range ifaceGen.Funcs {
			fn.InheritedFrom = qualifiedID{
				Namespace: pkg,
				Name:      toGoName(ifaceTD.Name.String(), ifacePublic),
			}
		}
		implIfaces = append(implIfaces, ifaceGen)

		if exTD, err := g.store.TypeDefByName(iface.Namespace + "." + iface.Name); err == nil {
			if _, ok := g.interfaceExclusiveTo(exTD); ok {
				exclusiveTypes = append(exclusiveTypes, exTD)
				activatedMap[exTD.Namespace.String()+"."+exTD.Name.String()] = false
			}
		}
	}

	// Static interfaces
	for _, blob := range td.GetTypeDefAttributesWithType(mdStore.AttributeTypeStaticAttribute) {
		className := extractClassFromBlob(blob)
		g.logger.Debug().Str("class", className).Msg("found static interface")
		staticTD, err := g.store.TypeDefByName(className)
		if err != nil {
			return nil, err
		}
		exclusiveTypes = append(exclusiveTypes, staticTD)
		activatedMap[staticTD.Namespace.String()+"."+staticTD.Name.String()] = true
	}

	// Activatable interfaces
	hasEmptyCtor := false
	for _, blob := range td.GetTypeDefAttributesWithType(mdStore.AttributeTypeActivatableAttribute) {
		if activatableAttrIsEmpty(blob) {
			hasEmptyCtor = true
			continue
		}
		className := extractClassFromBlob(blob)
		g.logger.Debug().Str("class", className).Msg("found activatable interface")
		actTD, err := g.store.TypeDefByName(className)
		if err != nil {
			g.logger.Warn().Err(err).Str("class", className).Msg("activatable class not found, skipping")
			continue
		}
		exclusiveTypes = append(exclusiveTypes, actTD)
		activatedMap[actTD.Namespace.String()+"."+actTD.Name.String()] = true
	}

	var exclusiveIfaces []*genInterface
	for _, exTD := range exclusiveTypes {
		key := exTD.Namespace.String() + "." + exTD.Name.String()
		requiresAct := activatedMap[key]
		isExtended := !requiresAct

		ifaceGen, err := g.createGenInterface(exTD, requiresAct)
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

	sig, err := g.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	isAbstract := td.Flags&winmd.TypeAttributes_Abstract != 0
	return &genClass{
		Name:                toGoName(td.Name.String(), isPublic),
		Signature:           sig,
		RequiresImports:     reqImports,
		FullyQualifiedName:  td.Namespace.String() + "." + td.Name.String(),
		ImplInterfaces:      implIfaces,
		ExclusiveInterfaces: exclusiveIfaces,
		HasEmptyConstructor: hasEmptyCtor,
		IsAbstract:          isAbstract,
	}, nil
}

func (g *generator) createGenEnum(td *mdStore.TypeDef) (*genEnum, error) {
	fields, err := g.resolveFields(td)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("enum %s has no fields", td.Name.String())
	}

	// First field is the underlying integer type
	first := fields[0]
	firstFlags := first.Flags
	isPrivate := firstFlags&winmd.FieldAttributes_FieldAccessMask == winmd.FieldAttributes_Private
	isSpecialName := firstFlags&winmd.FieldAttributes_SpecialName != 0
	isRTSpecialName := firstFlags&winmd.FieldAttributes_RTSpecialName != 0
	if !(isPrivate && isSpecialName && isRTSpecialName) {
		return nil, fmt.Errorf("enum %s.%s first field does not match spec", td.Namespace.String(), td.Name.String())
	}

	fieldSig, err := parseFieldSig(first.Signature)
	if err != nil {
		return nil, err
	}
	elType, err := g.elementType(td.Ctx(), fieldSig.Type, false)
	if err != nil {
		return nil, err
	}
	enumType := elType.name

	var values []*genEnumValue
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
		values = append(values, &genEnumValue{
			Name:  enumValueName(td.Name.String(), f.Name.String()),
			Value: rawValue,
		})
	}

	sig, err := g.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	return &genEnum{
		Name:      toGoName(td.Name.String(), isPublic),
		Type:      enumType,
		Signature: sig,
		Values:    values,
	}, nil
}

func (g *generator) createGenStruct(td *mdStore.TypeDef) (*genStruct, error) {
	fields, err := g.resolveFields(td)
	if err != nil {
		return nil, err
	}

	curPkg := typePackage(td.Namespace.String())
	var genFields []*genParam
	for _, f := range fields {
		fSig, err := parseFieldSig(f.Signature)
		if err != nil {
			return nil, err
		}

		fType, err := g.elementType(td.Ctx(), fSig.Type, false)
		if err != nil {
			return nil, err
		}

		genFields = append(genFields, &genParam{
			callerPackage: curPkg,
			varName:       cleanReservedWords(f.Name.String()),
			IsOut:         false,
			Type:          fType,
		})
	}

	sig, err := g.typeSignature(td)
	if err != nil {
		return nil, err
	}

	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	return &genStruct{
		Name:      toGoName(td.Name.String(), isPublic),
		Signature: sig,
		Fields:    genFields,
	}, nil
}

func (g *generator) createGenDelegate(td *mdStore.TypeDef) (*genDelegate, error) {
	guid, err := td.GUID()
	if err != nil {
		return nil, err
	}

	methods, err := g.resolveMethods(td)
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

	fn, err := g.genFuncFromMethod(td, &invokeMethod, "", false)
	if err != nil {
		return nil, fmt.Errorf("parsing delegate %s invoke: %w", td.Name.String(), err)
	}

	sig, err := g.typeSignature(td)
	if err != nil {
		return nil, err
	}

	return &genDelegate{
		Name:      toGoName(td.Name.String(), true),
		GUID:      guid,
		Signature: sig,
		InParams:  fn.InParams,
	}, nil
}
