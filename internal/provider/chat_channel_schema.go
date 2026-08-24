package provider

import (
	"context"

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

//nolint:goconst // Terraform attribute names repeated across schemas.
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
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *providerdata.Data")
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
