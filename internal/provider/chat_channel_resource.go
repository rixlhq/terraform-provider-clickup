//nolint:goconst // Terraform attribute names repeated across schemas.
package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupv3"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

type chatChannelResource struct {
	client *clickupv3.Client
}

func newChatChannelResource() resource.Resource {
	return &chatChannelResource{}
}

func (r *chatChannelResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "clickup_chat_channel"
}

func (r *chatChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a ClickUp Chat channel via the V3 API.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Workspace.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"channel_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Channel.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Channel.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the Channel.",
				Optional:            true,
			},
			"topic": schema.StringAttribute{
				MarkdownDescription: "The topic of the Channel.",
				Optional:            true,
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "The visibility of the Channel. One of `PUBLIC`, `PRIVATE`.",
				Optional:            true,
				Computed:            true,
			},
			"user_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "User IDs to add to the channel (up to 100).",
				Optional:            true,
			},
			"creator": schema.StringAttribute{
				MarkdownDescription: "ID of the user who created the channel.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp of when the channel was created.",
				Computed:            true,
			},
			"archived": schema.BoolAttribute{
				MarkdownDescription: "Whether the channel is archived.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the channel (CHANNEL, DM, GROUP_DM).",
				Computed:            true,
			},
		},
	}
}

func (r *chatChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	r.client = pd.ClickUpV3
}

