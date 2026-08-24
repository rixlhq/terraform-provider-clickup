package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

const statusKey = "status"

type webhookResource struct {
	client *clickupclient.Client
}

type webhookResourceModel struct {
	WebhookId    types.String `tfsdk:"webhook_id"`
	TeamId       types.Int64  `tfsdk:"team_id"`
	Endpoint     types.String `tfsdk:"endpoint"`
	Events       types.List   `tfsdk:"events"`
	Status       types.String `tfsdk:"status"`
	SpaceId      types.String `tfsdk:"space_id"`
	FolderId     types.String `tfsdk:"folder_id"`
	ListId       types.String `tfsdk:"list_id"`
	TaskId       types.String `tfsdk:"task_id"`
	ClientId     types.String `tfsdk:"client_id"`
	Health       types.String `tfsdk:"health"`
	HealthStatus types.String `tfsdk:"health_status"`
	Secret       types.String `tfsdk:"secret"`
	UserId       types.Int64  `tfsdk:"user_id"`
}

func newWebhookResource() resource.Resource {
	return &webhookResource{}
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

//nolint:goconst // repeated Terraform attribute names are intentional
func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"webhook_id": schema.StringAttribute{
				Computed: true,
			},
			"team_id": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"endpoint": schema.StringAttribute{
				Required: true,
			},
			"events": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
			},
			statusKey: schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"space_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"folder_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"list_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"task_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_id": schema.StringAttribute{
				Computed: true,
			},
			"health": schema.StringAttribute{
				Computed: true,
			},
			"health_status": schema.StringAttribute{
				Computed: true,
			},
			"secret": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"user_id": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
