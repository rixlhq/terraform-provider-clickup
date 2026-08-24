//nolint:goconst // Terraform attribute/path strings repeated in schemas and factory configs.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- task_checklist ---
// POST /v2/task/{task_id}/checklist (create)
// PUT  /v2/checklist/{checklist_id} (update)
// DELETE /v2/checklist/{checklist_id} (delete)
// No GET on checklist endpoint; read from task GET (checklists array).
func newTaskChecklistResource() resource.Resource {
	return &genericResource{
		name:               "task_checklist",
		createPath:         "/v2/task/{task_id}/checklist",
		readPath:           "/v2/task/{task_id}",
		updatePath:         "/v2/checklist/{checklist_id}",
		deletePath:         "/v2/checklist/{checklist_id}",
		updateMethod:       "put",
		createBodyFields:   []string{"name"},
		updateBodyFields:   []string{"name", "position"},
		idField:            "checklist_id",
		readFromList:       true,
		readListRoot:       "checklists",
		readListIDField:    "id",
		createResponseRoot: "checklist",
		schemaFunc:         taskChecklistSchema,
	}
}

func taskChecklistSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a checklist on a ClickUp task.",
		Attributes: map[string]schema.Attribute{
			"checklist_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"task_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"position": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

// --- checklist_item ---
// POST /v2/checklist/{checklist_id}/checklist_item (create)
// PUT  /v2/checklist/{checklist_id}/checklist_item/{checklist_item_id} (update)
// DELETE /v2/checklist/{checklist_id}/checklist_item/{checklist_item_id} (delete)
// No GET on checklist_item; read from task GET -> checklists -> items.
// Create response wraps the item inside { "checklist": { "items": [...] } }.
func newChecklistItemResource() resource.Resource {
	return &genericResource{
		name:               "checklist_item",
		createPath:         "/v2/checklist/{checklist_id}/checklist_item",
		readPath:           "/v2/task/{task_id}",
		updatePath:         "/v2/checklist/{checklist_id}/checklist_item/{checklist_item_id}",
		deletePath:         "/v2/checklist/{checklist_id}/checklist_item/{checklist_item_id}",
		updateMethod:       "put",
		createBodyFields:   []string{"name", "assignee"},
		updateBodyFields:   []string{"name", "assignee", "resolved", "parent"},
		idField:            "checklist_item_id",
		readFromList:       true,
		readListRoot:       "checklists.items",
		readListIDField:    "id",
		createResponseRoot: "checklist",
		schemaFunc:         checklistItemSchema,
	}
}

func checklistItemSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an item within a ClickUp checklist.",
		Attributes: map[string]schema.Attribute{
			"checklist_item_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checklist_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"task_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"assignee": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"resolved": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"parent": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

// --- key_result ---
// POST /v2/goal/{goal_id}/key_result (create)
// PUT  /v2/key_result/{key_result_id} (update)
// DELETE /v2/key_result/{key_result_id} (delete)
func newKeyResultResource() resource.Resource {
	return &genericResource{
		name:               "key_result",
		createPath:         "/v2/goal/{goal_id}/key_result",
		readPath:           "/v2/goal/{goal_id}",
		updatePath:         "/v2/key_result/{key_result_id}",
		deletePath:         "/v2/key_result/{key_result_id}",
		updateMethod:       "put",
		createBodyFields:   []string{"name", "owners", "type", "steps_start", "steps_end", "unit", "task_ids", "list_ids"},
		updateBodyFields:   []string{"steps_current", "note"},
		idField:            "key_result_id",
		readFromList:       true,
		readListRoot:       "goal.key_results",
		readListIDField:    "id",
		createResponseRoot: "key_result",
		schemaFunc:         keyResultSchema,
	}
}

func keyResultSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a key result on a ClickUp goal.",
		Attributes: map[string]schema.Attribute{
			"key_result_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"goal_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"owners": schema.ListAttribute{
				ElementType: types.Int64Type,
				Required:    true,
			},
			"steps_start": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"steps_end": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"steps_current": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"unit": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"task_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"list_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"note": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

// --- task_time (tracked time interval on a task) ---
// POST /v2/task/{task_id}/time (create)
// PUT  /v2/task/{task_id}/time/{interval_id} (update)
// DELETE /v2/task/{task_id}/time/{interval_id} (delete)
// The GET response nests intervals inside data[].intervals[], which the
// generic list reader cannot traverse. Read returns state as-is; drift
// is detected on the next plan when the task is refreshed.
func newTaskTimeResource() resource.Resource {
	return &genericResource{
		name:             "task_time",
		createPath:       "/v2/task/{task_id}/time",
		readPath:         "/v2/task/{task_id}/time",
		updatePath:       "/v2/task/{task_id}/time/{interval_id}",
		deletePath:       "/v2/task/{task_id}/time/{interval_id}",
		updateMethod:     "put",
		createBodyFields: []string{"start", "end", "time"},
		updateBodyFields: []string{"start", "end", "time"},
		idField:          "interval_id",
		schemaFunc:       taskTimeSchema,
	}
}

func taskTimeSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages tracked time on a ClickUp task.",
		Attributes: map[string]schema.Attribute{
			"interval_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"task_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"start": schema.Int64Attribute{
				Required: true,
			},
			"end": schema.Int64Attribute{
				Required: true,
			},
			"time": schema.Int64Attribute{
				Required: true,
			},
		},
	}
}

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
		createBodyFields: []string{"description", "tags", "start", "stop", "end", "billable", "duration", "assignee", "tid"},
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
			"stop": schema.Int64Attribute{
				Optional: true,
				Computed: true,
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
