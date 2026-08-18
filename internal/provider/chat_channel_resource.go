package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	resource_chat_channel "github.com/rixlhq/terraform-provider-clickup/internal/provider/generated/resource_chat_channel"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

var _ resource.Resource = &ChatChannelResource{}
var _ resource.ResourceWithConfigure = &ChatChannelResource{}

const (
	chatChannelCreatePath = "/api/v3/workspaces/{workspace_id}/chat/channels"
	chatChannelReadPath   = "/api/v3/workspaces/{workspace_id}/chat/channels/{channel_id}"
	chatChannelUpdatePath = "/api/v3/workspaces/{workspace_id}/chat/channels/{channel_id}"
	chatChannelDeletePath = "/api/v3/workspaces/{workspace_id}/chat/channels/{channel_id}"
)

// ChatChannelResource manages a ClickUp chat channel.
type ChatChannelResource struct {
	client *clickupclient.Client
}

func (r *ChatChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_chat_channel"
}

func (r *ChatChannelResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_chat_channel.ChatChannelResourceSchema(ctx)
}

func (r *ChatChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChatChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	var model resource_chat_channel.ChatChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID, diags := r.workspaceID(model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.Name.IsNull() || model.Name.IsUnknown() {
		resp.Diagnostics.AddError("Missing Name", "name is required to create a ClickUp chat channel.")
		return
	}

	body, err := r.buildCreateBody(model)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", err.Error())
		return
	}

	path := strings.ReplaceAll(chatChannelCreatePath, "{workspace_id}", workspaceID)
	raw, err := r.client.Post(ctx, path, body)
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	channelID, err := r.parseCreatedChannelID(raw)
	if err != nil {
		resp.Diagnostics.AddError("Response Error", err.Error())
		return
	}

	newModel, diags := r.readModel(ctx, workspaceID, channelID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)
}

func (r *ChatChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	var model resource_chat_channel.ChatChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID, diags := r.workspaceID(model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := model.ChannelId.ValueString()
	newModel, diags := r.readModel(ctx, workspaceID, channelID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)
}

func (r *ChatChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	var model resource_chat_channel.ChatChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID, diags := r.workspaceID(model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := model.ChannelId.ValueString()
	body, err := r.buildUpdateBody(model)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", err.Error())
		return
	}

	path := strings.ReplaceAll(chatChannelUpdatePath, "{workspace_id}", workspaceID)
	path = strings.ReplaceAll(path, "{channel_id}", channelID)
	if _, err := r.client.Patch(ctx, path, body); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	newModel, diags := r.readModel(ctx, workspaceID, channelID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)
}

func (r *ChatChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this resource.")
		return
	}

	var model resource_chat_channel.ChatChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspaceID, diags := r.workspaceID(model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := model.ChannelId.ValueString()
	path := strings.ReplaceAll(chatChannelDeletePath, "{workspace_id}", workspaceID)
	path = strings.ReplaceAll(path, "{channel_id}", channelID)
	if _, err := r.client.Delete(ctx, path); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}
}

func newChatChannelResource() resource.Resource {
	return &ChatChannelResource{}
}
