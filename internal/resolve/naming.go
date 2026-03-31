package resolve

import "strings"

// TypeToFolder converts a WinRT namespace to a filesystem path.
func TypeToFolder(ns string) string {
	return strings.ToLower(strings.ReplaceAll(ns, ".", "/"))
}

// TypePackage returns the Go package name for a WinRT namespace.
func TypePackage(ns string) string {
	parts := strings.Split(ns, ".")
	return strings.ToLower(parts[len(parts)-1])
}

// EnumValueName generates the Go constant name for an enum value.
func EnumValueName(typeName, valueName string) string {
	return typeName + valueName
}

// ToGoName converts a WinRT name to a Go-style name.
func ToGoName(name string, public bool) string {
	if IsParameterized(name) {
		name = strings.Split(name, "`")[0]
	}
	if !public {
		name = strings.ToLower(name[:1]) + name[1:]
	}
	return name
}

// IsParameterized returns true if the type name contains a generic arity indicator.
func IsParameterized(name string) bool {
	return strings.Contains(name, "`")
}

// TypeFilename returns the lowercase filename for a type.
func TypeFilename(typeName string) string {
	return strings.ToLower(ToGoName(typeName, true))
}

// CleanReservedWords replaces Go reserved words in identifiers.
func CleanReservedWords(name string) string {
	switch name {
	case "type":
		return "mType"
	}
	return name
}
