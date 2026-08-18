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
func TestAccSpaceTag_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	tagName := "blocked"
	tagFG := "#ffffff"
	tagBG := "#ff0000"

	ts.Register("POST", "/v2/space/123/tag", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	ts.Register("GET", "/v2/space/123/tag", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fmt.Appendf(nil, `{"tags":[{"name":"%s","tag_fg":"%s","tag_bg":"%s"}]}`, tagName, tagFG, tagBG))
	})
	ts.Register("PUT", "/v2/space/123/tag/blocked", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if tag, ok := payload["tag"].(map[string]any); ok {
				if name, ok := tag["name"].(string); ok {
					tagName = name
				}
				if fg, ok := tag["fg_color"].(string); ok {
					tagFG = fg
				}
				if bg, ok := tag["bg_color"].(string); ok {
					tagBG = bg
				}
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	ts.Register("DELETE", "/v2/space/123/tag/blocked2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	baseConfig := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
`, ts.URL)

	createConfig := baseConfig + `
resource "clickup_space_tag" "test" {
  space_id = "123"
  tag = {
    name   = "blocked"
    tag_fg = "#ffffff"
    tag_bg = "#ff0000"
  }
}
`
	updateConfig := baseConfig + `
resource "clickup_space_tag" "test" {
  space_id = "123"
  tag = {
    name   = "blocked2"
    tag_fg = "#000000"
    tag_bg = "#0000ff"
  }
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag_name", "blocked"),
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag.name", "blocked"),
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag.tag_fg", "#ffffff"),
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag.tag_bg", "#ff0000"),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag_name", "blocked2"),
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag.name", "blocked2"),
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag.tag_fg", "#000000"),
					resource.TestCheckResourceAttr("clickup_space_tag.test", "tag.tag_bg", "#0000ff"),
				),
			},
		},
	})
}
