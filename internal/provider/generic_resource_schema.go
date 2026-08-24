package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	resource_schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

func (r *genericResource) addMissingPathParamAttributes(s resource_schema.Schema) resource_schema.Schema {
	if r.createPath == "" || s.Attributes == nil {
		return s
	}
	for _, match := range pathParamRegex.FindAllStringSubmatch(r.createPath, -1) {
		param := clickupcommon.TerraformIdentifier(match[1])
		if attr, ok := s.Attributes[param]; ok {
			// The attribute already exists but it identifies the collection where
			// the object is created, so changes must trigger replacement.
			s.Attributes[param] = withRequiresReplace(attr)
			continue
		}
		// Default to a Required StringAttribute so the path parameter accepts
		// numeric and non-numeric IDs and is converted to a string when building
		// the API path.
		s.Attributes[param] = resource_schema.StringAttribute{
			Required:            true,
			MarkdownDescription: fmt.Sprintf("ClickUp %s path parameter required to create this %s.", param, r.name),
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		}
	}
	return s
}

func withRequiresReplace(attr resource_schema.Attribute) resource_schema.Attribute {
	switch a := attr.(type) {
	case resource_schema.StringAttribute:
		a.PlanModifiers = append(a.PlanModifiers, stringplanmodifier.RequiresReplace())
		return a
	case resource_schema.Int64Attribute:
		a.PlanModifiers = append(a.PlanModifiers, int64planmodifier.RequiresReplace())
		return a
	case resource_schema.NumberAttribute:
		a.PlanModifiers = append(a.PlanModifiers, numberplanmodifier.RequiresReplace())
		return a
	}
	return attr
}

func (r *genericResource) withPathParam(ctx context.Context, v tftypes.Value, param, id string) (tftypes.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid Value", err.Error())
		return tftypes.Value{}, diags
	}

	fullType, ok := r.tfType(ctx).(tftypes.Object)
	if !ok {
		diags.AddError("Invalid Value", "expected object type")
		return tftypes.Value{}, diags
	}

	attrType, ok := fullType.AttributeTypes[param]
	if !ok {
		diags.AddError("Invalid Resource Schema", fmt.Sprintf("path parameter %q is not an attribute", param))
		return tftypes.Value{}, diags
	}

	newVal, err := clickupcommon.JSONToTfValue(ctx, attrType, id)
	if err != nil {
		diags.AddError("ID Conversion Error", err.Error())
		return tftypes.Value{}, diags
	}

	vals := make(map[string]tftypes.Value, len(fullType.AttributeTypes))
	for k, at := range fullType.AttributeTypes {
		if ov, ok := obj[k]; ok {
			vals[k] = ov
		} else {
			vals[k] = tftypes.NewValue(at, tftypes.UnknownValue)
		}
	}
	vals[param] = newVal

	return tftypes.NewValue(fullType, vals), diags
}
