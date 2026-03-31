package emit

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"golang.org/x/tools/imports"

	"github.com/zandercodes/gowinrt/internal/ir"
	"github.com/zandercodes/gowinrt/internal/mapgo"
)

// Emitter writes generated Go source files to disk.
type Emitter struct {
	validateOnly bool
	tmpl         *template.Template
}

// NewEmitter creates a new Emitter. If validateOnly is true, files are checked
// against existing content instead of being written.
func NewEmitter(tmpl *template.Template, validateOnly bool) *Emitter {
	return &Emitter{
		validateOnly: validateOnly,
		tmpl:         tmpl,
	}
}

// RenderFile renders a DataFile to formatted Go source bytes without writing to disk.
func (e *Emitter) RenderFile(f *ir.DataFile, callerNS string) ([]byte, error) {
	mapgo.ComputeImports(&f.Data, callerNS)

	var buf bytes.Buffer
	if err := e.tmpl.ExecuteTemplate(&buf, "file.tmpl", f.Data); err != nil {
		return nil, fmt.Errorf("failed to execute template for %s: %w", f.Filename, err)
	}

	processed, err := imports.Process(f.Filename, buf.Bytes(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to process imports for %s: %w", f.Filename, err)
	}

	formatted, err := format.Source(processed)
	if err != nil {
		return nil, fmt.Errorf("failed to format source for %s: %w", f.Filename, err)
	}

	return formatted, nil
}

// EmitFile renders a DataFile to disk (or validates it).
func (e *Emitter) EmitFile(f *ir.DataFile, callerNS string) error {
	formatted, err := e.RenderFile(f, callerNS)
	if err != nil {
		return err
	}

	if e.validateOnly {
		existing, err := os.ReadFile(f.Filename)
		if err != nil {
			return err
		}
		if string(existing) != string(formatted) {
			return fmt.Errorf("file %s does not match generated content", f.Filename)
		}
		return nil
	}

	dir := filepath.Dir(f.Filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(f.Filename, formatted, 0644)
}
