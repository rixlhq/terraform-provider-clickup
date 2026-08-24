package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
// No GET on checklist_item. The task GET returns checklists[].items[],
// a nested array the generic list reader cannot traverse.
// Read returns state as-is; drift is detected on the next plan.
// Create response wraps the item inside { "checklist": { "items": [...] } },
// so we use idFromBody to extract from checklist.items[].id (last entry).
func newChecklistItemResource() resource.Resource {
	return &genericResource{
		name:                    "checklist_item",
		createPath:              "/v2/checklist/{checklist_id}/checklist_item",
		readPath:                "/v2/checklist/{checklist_id}/checklist_item",
		updatePath:              "/v2/checklist/{checklist_id}/checklist_item/{checklist_item_id}",
		deletePath:              "/v2/checklist/{checklist_id}/checklist_item/{checklist_item_id}",
		updateMethod:            "put",
		createBodyFields:        []string{"name", "assignee"},
		updateBodyFields:        []string{"name", "assignee", "resolved", "parent"},
		idField:                 "checklist_item_id",
		createResponseRoot:      "checklist",
		createResponseItemArray: "items",
		schemaFunc:              checklistItemSchema,
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
