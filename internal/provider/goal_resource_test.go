package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

//nolint:funlen
func TestAccGoal_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	goalName := "test"

	goalBody := func() string {
		return fmt.Sprintf(`{
  "goal": {
    "id": "456",
    "name": "%s",
    "due_date": "1234567890000",
    "description": "desc",
    "multiple_owners": true,
    "owners": [{"id": 1, "username": "one"}],
    "color": "#32a852",
    "team_id": "123",
    "creator": 1,
    "date_created": "1234567890000",
    "archived": false,
    "private": false,
    "percent_completed": 0
  }
}`, goalName)
	}

	ts.Register("POST", "/v2/team/123/goal", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(goalBody()))
	})
	ts.Register("GET", "/v2/goal/456", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(goalBody()))
	})
	ts.Register("PUT", "/v2/goal/456", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if name, ok := payload["name"].(string); ok {
				goalName = name
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(goalBody()))
	})
	ts.Register("DELETE", "/v2/goal/456", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	baseConfig := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
`, ts.URL)

	createConfig := baseConfig + `
resource "clickup_goal" "test" {
  team_id         = "123"
  name            = "test"
  due_date        = 1234567890000
  description     = "desc"
  multiple_owners = true
  owners          = [1]
  color           = "#32a852"
}
`

	updateConfig := baseConfig + `
resource "clickup_goal" "test" {
  team_id         = "123"
  name            = "renamed"
  due_date        = 1234567890000
  description     = "desc"
  multiple_owners = true
  owners          = [1]
  color           = "#32a852"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_goal.test", "name", "test"),
					resource.TestCheckResourceAttr("clickup_goal.test", "goal_id", "456"),
					resource.TestCheckResourceAttr("clickup_goal.test", "color", "#32a852"),
					resource.TestCheckResourceAttr("clickup_goal.test", "owners.#", "1"),
					resource.TestCheckResourceAttr("clickup_goal.test", "owners.0", "1"),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_goal.test", "name", "renamed"),
					resource.TestCheckResourceAttr("clickup_goal.test", "goal_id", "456"),
				),
			},
		},
	})
}
