package provider

import (
	"context"
	"fmt"

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
	client           *clickupclient.Client
	name             string
	createPath       string
	readPath         string
	updatePath       string
	deletePath       string
	updateMethod     string
	createBodyFields []string
	updateBodyFields []string
	schemaFunc       func(context.Context) resource_schema.Schema
	schema           resource_schema.Schema
}

func (r *genericResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.name
}

func (r *genericResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.schema = r.schemaFunc(ctx)
	resp.Schema = r.schema
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

	body, diags := r.buildBody(ctx, plan, r.createBodyFields)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	raw, err := r.client.Post(ctx, createPath, body)
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	id, err := r.extractID(raw)
	if err != nil {
		resp.Diagnostics.AddError("Create Response Error", err.Error())
		return
	}

	final, diags := r.readModel(ctx, plan, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Raw = final
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

	resp.State.Raw = final
}

func (r *genericResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	plan := req.Plan.Raw
	updatePath, diags := r.buildPath(r.updatePath, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, plan, r.updateBodyFields)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var raw []byte
	var err error
	if r.updateMethod == "put" {
		raw, err = r.client.Put(ctx, updatePath, body)
	} else {
		raw, err = r.client.Patch(ctx, updatePath, body)
	}
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}
	_ = raw

	id, diags := r.idFromValue(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	final, diags := r.readModel(ctx, plan, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.State.Raw = final
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
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
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

	raw, err := r.client.Get(ctx, readPath, nil)
	if err != nil {
		if clickupclient.IsNotFound(err) {
			diags.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for %s %s", r.name, id))
			return tftypes.Value{}, diags
		}
		diags.AddError("ClickUp API Error", err.Error())
		return tftypes.Value{}, diags
	}

	jv, err := clickupcommon.DecodeJSONResponse(raw)
	if err != nil {
		diags.AddError("Response Decode Error", err.Error())
		return tftypes.Value{}, diags
	}

	tfVal, err := clickupcommon.JSONToTfValue(ctx, r.tfType(ctx), jv)
	if err != nil {
		diags.AddError("State Conversion Error", err.Error())
		return tftypes.Value{}, diags
	}

	final, d := r.withPathParam(ctx, tfVal, param, id)
	diags.Append(d...)
	if diags.HasError() {
		return tftypes.Value{}, diags
	}

	return final, diags
}
