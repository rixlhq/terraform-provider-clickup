//nolint:goconst // Terraform attribute names repeated across schemas.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// --- comment_reply ---
// POST   /v2/comment/{comment_id}/reply (create reply)
// PUT    /v2/comment/{comment_id} (update — uses the reply's own comment_id)
// DELETE /v2/comment/{comment_id} (delete — uses the reply's own comment_id)
// No single-GET; replies are read from the parent comment's reply list.
func newCommentReplyResource() resource.Resource {
	return &genericResource{
		name:             "comment_reply",
		createPath:       "/v2/comment/{parent_comment_id}/reply",
		readPath:         "/v2/comment/{parent_comment_id}/reply",
		updatePath:       "/v2/comment/{comment_id}",
		deletePath:       "/v2/comment/{comment_id}",
		updateMethod:     "put",
		idField:          "comment_id",
		readFromList:     true,
		readListRoot:     "comments",
		readListIDField:  "id",
		createBodyFields: []string{"comment_text", "assignee", "group_assignee", "resolved"},
		updateBodyFields: []string{"comment_text", "assignee", "group_assignee", "resolved"},
		schemaFunc:       commentReplySchema,
	}
}

func commentReplySchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Replies to an existing ClickUp comment. The reply is itself a comment, updatable and deletable via the comment endpoint.",
		Attributes: map[string]schema.Attribute{
			"comment_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_comment_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"comment_text": schema.StringAttribute{
				Required: true,
			},
			"assignee": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"group_assignee": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"resolved": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}
