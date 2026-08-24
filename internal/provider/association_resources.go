//nolint:goconst // Terraform attribute names repeated across schemas.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// --- task_dependency ---
// POST/DELETE /v2/task/{task_id}/dependency
func newTaskDependencyResource() resource.Resource {
	return &associationResource{
		name:       "task_dependency",
		createPath: "/v2/task/{task_id}/dependency",
		deletePath: "/v2/task/{task_id}/dependency",
		bodyKeyMap: map[string]string{"depends_on_id": "depends_on", "dependency_of": "dependency_of", "type": "type"},
		schemaFunc: taskDependencySchema,
	}
}

func taskDependencySchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a dependency between two ClickUp tasks. Use `depends_on_id` to make this task depend on another, or `dependency_of` to mark this task as a dependency of another.",
		Attributes: map[string]schema.Attribute{
			"task_id": requiresReplaceString(),
			"depends_on_id": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"dependency_of": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// --- task_link ---
// POST/DELETE /v2/task/{task_id}/link/{links_to}
func newTaskLinkResource() resource.Resource {
	return &associationResource{
		name:       "task_link",
		createPath: "/v2/task/{task_id}/link/{links_to}",
		deletePath: "/v2/task/{task_id}/link/{links_to}",
		schemaFunc: taskLinkSchema,
	}
}

func taskLinkSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Links two ClickUp tasks together.",
		Attributes: map[string]schema.Attribute{
			"task_id":  requiresReplaceString(),
			"links_to": requiresReplaceString(),
		},
	}
}

// --- task_tag ---
// POST/DELETE /v2/task/{task_id}/tag/{tag_name}
func newTaskTagResource() resource.Resource {
	return &associationResource{
		name:       "task_tag",
		createPath: "/v2/task/{task_id}/tag/{tag_name}",
		deletePath: "/v2/task/{task_id}/tag/{tag_name}",
		schemaFunc: taskTagSchema,
	}
}

func taskTagSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Adds a tag to a ClickUp task.",
		Attributes: map[string]schema.Attribute{
			"task_id":  requiresReplaceString(),
			"tag_name": requiresReplaceString(),
		},
	}
}

// --- task_custom_field ---
// POST/DELETE /v2/task/{task_id}/field/{field_id}
func newTaskCustomFieldResource() resource.Resource {
	return &associationResource{
		name:       "task_custom_field",
		createPath: "/v2/task/{task_id}/field/{field_id}",
		deletePath: "/v2/task/{task_id}/field/{field_id}",
		bodyKeyMap: map[string]string{"value": "value"},
		schemaFunc: taskCustomFieldSchema,
	}
}

func taskCustomFieldSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Sets a custom field value on a ClickUp task.",
		Attributes: map[string]schema.Attribute{
			"task_id":  requiresReplaceString(),
			"field_id": requiresReplaceString(),
			"value": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// --- task_guest ---
// POST/DELETE /v2/task/{task_id}/guest/{guest_id}
func newTaskGuestResource() resource.Resource {
	return &associationResource{
		name:       "task_guest",
		createPath: "/v2/task/{task_id}/guest/{guest_id}",
		deletePath: "/v2/task/{task_id}/guest/{guest_id}",
		bodyKeyMap: map[string]string{"permission_level": "permission_level"},
		schemaFunc: taskGuestSchema,
	}
}

func taskGuestSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Grants a guest access to a ClickUp task.",
		Attributes: map[string]schema.Attribute{
			"task_id":  requiresReplaceString(),
			"guest_id": requiresReplaceString(),
			"permission_level": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// --- folder_guest ---
// POST/DELETE /v2/folder/{folder_id}/guest/{guest_id}
func newFolderGuestResource() resource.Resource {
	return &associationResource{
		name:       "folder_guest",
		createPath: "/v2/folder/{folder_id}/guest/{guest_id}",
		deletePath: "/v2/folder/{folder_id}/guest/{guest_id}",
		bodyKeyMap: map[string]string{"permission_level": "permission_level"},
		schemaFunc: folderGuestSchema,
	}
}

func folderGuestSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Grants a guest access to a ClickUp folder.",
		Attributes: map[string]schema.Attribute{
			"folder_id": requiresReplaceString(),
			"guest_id":  requiresReplaceString(),
			"permission_level": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// --- list_guest ---
// POST/DELETE /v2/list/{list_id}/guest/{guest_id}
func newListGuestResource() resource.Resource {
	return &associationResource{
		name:       "list_guest",
		createPath: "/v2/list/{list_id}/guest/{guest_id}",
		deletePath: "/v2/list/{list_id}/guest/{guest_id}",
		bodyKeyMap: map[string]string{"permission_level": "permission_level"},
		schemaFunc: listGuestSchema,
	}
}

func listGuestSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Grants a guest access to a ClickUp list.",
		Attributes: map[string]schema.Attribute{
			"list_id":  requiresReplaceString(),
			"guest_id": requiresReplaceString(),
			"permission_level": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// --- list_task ---
// POST/DELETE /v2/list/{list_id}/task/{task_id}
func newListTaskResource() resource.Resource {
	return &associationResource{
		name:       "list_task",
		createPath: "/v2/list/{list_id}/task/{task_id}",
		deletePath: "/v2/list/{list_id}/task/{task_id}",
		schemaFunc: listTaskSchema,
	}
}

func listTaskSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Adds a task to a ClickUp list.",
		Attributes: map[string]schema.Attribute{
			"list_id": requiresReplaceString(),
			"task_id": requiresReplaceString(),
		},
	}
}
