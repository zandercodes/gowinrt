/**
 * File: methoddef.go
 * Project: winmd
 * Created Date: 2026‑03‑29T00:14:27.2727+01:00
 * Author: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Last Modified: 2026‑03‑29T00:16:39.3939+01:00
 * Modified By: ZanderCodes (Julian Zander) <admin@zandercodes.com>
 *
 * Copyright © 2026 ZanderCodes (Julian Zander). All rights reserved.
 */

package winmd

import (
	winmd "github.com/microsoft/go-winmd/winmd"
)

// GetMethodOverloadName finds and returns the overload attribute for the given method.
func GetMethodOverloadName(ctx *winmd.Metadata, methodDef *winmd.MethodDef) string {
	for i := range ctx.Tables.CustomAttribute.Indices() {
		cAttr, err := ctx.Tables.CustomAttribute.At(i)
		if err != nil {
			continue
		}

		// Parent: The owner of the Attribute must be the given func
		if cAttr.Parent.Tag != winmd.HasCustomAttribute_MethodDef {
			continue
		}

		parentMethodDef, err := ctx.Tables.MethodDef.At(cAttr.Parent.Index)
		if err != nil {
			continue
		}

		// does the attribute belong to the method we're looking for?
		if parentMethodDef.Name.String() != methodDef.Name.String() ||
			string(parentMethodDef.Signature) != string(methodDef.Signature) {
			continue
		}

		// Type: the attribute type must be the given type
		// the cAttr.Type can be either a MemberRef or a MethodDef.
		// Since we are looking for a type, we will only consider the MemberRef.
		if cAttr.Type.Tag != winmd.CustomAttributeType_MemberRef {
			continue
		}

		attrTypeMemberRef, err := ctx.Tables.MemberRef.At(cAttr.Type.Index)
		if err != nil {
			continue
		}

		// we need to check the MemberRef Class
		// the value can belong to several tables, but we are only going to check for TypeRef
		if attrTypeMemberRef.Class.Tag != winmd.MemberRefParent_TypeRef {
			continue
		}

		attrTypeRef, err := ctx.Tables.TypeRef.At(attrTypeMemberRef.Class.Index)
		if err != nil {
			continue
		}

		if attrTypeRef.Namespace.String()+"."+attrTypeRef.Name.String() == AttributeTypeOverloadAttribute {
			// Metadata values start with 0x01 0x00 and ends with 0x00 0x00
			mdVal := cAttr.Value[2 : len(cAttr.Value)-2]
			// the next value is the length of the string
			mdVal = mdVal[1:]
			return string(mdVal)
		}
	}
	return methodDef.Name.String()
}
