package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

var _ resource.ResourceWithImportState = &genericResource{}

// ImportState imports an existing generic resource into Terraform state using its identifier.
// For resources with composite paths (e.g. space_id:tag_name), the ID can be provided
// as colon-separated components matching the required path parameters.
func (r *genericResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	fullType, ok := r.tfType(ctx).(tftypes.Object)
	if !ok {
		resp.Diagnostics.AddError("Import Error", "resource schema is not an object")
		return
	}

	vals := make(map[string]tftypes.Value, len(fullType.AttributeTypes))
	for k, at := range fullType.AttributeTypes {
		vals[k] = tftypes.NewValue(at, nil)
	}

	if strings.Contains(req.ID, ":") {
		if r.populateCompositeID(ctx, fullType, vals, req.ID, resp) {
			resp.State.Raw = tftypes.NewValue(fullType, vals)
			return
		}
	}

	r.populateSingleID(ctx, fullType, vals, req.ID, resp)
	if !resp.Diagnostics.HasError() {
		resp.State.Raw = tftypes.NewValue(fullType, vals)
	}
}

// populateCompositeID attempts to match colon-separated ID components against path parameters.
func (r *genericResource) populateCompositeID(ctx context.Context, fullType tftypes.Object, vals map[string]tftypes.Value, rawID string, resp *resource.ImportStateResponse) bool {
	parts := strings.Split(rawID, ":")
	for _, pathTmpl := range []string{r.updatePath, r.deletePath, r.readPath} {
		matches := pathParamRegex.FindAllStringSubmatch(pathTmpl, -1)
		if len(matches) == len(parts) {
			for i, match := range matches {
				attrName := clickupcommon.TerraformIdentifier(match[1])
				if at, ok := fullType.AttributeTypes[attrName]; ok {
					newVal, err := clickupcommon.JSONToTfValue(ctx, at, parts[i])
					if err != nil {
						resp.Diagnostics.AddError("ID Conversion Error", err.Error())
						return true
					}
					vals[attrName] = newVal
				}
			}
			return true
		}
	}

	if len(parts) == 2 {
		return r.populateParentChildID(ctx, fullType, vals, parts)
	}
	return false
}

// populateParentChildID sets parent path parameter and child resource ID for 2-part imports.
func (r *genericResource) populateParentChildID(ctx context.Context, fullType tftypes.Object, vals map[string]tftypes.Value, parts []string) bool {
	readMatches := pathParamRegex.FindAllStringSubmatch(r.readPath, -1)
	if len(readMatches) != 1 {
		return false
	}
	parentAttr := clickupcommon.TerraformIdentifier(readMatches[0][1])
	idAttr := r.idParam()
	if pType, ok := fullType.AttributeTypes[parentAttr]; ok {
		if pVal, err := clickupcommon.JSONToTfValue(ctx, pType, parts[0]); err == nil {
			vals[parentAttr] = pVal
		}
	}
	if iType, ok := fullType.AttributeTypes[idAttr]; ok {
		if iVal, err := clickupcommon.JSONToTfValue(ctx, iType, parts[1]); err == nil {
			vals[idAttr] = iVal
		}
	}
	return true
}

// populateSingleID assigns the import ID to the resource's primary identifier attribute.
func (r *genericResource) populateSingleID(ctx context.Context, fullType tftypes.Object, vals map[string]tftypes.Value, rawID string, resp *resource.ImportStateResponse) {
	param := r.idParam()
	attrType, ok := fullType.AttributeTypes[param]
	if !ok {
		if at, hasID := fullType.AttributeTypes["id"]; hasID {
			param = "id"
			attrType = at
		} else {
			resp.Diagnostics.AddError("Import Error", fmt.Sprintf("cannot find identifier attribute %q in schema", param))
			return
		}
	}

	newVal, err := clickupcommon.JSONToTfValue(ctx, attrType, rawID)
	if err != nil {
		resp.Diagnostics.AddError("ID Conversion Error", err.Error())
		return
	}
	vals[param] = newVal

	if param != "id" {
		if idType, hasID := fullType.AttributeTypes["id"]; hasID {
			if idVal, err := clickupcommon.JSONToTfValue(ctx, idType, rawID); err == nil {
				vals["id"] = idVal
			}
		}
	}
}
