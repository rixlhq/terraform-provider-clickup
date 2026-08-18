package provider

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasource_schema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

// rawJSONDataSource is a fallback data source for ClickUp endpoints whose
// response schemas cannot be generated automatically. It exposes the raw JSON
// response as a string so practitioners can use jsondecode() when needed.
type rawQueryParam struct {
	name     string
	list     bool
	required bool
	brackets bool
}

type rawJSONDataSource struct {
	client      *clickupclient.Client
	name        string
	path        string
	params      []string
	queryParams []rawQueryParam
	schema      datasource_schema.Schema
}

func newRawJSONDataSource(name, path string, params []string, queryParams ...rawQueryParam) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &rawJSONDataSource{
			name:        name,
			path:        path,
			params:      params,
			queryParams: queryParams,
		}
	}
}

func (d *rawJSONDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.name
}

func (d *rawJSONDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := make(map[string]datasource_schema.Attribute, len(d.params)+len(d.queryParams)+1)
	for _, p := range d.params {
		attrs[p] = datasource_schema.StringAttribute{
			Required:            true,
			MarkdownDescription: fmt.Sprintf("Path parameter %s.", p),
		}
	}
	for _, qp := range d.queryParams {
		if qp.list {
			attrs[qp.name] = datasource_schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            qp.required,
				Optional:            !qp.required,
				MarkdownDescription: fmt.Sprintf("Query parameter %s.", qp.name),
			}
		} else {
			attrs[qp.name] = datasource_schema.StringAttribute{
				Required:            qp.required,
				Optional:            !qp.required,
				MarkdownDescription: fmt.Sprintf("Query parameter %s.", qp.name),
			}
		}
	}
	attrs["result"] = datasource_schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "The raw JSON response from the ClickUp API. Use jsondecode() to parse it.",
	}
	d.schema = datasource_schema.Schema{Attributes: attrs}
	resp.Schema = d.schema
}

func (d *rawJSONDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *providerdata.Data, got %T", req.ProviderData))
		return
	}
	d.client = pd.ClickUp
}

func (d *rawJSONDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this data source.")
		return
	}

	path, query, err := d.buildPathAndQuery(req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", err.Error())
		return
	}

	body, err := d.client.Get(ctx, path, query)
	if err != nil {
		if clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for %s: %s", d.name, err.Error()))
			return
		}
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	tfType := d.schema.Type().TerraformType(ctx)
	objType, ok := tfType.(tftypes.Object)
	if !ok {
		resp.Diagnostics.AddError("Schema Error", "raw JSON data source schema is not an object")
		return
	}

	configObj, err := asObject(req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Config Error", err.Error())
		return
	}

	state := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for attr, attrType := range objType.AttributeTypes {
		if attr == "result" {
			state[attr] = tftypes.NewValue(tftypes.String, string(body))
			continue
		}
		if v, ok := configObj[attr]; ok {
			state[attr] = v
			continue
		}
		state[attr] = tftypes.NewValue(attrType, nil)
	}

	resp.State.Raw = tftypes.NewValue(tfType, state)
}

func (d *rawJSONDataSource) buildPathAndQuery(v tftypes.Value) (string, url.Values, error) {
	obj, err := asObject(v)
	if err != nil {
		return "", nil, err
	}

	pathParams := make(map[string]bool, len(d.params))
	path := d.path
	for _, match := range pathParamRegex.FindAllStringSubmatch(d.path, -1) {
		param := match[1]
		normParam := clickupcommon.TerraformIdentifier(param)
		pathParams[normParam] = true
		val, ok := obj[normParam]
		if !ok {
			return "", nil, fmt.Errorf("missing path parameter %q", normParam)
		}
		str, err := valueAsString(val)
		if err != nil {
			return "", nil, fmt.Errorf("path parameter %q: %w", normParam, err)
		}
		path = strings.Replace(path, match[0], url.PathEscape(str), 1)
	}

	queryParamsMap := make(map[string]rawQueryParam, len(d.queryParams))
	for _, qp := range d.queryParams {
		queryParamsMap[qp.name] = qp
	}
	query := url.Values{}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		if k == "result" || pathParams[k] {
			continue
		}
		if _, ok := queryParamsMap[k]; !ok {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		val := obj[k]
		if val.IsNull() || !val.IsKnown() {
			continue
		}
		qp := queryParamsMap[k]
		if err := appendQueryValue(query, k, val, qp.brackets); err != nil {
			return "", nil, fmt.Errorf("query parameter %q: %w", k, err)
		}
	}

	return path, query, nil
}
