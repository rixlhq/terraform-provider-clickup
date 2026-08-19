//nolint:goconst
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func newListCommentResource() resource.Resource {
	return &genericResource{
		name:             "list_comment",
		createPath:       "/v2/list/{list_id}/comment",
		readPath:         "/v2/list/{list_id}/comment",
		updatePath:       "/v2/comment/{comment_id}",
		deletePath:       "/v2/comment/{comment_id}",
		updateMethod:     "put",
		idField:          "comment_id",
		readFromList:     true,
		readListRoot:     "comments",
		readListIDField:  "id",
		createBodyFields: []string{"comment_text", "assignee", "notify_all"},
		updateBodyFields: []string{"comment_text", "resolved"},
		createBodyDefaults: map[string]any{
			"notify_all": false,
		},
		readTransforms: map[string]func(any) any{
			"assignee": objectFieldToInt,
		},
		schemaFunc: listCommentSchema,
	}
}

func listCommentSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"comment_id": schema.StringAttribute{
				Computed: true,
			},
			"list_id": schema.StringAttribute{
				Required: true,
			},
			"comment_text": schema.StringAttribute{
				Required: true,
			},
			"assignee": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"notify_all": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"resolved": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"date": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}
