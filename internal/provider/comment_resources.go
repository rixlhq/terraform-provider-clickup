//nolint:goconst // Terraform attribute names repeated across schemas.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// --- comment_reply ---
// POST   /v2/comment/{comment_id}/reply (create reply)
// PUT    /v2/comment/{comment_id} (update — uses the reply's own comment_id)
// DELETE /v2/comment/{comment_id} (delete — uses the reply's own comment_id)
// No single-GET; replies are read from the parent comment's reply list.
// newCommentReplyResource returns a resource instance for replying to comments.
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

// commentReplySchema returns the schema for the clickup_comment_reply resource.
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
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"assignee": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Validators: []validator.Int64{
					int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("group_assignee")),
				},
			},
			"group_assignee": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Validators: []validator.Int64{
					int64validator.ConflictsWith(path.MatchRelative().AtParent().AtName("assignee")),
				},
			},
			"resolved": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}
