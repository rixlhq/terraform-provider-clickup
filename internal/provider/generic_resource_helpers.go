package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

func (r *genericResource) idFromValue(v tftypes.Value) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid State", err.Error())
		return "", diags
	}
	param := r.idParam()
	paramVal, ok := obj[param]
	if !ok {
		diags.AddError("Missing Resource ID", fmt.Sprintf("%q is required", param))
		return "", diags
	}
	s, err := valueAsString(paramVal)
	if err != nil {
		diags.AddError("Invalid Resource ID", err.Error())
		return "", diags
	}
	return s, diags
}

func (r *genericResource) idParam() string {
	if r.idField != "" {
		return r.idField
	}
	for _, path := range []string{r.deletePath, r.updatePath, r.readPath} {
		for _, match := range pathParamRegex.FindAllStringSubmatch(path, -1) {
			return match[1]
		}
	}
	return "id"
}

func (r *genericResource) buildPath(path string, v tftypes.Value) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid Value", err.Error())
		return "", diags
	}

	out := path
	for _, match := range pathParamRegex.FindAllStringSubmatch(path, -1) {
		param := match[1]
		normParam := clickupcommon.TerraformIdentifier(param)
		paramVal, ok := obj[normParam]
		if !ok {
			diags.AddError("Missing Path Parameter", fmt.Sprintf("%q is required", normParam))
			return "", diags
		}
		s, err := valueAsString(paramVal)
		if err != nil {
			diags.AddError("Invalid Path Parameter", fmt.Sprintf("%q: %s", normParam, err))
			return "", diags
		}
		out = strings.Replace(out, match[0], url.PathEscape(s), 1)
	}

	return out, diags
}

func (r *genericResource) buildBody(ctx context.Context, v tftypes.Value, fields []string, defaults map[string]any, transforms map[string]func(any) any) ([]byte, diag.Diagnostics) {
	var diags diag.Diagnostics
	j, err := clickupcommon.TfValueToJSON(ctx, v)
	if err != nil {
		diags.AddError("Request Body Error", err.Error())
		return nil, diags
	}

	all, ok := j.(map[string]any)
	if !ok {
		diags.AddError("Request Body Error", "plan value is not an object")
		return nil, diags
	}

	body := make(map[string]any, len(fields))
	for _, f := range fields {
		val, ok := all[f]
		if !ok || val == nil {
			if d, ok := defaults[f]; ok {
				val = d
			}
		}
		if val == nil {
			// Field is missing and has no default; omit it from the body.
			continue
		}
		if transform, ok := transforms[f]; ok && transform != nil {
			val = transform(val)
		}
		body[f] = val
	}

	raw, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Request Body Error", err.Error())
		return nil, diags
	}
	return raw, diags
}

// buildUpdateBody creates the request body for an update. It uses the merged
// state+plan value for known fields, but omits any update body field that is
// unknown or null in the plan. This prevents sending unchanged computed values
// to incremental update endpoints while still preserving nested computed
// sub-attributes (e.g. the "rem" side of an add/rem object) when a field is
// changed.
func (r *genericResource) buildUpdateBody(ctx context.Context, merged, plan tftypes.Value, fields []string, defaults map[string]any, transforms map[string]func(any) any) ([]byte, diag.Diagnostics) {
	var diags diag.Diagnostics
	bodyValue, err := r.updateBodyValue(ctx, merged, plan, fields)
	if err != nil {
		diags.AddError("Update Body Error", err.Error())
		return nil, diags
	}
	return r.buildBody(ctx, bodyValue, fields, defaults, transforms)
}

// updateBodyValue returns a tftypes.Value that contains the merged value for
// every update body field that is known in the plan, and null for update body
// fields that are unknown or null in the plan. Non-body attributes are left as
// they are in the merged value.
func (r *genericResource) updateBodyValue(_ context.Context, merged, plan tftypes.Value, fields []string) (tftypes.Value, error) {
	if !merged.IsKnown() || merged.IsNull() {
		return merged, nil
	}

	typ, ok := merged.Type().(tftypes.Object)
	if !ok {
		return merged, nil
	}

	mergedObj, err := asObject(merged)
	if err != nil {
		return merged, err
	}

	planObj := map[string]tftypes.Value{}
	if plan.IsKnown() && !plan.IsNull() {
		if m, err := asObject(plan); err == nil {
			planObj = m
		}
	}

	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[f] = true
	}

	vals := make(map[string]tftypes.Value, len(typ.AttributeTypes))
	for attr, attrType := range typ.AttributeTypes {
		planVal, ok := planObj[attr]
		if !ok {
			planVal = tftypes.NewValue(attrType, nil)
		}
		mergedVal, ok := mergedObj[attr]
		if !ok {
			mergedVal = tftypes.NewValue(attrType, nil)
		}

		if fieldSet[attr] && (!planVal.IsKnown() || planVal.IsNull()) {
			// The practitioner did not configure this field for the update.
			vals[attr] = tftypes.NewValue(attrType, nil)
		} else {
			vals[attr] = mergedVal
		}
	}

	return tftypes.NewValue(merged.Type(), vals), nil
}

func (r *genericResource) tfType(ctx context.Context) tftypes.Type {
	return r.schema(ctx).Type().TerraformType(ctx)
}
