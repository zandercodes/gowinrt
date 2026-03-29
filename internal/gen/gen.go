package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"

	winmd "github.com/microsoft/go-winmd/winmd"
	"golang.org/x/tools/imports"

	"github.com/zandercodes/gowinrt/internal/logger"
	mdStore "github.com/zandercodes/gowinrt/internal/winmd"
)

const invokeMethodName = "Invoke"

// tdWindowsRuntime indicates that a type is a WinRT type (0x4000 flag)
// https://docs.microsoft.com/en-us/uwp/winrt-cref/winmd-files#runtime-classes
const tdWindowsRuntime = 0x4000

// Primitive type signatures used by WinRT type system.
const (
	SignatureUInt8   = "u1"
	SignatureUInt16  = "u2"
	SignatureUInt32  = "u4"
	SignatureUInt64  = "u8"
	SignatureInt8    = "i1"
	SignatureInt16   = "i2"
	SignatureInt32   = "i4"
	SignatureInt64   = "i8"
	SignatureFloat32 = "f4"
	SignatureFloat64 = "f8"
	SignatureBool    = "b1"
	SignatureChar    = "c2"
	SignatureString  = "string"
	SignatureGUID    = "g16"
)

type generator struct {
	class        string
	validateOnly bool
	inheritance  bool
	filter       Filter
	logger       logger.Log
	dataFiles    []*genDataFile
	store        *mdStore.Store
}

// Generate generates Go bindings for the WinRT class described by cfg.
func Generate(cfg *Config, log logger.Log) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	store, err := mdStore.NewStore(log)
	if err != nil {
		return err
	}

	g := &generator{
		class:        cfg.Class,
		validateOnly: cfg.ValidateOnly,
		inheritance:  cfg.Inheritance,
		filter:       cfg.MethodFilter(),
		logger:       log,
		store:        store,
	}
	return g.run()
}

func (g *generator) run() error {
	g.logger.Debug().Str("class", g.class).Msg("starting code generation")

	typeDef, err := g.store.TypeDefByName(g.class)
	if err != nil {
		return fmt.Errorf("failed to get typedef for class %s: %w", g.class, err)
	}
	return g.generate(typeDef)
}

func (g *generator) generate(td *mdStore.TypeDef) error {
	if td.Flags&tdWindowsRuntime == 0 {
		return fmt.Errorf("%s.%s is not a WinRT type", td.Namespace.String(), td.Name.String())
	}

	if err := g.loadCodeGenData(td); err != nil {
		return fmt.Errorf("failed to load codegen data for %s.%s: %w", td.Namespace.String(), td.Name.String(), err)
	}

	for _, f := range g.dataFiles {
		if err := g.generateFile(f, td); err != nil {
			return fmt.Errorf("failed to generate file %s: %w", f.Filename, err)
		}
	}
	return nil
}

func (g *generator) generateFile(f *genDataFile, td *mdStore.TypeDef) error {
	tmpl, err := loadTemplates()
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	f.Data.computeImports(td.Namespace.String())

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "file.tmpl", f.Data); err != nil {
		return fmt.Errorf("failed to execute template for %s: %w", f.Filename, err)
	}

	processed, err := imports.Process(f.Filename, buf.Bytes(), nil)
	if err != nil {
		return fmt.Errorf("failed to process imports for %s: %w", f.Filename, err)
	}

	formatted, err := format.Source(processed)
	if err != nil {
		return fmt.Errorf("failed to format source for %s: %w", f.Filename, err)
	}

	if g.validateOnly {
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

func (g *generator) loadCodeGenData(td *mdStore.TypeDef) error {
	f := g.addFile(td)

	switch {
	case td.IsInterface():
		g.logger.Info().Str("interface", td.Namespace.String()+"."+td.Name.String()).Msg("generating interface")
		if err := g.validateInterface(td); err != nil {
			return err
		}
		iface, err := g.createGenInterface(td, false)
		if err != nil {
			return err
		}
		f.Data.Interfaces = append(f.Data.Interfaces, iface)

	case td.IsEnum():
		g.logger.Info().Str("enum", td.Namespace.String()+"."+td.Name.String()).Msg("generating enum")
		enum, err := g.createGenEnum(td)
		if err != nil {
			return err
		}
		f.Data.Enums = append(f.Data.Enums, enum)

	case td.IsStruct():
		g.logger.Info().Str("struct", td.Namespace.String()+"."+td.Name.String()).Msg("generating struct")
		s, err := g.createGenStruct(td)
		if err != nil {
			return err
		}
		f.Data.Structs = append(f.Data.Structs, s)

	case td.IsDelegate():
		g.logger.Info().Str("delegate", td.Namespace.String()+"."+td.Name.String()).Msg("generating delegate")
		d, err := g.createGenDelegate(td)
		if err != nil {
			return err
		}
		f.Data.Delegates = append(f.Data.Delegates, d)

	default:
		g.logger.Info().Str("class", td.Namespace.String()+"."+td.Name.String()).Msg("generating class")
		cls, err := g.createGenClass(td)
		if err != nil {
			return err
		}
		f.Data.Classes = append(f.Data.Classes, cls)
	}

	return nil
}

func (g *generator) addFile(td *mdStore.TypeDef) *genDataFile {
	folder := typeToFolder(td.Namespace.String())
	filename := folder + "/" + typeFilename(td.Name.String()) + ".go"
	f := &genDataFile{
		Filename: filename,
		Data: genData{
			Package: typePackage(td.Namespace.String()),
		},
	}
	g.dataFiles = append(g.dataFiles, f)
	return f
}

func (g *generator) validateInterface(td *mdStore.TypeDef) error {
	if td.Flags&winmd.TypeAttributes_VisibilityMask == winmd.TypeAttributes_NotPublic {
		return fmt.Errorf("interface %s.%s is not public", td.Namespace.String(), td.Name.String())
	}
	return nil
}