type chatChannelModel struct {
	WorkspaceID types.String   `tfsdk:"workspace_id"`
	ChannelID   types.String   `tfsdk:"channel_id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Topic       types.String   `tfsdk:"topic"`
	Visibility  types.String   `tfsdk:"visibility"`
	UserIDs     []types.String `tfsdk:"user_ids"`
	Creator     types.String   `tfsdk:"creator"`
	CreatedAt   types.String   `tfsdk:"created_at"`
	Archived    types.Bool     `tfsdk:"archived"`
	Type        types.String   `tfsdk:"type"`
}

func (r *chatChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp V3 Client", "Configure the provider with api_token to use this resource.")
		return
	}

	var plan chatChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsID, err := strconv.Atoi(plan.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid workspace_id", "workspace_id must be a numeric string")
		return
	}

	body := clickupv3.ChatCreateChatChannel{
		Name:    plan.Name.ValueString(),
		UserIds: stringListValue(plan.UserIDs),
	}
	if !plan.Description.IsNull() {
		body.Description = clickupv3.NewOptString(plan.Description.ValueString())
	}
	if !plan.Topic.IsNull() {
		body.Topic = clickupv3.NewOptString(plan.Topic.ValueString())
	}
	if !plan.Visibility.IsNull() {
		body.Visibility = clickupv3.NewOptChatCreateChatChannelVisibility(clickupv3.ChatCreateChatChannelVisibility(plan.Visibility.ValueString()))
	}

	res, err := r.client.CreateChatChannel(ctx, &body, clickupv3.CreateChatChannelParams{
		WorkspaceID: clickupv3.ChatPublicApiChatChannelsControllerCreateChatChannelWorkspaceIdPath(wsID),
	})
	if err != nil {
		resp.Diagnostics.AddError("ClickUp V3 API Error", err.Error())
		return
	}

	ch, ok := res.(*clickupv3.ChatPublicApiChatChannelsControllerCreateChatChannel201Response)
	if !ok {
		if ch200, ok2 := res.(*clickupv3.ChatPublicApiChatChannelsControllerCreateChatChannel200Response); ok2 {
			ch.Data = ch200.Data
		} else {
			resp.Diagnostics.AddError("Create Chat Channel Error", fmt.Sprintf("unexpected response type %T", res))
			return
		}
	}

	r.applyChannelToModel(&plan, ch.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *chatChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp V3 Client", "Configure the provider with api_token to use this resource.")
		return
	}

	var state chatChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsID, err := strconv.Atoi(state.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid workspace_id", "workspace_id must be a numeric string")
		return
	}

	res, err := r.client.GetChatChannel(ctx, clickupv3.GetChatChannelParams{
		WorkspaceID: clickupv3.ChatPublicApiChatChannelsControllerGetChatChannelWorkspaceIdPath(wsID),
		ChannelID:   state.ChannelID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("ClickUp V3 API Error", err.Error())
		return
	}

	switch v := res.(type) {
	case *clickupv3.ChatPublicApiChatChannelsControllerGetChatChannel200Response:
		r.applyChannelToModel(&state, v.Data)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	case *clickupv3.ChatPublicApiErrorResponseStatusCode:
		if v.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("ClickUp V3 API Error", fmt.Sprintf("status %d: %s", v.StatusCode, v.Response.Message))
	default:
		resp.Diagnostics.AddError("Read Chat Channel Error", fmt.Sprintf("unexpected response type %T", res))
	}
}

func (r *chatChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp V3 Client", "Configure the provider with api_token to use this resource.")
		return
	}

	var plan chatChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state chatChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsID, err := strconv.Atoi(plan.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid workspace_id", "workspace_id must be a numeric string")
		return
	}

	body := clickupv3.ChatUpdateChatChannel{}
	if !plan.Name.IsNull() {
		body.Name = clickupv3.NewOptString(plan.Name.ValueString())
	}
	if !plan.Description.IsNull() {
		body.Description = clickupv3.NewOptString(plan.Description.ValueString())
	}
	if !plan.Topic.IsNull() {
		body.Topic = clickupv3.NewOptString(plan.Topic.ValueString())
	}
	if !plan.Visibility.IsNull() {
		body.Visibility = clickupv3.NewOptChatUpdateChatChannelVisibility(clickupv3.ChatUpdateChatChannelVisibility(plan.Visibility.ValueString()))
	}

	res, err := r.client.UpdateChatChannel(ctx, &body, clickupv3.UpdateChatChannelParams{
		WorkspaceID: clickupv3.ChatPublicApiChatChannelsControllerUpdateChatChannelWorkspaceIdPath(wsID),
		ChannelID:   state.ChannelID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("ClickUp V3 API Error", err.Error())
		return
	}

	switch v := res.(type) {
	case *clickupv3.ChatPublicApiChatChannelsControllerUpdateChatChannel200Response:
		r.applyChannelToModel(&plan, v.Data)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	case *clickupv3.ChatPublicApiErrorResponseStatusCode:
		resp.Diagnostics.AddError("ClickUp V3 API Error", fmt.Sprintf("status %d: %s", v.StatusCode, v.Response.Message))
	default:
		resp.Diagnostics.AddError("Update Chat Channel Error", fmt.Sprintf("unexpected response type %T", res))
	}
}

func (r *chatChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp V3 Client", "Configure the provider with api_token to use this resource.")
		return
	}

	var state chatChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsID, err := strconv.Atoi(state.WorkspaceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid workspace_id", "workspace_id must be a numeric string")
		return
	}

	res, err := r.client.DeleteChatChannel(ctx, clickupv3.DeleteChatChannelParams{
		WorkspaceID: clickupv3.ChatPublicApiChatChannelsControllerDeleteChatChannelWorkspaceIdPath(wsID),
		ChannelID:   state.ChannelID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("ClickUp V3 API Error", err.Error())
		return
	}

	if v, ok := res.(*clickupv3.ChatPublicApiErrorResponseStatusCode); ok {
		if v.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("ClickUp V3 API Error",
			fmt.Sprintf("delete chat channel failed with status %d", v.StatusCode))
		return
	}
}

func (r *chatChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: workspace_id:channel_id
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid Import ID",
			"expected format: workspace_id:channel_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("channel_id"), parts[1])...)
}

func (r *chatChannelResource) applyChannelToModel(m *chatChannelModel, ch clickupv3.ChatChannel) {
	m.ChannelID = types.StringValue(ch.ID)
	m.Name = types.StringValue(ch.Name)
	if ch.Description.IsSet() {
		m.Description = types.StringValue(ch.Description.Value)
	} else {
		m.Description = types.StringNull()
	}
	if ch.Topic.IsSet() {
		m.Topic = types.StringValue(ch.Topic.Value)
	} else {
		m.Topic = types.StringNull()
	}
	m.Visibility = types.StringValue(string(ch.Visibility))
	m.Creator = types.StringValue(ch.Creator)
	m.CreatedAt = types.StringValue(ch.CreatedAt)
	m.Archived = types.BoolValue(ch.Archived)
	m.Type = types.StringValue(string(ch.Type))
}

func stringListValue(vals []types.String) []string {
	if len(vals) == 0 {
		return nil
	}
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if !v.IsNull() && !v.IsUnknown() {
			out = append(out, v.ValueString())
		}
	}
	return out
}
