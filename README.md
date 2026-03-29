# gowinrt

A Go code generator for [Windows Runtime (WinRT)](https://learn.microsoft.com/en-us/windows/uwp/winrt-cref/winrt-type-system) APIs. Generate type-safe Go bindings from WinRT metadata (`.winmd`) files and call Windows APIs directly from Go — no CGo required.

> **Built on [`go-winmd`](https://github.com/microsoft/go-winmd)** — Microsoft's official Go library for reading ECMA-335 / WinRT metadata.

## Features

- **Full WinRT type support** — interfaces, classes, enums, structs, and delegates
- **Interface inheritance** — inherited methods are generated as convenient wrappers using `QueryInterface`
- **Generic types** — custom ECMA-335 signature parser handles `GENERICINST`, `VAR`, `MVAR`, `SZARRAY`, `BYREF`, and `ARRAY`
- **Method filtering** — generate only the methods you need with include/exclude patterns
- **`go generate` friendly** — designed to run as a `//go:generate` directive
- **Pure Go** — uses `syscall.SyscallN` via [go-ole](https://github.com/go-ole/go-ole), no CGo dependency
- **Formatted output** — generated code is automatically processed by `goimports` and `go/format`

## Installation

```bash
go install github.com/zandercodes/gowinrt/cmd/gowinrt@latest
```

## Quick Start

Generate bindings for a WinRT class:

```bash
gowinrt -c Windows.Foundation.Deferral
```

With inheritance (adds wrapper methods for parent interfaces):

```bash
gowinrt -c "Windows.Foundation.IAsyncOperation`1" --inheritance
```

With method filtering:

```bash
gowinrt -c Windows.Storage.Streams.DataReader \
  -f FromBuffer -f ReadBytes -f '!*'
```

### Using `go generate`

Add directives to any Go file:

```go
//go:generate gowinrt -c Windows.Foundation.IClosable
//go:generate gowinrt -c "Windows.Foundation.IAsyncOperation`1" --inheritance
//go:generate gowinrt -c Windows.Foundation.Deferral
//go:generate gowinrt -c Windows.Foundation.Collections.IVector`1
```

Then run:

```bash
go generate ./...
```

## CLI Reference

```
Usage:
  gowinrt [flags]

Flags:
  -c, --class string             WinRT class to generate (e.g. Windows.Foundation.Uri)
  -f, --method-filter strings    Filter methods (e.g. -f GetResults -f '!*')
      --inheritance              Include inherited interface methods as wrappers
      --validate                 Validate generated files match without writing
  -v, --verbose                  Enable debug output
  -h, --help                     Show help
```

### Method Filter Syntax

| Pattern | Meaning |
|---------|---------|
| `MethodName` | Include this method |
| `!MethodName` | Exclude this method |
| `!*` | Exclude all remaining (use after includes) |

Methods are evaluated in order. Use `-f GetResults -f Close -f '!*'` to generate only `GetResults` and `Close`.

## Supported WinRT Types

| Type | Template | Description |
|------|----------|-------------|
| **Interface** | `interface.tmpl` | VTable struct, method implementations, inherited wrappers |
| **Class** | `class.tmpl` | Runtime class with constructor, implemented + exclusive interfaces |
| **Enum** | `enum.tmpl` | Typed integer constants |
| **Struct** | `struct.tmpl` | Value types with fields |
| **Delegate** | `delegate.tmpl` | Callback types with VTable and ref counting |

## Architecture

```
cmd/gowinrt/          CLI entry point
internal/
├── cli/              Cobra command setup
├── gen/
│   ├── gen.go            Main generation pipeline
│   ├── gen_types.go      Type creators (interface, class, enum, struct, delegate)
│   ├── gen_methods.go    Method & parameter helpers
│   ├── gen_resolve.go    Field/method resolution, element type mapping
│   ├── gen_signature.go  WinRT type signature generation
│   ├── sigparse.go       Custom ECMA-335 signature blob parser
│   ├── types.go          Template data types (genData, genFunc, genParam, …)
│   ├── templates.go      Template engine (embed, load, custom functions)
│   ├── naming.go         Naming helpers (toGoName, typePackage, …)
│   ├── config.go         Configuration
│   ├── filter.go         Method filter logic
│   └── templates/        Go text/template files (11 templates)
├── winmd/            WinRT metadata store (wraps go-winmd)
├── kernel32/         Heap allocation for delegate VTables
└── delegate/         Delegate callback registration
```

### Signature Parser

WinRT metadata encodes method signatures as ECMA-335 blobs. The built-in `go-winmd` parser does not yet support generic instantiations (`GENERICINST`). `gowinrt` includes a custom signature parser ([`sigparse.go`](internal/gen/sigparse.go)) that handles:

- `GENERICINST` — Generic type instantiations (e.g. `IAsyncOperation<T>`)
- `VAR` / `MVAR` — Generic type/method parameters → mapped to `unsafe.Pointer`
- `SZARRAY` — Single-dimensional arrays
- `BYREF` — Pass-by-reference parameters
- `ARRAY` — Multi-dimensional arrays

### Template System

Templates use Go's `text/template` with custom functions:

| Function | Description |
|----------|-------------|
| `funcName` | Converts WinRT method names (`get_X` → `GetX`, `put_X` → `SetX`) |
| `concat` | Merges parameter slices for iteration |
| `dict` | Creates a map from key-value pairs for template data passing |
| `toLower` | Lowercases first character |

## Example Output

Running `gowinrt -c Windows.Foundation.IClosable` generates:

```go
// Code generated by gowinrt. DO NOT EDIT.

//go:build windows

package foundation

import (
    "syscall"
    "unsafe"

    "github.com/go-ole/go-ole"
)

const GUIDIClosable string = "30d5a829-7fa4-4026-83bb-d75bae4ea99c"
const SignatureIClosable string = "{30d5a829-7fa4-4026-83bb-d75bae4ea99c}"

type IClosable struct {
    ole.IInspectable
}

type IClosableVtbl struct {
    ole.IInspectableVtbl
    Close uintptr
}

func (v *IClosable) VTable() *IClosableVtbl {
    return (*IClosableVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *IClosable) Close() error {
    hr, _, _ := syscall.SyscallN(
        v.VTable().Close,
        uintptr(unsafe.Pointer(v)),
    )
    if hr != 0 {
        return ole.NewError(hr)
    }
    return nil
}
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [`go-winmd`](https://github.com/microsoft/go-winmd) | ECMA-335 metadata reader |
| [`go-ole`](https://github.com/go-ole/go-ole) | COM/OLE interop (IUnknown, IInspectable, HString) |
| [`cobra`](https://github.com/spf13/cobra) | CLI framework |
| [`zerolog`](https://github.com/rs/zerolog) | Structured logging |
| [`x/tools`](https://pkg.go.dev/golang.org/x/tools) | `goimports` for import cleanup |

## Credits

This project builds on the excellent work of [winrt-go](https://github.com/saltosystems/winrt-go) by Salto Systems, reimplemented with Microsoft's official `go-winmd` metadata library.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).

Portions derived from [winrt-go](https://github.com/saltosystems/winrt-go) by Salto Systems (MIT License).
