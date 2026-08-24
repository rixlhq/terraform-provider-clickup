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
	krCurrent := 0
	var krMu sync.Mutex
	ts.Register("GET", "/v2/goal/123", func(w http.ResponseWriter, _ *http.Request) {
		krMu.Lock()
		current := krCurrent
		krMu.Unlock()
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
						"steps_current": current,
						"unit":          "%",
					},
				},
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/key_result/kr1", func(w http.ResponseWriter, _ *http.Request) {
		krMu.Lock()
		krCurrent = 75
		krMu.Unlock()
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
