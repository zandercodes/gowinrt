package metadata

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

		if cAttr.Parent.Tag != winmd.HasCustomAttribute_MethodDef {
			continue
		}

		parentMethodDef, err := ctx.Tables.MethodDef.At(cAttr.Parent.Index)
		if err != nil {
			continue
		}

		if parentMethodDef.Name.String() != methodDef.Name.String() ||
			string(parentMethodDef.Signature) != string(methodDef.Signature) {
			continue
		}

		if cAttr.Type.Tag != winmd.CustomAttributeType_MemberRef {
			continue
		}

		attrTypeMemberRef, err := ctx.Tables.MemberRef.At(cAttr.Type.Index)
		if err != nil {
			continue
		}

		if attrTypeMemberRef.Class.Tag != winmd.MemberRefParent_TypeRef {
			continue
		}

		attrTypeRef, err := ctx.Tables.TypeRef.At(attrTypeMemberRef.Class.Index)
		if err != nil {
			continue
		}

		if attrTypeRef.Namespace.String()+"."+attrTypeRef.Name.String() == AttributeTypeOverloadAttribute {
			mdVal := cAttr.Value[2 : len(cAttr.Value)-2]
			mdVal = mdVal[1:]
			return string(mdVal)
		}
	}
	return methodDef.Name.String()
}
