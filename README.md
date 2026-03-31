# gowinrt

A Go code generator for [Windows Runtime (WinRT)](https://learn.microsoft.com/en-us/uwp/winrt-cref/winrt-type-system) APIs. Generate type-safe Go bindings from WinRT metadata (`.winmd`) files and call Windows APIs directly from Go — no CGo required.

> **Built on [`go-winmd`](https://github.com/microsoft/go-winmd)** — Microsoft's official Go library for reading ECMA-335 / WinRT metadata.

## Features

- **Full WinRT type support** — interfaces, classes, enums, structs, and delegates
- **Interface inheritance** — inherited methods are generated as convenient wrappers using `QueryInterface`
- **Generic types** — custom ECMA-335 signature parser handles `GENERICINST`, `VAR`, `MVAR`, `SZARRAY`, `BYREF`, and `ARRAY`
- **Method filtering** — generate only the methods you need with include/exclude patterns
- **`go generate` friendly** — designed to run as a `//go:generate` directive
- **Pure Go** — uses `syscall.SyscallN` with a built-in `winrt` package, no CGo or external COM dependencies
- **Parameterized GUIDs** — runtime helper to compute GUIDs for generic WinRT types (RFC 4122 v5)
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
  -i, --inheritance              Include inherited interface methods as wrappers
  -V, --validate                 Validate generated files match without writing
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
winrt/                WinRT runtime types (IUnknown, IInspectable, GUID, HString, 
│                       parameterized GUID computation, type signature constants)
tests/                All tests (generation, runtime, GUID)
internal/
├─ cli/              Cobra command setup
├─ delegate/         Delegate callback registration
├─ emit/             Template rendering & file output
├─ gen/              Template engine & template files
├─ ir/               Intermediate representation types
├─ kernel32/         Heap allocation for delegate VTables
├─ logger/           Logging setup
├─ mapgo/            Go import path computation
├─ metadata/         WinRT metadata store (wraps go-winmd)
└─ resolve/          Type resolution, naming, filtering, signature parsing
windows/              Generated WinRT bindings
```

### Parameterized GUIDs

Generic WinRT types (e.g. `IAsyncOperation<T>`) don't have a fixed GUID — it must be computed at runtime from the base GUID and the type arguments' signatures. The `winrt` package implements this per the [WinRT type system spec](https://docs.microsoft.com/en-us/uwp/winrt-cref/winrt-type-system#guid-generation-for-parameterized-types):

```go
import "github.com/zandercodes/gowinrt/winrt"

// TypedEventHandler<Watcher, ReceivedEventArgs>
guid := winrt.ParameterizedInstanceGUID(
    "9de1c534-6ae1-11e0-84e1-18a905bcc53f",
    "rc(Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementWatcher;{a6ac336f-f3d3-4297-8d6c-c81ea6623f40})",
    "rc(Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementReceivedEventArgs;{27987ddf-e596-41be-8d43-9e6731d4a913})",
)
// guid == "{90EB4ECA-D465-5EA0-A61C-033C8C5ECEF2}"
```

### Signature Parser

WinRT metadata encodes method signatures as ECMA-335 blobs. The built-in `go-winmd` parser does not yet support generic instantiations (`GENERICINST`). `gowinrt` includes a custom signature parser ([`sigparse.go`](internal/metadata/sigparse.go)) that handles:

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

    "github.com/zandercodes/gowinrt/winrt"
)

const GUIDIClosable string = "30d5a829-7fa4-4026-83bb-d75bae4ea99c"
const SignatureIClosable string = "{30d5a829-7fa4-4026-83bb-d75bae4ea99c}"

type IClosable struct {
    winrt.IInspectable
}

type IClosableVtbl struct {
    winrt.IInspectableVtbl
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
        return winrt.NewError(hr)
    }
    return nil
}
```

## Known Limitations

- **Receive-array out-parameters** — WinRT methods that use the "receive array" pattern (`BYREF SZARRAY` with `Out` flag, e.g. `IPropertyValue.GetUInt8Array`) are not yet implemented. The VTable entries are generated to preserve correct interface layout, but no Go wrapper is emitted. The generator logs a warning for each skipped method.

## Dependencies

| Package | Purpose |
|---------|---------|
| [`go-winmd`](https://github.com/microsoft/go-winmd) | ECMA-335 metadata reader |
| [`cobra`](https://github.com/spf13/cobra) | CLI framework |
| [`zerolog`](https://github.com/rs/zerolog) | Structured logging |
| [`x/tools`](https://pkg.go.dev/golang.org/x/tools) | `goimports` for import cleanup |
| [`x/sys`](https://pkg.go.dev/golang.org/x/sys) | Windows DLL loading (`combase.dll`) |

## Credits

This project based on the excellent work of [winrt-go](https://github.com/saltosystems/winrt-go) by Salto Systems, reimplemented with Microsoft's official `go-winmd` metadata library.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).

Portions derived from [winrt-go](https://github.com/saltosystems/winrt-go) by Salto Systems (MIT License).
