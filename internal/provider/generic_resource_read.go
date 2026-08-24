package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/provider/clickupcommon"
)

// queryParams extracts the configured query parameters from a Terraform value.
// It is used for resource read endpoints that require query parameters such as
// the user group list endpoint, which requires a team_id query.
func (r *genericResource) queryParams(_ context.Context, v tftypes.Value) (url.Values, diag.Diagnostics) {
	var diags diag.Diagnostics
	query := url.Values{}
	if len(r.readQueryParams) == 0 {
		return query, diags
	}

	obj, err := asObject(v)
	if err != nil {
		diags.AddError("Invalid State", err.Error())
		return nil, diags
	}

	for q, attr := range r.readQueryParams {
		val, ok := obj[attr]
		if !ok {
			diags.AddError("Missing Query Parameter", fmt.Sprintf("%q is required", attr))
			return nil, diags
		}
		s, err := valueAsString(val)
		if err != nil {
			diags.AddError("Invalid Query Parameter", fmt.Sprintf("%q: %s", attr, err))
			return nil, diags
		}
		if s == "" {
			diags.AddError("Invalid Query Parameter", fmt.Sprintf("%q must not be empty", attr))
			return nil, diags
		}
		query.Set(q, s)
	}

	return query, diags
}

func (r *genericResource) getAndDecode(ctx context.Context, readPath, id string, query url.Values) (any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !r.readFromList {
		raw, err := r.client.Get(ctx, readPath, query)
		if err != nil {
			if clickupclient.IsNotFound(err) {
				diags.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for %s %s", r.name, id))
				return nil, diags
			}
			diags.AddError("ClickUp API Error", err.Error())
			return nil, diags
		}

		jv, err := clickupcommon.DecodeJSONResponse(raw)
		if err != nil {
			diags.AddError("Response Decode Error", err.Error())
			return nil, diags
		}
		return jv, diags
	}

	pageQuery := cloneURLValues(query)
	seen := make(map[string]bool)
	for {
		raw, err := r.client.Get(ctx, readPath, pageQuery)
		if err != nil {
			if clickupclient.IsNotFound(err) {
				diags.AddWarning("Not Found", fmt.Sprintf("ClickUp API returned 404 for %s %s", r.name, id))
				return nil, diags
			}
			diags.AddError("ClickUp API Error", err.Error())
			return nil, diags
		}

		jv, err := clickupcommon.DecodeJSONResponse(raw)
		if err != nil {
			diags.AddError("Response Decode Error", err.Error())
			return nil, diags
		}

		item, err := r.findInList(jv, id)
		if err == nil {
			return item, diags
		}
		var notFound *notFoundError
		if !errors.As(err, &notFound) {
			diags.AddError("Read Error", err.Error())
			return nil, diags
		}

		next, ok := r.nextListPageQuery(jv, query, seen)
		if !ok {
			diags.AddWarning("Not Found", notFound.message)
			return nil, diags
		}
		pageQuery = next
	}
}
