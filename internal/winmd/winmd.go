/**
 * File: winmd.go
 * Project: winmd
 * Created Date: 2026‑03‑28T22:58:00.000+01:00
 * Author: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Last Modified: 2026‑03‑29T00:17:11.1111+01:00
 * Modified By: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Copyright © 2026 ZanderCodes (Julian Zander). All rights reserved.
 */

package winmd

import (
	"bytes"
	"debug/pe"
	"embed"
	"io"
	"io/fs"

	winmd "github.com/microsoft/go-winmd/winmd"
)

// Custom Attributes
const (
	AttributeTypeGUID                 = "Windows.Foundation.Metadata.GuidAttribute"
	AttributeTypeExclusiveTo          = "Windows.Foundation.Metadata.ExclusiveToAttribute"
	AttributeTypeStaticAttribute      = "Windows.Foundation.Metadata.StaticAttribute"
	AttributeTypeActivatableAttribute = "Windows.Foundation.Metadata.ActivatableAttribute"
	AttributeTypeDefaultAttribute     = "Windows.Foundation.Metadata.DefaultAttribute"
	AttributeTypeOverloadAttribute    = "Windows.Foundation.Metadata.OverloadAttribute"
)

// HasContext is a helper struct that holds the original context of a metadata element.
type HasContext struct {
	originalCtx *winmd.Metadata
}

// Ctx returns the original context of the element.
func (hctx *HasContext) Ctx() *winmd.Metadata {
	return hctx.originalCtx
}

//go:embed metadata/*.winmd
var files embed.FS

func allFiles() ([]fs.DirEntry, error) {
	return files.ReadDir("metadata")
}

func open(path string) (*pe.File, error) {
	file, err := files.Open("metadata/" + path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	pef, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return pef, nil
}
