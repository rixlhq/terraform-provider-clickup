package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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
	for _, match := range pathParamRegex.FindAllStringSubmatch(r.readPath, -1) {
		return match[1]
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
		paramVal, ok := obj[param]
		if !ok {
			diags.AddError("Missing Path Parameter", fmt.Sprintf("%q is required", param))
			return "", diags
		}
		s, err := valueAsString(paramVal)
		if err != nil {
			diags.AddError("Invalid Path Parameter", fmt.Sprintf("%q: %s", param, err))
			return "", diags
		}
		out = strings.Replace(out, match[0], s, 1)
	}

	return out, diags
}

func (r *genericResource) buildBody(ctx context.Context, v tftypes.Value, fields []string) ([]byte, diag.Diagnostics) {
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
		if val, ok := all[f]; ok {
			body[f] = val
		}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Request Body Error", err.Error())
		return nil, diags
	}
	return raw, diags
}

func (r *genericResource) tfType(ctx context.Context) tftypes.Type {
	return r.schema.Type().TerraformType(ctx)
}

func (r *genericResource) withPathParam(ctx context.Context, v tftypes.Value, param, id string) (tftypes.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid Value", err.Error())
		return tftypes.Value{}, diags
	}

	objType, ok := v.Type().(tftypes.Object)
	if !ok {
		diags.AddError("Invalid Value", "expected object type")
		return tftypes.Value{}, diags
	}

	attrType, ok := objType.AttributeTypes[param]
	if !ok {
		diags.AddError("Invalid Resource Schema", fmt.Sprintf("path parameter %q is not an attribute", param))
		return tftypes.Value{}, diags
	}

	newVal, err := clickupcommon.JSONToTfValue(ctx, attrType, id)
	if err != nil {
		diags.AddError("ID Conversion Error", err.Error())
		return tftypes.Value{}, diags
	}

	vals := make(map[string]tftypes.Value, len(obj))
	maps.Copy(vals, obj)
	vals[param] = newVal

	return tftypes.NewValue(objType, vals), diags
}

func (r *genericResource) extractID(raw []byte) (string, error) {
	v, err := clickupcommon.DecodeJSONResponse(raw)
	if err != nil {
		return "", err
	}

	data, ok := v.(map[string]any)
	if !ok {
		return "", errors.New("create response is not a JSON object")
	}

	if id, ok := data["id"]; ok {
		return valueToIDString(id), nil
	}
	if d, ok := data["data"].(map[string]any); ok {
		if id, ok := d["id"]; ok {
			return valueToIDString(id), nil
		}
	}

	return "", errors.New("create response did not contain an id")
}

func valueToIDString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return fmt.Sprintf("%g", x)
	case int, int32, int64:
		return fmt.Sprintf("%d", x)
	}
	return fmt.Sprint(v)
}
