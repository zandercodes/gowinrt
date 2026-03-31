package tests

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/zandercodes/gowinrt/internal/emit"
	"github.com/zandercodes/gowinrt/internal/gen"
	"github.com/zandercodes/gowinrt/internal/logger"
	"github.com/zandercodes/gowinrt/internal/metadata"
	"github.com/zandercodes/gowinrt/internal/resolve"
)

// helper runs the full pipeline: metadata → resolve → gen → emit (render to string).
func generate(t *testing.T, className string, filters []string, inheritance bool) string {
	t.Helper()

	log := logger.NewLoggerWithLevel(zerolog.Disabled)
	store, err := metadata.NewStore(log)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	td, err := store.TypeDefByName(className)
	if err != nil {
		t.Fatalf("TypeDefByName(%s): %v", className, err)
	}

	filter := resolve.NewFilter(filters)
	resolver := resolve.NewResolver(store, filter, inheritance, log)

	dataFiles, err := resolver.ResolveType(td)
	if err != nil {
		t.Fatalf("ResolveType(%s): %v", className, err)
	}
	if len(dataFiles) == 0 {
		t.Fatalf("ResolveType(%s) returned no files", className)
	}

	tmpl, err := gen.LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates: %v", err)
	}

	emitter := emit.NewEmitter(tmpl, false)
	out, err := emitter.RenderFile(dataFiles[0], td.Namespace.String())
	if err != nil {
		t.Fatalf("RenderFile(%s): %v", className, err)
	}

	return string(out)
}

// ---------- Enum tests ----------

func TestGenerate_Enum_AsyncStatus(t *testing.T) {
	src := generate(t, "Windows.Foundation.AsyncStatus", nil, false)

	mustContain(t, src,
		"package foundation",
		`type AsyncStatus int32`,
		`SignatureAsyncStatus`,
		`enum(Windows.Foundation.AsyncStatus;i4)`,
		"AsyncStatusStarted",
		"AsyncStatusCompleted",
		"AsyncStatusCanceled",
		"AsyncStatusError",
	)
}

func TestGenerate_Enum_BluetoothConnectionStatus(t *testing.T) {
	src := generate(t, "Windows.Devices.Bluetooth.BluetoothConnectionStatus", nil, false)

	mustContain(t, src,
		"package bluetooth",
		"type BluetoothConnectionStatus int32",
		"BluetoothConnectionStatusDisconnected",
		"BluetoothConnectionStatusConnected",
	)
}

func TestGenerate_Enum_BluetoothCacheMode(t *testing.T) {
	src := generate(t, "Windows.Devices.Bluetooth.BluetoothCacheMode", nil, false)

	mustContain(t, src,
		"type BluetoothCacheMode int32",
		"BluetoothCacheModeUncached",
		"BluetoothCacheModeCached",
	)
}

func TestGenerate_Enum_GattCharacteristicProperties(t *testing.T) {
	src := generate(t, "Windows.Devices.Bluetooth.GenericAttributeProfile.GattCharacteristicProperties", nil, false)

	mustContain(t, src,
		"package genericattributeprofile",
		"type GattCharacteristicProperties uint32",
		"GattCharacteristicPropertiesNone",
		"GattCharacteristicPropertiesRead",
		"GattCharacteristicPropertiesWrite",
		"GattCharacteristicPropertiesNotify",
	)
}

func TestGenerate_Enum_PropertyType(t *testing.T) {
	src := generate(t, "Windows.Foundation.PropertyType", nil, false)

	mustContain(t, src,
		"type PropertyType int32",
		"PropertyTypeEmpty",
		"PropertyTypeString",
		"PropertyTypeBoolean",
	)
}

// ---------- Struct tests ----------

func TestGenerate_Struct_DateTime(t *testing.T) {
	src := generate(t, "Windows.Foundation.DateTime", nil, false)

	mustContain(t, src,
		"package foundation",
		"type DateTime struct",
		"UniversalTime int64",
		`struct(Windows.Foundation.DateTime;i8)`,
	)
}

func TestGenerate_Struct_TimeSpan(t *testing.T) {
	src := generate(t, "Windows.Foundation.TimeSpan", nil, false)

	mustContain(t, src,
		"type TimeSpan struct",
		"Duration int64",
		`struct(Windows.Foundation.TimeSpan;i8)`,
	)
}

func TestGenerate_Struct_Point(t *testing.T) {
	src := generate(t, "Windows.Foundation.Point", nil, false)

	mustContain(t, src,
		"type Point struct",
		"X float32",
		"Y float32",
		`struct(Windows.Foundation.Point;f4;f4)`,
	)
}

func TestGenerate_Struct_Size(t *testing.T) {
	src := generate(t, "Windows.Foundation.Size", nil, false)

	mustContain(t, src,
		"type Size struct",
		"Width float32",
		"Height float32",
	)
}

func TestGenerate_Struct_Rect(t *testing.T) {
	src := generate(t, "Windows.Foundation.Rect", nil, false)

	mustContain(t, src,
		"type Rect struct",
		"X float32",
		"Y float32",
		"Width float32",
		"Height float32",
		`struct(Windows.Foundation.Rect;f4;f4;f4;f4)`,
	)
}

