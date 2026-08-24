package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

// associationResource manages a ClickUp API association (link/relationship)
// that is created via POST and removed via DELETE with no update endpoint.
// All configurable attributes use RequiresReplace so Terraform recreates the
// association instead of calling Update. Read returns the state as-is since
// these endpoints have no single-GET; drift is detected on the next plan.
type associationResource struct {
	client     *clickupclient.Client
	name       string
	createPath string
	deletePath string
	createBody map[string]string // terraform attr -> json body key (empty = use attr name)
	schemaFunc func(context.Context) schema.Schema
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

	var raw map[string]types.String
	resp.Diagnostics.Append(req.Plan.Get(ctx, &raw)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := buildAssociationPath(r.createPath, raw)
	body := buildAssociationBody(raw, r.createBody)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Body Encode Error", err.Error())
		return
	}

	if _, err := r.client.Post(ctx, path, bodyBytes); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, raw)...)
}

func (r *associationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Association resources have no single-GET endpoint. Return state as-is.
	// Drift is detected when the parent resource is read or on next plan.
	var state map[string]types.String
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *associationResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Never called — all attributes use RequiresReplace.
}

func (r *associationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token to use this resource.")
		return
	}

	var state map[string]types.String
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := buildAssociationPath(r.deletePath, state)
	if _, err := r.client.Delete(ctx, path); err != nil {
		if !clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		}
	}
}

// buildAssociationPath replaces {param} placeholders with values from the
// Terraform state/plan map.
func buildAssociationPath(path string, vals map[string]types.String) string {
	for attr, v := range vals {
		placeholder := "{" + attr + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, v.ValueString())
		}
	}
	return path
}

// buildAssociationBody constructs the JSON request body from the Terraform
// values that are NOT path parameters.
func buildAssociationBody(vals map[string]types.String, bodyKeyMap map[string]string) map[string]any {
	body := map[string]any{}
	for attr, v := range vals {
		if v.IsNull() || v.IsUnknown() {
			continue
		}
		jsonKey := attr
		if mapped, ok := bodyKeyMap[attr]; ok && mapped != "" {
			jsonKey = mapped
		}
		body[jsonKey] = v.ValueString()
	}
	return body
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
