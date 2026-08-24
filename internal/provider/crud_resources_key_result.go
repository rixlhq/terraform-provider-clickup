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
