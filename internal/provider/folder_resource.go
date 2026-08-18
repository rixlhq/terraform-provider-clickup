package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

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

func newFolderResource() resource.Resource {
	return &folderResource{}
}

func (r *folderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_folder"
}

func (r *folderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"folder_id": schema.StringAttribute{
				Computed: true,
			},
			"space_id": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
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

	path := "/v2/space/" + data.SpaceId.ValueString() + "/folder"
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *folderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data folderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.buildUpdateBody(data)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := "/v2/folder/" + data.FolderId.ValueString()
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

func (r *folderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data folderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := "/v2/folder/" + data.FolderId.ValueString()
	if _, err := r.client.Delete(ctx, path); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
	}
}
