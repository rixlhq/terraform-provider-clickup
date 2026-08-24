package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

var assocPathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// buildPath replaces {param} placeholders in path with values from the
// Terraform plan/state object.
func (r *associationResource) buildPath(path string, v tftypes.Value) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid Value", err.Error())
		return "", diags
	}

	out := path
	for _, match := range assocPathParamRegex.FindAllStringSubmatch(path, -1) {
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
		out = strings.Replace(out, match[0], s, 1)
	}

	return out, diags
}

// buildBody converts the Terraform plan to a JSON body, excluding path
// parameters (they appear in the URL, not the body). The bodyKeyMap renames
// Terraform attribute names to API body key names when they differ.
func (r *associationResource) buildBody(ctx context.Context, v tftypes.Value) ([]byte, diag.Diagnostics) {
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

	// Collect path parameter names to exclude from body.
	pathParams := map[string]bool{}
	for _, match := range assocPathParamRegex.FindAllStringSubmatch(r.createPath, -1) {
		pathParams[clickupcommon.TerraformIdentifier(match[1])] = true
	}

	body := make(map[string]any, len(all))
	for k, val := range all {
		if pathParams[k] {
			continue
		}
		// Rename Terraform attr to API body key if a mapping exists.
		apiKey := k
		if mapped, ok := r.bodyKeyMap[k]; ok && mapped != "" {
			apiKey = mapped
		}
		body[apiKey] = val
	}

	out, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Body Encode Error", err.Error())
		return nil, diags
	}
	return out, diags
}
