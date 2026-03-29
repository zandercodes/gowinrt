package gen

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

// ---------- templates ----------

//go:embed templates/*
var templatesFS embed.FS

func loadTemplates() (*template.Template, error) {
	return template.New("").
		Funcs(templateFuncs()).
		ParseFS(templatesFS, "templates/*")
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"funcName": funcName,
		"concat": func(a, b []*genParam) []*genParam {
			return append(a, b...)
		},
		"toLower": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
}

// funcName generates the Go-style function name from a genFunc.
func funcName(m genFunc) string {
	replacer := strings.NewReplacer(
		"get_", "Get",
		"put_", "Set",
		"add_", "Add",
		"remove_", "Remove",
	)
	name := replacer.Replace(m.Name)

	prefix := ""
	if m.ExclusiveTo != "" && m.RequiresActivation {
		parts := strings.Split(m.ExclusiveTo, ".")
		prefix = toGoName(parts[len(parts)-1], true)
	}
	return prefix + name
}
