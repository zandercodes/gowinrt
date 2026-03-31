package resolve

// Filter controls which methods get full implementations vs stubs.
type Filter struct {
	Methods []string
}

// NewFilter creates a Filter from a list of method patterns.
func NewFilter(methods []string) Filter {
	return Filter{
		Methods: methods,
	}
}

// Matches returns true if the given method name passes the filter.
func (f *Filter) Matches(methodName string) bool {
	for _, m := range f.Methods {
		ok := true
		if len(m) > 0 && m[0] == '!' {
			m, ok = m[1:], false
		}
		if m == "*" || m == methodName {
			return ok
		}
	}
	return true
}
