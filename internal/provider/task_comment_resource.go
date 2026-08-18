//nolint:goconst
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func newTaskCommentResource() resource.Resource {
	return &genericResource{
		name:             "task_comment",
		createPath:       "/v2/task/{task_id}/comment",
		readPath:         "/v2/task/{task_id}/comment",
		updatePath:       "/v2/comment/{comment_id}",
		deletePath:       "/v2/comment/{comment_id}",
		updateMethod:     "put",
		idField:          "comment_id",
		readFromList:     true,
		readListRoot:     "comments",
		readListIDField:  "id",
		createBodyFields: []string{"comment_text", "notify_all"},
		updateBodyFields: []string{"comment_text", "resolved"},
		createBodyDefaults: map[string]any{
			"notify_all": false,
		},
		schemaFunc: taskCommentSchema,
	}
}

func taskCommentSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"comment_id": schema.StringAttribute{
				Computed: true,
			},
			"task_id": schema.StringAttribute{
				Required: true,
			},
			"comment_text": schema.StringAttribute{
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
