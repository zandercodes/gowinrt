//go:build windows

package tests

import (
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/zandercodes/gowinrt/windows/foundation"
	"github.com/zandercodes/gowinrt/winrt"
)

// ---------- Enum value tests ----------

func TestEnumValues_AsyncStatus(t *testing.T) {
	tests := []struct {
		name string
		val  foundation.AsyncStatus
		want int32
	}{
		{"Started", foundation.AsyncStatusStarted, 0},
		{"Completed", foundation.AsyncStatusCompleted, 1},
		{"Canceled", foundation.AsyncStatusCanceled, 2},
		{"Error", foundation.AsyncStatusError, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.val) != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.val, tt.want)
			}
		})
	}
}

func TestEnumValues_PropertyType(t *testing.T) {
	tests := []struct {
		name string
		val  foundation.PropertyType
		want int32
	}{
		{"Empty", foundation.PropertyTypeEmpty, 0},
		{"UInt8", foundation.PropertyTypeUInt8, 1},
		{"Int32", foundation.PropertyTypeInt32, 4},
		{"Boolean", foundation.PropertyTypeBoolean, 11},
		{"String", foundation.PropertyTypeString, 12},
		{"DateTime", foundation.PropertyTypeDateTime, 14},
		{"Guid", foundation.PropertyTypeGuid, 16},
		{"UInt8Array", foundation.PropertyTypeUInt8Array, 1025},
		{"StringArray", foundation.PropertyTypeStringArray, 1036},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int32(tt.val) != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.val, tt.want)
			}
		})
	}
}

// ---------- Struct layout tests ----------

func TestStructLayout_DateTime(t *testing.T) {
	dt := foundation.DateTime{UniversalTime: 132800000000000000}
	if dt.UniversalTime != 132800000000000000 {
		t.Errorf("UniversalTime = %d, want 132800000000000000", dt.UniversalTime)
	}
	if s := unsafe.Sizeof(dt); s != 8 {
		t.Errorf("sizeof(DateTime) = %d, want 8", s)
	}
}

func TestStructLayout_TimeSpan(t *testing.T) {
	ts := foundation.TimeSpan{Duration: 10000000} // 1 second in 100-ns ticks
	if ts.Duration != 10000000 {
		t.Errorf("Duration = %d, want 10000000", ts.Duration)
	}
	if s := unsafe.Sizeof(ts); s != 8 {
		t.Errorf("sizeof(TimeSpan) = %d, want 8", s)
	}
}

func TestStructLayout_Point(t *testing.T) {
	p := foundation.Point{X: 1.5, Y: 2.5}
	if p.X != 1.5 || p.Y != 2.5 {
		t.Errorf("Point = {%f, %f}, want {1.5, 2.5}", p.X, p.Y)
	}
	if s := unsafe.Sizeof(p); s != 8 {
		t.Errorf("sizeof(Point) = %d, want 8", s)
	}
}

func TestStructLayout_Size(t *testing.T) {
	sz := foundation.Size{Width: 1920, Height: 1080}
	if sz.Width != 1920 || sz.Height != 1080 {
		t.Errorf("Size = {%f, %f}, want {1920, 1080}", sz.Width, sz.Height)
	}
	if s := unsafe.Sizeof(sz); s != 8 {
		t.Errorf("sizeof(Size) = %d, want 8", s)
	}
}

func TestStructLayout_Rect(t *testing.T) {
	r := foundation.Rect{X: 10, Y: 20, Width: 100, Height: 200}
	if r.X != 10 || r.Y != 20 || r.Width != 100 || r.Height != 200 {
		t.Errorf("Rect fields incorrect: %+v", r)
	}
	if s := unsafe.Sizeof(r); s != 16 {
		t.Errorf("sizeof(Rect) = %d, want 16", s)
	}
}

func TestStructLayout_HResult(t *testing.T) {
	hr := foundation.HResult{Value: -2147024809} // E_INVALIDARG
	if hr.Value != -2147024809 {
		t.Errorf("Value = %d, want -2147024809", hr.Value)
	}
	if s := unsafe.Sizeof(hr); s != 4 {
		t.Errorf("sizeof(HResult) = %d, want 4", s)
	}
}

func TestStructLayout_EventRegistrationToken(t *testing.T) {
	tok := foundation.EventRegistrationToken{Value: 42}
	if tok.Value != 42 {
		t.Errorf("Value = %d, want 42", tok.Value)
	}
	if s := unsafe.Sizeof(tok); s != 8 {
		t.Errorf("sizeof(EventRegistrationToken) = %d, want 8", s)
	}
}

// ---------- Signature & GUID constant tests ----------

func TestSignatureConstants(t *testing.T) {
	tests := []struct {
		name, got, want string
	}{
		{"SignatureAsyncStatus", foundation.SignatureAsyncStatus, "enum(Windows.Foundation.AsyncStatus;i4)"},
		{"SignatureDateTime", foundation.SignatureDateTime, "struct(Windows.Foundation.DateTime;i8)"},
		{"SignatureTimeSpan", foundation.SignatureTimeSpan, "struct(Windows.Foundation.TimeSpan;i8)"},
		{"SignaturePoint", foundation.SignaturePoint, "struct(Windows.Foundation.Point;f4;f4)"},
		{"SignatureSize", foundation.SignatureSize, "struct(Windows.Foundation.Size;f4;f4)"},
		{"SignatureRect", foundation.SignatureRect, "struct(Windows.Foundation.Rect;f4;f4;f4;f4)"},
		{"SignatureHResult", foundation.SignatureHResult, "struct(Windows.Foundation.HResult;i4)"},
		{"SignatureEventRegistrationToken", foundation.SignatureEventRegistrationToken, "struct(Windows.Foundation.EventRegistrationToken;i8)"},
		{"SignaturePropertyType", foundation.SignaturePropertyType, "enum(Windows.Foundation.PropertyType;i4)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestGUIDConstants(t *testing.T) {
	tests := []struct {
		name, got, want string
	}{
		{"GUIDIClosable", foundation.GUIDIClosable, "30d5a829-7fa4-4026-83bb-d75bae4ea99e"},
		{"GUIDIPropertyValue", foundation.GUIDIPropertyValue, "4bd682dd-7554-40e9-9a9b-82654ede7e62"},
		{"GUIDIAsyncInfo", foundation.GUIDIAsyncInfo, "00000036-0000-0000-c000-000000000046"},
		{"GUIDDeferralCompletedHandler", foundation.GUIDDeferralCompletedHandler, "ed32a372-f3c8-4faa-9cfb-470148da3888"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}
}

// ---------- COM runtime tests ----------

func TestDeferral_CreateCompleteClose(t *testing.T) {
	var mu sync.Mutex
	callbackCalled := false

	iid := winrt.NewGUID(foundation.GUIDDeferralCompletedHandler)
	handler := foundation.NewDeferralCompletedHandler(iid, func(instance *foundation.DeferralCompletedHandler) {
		mu.Lock()
		defer mu.Unlock()
		callbackCalled = true
	})

	deferral, err := foundation.DeferralCreate(handler)
	if err != nil {
		t.Fatalf("DeferralCreate: %v", err)
	}

	if err := deferral.Complete(); err != nil {
		t.Fatalf("Deferral.Complete: %v", err)
	}

	// Give a moment for the callback to fire (it's dispatched synchronously in WinRT,
	// but be safe).
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	called := callbackCalled
	mu.Unlock()

	if !called {
		t.Error("DeferralCompletedHandler callback was not invoked after Complete()")
	}

	if err := deferral.Close(); err != nil {
		t.Fatalf("Deferral.Close: %v", err)
	}
}