func TestGenerate_Struct_HResult(t *testing.T) {
	src := generate(t, "Windows.Foundation.HResult", nil, false)

	mustContain(t, src,
		"type HResult struct",
		"Value int32",
	)
}

// ---------- Interface tests ----------

func TestGenerate_Interface_IClosable(t *testing.T) {
	src := generate(t, "Windows.Foundation.IClosable", nil, false)

	mustContain(t, src,
		"package foundation",
		"type IClosable struct",
		"winrt.IInspectable",
		"type IClosableVtbl struct",
		"Close uintptr",
		"GUIDIClosable",
		"30d5a829-7fa4-4026-83bb-d75bae4ea99e",
	)
}

func TestGenerate_Interface_IAsyncInfo(t *testing.T) {
	src := generate(t, "Windows.Foundation.IAsyncInfo", nil, false)

	mustContain(t, src,
		"type IAsyncInfo struct",
		"winrt.IInspectable",
		"GUIDIAsyncInfo",
	)
}

// ---------- Delegate tests ----------

func TestGenerate_Delegate_DeferralCompletedHandler(t *testing.T) {
	src := generate(t, "Windows.Foundation.DeferralCompletedHandler", nil, false)

	mustContain(t, src,
		"package foundation",
		"type DeferralCompletedHandler struct",
		"GUIDDeferralCompletedHandler",
		"ed32a372-f3c8-4faa-9cfb-470148da3888",
		`delegate({ed32a372-f3c8-4faa-9cfb-470148da3888})`,
	)
}

// ---------- Class tests ----------

func TestGenerate_Class_Deferral(t *testing.T) {
	src := generate(t, "Windows.Foundation.Deferral", nil, false)

	mustContain(t, src,
		"package foundation",
		"type Deferral struct",
		"winrt.IInspectable",
	)
}

func TestGenerate_Class_BluetoothLEDevice_WithFilter(t *testing.T) {
	filters := []string{
		"FromBluetoothAddressAsync",
		"get_ConnectionStatus",
		"!*",
	}
	src := generate(t, "Windows.Devices.Bluetooth.BluetoothLEDevice", filters, false)

	mustContain(t, src,
		"package bluetooth",
		"type BluetoothLEDevice struct",
		"winrt.IInspectable",
		"FromBluetoothAddressAsync",
		"GetConnectionStatus",
	)
}

func TestGenerate_Class_BluetoothDeviceId(t *testing.T) {
	filters := []string{"!FromId"}
	src := generate(t, "Windows.Devices.Bluetooth.BluetoothDeviceId", filters, false)

	mustContain(t, src,
		"type BluetoothDeviceId struct",
	)
}

// ---------- Parameterized (generic) types ----------

func TestGenerate_Interface_IAsyncOperation(t *testing.T) {
	src := generate(t, "Windows.Foundation.IAsyncOperation`1", nil, false)

	mustContain(t, src,
		"package foundation",
		"type IAsyncOperation struct",
		"winrt.IInspectable",
		"GUIDIAsyncOperation",
	)
}

// ---------- Regression: output is valid Go ----------

func TestGenerate_OutputIsValidGo(t *testing.T) {
	classes := []string{
		"Windows.Foundation.AsyncStatus",
		"Windows.Foundation.DateTime",
		"Windows.Foundation.TimeSpan",
		"Windows.Foundation.Point",
		"Windows.Foundation.Size",
		"Windows.Foundation.Rect",
		"Windows.Foundation.HResult",
		"Windows.Foundation.IClosable",
		"Windows.Foundation.IAsyncInfo",
		"Windows.Foundation.DeferralCompletedHandler",
		"Windows.Foundation.Deferral",
		"Windows.Foundation.PropertyType",
		"Windows.Devices.Bluetooth.BluetoothConnectionStatus",
		"Windows.Devices.Bluetooth.BluetoothCacheMode",
		"Windows.Devices.Bluetooth.BluetoothAddressType",
		"Windows.Devices.Bluetooth.BluetoothError",
		"Windows.Devices.Bluetooth.GenericAttributeProfile.GattCommunicationStatus",
		"Windows.Devices.Bluetooth.GenericAttributeProfile.GattCharacteristicProperties",
		"Windows.Devices.Bluetooth.GenericAttributeProfile.GattWriteOption",
		"Windows.Devices.Bluetooth.GenericAttributeProfile.GattProtectionLevel",
	}

	for _, cls := range classes {
		t.Run(cls, func(t *testing.T) {
			src := generate(t, cls, nil, false)
			if len(src) == 0 {
				t.Errorf("empty output for %s", cls)
			}
			// generate() already runs go/format + goimports — if those succeed,
			// the output is valid Go syntax.
		})
	}
}

// ---------- helper ----------

func mustContain(t *testing.T, src string, patterns ...string) {
	t.Helper()
	for _, p := range patterns {
		if !strings.Contains(src, p) {
			t.Errorf("output missing expected substring %q\n--- output ---\n%s", p, src)
		}
	}
}
