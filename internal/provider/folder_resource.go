package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

var _ resource.ResourceWithImportState = &folderResource{}

type folderResource struct {
	client *clickupclient.Client
}

type folderResourceModel struct {
	FolderId         types.String `tfsdk:"folder_id"`
	SpaceId          types.String `tfsdk:"space_id"`
	Name             types.String `tfsdk:"name"`
	ParentFolderId   types.String `tfsdk:"parent_folder_id"`
	Hidden           types.Bool   `tfsdk:"hidden"`
	OverrideStatuses types.Bool   `tfsdk:"override_statuses"`
	Orderindex       types.Int64  `tfsdk:"orderindex"`
	TaskCount        types.Int64  `tfsdk:"task_count"`
}

// newFolderResource returns a new ClickUp folder resource instance.
func newFolderResource() resource.Resource {
	return &folderResource{}
}

// Metadata sets the resource type name for clickup_folder.
func (r *folderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

// Schema defines the schema and attributes for clickup_folder.
func (r *folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"folder_id": schema.StringAttribute{
				Computed: true,
			},
			"space_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"parent_folder_id": schema.StringAttribute{
				Optional: true,
			},
			"hidden": schema.BoolAttribute{
				Computed: true,
			},
			"override_statuses": schema.BoolAttribute{
				Computed: true,
			},
			"orderindex": schema.Int64Attribute{
				Computed: true,
			},
			"task_count": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

// Configure initializes the ClickUp client for the folder resource.
func (r *folderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a new folder in the specified ClickUp space.
func (r *folderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data folderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.buildCreateBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := "/v2/space/" + url.PathEscape(data.SpaceId.ValueString()) + "/folder"
	raw, err := r.client.Post(ctx, path, body)
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	if err := r.mapResponse(raw, &data); err != nil {
		resp.Diagnostics.AddError("Create Response Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the folder state from the ClickUp API.
func (r *folderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refresh(ctx, &data); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	if data.FolderId.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates an existing folder in ClickUp.
func (r *folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := stateData
	if !planData.Name.IsUnknown() && !planData.Name.IsNull() {
		data.Name = planData.Name
	}

	body, err := r.buildUpdateBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := "/v2/folder/" + url.PathEscape(stateData.FolderId.ValueString())
	raw, err := r.client.Put(ctx, path, body)
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	if err := r.mapResponse(raw, &data); err != nil {
		resp.Diagnostics.AddError("Update Response Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete removes a folder from ClickUp.
func (r *folderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := "/v2/folder/" + url.PathEscape(data.FolderId.ValueString())
	if _, err := r.client.Delete(ctx, path); err != nil {
		if !clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		}
	}
}

// ImportState imports an existing folder into Terraform state using its folder_id.
func (r *folderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("folder_id"), req, resp)
}
