package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// manualDataSourceFactories contains data sources whose generated schemas are
// not usable (due to untyped/union responses in the OpenAPI spec). They expose
// the raw JSON response instead of dropping the route.
//
//nolint:goconst // repeated parameter names are intentional Terraform attribute names
var manualDataSourceFactories = []func() datasource.DataSource{
	newRawJSONDataSource("folder", "/v2/folder/{folder_id}", []string{"folder_id"}),
	newRawJSONDataSource("goals", "/v2/team/{team_id}/goal", []string{"team_id"}),
	newRawJSONDataSource("task", "/v2/task/{task_id}", []string{"task_id"},
		rawQueryParam{name: "custom_task_ids"},
		rawQueryParam{name: "team_id"},
	),
	newRawJSONDataSource("view", "/v2/view/{view_id}", []string{"view_id"},
		rawQueryParam{name: "page"},
	),
	newRawJSONDataSource("view_tasks", "/v2/view/{view_id}/task", []string{"view_id"}),
	newRawJSONDataSource("task_s_timein_status", "/v2/task/{task_id}/time_in_status", []string{"task_id"},
		rawQueryParam{name: "custom_task_ids"},
		rawQueryParam{name: "team_id"},
	),
	newRawJSONDataSource("bulk_tasks_timein_status", "/v2/task/bulk_time_in_status", nil,
		rawQueryParam{name: "task_ids", list: true, required: true, brackets: false},
		rawQueryParam{name: "custom_task_ids"},
		rawQueryParam{name: "team_id"},
	),
	newRawJSONDataSource("time_entries", "/v2/team/{team_id}/time_entries", []string{"team_id"},
		rawQueryParam{name: "start_date"},
		rawQueryParam{name: "end_date"},
		rawQueryParam{name: "assignee"},
		rawQueryParam{name: "include_task_tags"},
		rawQueryParam{name: "include_location_names"},
		rawQueryParam{name: "include_approval_history"},
		rawQueryParam{name: "include_approval_details"},
		rawQueryParam{name: "space_id"},
		rawQueryParam{name: "folder_id"},
		rawQueryParam{name: "list_id"},
		rawQueryParam{name: "task_id"},
		rawQueryParam{name: "custom_task_ids"},
		rawQueryParam{name: "is_billable"},
	),
}
