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

func TestAccFolder_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	folderName := "test"

	ts.Register("POST", "/v2/space/123/folder", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"456","name":"test","space":{"id":"123"}}`))
	})
	ts.Register("GET", "/v2/folder/456", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fmt.Appendf(nil, `{"id":"456","name":"%s","space":{"id":"123"}}`, folderName))
	})
	ts.Register("PUT", "/v2/folder/456", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if name, ok := payload["name"].(string); ok {
				folderName = name
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fmt.Appendf(nil, `{"id":"456","name":"%s","space":{"id":"123"}}`, folderName))
	})
	ts.Register("DELETE", "/v2/folder/456", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	baseConfig := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
`, ts.URL)

	createConfig := baseConfig + `
resource "clickup_folder" "test" {
  space_id = "123"
  name     = "test"
}
`
	updateConfig := baseConfig + `
resource "clickup_folder" "test" {
  space_id = "123"
  name     = "renamed"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_folder.test", "name", "test"),
					resource.TestCheckResourceAttr("clickup_folder.test", "folder_id", "456"),
					resource.TestCheckResourceAttr("clickup_folder.test", "space_id", "123"),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_folder.test", "name", "renamed"),
					resource.TestCheckResourceAttr("clickup_folder.test", "folder_id", "456"),
				),
			},
		},
	})
}
