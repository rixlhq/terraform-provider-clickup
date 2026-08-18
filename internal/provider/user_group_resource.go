//nolint:goconst
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newUserGroupResource() resource.Resource {
	return &genericResource{
		name:             "user_group",
		createPath:       "/v2/team/{team_id}/group",
		readPath:         "/v2/group",
		updatePath:       "/v2/group/{group_id}",
		deletePath:       "/v2/group/{group_id}",
		updateMethod:     "put",
		idField:          "group_id",
		readFromList:     true,
		readListRoot:     "groups",
		readListIDField:  "id",
		readQueryParams:  map[string]string{"team_id": "team_id"},
		createBodyFields: []string{"name", "handle", "members"},
		updateBodyFields: []string{"name", "handle", "members"},
		createBodyTransforms: map[string]func(any) any{
			"members": extractAddList,
		},
		readTransforms: map[string]func(any) any{
			"members": listToAddRemObject,
		},
		schemaFunc: userGroupSchema,
	}
}

func userGroupSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Computed: true,
			},
			"team_id": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"handle": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"members": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"add": schema.ListAttribute{
						ElementType:         types.Int64Type,
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "User IDs to add to the group.",
					},
					"rem": schema.ListAttribute{
						ElementType:         types.Int64Type,
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "User IDs to remove from the group.",
					},
				},
			},
		},
	}
}
