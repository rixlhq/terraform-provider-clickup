package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resource_schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

// genericResource implements resource.Resource for any ClickUp v2 endpoint with a
// full create/read/update/delete cycle. It uses the generated Terraform schema and
// the API paths configured in generator_config.yml.
type genericResource struct {
	client               *clickupclient.Client
	name                 string
	createPath           string
	readPath             string
	updatePath           string
	deletePath           string
	updateMethod         string
	createBodyFields     []string
	updateBodyFields     []string
	createBodyDefaults   map[string]any
	updateBodyDefaults   map[string]any
	createBodyTransforms map[string]func(any) any
	updateBodyTransforms map[string]func(any) any
	readTransforms       map[string]func(any) any
	preReadTransforms    map[string]func(any) any
	// readQueryParams maps query parameter names to Terraform attribute names
	// that must be appended to GET requests used by readModel.
	readQueryParams map[string]string
	schemaFunc      func(context.Context) resource_schema.Schema
	// readFromList is set when the API has no single-GET endpoint. The resource
	// reads the collection at readPath and filters the item whose
	// readListIDField equals the resource ID.
	readFromList    bool
	readListRoot    string
	readListIDField string
	// createResponseRoot is the JSON key that wraps the created object in the
	// response (e.g. "view", "key_result"). Empty means the object is top-level.
	createResponseRoot string
	// createResponseItemArray is the JSON key inside createResponseRoot that
	// contains an array of items. When set, extractID takes the last element
	// of this array and reads its "id" field. Used for endpoints like
	// checklist_item where the response is { "checklist": { "items": [...] } }.
	createResponseItemArray string
	// idFromBody is a path of map keys used to extract the resource ID from
	// the request body when the API response is empty or does not contain an id.
	// This is used for both create and update. Example: ["tag", "name"] for space tags.
	idFromBody []string
	// readResponseRoot is the JSON key that wraps the read response. When the
	// response is a list item it is wrapped *under* this root. When the response
	// is a single-GET that already contains this root, the root is unwrapped.
	readResponseRoot string
	// idField is the Terraform attribute name that holds the resource ID. When
	// empty it is derived from the first path parameter of updatePath/deletePath
	// or readPath.
	idField string
}

func (r *genericResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.name
}

func (r *genericResource) schema(ctx context.Context) resource_schema.Schema {
	return r.addMissingPathParamAttributes(r.schemaFunc(ctx))
}

func (r *genericResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema(ctx)
}
func (r *genericResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	r.client = pd.ClickUp
}

func (r *genericResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	plan := req.Plan.Raw
	createPath, diags := r.buildPath(r.createPath, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, plan, r.createBodyFields, r.createBodyDefaults, r.createBodyTransforms)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.Post(ctx, createPath, body)
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	id, err := r.resolveResourceID(raw, body)
	if err != nil {
		resp.Diagnostics.AddError("Create Response Error", err.Error())
		return
	}

	// Persist the ID immediately so a failed follow-up read leaves a
	// recoverable state instead of an orphan.
	param := r.idParam()
	planWithID, d := r.withPathParam(ctx, plan, param, id)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.Raw = planWithID

	final, diags := r.readModel(ctx, planWithID, id)
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

func (r *genericResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	state := req.State.Raw
	id, diags := r.idFromValue(state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	final, diags := r.readModel(ctx, state, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, d := range diags {
		if d.Severity() == diag.SeverityWarning && d.Summary() == "Not Found" {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	resp.State.Raw = final
}

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

// queryParams extracts the configured query parameters from a Terraform value.
// It is used for resource read endpoints that require query parameters such as
// the user group list endpoint, which requires a team_id query.
func (r *genericResource) queryParams(_ context.Context, v tftypes.Value) (url.Values, diag.Diagnostics) {
	var diags diag.Diagnostics
	query := url.Values{}
	if len(r.readQueryParams) == 0 {
		return query, diags
	}

	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid State", err.Error())
		return nil, diags
	}

	for q, attr := range r.readQueryParams {
		val, ok := obj[attr]
		if !ok {
			diags.AddError("Missing Query Parameter", fmt.Sprintf("%q is required", attr))
			return nil, diags
		}
		s, err := valueAsString(val)
		if err != nil {
			diags.AddError("Invalid Query Parameter", fmt.Sprintf("%q: %s", attr, err))
			return nil, diags
		}
		if s == "" {
			diags.AddError("Invalid Query Parameter", fmt.Sprintf("%q must not be empty", attr))
			return nil, diags
		}
		query.Set(q, s)
	}

	return query, diags
}

func (r *genericResource) getAndDecode(ctx context.Context, readPath, id string, query url.Values) (any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !r.readFromList {
		raw, err := r.client.Get(ctx, readPath, query)
		if err != nil {
			if clickupclient.IsNotFound(err) {
				diags.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for %s %s", r.name, id))
				return nil, diags
			}
			diags.AddError("ClickUp API Error", err.Error())
			return nil, diags
		}

		jv, err := clickupcommon.DecodeJSONResponse(raw)
		if err != nil {
			diags.AddError("Response Decode Error", err.Error())
			return nil, diags
		}
		return jv, diags
	}

	pageQuery := cloneURLValues(query)
	seen := make(map[string]bool)
	for {
		raw, err := r.client.Get(ctx, readPath, pageQuery)
		if err != nil {
			if clickupclient.IsNotFound(err) {
				diags.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for %s %s", r.name, id))
				return nil, diags
			}
			diags.AddError("ClickUp API Error", err.Error())
			return nil, diags
		}

		jv, err := clickupcommon.DecodeJSONResponse(raw)
		if err != nil {
			diags.AddError("Response Decode Error", err.Error())
			return nil, diags
		}

		item, err := r.findInList(jv, id)
		if err == nil {
			return item, diags
		}
		var notFound *notFoundError
		if !errors.As(err, &notFound) {
			diags.AddError("Read Error", err.Error())
			return nil, diags
		}

		next, ok := r.nextListPageQuery(jv, query, seen)
		if !ok {
			diags.AddWarning("Not Found", notFound.message)
			return nil, diags
		}
		pageQuery = next
	}
}
