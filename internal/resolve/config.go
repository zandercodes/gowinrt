package resolve

import (
	"fmt"
)

// Config holds the code generation settings.
type Config struct {
	Debug        bool
	Class        string
	ValidateOnly bool
	Inheritance  bool
	Filters      []string
}

// Validate returns an error if the config is invalid.
func (cfg *Config) Validate() error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if cfg.Class == "" {
		return fmt.Errorf("class name must not be empty")
	}
	return nil
}

// MethodFilter creates a Filter from the configured filter rules.
func (cfg *Config) MethodFilter() Filter {
	return NewFilter(cfg.Filters)
}
