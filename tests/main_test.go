//go:build windows

package tests

import (
	"os"
	"testing"

	"github.com/zandercodes/gowinrt/winrt"
)

func TestMain(m *testing.M) {
	if err := winrt.RoInitialize(1); err != nil {
		panic("RoInitialize failed: " + err.Error())
	}
	os.Exit(m.Run())
}
