//nolint:goconst // Terraform attribute/path strings repeated in schemas, maps, and tests.
package provider

import (
	"context"
	"maps"

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
		updateBodyTransforms: map[string]func(any) any{
			"tag": spaceTagUpdateColorNames,
		},
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

// spaceTagUpdateColorNames renames the create-scoped tag color keys
// (tag_fg / tag_bg) to the update-scoped keys (fg_color / bg_color) used by
// the ClickUp PUT /v2/space/{space_id}/tag/{tag_name} endpoint.
func spaceTagUpdateColorNames(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}

	out := maps.Clone(m)
	if fg, ok := out["tag_fg"]; ok {
		out["fg_color"] = fg
		delete(out, "tag_fg")
	}
	if bg, ok := out["tag_bg"]; ok {
		out["bg_color"] = bg
		delete(out, "tag_bg")
	}
	return out
}
