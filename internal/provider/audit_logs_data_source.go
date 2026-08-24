package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupv3"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

type auditLogsDataSource struct {
	client *clickupv3.Client
}

func newAuditLogsDataSource() datasource.DataSource {
	return &auditLogsDataSource{}
}

func (d *auditLogsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "clickup_audit_logs"
}

func (d *auditLogsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Queries ClickUp audit logs via the V3 API. Returns the raw JSON response; use `jsondecode()` to parse.",
		Attributes: map[string]schema.Attribute{
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Workspace to query audit logs for.",
				Required:            true,
			},
			"applicability": schema.StringAttribute{
				MarkdownDescription: "Type of logs to filter by. Options: `auth-and-security`, `custom-fields`, `hierarchy-activity`, `user-activity`.",
				Optional:            true,
			},
			"result": schema.StringAttribute{
				MarkdownDescription: "The raw JSON response from the ClickUp V3 API. Use `jsondecode()` to parse it.",
				Computed:            true,
			},
		},
	}
}

func (d *auditLogsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	d.client = pd.ClickUpV3
}

type auditLogsModel struct {
	WorkspaceID   types.String `tfsdk:"workspace_id"`
	Applicability types.String `tfsdk:"applicability"`
	Result        types.String `tfsdk:"result"`
}

func (d *auditLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp V3 Client", "Configure the provider with api_token to use this data source.")
		return
	}

	var data auditLogsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wsID, err := strconv.ParseFloat(data.WorkspaceID.ValueString(), 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid workspace_id", "workspace_id must be a numeric string")
		return
	}

	body := clickupv3.AutoAuditlogAuditLogQueryRequest{}
	if !data.Applicability.IsNull() && data.Applicability.ValueString() != "" {
		body.Applicability = clickupv3.NewOptAutoAuditlogAuditLogQueryRequestApplicability(
			clickupv3.AutoAuditlogAuditLogQueryRequestApplicability(data.Applicability.ValueString()),
		)
	}

	raw, err := d.client.QueryAuditLog(ctx, &body, clickupv3.QueryAuditLogParams{
		WorkspaceID: clickupv3.AutoAuditlogWorkspaceAuditLogPublicControllerQueryAuditLogWorkspaceIdPath(wsID),
	})
	if err != nil {
		resp.Diagnostics.AddError("ClickUp V3 API Error", err.Error())
		return
	}

	// Re-serialize to compact JSON for the state string.
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		resp.Diagnostics.AddError("Response Decode Error", err.Error())
		return
	}
	compact, err := json.Marshal(parsed)
	if err != nil {
		resp.Diagnostics.AddError("Response Encode Error", err.Error())
		return
	}

	data.Result = types.StringValue(string(compact))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
