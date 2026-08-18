package provider

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasource_schema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
	"github.com/rixlhq/terraform-provider-clickup/internal/providerdata"
)

var _ datasource.DataSource = &genericDataSource{}

var pathParamRegex = regexp.MustCompile(`\{(\w+)\}`)

// genericDataSource maps a ClickUp GET endpoint to a Terraform data source using
// a generated Terraform Plugin Framework schema. It operates on tftypes.Value so
// the same implementation can serve every generated data source.
type genericDataSource struct {
	client     *clickupclient.Client
	name       string
	path       string
	schemaFunc func(context.Context) datasource_schema.Schema
	schema     datasource_schema.Schema
}

func (d *genericDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.name
}

func (d *genericDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	d.schema = d.schemaFunc(ctx)
	resp.Schema = d.schema
}

func (d *genericDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *genericDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Missing ClickUp Client", "Configure the provider with api_token or CLICKUP_API_TOKEN to use this data source.")
		return
	}

	schema := d.schema
	if schema.Type() == nil {
		schema = d.schemaFunc(ctx)
		d.schema = schema
	}
	tfType := schema.Type().TerraformType(ctx)

	path, query, err := d.buildPathAndQuery(req.Config.Raw)
	if err != nil {
		resp.Diagnostics.AddError("Request Error", err.Error())
		return
	}

	body, err := d.client.Get(ctx, path, query)
	if err != nil {
		if clickupclient.IsNotFound(err) {
			resp.Diagnostics.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for %s: %s", path, err.Error()))
			return
		}
		resp.Diagnostics.AddError("ClickUp API Error", err.Error())
		return
	}

	jsonVal, err := clickupcommon.DecodeJSONResponse(body)
	if err != nil {
		resp.Diagnostics.AddError("Response Decode Error", err.Error())
		return
	}

	tfVal, err := clickupcommon.JSONToTfValue(ctx, tfType, jsonVal)
	if err != nil {
		resp.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}

	resp.State.Raw = tfVal
}

func (d *genericDataSource) buildPathAndQuery(v tftypes.Value) (string, url.Values, error) {
	obj, err := asObject(v)
	if err != nil {
		return "", nil, err
	}

	pathParams := map[string]bool{}
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

	query := url.Values{}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		if !pathParams[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		val := obj[k]
		if val.IsNull() || !val.IsKnown() {
			continue
		}
		if err := appendQueryValue(query, k, val, true); err != nil {
			return "", nil, fmt.Errorf("query parameter %q: %w", k, err)
		}
	}

	return path, query, nil
}

func asObject(v tftypes.Value) (map[string]tftypes.Value, error) {
	if v.IsNull() || !v.IsKnown() {
		return nil, errors.New("value is null or unknown")
	}
	objType, ok := v.Type().(tftypes.Object)
	if !ok {
		return nil, fmt.Errorf("expected object value, got %s", v.Type())
	}
	m := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	if err := v.As(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func valueAsStrings(v tftypes.Value) ([]string, error) {
	switch v.Type().(type) {
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		var elems []tftypes.Value
		if err := v.As(&elems); err != nil {
			return nil, err
		}
		out := make([]string, 0, len(elems))
		for _, e := range elems {
			if e.IsNull() || !e.IsKnown() {
				continue
			}
			s, err := valueAsString(e)
			if err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, fmt.Errorf("cannot use %s as a collection of parameter values", v.Type())
}

// appendQueryValue encodes a Terraform value as a query parameter. Collection
// values are added once per element. When bracketed is true the query key uses
// the "name[]" style that ClickUp uses for array query parameters.
func appendQueryValue(query url.Values, key string, v tftypes.Value, bracketed bool) error {
	if v.IsNull() || !v.IsKnown() {
		return nil
	}
	t := v.Type()
	if t.Is(tftypes.String) || t.Is(tftypes.Number) || t.Is(tftypes.Bool) {
		s, err := valueAsString(v)
		if err != nil {
			return err
		}
		query.Set(key, s)
		return nil
	}
	switch t.(type) {
	case tftypes.List, tftypes.Set, tftypes.Tuple:
		vals, err := valueAsStrings(v)
		if err != nil {
			return err
		}
		k := key
		if bracketed {
			k = key + "[]"
		}
		for _, s := range vals {
			query.Add(k, s)
		}
		return nil
	}
	return fmt.Errorf("cannot use %s as a query parameter value", t)
}

func valueAsString(v tftypes.Value) (string, error) {
	if v.IsNull() || !v.IsKnown() {
		return "", errors.New("value is null or unknown")
	}
	t := v.Type()
	if t.Is(tftypes.String) {
		var s string
		if err := v.As(&s); err != nil {
			return "", err
		}
		return s, nil
	}
	if t.Is(tftypes.Number) {
		var n big.Float
		if err := v.As(&n); err != nil {
			return "", err
		}
		return n.Text('f', -1), nil
	}
	if t.Is(tftypes.Bool) {
		var b bool
		if err := v.As(&b); err != nil {
			return "", err
		}
		return strconv.FormatBool(b), nil
	}
	return "", fmt.Errorf("cannot use %s as parameter value", t)
}
