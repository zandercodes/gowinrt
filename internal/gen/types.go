package gen

import "strings"

// ---------- data types used by templates ----------

type genDataFile struct {
	Filename string
	Data     genData
}

type genData struct {
	Package    string
	Imports    []string
	Classes    []*genClass
	Enums      []*genEnum
	Interfaces []*genInterface
	Structs    []*genStruct
	Delegates  []*genDelegate
}

func (g *genData) computeImports(callerNS string) {
	imports := make([]*genImport, 0)
	for _, c := range g.Classes {
		imports = append(imports, c.requiredImports()...)
	}
	for _, i := range g.Interfaces {
		imports = append(imports, i.requiredImports()...)
	}
	seen := map[string]bool{}
	for _, imp := range imports {
		if imp.Namespace == callerNS {
			continue
		}
		goImp := imp.toGoImport()
		if !seen[goImp] {
			g.Imports = append(g.Imports, goImp)
			seen[goImp] = true
		}
	}
}

type genInterface struct {
	Name      string
	GUID      string
	Signature string
	Funcs     []*genFunc
}

func (g *genInterface) requiredImports() []*genImport {
	var out []*genImport
	for _, f := range g.Funcs {
		out = append(out, f.RequiresImports...)
	}
	return out
}

type genClass struct {
	Name                string
	Signature           string
	RequiresImports     []*genImport
	FullyQualifiedName  string
	ImplInterfaces      []*genInterface
	ExclusiveInterfaces []*genInterface
	HasEmptyConstructor bool
	IsAbstract          bool
}

func (g *genClass) requiredImports() []*genImport {
	out := append([]*genImport{}, g.RequiresImports...)
	for _, i := range g.ExclusiveInterfaces {
		out = append(out, i.requiredImports()...)
	}
	return out
}

type genDelegate struct {
	Name        string
	GUID        string
	Signature   string
	InParams    []*genParam
	ReturnParam *genParam
}

type genEnum struct {
	Name      string
	Type      string
	Signature string
	Values    []*genEnumValue
}

type genEnumValue struct {
	Name  string
	Value string
}

type genFunc struct {
	Name            string
	RequiresImports []*genImport
	Implement       bool
	FuncOwner       string
	InParams        []*genParam
	ReturnParams    []*genParam

	ExclusiveTo        string
	RequiresActivation bool
	InheritedFrom      qualifiedID
}

type qualifiedID struct {
	Namespace, Name string
}

type genImport struct {
	Namespace, Name string
}

func (i genImport) toGoImport() string {
	if !strings.Contains(i.Namespace, ".") && i.Namespace != "Windows" {
		return i.Namespace
	}
	return "github.com/zandercodes/gowinrt/" + typeToFolder(i.Namespace)
}

type genDefaultValue struct {
	value       string
	isPrimitive bool
}

type genParamType struct {
	namespace string
	name      string

	IsPointer          bool
	IsGeneric          bool
	IsArray            bool
	IsPrimitive        bool
	IsEnum             bool
	UnderlyingEnumType string

	defaultValue genDefaultValue
}

type genParam struct {
	callerPackage string
	varName       string
	Type          *genParamType
	IsOut         bool
}

func (p *genParam) GoVarName() string {
	return toGoName(p.varName, true)
}

func (p *genParam) GoTypeName() string {
	if p.Type.IsPrimitive {
		return p.Type.name
	}
	name := toGoName(p.Type.name, true)
	pkg := typePackage(p.Type.namespace)
	if p.callerPackage != pkg {
		name = pkg + "." + name
	}
	return name
}

func (p *genParam) GoDefaultValue() string {
	if p.Type.defaultValue.isPrimitive {
		return p.Type.defaultValue.value
	}
	pkg := typePackage(p.Type.namespace)
	if p.callerPackage != pkg {
		return pkg + "." + p.Type.defaultValue.value
	}
	return p.Type.defaultValue.value
}

type genStruct struct {
	Name      string
	Signature string
	Fields    []*genParam
}
