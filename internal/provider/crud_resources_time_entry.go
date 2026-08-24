package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- time_entry (team-level time entry) ---
// POST /v2/team/{team_id}/time_entries (create)
// GET  /v2/team/{team_id}/time_entries/{timer_id} (read)
// PUT  /v2/team/{team_id}/time_entries/{timer_id} (update)
// DELETE /v2/team/{team_id}/time_entries/{timer_id} (delete)
func newTimeEntryResource() resource.Resource {
	return &genericResource{
		name:             "time_entry",
		createPath:       "/v2/team/{team_id}/time_entries",
		readPath:         "/v2/team/{team_id}/time_entries/{timer_id}",
		updatePath:       "/v2/team/{team_id}/time_entries/{timer_id}",
		deletePath:       "/v2/team/{team_id}/time_entries/{timer_id}",
		updateMethod:     "put",
		createBodyFields: []string{"description", "tags", "start", "end", "billable", "duration", "assignee", "tid"},
		updateBodyFields: []string{"description", "tags", "tag_action", "start", "end", "tid", "billable", "duration"},
		idField:          "timer_id",
		schemaFunc:       timeEntrySchema,
	}
}

func timeEntrySchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a time entry on a ClickUp team.",
		Attributes: map[string]schema.Attribute{
			"timer_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"tags": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"start": schema.Int64Attribute{
				Required: true,
			},
			"end": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"duration": schema.Int64Attribute{
				Required: true,
			},
			"billable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"assignee": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"tid": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"tag_action": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

// --- time_entry_tag ---
// REMOVED: The /time_entries/tags endpoint is not standard CRUD.
// POST/DELETE assign/remove tags to/from time entries (requires
// time_entry_ids + tags arrays), not standalone tag creation.
// PUT renames a tag. GET lists tags under "data", not "tags".
// These are two different operations that don't fit a single Terraform
// resource. Use time_entry's "tags" field for tag assignment, and
// a future time_entry_tag_rename resource for the PUT rename operation.

// --- team_guest ---
// Hand-written in team_invite_resources.go — the invite POST returns
// { "team": { "members": [...] } } with no top-level guest_id.

func teamGuestSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Invites a guest to a ClickUp workspace (team).",
		Attributes: map[string]schema.Attribute{
			"guest_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"can_edit_tags": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"can_see_time_spent": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"can_see_time_estimated": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"can_create_views": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"can_see_points_estimated": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"custom_role_id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

// --- team_user ---
// Hand-written in team_invite_resources.go — same team.members structure.

func teamUserSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Invites a user to a ClickUp workspace (team).",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"admin": schema.BoolAttribute{
				Required: true,
			},
			"custom_role_id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

// --- team_view ---
// REMOVED: The existing "view" resource already creates at /v2/team/{team_id}/view
// and manages via /v2/view/{view_id}. team_view was a duplicate with an
// incomplete schema (missing grouping, divide, sorting, filters, columns,
// team_sidebar, settings). Use the "view" resource instead.
