/**
 * File: filter.go
 * Project: gen
 * Created Date: 2026‑03‑28T22:10:39.3939+01:00
 * Author: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Last Modified: 2026‑03‑28T22:20:57.5757+01:00
 * Modified By: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Copyright © 2026 ZanderCodes (Julian Zander). All rights reserved.
 */

package gen

type Filter struct {
	Methods []string
}

func NewFilter(methods []string) Filter {
	return Filter{
		Methods: methods,
	}
}

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
