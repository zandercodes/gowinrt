package gen

import "strings"

// ---------- naming helpers ----------

func typeToFolder(ns string) string {
	return strings.ToLower(strings.ReplaceAll(ns, ".", "/"))
}

func typePackage(ns string) string {
	parts := strings.Split(ns, ".")
	return strings.ToLower(parts[len(parts)-1])
}

func enumValueName(typeName, valueName string) string {
	return typeName + valueName
}

func toGoName(name string, public bool) string {
	if isParameterized(name) {
		name = strings.Split(name, "`")[0]
	}
	if !public {
		name = strings.ToLower(name[:1]) + name[1:]
	}
	return name
}

func isParameterized(name string) bool {
	return strings.Contains(name, "`")
}

func typeFilename(typeName string) string {
	return strings.ToLower(toGoName(typeName, true))
}

func cleanReservedWords(name string) string {
	switch name {
	case "type":
		return "mType"
	}
	return name
}
