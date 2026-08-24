package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resource_schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
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
