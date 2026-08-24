package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

// associationResource manages a ClickUp API association (link/relationship)
// that is created via POST and removed via DELETE with no update endpoint.
// All configurable attributes use RequiresReplace so Terraform recreates the
// association instead of calling Update. Read returns the state as-is since
// these endpoints have no single-GET; drift is detected on the next plan.
type associationResource struct {
	client            *clickupclient.Client
	name              string
	createPath        string
	deletePath        string
	bodyKeyMap        map[string]string // terraform attr -> API body key (for renames like depends_on_id -> depends_on)
	deleteQueryParams []string          // terraform attr names to send as query params on DELETE
	schemaFunc        func(context.Context) schema.Schema
}

func (r *associationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.name
}

func (r *associationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schemaFunc(ctx)
}

func (r *associationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *associationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token to use this resource.")
		return
	}

	plan := req.Plan.Raw
	createPath, diags := r.buildPath(r.createPath, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Post(ctx, createPath, body); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *associationResource) Read(_ context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Association resources have no single-GET endpoint. Return state as-is.
	resp.State.Raw = req.State.Raw
}

func (r *associationResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Never called — all attributes use RequiresReplace.
}

func (r *associationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token to use this resource.")
		return
	}

	state := req.State.Raw
	deletePath, diags := r.buildPath(r.deletePath, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build query params for DELETE (e.g. task_dependency needs
	// depends_on and dependency_of as query params, not path params).
	var query url.Values
	if len(r.deleteQueryParams) > 0 {
		obj, err := asObject(state)
		if err != nil {
			resp.Diagnostics.AddError("Invalid State", err.Error())
			return
		}
		query = url.Values{}
		for _, attr := range r.deleteQueryParams {
			apiKey := attr
			if mapped, ok := r.bodyKeyMap[attr]; ok && mapped != "" {
				apiKey = mapped
			}
			if v, ok := obj[attr]; ok {
				if s, err := valueAsString(v); err == nil {
					query.Set(apiKey, s)
				}
			}
		}
	}

	var deleteErr error
	if query != nil {
		_, deleteErr = r.client.Delete(ctx, deletePath, query)
	} else {
		_, deleteErr = r.client.Delete(ctx, deletePath)
	}
	if deleteErr != nil {
		if !clickupclient.IsNotFound(deleteErr) {
			resp.Diagnostics.AddError("ClickUp API Error", deleteErr.Error())
		}
	}
}

// requiresReplaceString is a helper for association resource schemas.
func requiresReplaceString() schema.StringAttribute {
	return schema.StringAttribute{
		Required: true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}
