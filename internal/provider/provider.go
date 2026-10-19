package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/clickupv3"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

var (
	_ provider.Provider = &ClickUpProvider{}
)

// ClickUpProvider implements the ClickUp Terraform provider.
type ClickUpProvider struct {
	version string
}

// providerData holds the configured API client and is passed to resources and data sources.
type providerData = providerdata.Data

// ClickUpProviderModel describes the provider configuration.
type ClickUpProviderModel struct {
	APIToken  types.String `tfsdk:"api_token"`
	BaseURL   types.String `tfsdk:"base_url"`
	V3BaseURL types.String `tfsdk:"v3_base_url"`
}

// Metadata sets the provider type name and version.
func (p *ClickUpProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "clickup"
	resp.Version = p.version
}

// Schema defines the provider configuration schema.
func (p *ClickUpProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for managing ClickUp infrastructure via the ClickUp Public APIs (V2 and V3). Automate workspaces, spaces, folders, lists, tasks, goals, webhooks, chat channels, time entries, and permissions under Terraform Plugin Framework protocol version 6.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "ClickUp API token. Can be set via the `CLICKUP_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Override the ClickUp V2 API base URL. Can be set via the `CLICKUP_BASE_URL` environment variable. Defaults to `https://api.clickup.com/api`.",
				Optional:            true,
			},
			"v3_base_url": schema.StringAttribute{
				MarkdownDescription: "Override the ClickUp V3 API base URL. Can be set via the `CLICKUP_V3_BASE_URL` environment variable. Defaults to `https://api.clickup.com/`.",
				Optional:            true,
			},
		},
	}
}

// Configure initializes ClickUp V2 and V3 clients from configuration and environment variables.
func (p *ClickUpProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ClickUpProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = applyEnvOverrides(data)

	if data.APIToken.IsNull() || data.APIToken.ValueString() == "" {
		resp.Diagnostics.AddError("Missing API Token", "A ClickUp API token must be configured via the api_token attribute or CLICKUP_API_TOKEN environment variable.")
		return
	}

	client := clickupclient.New(data.APIToken.ValueString(), data.BaseURL.ValueString(), nil)
	if p.version != "" {
		client.SetUserAgent("terraform-provider-clickup/" + p.version)
	}

	v3Client, err := clickupv3.New(data.APIToken.ValueString(), data.V3BaseURL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("ClickUp V3 Client Error", err.Error())
		return
	}

	pd := &providerData{ClickUp: client, ClickUpV3: v3Client}
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

// applyEnvOverrides populates missing configuration values from environment variables.
func applyEnvOverrides(data ClickUpProviderModel) ClickUpProviderModel {
	data.APIToken = envOrString(data.APIToken, "CLICKUP_API_TOKEN")
	data.BaseURL = envOrString(data.BaseURL, "CLICKUP_BASE_URL")
	data.V3BaseURL = envOrString(data.V3BaseURL, "CLICKUP_V3_BASE_URL")
	return data
}

// envOrString returns the environment variable value if the Terraform string is null or empty.
func envOrString(v types.String, env string) types.String {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v
	}
	if val := os.Getenv(env); val != "" {
		return types.StringValue(val)
	}
	return v
}

// Resources returns all resource factory functions supported by this provider.
func (p *ClickUpProvider) Resources(ctx context.Context) []func() resource.Resource {
	return append(resourceFactories,
		newFolderResource,
		newListCommentResource,
		newSpaceTagResource,
		newTaskCommentResource,
		newUserGroupResource,
		newViewCommentResource,
		newWebhookResource,
		newChatChannelResource,
		// Association resources (POST/DELETE only)
		newTaskDependencyResource,
		newTaskLinkResource,
		newTaskTagResource,
		newTaskCustomFieldResource,
		newTaskGuestResource,
		newFolderGuestResource,
		newListGuestResource,
		newListTaskResource,
		// Full CRUD resources with hand-written schemas
		newTaskChecklistResource,
		newChecklistItemResource,
		newKeyResultResource,
		newTaskTimeResource,
		newTimeEntryResource,
		newTeamGuestResource,
		newTeamUserResource,
		newCommentReplyResource,
	)
}

// DataSources returns all data source factory functions supported by this provider.
func (p *ClickUpProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return append(dataSourceFactories, append(manualDataSourceFactories,
		newAuditLogsDataSource,
	)...)
}

// New returns a factory for the ClickUp provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ClickUpProvider{version: version}
	}
}
