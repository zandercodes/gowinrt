package mapgo

import (
	"strings"

	"github.com/zandercodes/gowinrt/internal/ir"
	"github.com/zandercodes/gowinrt/internal/resolve"
)

// GoImportPath converts an ir.Import to a Go import path string.
func GoImportPath(imp *ir.Import) string {
	if !strings.Contains(imp.Namespace, ".") && imp.Namespace != "Windows" {
		return imp.Namespace
	}
	return "github.com/zandercodes/gowinrt/" + resolve.TypeToFolder(imp.Namespace)
}

// ComputeImports resolves all required Go import paths for a Data block.
func ComputeImports(data *ir.Data, callerNS string) {
	imports := make([]*ir.Import, 0)
	for _, c := range data.Classes {
		imports = append(imports, c.RequiredImports...)
		for _, ei := range c.ExclusiveInterfaces {
			for _, f := range ei.Funcs {
				imports = append(imports, f.RequiredImports...)
			}
		}
	}
	for _, iface := range data.Interfaces {
		for _, f := range iface.Funcs {
			imports = append(imports, f.RequiredImports...)
		}
	}
	seen := map[string]bool{}
	for _, imp := range imports {
		if imp.Namespace == callerNS {
			continue
		}
		goImp := GoImportPath(imp)
		if !seen[goImp] {
			data.Imports = append(data.Imports, goImp)
			seen[goImp] = true
		}
	}
}
