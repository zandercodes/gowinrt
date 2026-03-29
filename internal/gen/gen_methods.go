package gen

import (
	"fmt"

	winmd "github.com/microsoft/go-winmd/winmd"

	mdStore "github.com/zandercodes/gowinrt/internal/winmd"
)

// ---- method helpers ----

func (g *generator) getGenFuncs(td *mdStore.TypeDef, requiresActivation bool) ([]*genFunc, error) {
	methods, err := g.resolveMethods(td)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve method list: %w", err)
	}

	exclusiveTo := ""
	if ex, ok := g.interfaceExclusiveTo(td); ok {
		exclusiveTo = ex
	}

	// Pre-allocate slice with exact capacity needed
	out := make([]*genFunc, 0, len(methods))
	for _, m := range methods {
		md := m
		fn, err := g.genFuncFromMethod(td, &md, exclusiveTo, requiresActivation)
		if err != nil {
			return nil, fmt.Errorf("failed to generate function from method %s: %w", md.Name.String(), err)
		}
		out = append(out, fn)
	}
	return out, nil
}

func (g *generator) getInheritedMethods(td *mdStore.TypeDef, requiresActivation bool) ([]*genFunc, error) {
	var out []*genFunc
	seen := map[string]bool{}

	interfaces, err := td.GetImplementedInterfaces()
	if err != nil {
		return out, nil
	}

	for _, iface := range interfaces {
		baseTD, err := g.store.TypeDefByName(iface.Namespace + "." + iface.Name)
		if err != nil {
			g.logger.Warn().Err(err).Str("interface", iface.Namespace+"."+iface.Name).Msg("base interface not found")
			continue
		}

		baseFuncs, err := g.getGenFuncs(baseTD, requiresActivation)
		if err != nil {
			return nil, err
		}

		for _, fn := range baseFuncs {
			if seen[fn.Name] {
				continue
			}
			w := *fn
			isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
			w.FuncOwner = toGoName(td.Name.String(), isPublic)

			pkg := ""
			if td.Namespace.String() != baseTD.Namespace.String() {
				pkg = typePackage(iface.Namespace)
			}
			basePublic := baseTD.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
			w.InheritedFrom = qualifiedID{
				Namespace: pkg,
				Name:      toGoName(baseTD.Name.String(), basePublic),
			}
			out = append(out, &w)
			seen[fn.Name] = true
		}
	}
	return out, nil
}

func (g *generator) genFuncFromMethod(td *mdStore.TypeDef, md *winmd.MethodDef, exclusiveTo string, requiresActivation bool) (*genFunc, error) {
	overloadName := mdStore.GetMethodOverloadName(td.Ctx(), md)
	isPublic := td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_Public
	owner := toGoName(td.Name.String(), isPublic)
	implement := g.filter.Matches(overloadName)

	stub := &genFunc{
		Name:               overloadName,
		Implement:          false,
		FuncOwner:          owner,
		ExclusiveTo:        exclusiveTo,
		RequiresActivation: requiresActivation,
	}

	if !implement {
		return stub, nil
	}

	curPkg := typePackage(td.Namespace.String())

	inParams, err := g.getInParams(curPkg, td, md)
	if err != nil {
		return nil, fmt.Errorf("parsing params for %s: %w", overloadName, err)
	}

	retParams, err := g.getReturnParams(curPkg, td, md)
	if err != nil {
		return nil, fmt.Errorf("parsing return for %s: %w", overloadName, err)
	}

	var allParams []*genParam
	allParams = append(allParams, inParams...)
	allParams = append(allParams, retParams...)

	var reqImports []*genImport
	for _, p := range allParams {
		p.callerPackage = curPkg
		if !p.Type.IsPrimitive {
			reqImports = append(reqImports, &genImport{p.Type.namespace, p.Type.name})
		}
	}

	return &genFunc{
		Name:               overloadName,
		RequiresImports:    reqImports,
		Implement:          true,
		InParams:           inParams,
		ReturnParams:       retParams,
		FuncOwner:          owner,
		ExclusiveTo:        exclusiveTo,
		RequiresActivation: requiresActivation,
	}, nil
}

func (g *generator) getInParams(curPkg string, td *mdStore.TypeDef, md *winmd.MethodDef) ([]*genParam, error) {
	params, err := g.resolveParams(td, md)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve parameter list: %w", err)
	}

	sig, err := parseMethodSig(md.Signature)
	if err != nil {
		return nil, err
	}

	var out []*genParam
	for i, sp := range sig.Param {
		param := paramBySequence(params, uint16(i+1))
		if param == nil {
			g.logger.Error().Int("index", i+1).Msg("parameter not found")
			continue
		}

		// Array parameters need a preceding size param
		if sp.Type.Kind == winmd.ElementType_SZARRAY || sp.Type.Kind == winmd.ElementType_ARRAY {
			isOutSize := param.Flags&winmd.ParamAttributes_Out != 0 && sp.Kind == winmd.SigParamKind_ByRef
			out = append(out, &genParam{
				callerPackage: curPkg,
				varName:       cleanReservedWords(param.Name.String() + "Size"),
				IsOut:         isOutSize,
				Type: &genParamType{
					name:         "uint32",
					IsPrimitive:  true,
					defaultValue: genDefaultValue{"0", true},
				},
			})
		}

		isByRef := sp.Kind == winmd.SigParamKind_ByRef
		elType, err := g.elementType(td.Ctx(), sp.Type, isByRef)
		if err != nil {
			return nil, err
		}

		isOut := param.Flags&winmd.ParamAttributes_Out != 0
		out = append(out, &genParam{
			callerPackage: curPkg,
			varName:       cleanReservedWords(param.Name.String()),
			IsOut:         isOut,
			Type:          elType,
		})
	}
	return out, nil
}

func (g *generator) getReturnParams(curPkg string, td *mdStore.TypeDef, md *winmd.MethodDef) ([]*genParam, error) {
	sig, err := parseMethodSig(md.Signature)
	if err != nil {
		return nil, err
	}

	if sig.RetType.Kind == winmd.SigRetTypeKind_Void {
		return nil, nil
	}

	elType, err := g.elementType(td.Ctx(), sig.RetType.Type, false)
	if err != nil {
		return nil, err
	}

	return []*genParam{{
		callerPackage: curPkg,
		varName:       "out",
		IsOut:         true,
		Type:          elType,
	}}, nil
}
