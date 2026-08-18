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

func TestAccList_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	listName := "test"

	ts.Register("POST", "/v2/folder/123/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fmt.Appendf(nil, `{"id":"456","name":"%s","folder":{"id":"123"},"space":{"id":"789"}}`, listName))
	})
	ts.Register("GET", "/v2/list/456", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fmt.Appendf(nil, `{"id":"456","name":"%s","folder":{"id":"123"},"space":{"id":"789"}}`, listName))
	})
	ts.Register("PUT", "/v2/list/456", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if name, ok := payload["name"].(string); ok {
				listName = name
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fmt.Appendf(nil, `{"id":"456","name":"%s","folder":{"id":"123"},"space":{"id":"789"}}`, listName))
	})
	ts.Register("DELETE", "/v2/list/456", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	baseConfig := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
`, ts.URL)

	createConfig := baseConfig + `
resource "clickup_list" "test" {
  folder_id = "123"
  name      = "test"
}
`
	updateConfig := baseConfig + `
resource "clickup_list" "test" {
  folder_id = "123"
  name      = "renamed"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_list.test", "name", "test"),
					resource.TestCheckResourceAttr("clickup_list.test", "id", "456"),
					resource.TestCheckResourceAttr("clickup_list.test", "folder_id", "123"),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_list.test", "name", "renamed"),
					resource.TestCheckResourceAttr("clickup_list.test", "id", "456"),
				),
			},
		},
	})
}
