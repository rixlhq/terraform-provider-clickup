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
// No single-GET; read from task checklists list.
func newTaskChecklistResource() resource.Resource {
	return &genericResource{
		name:               "task_checklist",
		createPath:         "/v2/task/{task_id}/checklist",
		readPath:           "/v2/task/{task_id}/checklist",
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
		readResponseRoot:   "checklist",
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
func newChecklistItemResource() resource.Resource {
	return &genericResource{
		name:               "checklist_item",
		createPath:         "/v2/checklist/{checklist_id}/checklist_item",
		readPath:           "/v2/checklist/{checklist_id}/checklist_item",
		updatePath:         "/v2/checklist/{checklist_id}/checklist_item/{checklist_item_id}",
		deletePath:         "/v2/checklist/{checklist_id}/checklist_item/{checklist_item_id}",
		updateMethod:       "put",
		createBodyFields:   []string{"name", "assignee"},
		updateBodyFields:   []string{"name", "assignee", "resolved", "parent"},
		idField:            "checklist_item_id",
		readFromList:       true,
		readListRoot:       "items",
		readListIDField:    "id",
		createResponseRoot: "item",
		readResponseRoot:   "item",
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
		readListRoot:       "key_results",
		readListIDField:    "id",
		createResponseRoot: "key_result",
		readResponseRoot:   "key_result",
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
		readFromList:     true,
		readListRoot:     "data",
		readListIDField:  "id",
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
// POST   /v2/team/{team_id}/time_entries/tags (create)
// PUT    /v2/team/{team_id}/time_entries/tags (update by name)
// GET    /v2/team/{team_id}/time_entries/tags (list)
// DELETE /v2/team/{team_id}/time_entries/tags (delete by name)
func newTimeEntryTagResource() resource.Resource {
	return &genericResource{
		name:             "time_entry_tag",
		createPath:       "/v2/team/{team_id}/time_entries/tags",
		readPath:         "/v2/team/{team_id}/time_entries/tags",
		updatePath:       "/v2/team/{team_id}/time_entries/tags",
		deletePath:       "/v2/team/{team_id}/time_entries/tags",
		updateMethod:     "put",
		createBodyFields: []string{"name", "tag_bg", "tag_fg"},
		updateBodyFields: []string{"name", "new_name", "tag_bg", "tag_fg"},
		idField:          "tag_name",
		readFromList:     true,
		readListRoot:     "tags",
		readListIDField:  "name",
		idFromBody:       []string{"name"},
		schemaFunc:       timeEntryTagSchema,
	}
}

func timeEntryTagSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a time tracking tag on a ClickUp team.",
		Attributes: map[string]schema.Attribute{
			"tag_name": schema.StringAttribute{
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
			"name": schema.StringAttribute{
				Required: true,
			},
			"new_name": schema.StringAttribute{
				Optional: true,
			},
			"tag_bg": schema.StringAttribute{
				Required: true,
			},
			"tag_fg": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// --- team_guest ---
// POST   /v2/team/{team_id}/guest (create/invite)
// GET    /v2/team/{team_id}/guest/{guest_id} (read)
// PUT    /v2/team/{team_id}/guest/{guest_id} (update)
// DELETE /v2/team/{team_id}/guest/{guest_id} (delete)
func newTeamGuestResource() resource.Resource {
	return &genericResource{
		name:             "team_guest",
		createPath:       "/v2/team/{team_id}/guest",
		readPath:         "/v2/team/{team_id}/guest/{guest_id}",
		updatePath:       "/v2/team/{team_id}/guest/{guest_id}",
		deletePath:       "/v2/team/{team_id}/guest/{guest_id}",
		updateMethod:     "put",
		createBodyFields: []string{"email", "can_edit_tags", "can_see_time_spent", "can_see_time_estimated", "can_create_views", "can_see_points_estimated", "custom_role_id"},
		updateBodyFields: []string{"can_edit_tags", "can_see_time_spent", "can_see_time_estimated", "can_create_views", "can_see_points_estimated", "custom_role_id"},
		idField:          "guest_id",
		schemaFunc:       teamGuestSchema,
	}
}

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
// POST   /v2/team/{team_id}/user (create/invite)
// GET    /v2/team/{team_id}/user/{user_id} (read)
// PUT    /v2/team/{team_id}/user/{user_id} (update)
// DELETE /v2/team/{team_id}/user/{user_id} (delete)
func newTeamUserResource() resource.Resource {
	return &genericResource{
		name:             "team_user",
		createPath:       "/v2/team/{team_id}/user",
		readPath:         "/v2/team/{team_id}/user/{user_id}",
		updatePath:       "/v2/team/{team_id}/user/{user_id}",
		deletePath:       "/v2/team/{team_id}/user/{user_id}",
		updateMethod:     "put",
		createBodyFields: []string{"email", "admin", "custom_role_id"},
		updateBodyFields: []string{"username", "admin", "custom_role_id"},
		idField:          "user_id",
		schemaFunc:       teamUserSchema,
	}
}

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
// POST /v2/team/{team_id}/view (create)
// GET/PUT/DELETE /v2/view/{view_id} (read/update/delete)
func newTeamViewResource() resource.Resource {
	return &genericResource{
		name:               "team_view",
		createPath:         "/v2/team/{team_id}/view",
		readPath:           "/v2/view/{view_id}",
		updatePath:         "/v2/view/{view_id}",
		deletePath:         "/v2/view/{view_id}",
		updateMethod:       "put",
		createBodyFields:   []string{"name", "type", "grouping", "divide", "sorting", "filters", "columns", "team_sidebar", "settings"},
		updateBodyFields:   []string{"name", "type", "grouping", "divide", "sorting", "filters", "columns", "team_sidebar", "settings"},
		idField:            "view_id",
		createResponseRoot: "view",
		readResponseRoot:   "view",
		schemaFunc:         teamViewSchema,
	}
}

func teamViewSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a team-level view in ClickUp.",
		Attributes: map[string]schema.Attribute{
			"view_id": schema.StringAttribute{
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
			"name": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required: true,
			},
		},
	}
}
