//nolint:goconst // Terraform attribute/path strings repeated in schemas, maps, and tests.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func newSpaceTagResource() resource.Resource {
	return &genericResource{
		name:             "space_tag",
		createPath:       "/v2/space/{space_id}/tag",
		readPath:         "/v2/space/{space_id}/tag",
		updatePath:       "/v2/space/{space_id}/tag/{tag_name}",
		deletePath:       "/v2/space/{space_id}/tag/{tag_name}",
		updateMethod:     "put",
		createBodyFields: []string{"tag"},
		updateBodyFields: []string{"tag"},
		readFromList:     true,
		readListRoot:     "tags",
		readListIDField:  "name",
		readResponseRoot: "tag",
		idFromBody:       []string{"tag", "name"},
		idField:          "tag_name",
		schemaFunc:       spaceTagSchema,
	}
}

func spaceTagSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"space_id": schema.StringAttribute{
				Required: true,
			},
			"tag_name": schema.StringAttribute{
				Computed: true,
			},
			"tag": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required: true,
					},
					"tag_fg": schema.StringAttribute{
						Required: true,
					},
					"tag_bg": schema.StringAttribute{
						Required: true,
					},
				},
			},
		},
	}
}
