package ir

import "strings"

// TypeKind identifies the category of a WinRT type.
type TypeKind int

const (
	KindInterface TypeKind = iota
	KindClass
	KindEnum
	KindStruct
	KindDelegate
)

// DataFile pairs a filename with its type data for code generation.
type DataFile struct {
	Filename string
	Data     Data
}

// Data holds all types resolved for a single output file.
type Data struct {
	Package    string
	Imports    []string
	Classes    []*Class
	Enums      []*Enum
	Interfaces []*Interface
	Structs    []*Struct
	Delegates  []*Delegate
}

// Interface represents a WinRT interface.
type Interface struct {
	Name      string
	GUID      string
	Signature string
	Funcs     []*Func
}

// Class represents a WinRT runtime class.
type Class struct {
	Name                string
	Signature           string
	RequiredImports     []*Import
	FullyQualifiedName  string
	ImplInterfaces      []*Interface
	ExclusiveInterfaces []*Interface
	HasEmptyConstructor bool
	IsAbstract          bool
}

// Delegate represents a WinRT delegate type.
type Delegate struct {
	Name        string
	GUID        string
	Signature   string
	InParams    []*Param
	ReturnParam *Param
}

// Enum represents a WinRT enumeration.
type Enum struct {
	Name      string
	Type      string
	Signature string
	Values    []*EnumValue
}

// EnumValue holds a name–value pair for an enum constant.
type EnumValue struct {
	Name  string
	Value string
}

// Func represents a WinRT method.
type Func struct {
	Name               string
	RequiredImports    []*Import
	Implement          bool
	FuncOwner          string
	InParams           []*Param
	ReturnParams       []*Param
	ExclusiveTo        string
	RequiresActivation bool
	InheritedFrom      QualifiedID
}

// QualifiedID holds namespace and name of a qualified element.
type QualifiedID struct {
	Namespace, Name string
}

// Param represents a method parameter or struct field.
type Param struct {
	CallerPackage string
	VarName       string
	Type          *ParamType
	IsOut         bool
}

// ParamType describes the type of a parameter.
type ParamType struct {
	Namespace string
	Name      string

	IsPointer          bool
	IsGeneric          bool
	IsArray            bool
	IsPrimitive        bool
	IsEnum             bool
	UnderlyingEnumType string

	DefaultValue DefaultValue
}

// DefaultValue holds the default value expression for a type.
type DefaultValue struct {
	Value       string
	IsPrimitive bool
}

// Import represents a dependency on another WinRT namespace.
type Import struct {
	Namespace, Name string
}

// Struct represents a WinRT value type (struct).
type Struct struct {
	Name      string
	Signature string
	Fields    []*Param
}

// GoVarName returns the Go-style variable name for a parameter.
func (p *Param) GoVarName() string {
	return toGoName(p.VarName, true)
}

// GoTypeName returns the Go type expression for a parameter.
func (p *Param) GoTypeName() string {
	if p.Type.IsPrimitive {
		return p.Type.Name
	}
	name := toGoName(p.Type.Name, true)
	pkg := typePackage(p.Type.Namespace)
	if p.CallerPackage != pkg {
		name = pkg + "." + name
	}
	return name
}

// GoDefaultValue returns the Go zero/default expression for a parameter's type.
func (p *Param) GoDefaultValue() string {
	if p.Type.DefaultValue.IsPrimitive {
		return p.Type.DefaultValue.Value
	}
	pkg := typePackage(p.Type.Namespace)
	if p.CallerPackage != pkg {
		return pkg + "." + p.Type.DefaultValue.Value
	}
	return p.Type.DefaultValue.Value
}

func toGoName(name string, public bool) string {
	if strings.Contains(name, "`") {
		name = strings.Split(name, "`")[0]
	}
	if !public {
		name = strings.ToLower(name[:1]) + name[1:]
	}
	return name
}

func typePackage(ns string) string {
	parts := strings.Split(ns, ".")
	return strings.ToLower(parts[len(parts)-1])
}
