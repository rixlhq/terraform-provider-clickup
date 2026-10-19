//nolint:goconst // Terraform attribute names repeated across schemas.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// --- task_dependency ---
// POST/DELETE /v2/task/{task_id}/dependency
// newTaskDependencyResource returns an association resource for task dependencies.
func newTaskDependencyResource() resource.Resource {
	return &associationResource{
		name:              "task_dependency",
		createPath:        "/v2/task/{task_id}/dependency",
		deletePath:        "/v2/task/{task_id}/dependency",
		bodyKeyMap:        map[string]string{"depends_on_id": "depends_on", "dependency_of": "dependency_of"},
		deleteQueryParams: []string{"depends_on_id", "dependency_of"},
		schemaFunc:        taskDependencySchema,
	}
}

// taskDependencySchema returns the schema for the clickup_task_dependency resource.
func taskDependencySchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a dependency between two ClickUp tasks. Use `depends_on_id` to make this task depend on another, or `dependency_of` to mark this task as a dependency of another.",
		Attributes: map[string]schema.Attribute{
			"task_id": requiresReplaceString(),
			"depends_on_id": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("dependency_of")),
				},
			},
			"dependency_of": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("depends_on_id")),
				},
			},
			"type": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

// --- task_link ---
// POST/DELETE /v2/task/{task_id}/link/{links_to}
// newTaskLinkResource returns an association resource for linking tasks.
func newTaskLinkResource() resource.Resource {
	return &associationResource{
		name:       "task_link",
		createPath: "/v2/task/{task_id}/link/{links_to}",
		deletePath: "/v2/task/{task_id}/link/{links_to}",
		schemaFunc: taskLinkSchema,
	}
}

// taskLinkSchema returns the schema for the clickup_task_link resource.
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
// newTaskTagResource returns an association resource for assigning tags to tasks.
func newTaskTagResource() resource.Resource {
	return &associationResource{
		name:       "task_tag",
		createPath: "/v2/task/{task_id}/tag/{tag_name}",
		deletePath: "/v2/task/{task_id}/tag/{tag_name}",
		schemaFunc: taskTagSchema,
	}
}

// taskTagSchema returns the schema for the clickup_task_tag resource.
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
// newTaskCustomFieldResource returns an association resource for setting custom fields on tasks.
func newTaskCustomFieldResource() resource.Resource {
	return &associationResource{
		name:       "task_custom_field",
		createPath: "/v2/task/{task_id}/field/{field_id}",
		deletePath: "/v2/task/{task_id}/field/{field_id}",
		bodyKeyMap: map[string]string{"value": "value"},
		schemaFunc: taskCustomFieldSchema,
	}
}

// taskCustomFieldSchema returns the schema for the clickup_task_custom_field resource.
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
// newTaskGuestResource returns an association resource for managing guest access to a task.
func newTaskGuestResource() resource.Resource {
	return &associationResource{
		name:       "task_guest",
		createPath: "/v2/task/{task_id}/guest/{guest_id}",
		deletePath: "/v2/task/{task_id}/guest/{guest_id}",
		bodyKeyMap: map[string]string{"permission_level": "permission_level"},
		schemaFunc: taskGuestSchema,
	}
}

// taskGuestSchema returns the schema for the clickup_task_guest resource.
func taskGuestSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Grants a guest access to a ClickUp task.",
		Attributes: map[string]schema.Attribute{
			"task_id":  requiresReplaceString(),
			"guest_id": requiresReplaceString(),
			"permission_level": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("read", "comment", "edit", "create"),
				},
			},
		},
	}
}

// --- folder_guest ---
// POST/DELETE /v2/folder/{folder_id}/guest/{guest_id}
// newFolderGuestResource returns an association resource for managing guest access to a folder.
func newFolderGuestResource() resource.Resource {
	return &associationResource{
		name:       "folder_guest",
		createPath: "/v2/folder/{folder_id}/guest/{guest_id}",
		deletePath: "/v2/folder/{folder_id}/guest/{guest_id}",
		bodyKeyMap: map[string]string{"permission_level": "permission_level"},
		schemaFunc: folderGuestSchema,
	}
}

// folderGuestSchema returns the schema for the clickup_folder_guest resource.
func folderGuestSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Grants a guest access to a ClickUp folder.",
		Attributes: map[string]schema.Attribute{
			"folder_id": requiresReplaceString(),
			"guest_id":  requiresReplaceString(),
			"permission_level": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("read", "comment", "edit", "create"),
				},
			},
		},
	}
}

// --- list_guest ---
// POST/DELETE /v2/list/{list_id}/guest/{guest_id}
// newListGuestResource returns an association resource for managing guest access to a list.
func newListGuestResource() resource.Resource {
	return &associationResource{
		name:       "list_guest",
		createPath: "/v2/list/{list_id}/guest/{guest_id}",
		deletePath: "/v2/list/{list_id}/guest/{guest_id}",
		bodyKeyMap: map[string]string{"permission_level": "permission_level"},
		schemaFunc: listGuestSchema,
	}
}

// listGuestSchema returns the schema for the clickup_list_guest resource.
func listGuestSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Grants a guest access to a ClickUp list.",
		Attributes: map[string]schema.Attribute{
			"list_id":  requiresReplaceString(),
			"guest_id": requiresReplaceString(),
			"permission_level": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.OneOf("read", "comment", "edit", "create"),
				},
			},
		},
	}
}

// --- list_task ---
// POST/DELETE /v2/list/{list_id}/task/{task_id}
// newListTaskResource returns an association resource for associating tasks with lists.
func newListTaskResource() resource.Resource {
	return &associationResource{
		name:       "list_task",
		createPath: "/v2/list/{list_id}/task/{task_id}",
		deletePath: "/v2/list/{list_id}/task/{task_id}",
		schemaFunc: listTaskSchema,
	}
}

// listTaskSchema returns the schema for the clickup_list_task resource.
func listTaskSchema(_ context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Adds a task to a ClickUp list.",
		Attributes: map[string]schema.Attribute{
			"list_id": requiresReplaceString(),
			"task_id": requiresReplaceString(),
		},
	}
}
