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
func TestAccUserGroup_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	groupName := "Design"
	members := []any{map[string]any{"id": 1, "username": "one"}, map[string]any{"id": 2, "username": "two"}}

	ts.Register("POST", "/v2/team/123/group", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"id":      "g1",
			"name":    groupName,
			"handle":  "design",
			"members": members,
		})
		_, _ = w.Write(body)
	})
	ts.Register("GET", "/v2/group", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"groups": []any{
				map[string]any{
					"id":      "g1",
					"name":    groupName,
					"handle":  "design",
					"members": members,
				},
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/group/g1", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if name, ok := payload["name"].(string); ok {
				groupName = name
			}
			if m, ok := payload["members"].(map[string]any); ok {
				if add, ok := m["add"].([]any); ok {
					newMembers := make([]any, 0, len(add))
					for _, id := range add {
						newMembers = append(newMembers, map[string]any{"id": id, "username": "user"})
					}
					members = newMembers
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		body, _ = json.Marshal(map[string]any{"id": "g1", "name": groupName, "members": members})
		_, _ = w.Write(body)
	})
	ts.Register("DELETE", "/v2/group/g1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	baseConfig := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
`, ts.URL)

	createConfig := baseConfig + `
resource "clickup_user_group" "test" {
  team_id = "123"
  name    = "Design"
  handle  = "design"
  members = {
    add = [1, 2]
    rem = []
  }
}
`
	updateConfig := baseConfig + `
resource "clickup_user_group" "test" {
  team_id = "123"
  name    = "Renamed"
  handle  = "design"
  members = {
    add = [1, 2, 3]
    rem = []
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_user_group.test", "group_id", "g1"),
					resource.TestCheckResourceAttr("clickup_user_group.test", "name", "Design"),
					resource.TestCheckResourceAttr("clickup_user_group.test", "members.add.#", "2"),
					resource.TestCheckResourceAttr("clickup_user_group.test", "members.add.0", "1"),
					resource.TestCheckResourceAttr("clickup_user_group.test", "members.add.1", "2"),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_user_group.test", "group_id", "g1"),
					resource.TestCheckResourceAttr("clickup_user_group.test", "name", "Renamed"),
					resource.TestCheckResourceAttr("clickup_user_group.test", "members.add.#", "3"),
				),
			},
		},
	})
}
