package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

func (r *genericResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	// Computed id attributes are unknown in the plan for updates. Merge prior
	// state into the plan so path parameters and identifiers are known, while
	// keeping any configuration changes from the plan.
	merged, err := mergeTfValues(req.State.Raw, req.Plan.Raw)
	if err != nil {
		resp.Diagnostics.AddError("State Merge Error", err.Error())
		return
	}

	updatePath, diags := r.buildPath(r.updatePath, merged)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only send update fields that are configured in the plan. Unknown fields
	// in the plan are omitted from the request body, but known fields are
	// filled from the merged state+plan value so nested computed values (such
	// as the "rem" side of an add/rem object) are preserved.
	body, diags := r.buildUpdateBody(ctx, merged, req.Plan.Raw, r.updateBodyFields, r.updateBodyDefaults, r.updateBodyTransforms)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var raw []byte
	if r.updateMethod == "put" {
		raw, err = r.client.Put(ctx, updatePath, body)
	} else {
		raw, err = r.client.Patch(ctx, updatePath, body)
	}
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	id, diags := r.idFromValue(merged)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Some updates (e.g. space tags) change the resource's identifier. Try to
	// resolve the new ID from the response or request body, then fall back to
	// the original ID.
	newID := id
	if resolved, err := r.resolveResourceID(raw, body); err == nil && resolved != "" {
		newID = resolved
	}

	// Persist the updated identifier immediately so a failed read leaves a
	// recoverable state.
	param := r.idParam()
	mergedWithID, d := r.withPathParam(ctx, merged, param, newID)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Raw = mergedWithID

	final, diags := r.readModel(ctx, mergedWithID, newID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, d := range diags {
		if d.Severity() == diag.SeverityWarning && d.Summary() == "Not Found" {
			return
		}
	}

	if final.Type() != nil {
		resp.State.Raw = final
	}
}

func (r *genericResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	state := req.State.Raw
	deletePath, diags := r.buildPath(r.deletePath, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Delete(ctx, deletePath); err != nil {
		if !clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddError("ClickUp API Error", err.Error())
			return
		}
	}
}

func (r *genericResource) readModel(ctx context.Context, base tftypes.Value, id string) (tftypes.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	param := r.idParam()

	baseWithParam, d := r.withPathParam(ctx, base, param, id)
	diags.Append(d...)
	if diags.HasError() {
		return tftypes.Value{}, diags
	}

	readPath, d := r.buildPath(r.readPath, baseWithParam)
	diags.Append(d...)
	if diags.HasError() {
		return tftypes.Value{}, diags
	}

	query, d := r.queryParams(ctx, baseWithParam)
	diags.Append(d...)
	if diags.HasError() {
		return tftypes.Value{}, diags
	}

	jv, d := r.getAndDecode(ctx, readPath, id, query)
	diags.Append(d...)
	if diags.HasError() {
		return tftypes.Value{}, diags
	}
	if jv == nil {
		return tftypes.Value{}, diags
	}

	jv = r.unwrapReadResponse(jv)

	if m, ok := jv.(map[string]any); ok {
		r.applyTransforms(m, r.preReadTransforms)
		r.applyTransforms(m, r.readTransforms)
	}

	// Use the unknown-missing converter so that fields the API omits are
	// treated as unknown and merged from state, while explicit API nulls are
	// preserved and overwrite state values.
	tfVal, err := clickupcommon.JSONToTfValueWithUnknownMissing(ctx, r.tfType(ctx), jv)
	if err != nil {
		diags.AddError("State Conversion Error", err.Error())
		return tftypes.Value{}, diags
	}

	merged, err := mergeTfValues(baseWithParam, tfVal)
	if err != nil {
		diags.AddError("State Merge Error", err.Error())
		return tftypes.Value{}, diags
	}

	final, d := r.withPathParam(ctx, merged, param, id)
	diags.Append(d...)
	if diags.HasError() {
		return tftypes.Value{}, diags
	}

	return final, diags
}

// unwrapReadResponse handles readResponseRoot. For list reads it wraps the item
// under the root. For single-GET reads it unwraps the item if the response is
// already wrapped under the root.
func (r *genericResource) unwrapReadResponse(jv any) any {
	if r.readResponseRoot == "" {
		return jv
	}

	if r.readFromList {
		return map[string]any{r.readResponseRoot: jv}
	}

	if m, ok := jv.(map[string]any); ok {
		if v, ok := m[r.readResponseRoot]; ok {
			return v
		}
	}

	return jv
}

// applyTransforms runs the configured value transforms on the decoded JSON map.
func (r *genericResource) applyTransforms(m map[string]any, transforms map[string]func(any) any) {
	for key, transform := range transforms {
		if transform == nil {
			continue
		}
		if v, ok := m[key]; ok {
			m[key] = transform(v)
		}
	}
}
