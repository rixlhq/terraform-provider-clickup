package provider

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

const statusKey = "status"

type webhookResource struct {
	client *clickupclient.Client
}

type webhookResourceModel struct {
	WebhookId types.String `tfsdk:"webhook_id"`
	TeamId    types.Int64  `tfsdk:"team_id"`
	Endpoint  types.String `tfsdk:"endpoint"`
	Events    types.List   `tfsdk:"events"`
	Status    types.String `tfsdk:"status"`
	SpaceId   types.String `tfsdk:"space_id"`
	FolderId  types.String `tfsdk:"folder_id"`
	ListId    types.String `tfsdk:"list_id"`
	TaskId    types.String `tfsdk:"task_id"`
	ClientId  types.String `tfsdk:"client_id"`
	Health    types.String `tfsdk:"health"`
	Secret    types.String `tfsdk:"secret"`
	UserId    types.Int64  `tfsdk:"user_id"`
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
			},
			"folder_id": schema.StringAttribute{
				Optional: true,
			},
			"list_id": schema.StringAttribute{
				Optional: true,
			},
			"task_id": schema.StringAttribute{
				Optional: true,
			},
			"client_id": schema.StringAttribute{
				Computed: true,
			},
			"health": schema.StringAttribute{
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

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := r.buildCreateBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := "/v2/team/" + strconv.FormatInt(data.TeamId.ValueInt64(), 10) + "/webhook"
	raw, err := r.client.Post(ctx, path, body)
	if err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	id, err := extractWebhookID(raw)
	if err != nil {
		resp.Diagnostics.AddError("Create Response Error", err.Error())
		return
	}
	data.WebhookId = types.StringValue(id)

	r.refresh(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.refresh(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.WebhookId.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Start from the prior state so the computed webhook_id used in the path is
	// known, then overlay any values the practitioner changed in the plan.
	data := stateData
	if !planData.Endpoint.IsUnknown() {
		data.Endpoint = planData.Endpoint
	}
	if !planData.Events.IsUnknown() {
		data.Events = planData.Events
	}
	if !planData.Status.IsUnknown() {
		data.Status = planData.Status
	}

	body, err := r.buildUpdateBody(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Request Body Error", err.Error())
		return
	}

	path := "/v2/webhook/" + url.PathEscape(stateData.WebhookId.ValueString())
	if _, err := r.client.Put(ctx, path, body); err != nil {
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	r.refresh(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := "/v2/webhook/" + url.PathEscape(data.WebhookId.ValueString())
	if _, err := r.client.Delete(ctx, path); err != nil {
		if !clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		}
	}
}
