package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

//nolint:funlen // acceptance test with mock handlers
func TestAccTaskChecklist_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	name := "my-checklist"

	var clPos int64
	var clMu sync.Mutex
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
	ts.Register("GET", "/v2/task/123", func(w http.ResponseWriter, _ *http.Request) {
		clMu.Lock()
		pos := clPos
		clMu.Unlock()
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"id":   "123",
			"name": "test-task",
			"checklists": []any{
				map[string]any{
					"id":       "cl1",
					"name":     name,
					"position": pos,
				},
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/checklist/cl1", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		clMu.Lock()
		if p, ok := payload["position"].(float64); ok {
			clPos = int64(p)
		}
		clMu.Unlock()
		if n, ok := payload["name"].(string); ok {
			name = n
		}
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"checklist": map[string]any{
				"id":       "cl1",
				"name":     name,
				"position": clPos,
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
