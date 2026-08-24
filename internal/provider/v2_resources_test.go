package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

//nolint:funlen // acceptance test with mock handlers
func TestAccTaskChecklist_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	name := "my-checklist"

	ts.Register("POST", "/v2/task/123/checklist", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"checklist": map[string]any{
				"id":       "cl1",
				"name":     name,
				"position": 0,
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("GET", "/v2/task/123/checklist", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"checklists": []any{
				map[string]any{
					"id":       "cl1",
					"name":     name,
					"position": 0,
				},
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/checklist/cl1", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if n, ok := payload["name"].(string); ok {
			name = n
		}
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"checklist": map[string]any{
				"id":       "cl1",
				"name":     name,
				"position": 0,
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("DELETE", "/v2/checklist/cl1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_task_checklist" "test" {
  task_id = "123"
  name    = "my-checklist"
}
`, ts.URL)

	updateCfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_task_checklist" "test" {
  task_id   = "123"
  name      = "updated-checklist"
  position  = 1
}
`, ts.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_task_checklist.test", "checklist_id", "cl1"),
					resource.TestCheckResourceAttr("clickup_task_checklist.test", "name", "my-checklist"),
				),
			},
			{
				Config: updateCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_task_checklist.test", "checklist_id", "cl1"),
					resource.TestCheckResourceAttr("clickup_task_checklist.test", "name", "updated-checklist"),
				),
			},
		},
	})
}

func TestAccTaskTag_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	ts.Register("POST", "/v2/task/123/tag/red", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ts.Register("DELETE", "/v2/task/123/tag/red", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_task_tag" "test" {
  task_id  = "123"
  tag_name = "red"
}
`, ts.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_task_tag.test", "task_id", "123"),
					resource.TestCheckResourceAttr("clickup_task_tag.test", "tag_name", "red"),
				),
			},
		},
	})
}

func TestAccTaskDependency_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	ts.Register("POST", "/v2/task/123/dependency", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	ts.Register("DELETE", "/v2/task/123/dependency", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_task_dependency" "test" {
  task_id        = "123"
  depends_on_id  = "456"
  type           = "waiting_on"
}
`, ts.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_task_dependency.test", "task_id", "123"),
					resource.TestCheckResourceAttr("clickup_task_dependency.test", "depends_on_id", "456"),
					resource.TestCheckResourceAttr("clickup_task_dependency.test", "type", "waiting_on"),
				),
			},
		},
	})
}

//nolint:funlen // acceptance test with mock handlers
func TestAccTimeEntry_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	desc := "working"
	ts.Register("POST", "/v2/team/123/time_entries", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"id":          "te1",
			"description": desc,
			"start":       1700000000000,
			"end":         1700003600000,
			"duration":    3600000,
			"billable":    false,
			"tags":        []any{},
		})
		_, _ = w.Write(body)
	})
	ts.Register("GET", "/v2/team/123/time_entries/te1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"id":          "te1",
			"description": desc,
			"start":       1700000000000,
			"end":         1700003600000,
			"duration":    3600000,
			"billable":    false,
			"tags":        []any{},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/team/123/time_entries/te1", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if d, ok := payload["description"].(string); ok {
			desc = d
		}
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"id":          "te1",
			"description": desc,
			"start":       1700000000000,
			"end":         1700003600000,
			"duration":    3600000,
			"billable":    false,
			"tags":        []any{},
		})
		_, _ = w.Write(body)
	})
	ts.Register("DELETE", "/v2/team/123/time_entries/te1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_time_entry" "test" {
  team_id     = "123"
  description = "working"
  start       = 1700000000000
  duration    = 3600000
}
`, ts.URL)

	updateCfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_time_entry" "test" {
  team_id     = "123"
  description = "updated work"
  start       = 1700000000000
  duration    = 3600000
}
`, ts.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_time_entry.test", "timer_id", "te1"),
					resource.TestCheckResourceAttr("clickup_time_entry.test", "description", "working"),
				),
			},
			{
				Config: updateCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_time_entry.test", "timer_id", "te1"),
					resource.TestCheckResourceAttr("clickup_time_entry.test", "description", "updated work"),
				),
			},
		},
	})
}

//nolint:funlen // acceptance test with mock handlers
func TestAccKeyResult_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	ts.Register("POST", "/v2/goal/123/key_result", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"key_result": map[string]any{
				"id":            "kr1",
				"name":          "my-kr",
				"type":          "number",
				"steps_start":   0,
				"steps_end":     100,
				"steps_current": 0,
				"unit":          "%",
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("GET", "/v2/goal/123", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"goal": map[string]any{
				"id": "123",
				"key_results": []any{
					map[string]any{
						"id":            "kr1",
						"name":          "my-kr",
						"type":          "number",
						"steps_start":   0,
						"steps_end":     100,
						"steps_current": 50,
						"unit":          "%",
					},
				},
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/key_result/kr1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"key_result": map[string]any{
				"id":            "kr1",
				"name":          "my-kr",
				"steps_current": 75,
				"note":          "progress",
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("DELETE", "/v2/key_result/kr1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_key_result" "test" {
  goal_id     = "123"
  name        = "my-kr"
  type        = "number"
  owners      = [1]
  steps_start = 0
  steps_end   = 100
  unit        = "%%"
}
`, ts.URL)

	updateCfg := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
resource "clickup_key_result" "test" {
  goal_id       = "123"
  name          = "my-kr"
  type          = "number"
  owners        = [1]
  steps_start   = 0
  steps_end     = 100
  unit          = "%%"
  steps_current = 75
  note          = "progress"
}
`, ts.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_key_result.test", "key_result_id", "kr1"),
					resource.TestCheckResourceAttr("clickup_key_result.test", "name", "my-kr"),
				),
			},
			{
				Config: updateCfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_key_result.test", "key_result_id", "kr1"),
					resource.TestCheckResourceAttr("clickup_key_result.test", "note", "progress"),
				),
			},
		},
	})
}
