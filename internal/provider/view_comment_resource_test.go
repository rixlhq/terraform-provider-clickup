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

//nolint:funlen,goconst // acceptance test uses repeated Terraform/map literals
func TestAccViewComment_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	commentText := "hello"
	resolved := false

	ts.Register("POST", "/v2/view/123/comment", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"id":           "vc1",
			"comment_text": commentText,
			"notify_all":   false,
			"resolved":     resolved,
			"date":         1234567890000,
		})
		_, _ = w.Write(body)
	})
	ts.Register("GET", "/v2/view/123/comment", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"comments": []any{
				map[string]any{
					"id":           "vc1",
					"comment_text": commentText,
					"notify_all":   false,
					"resolved":     resolved,
					"date":         1234567890000,
				},
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/comment/vc1", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if text, ok := payload["comment_text"].(string); ok {
				commentText = text
			}
			if r, ok := payload["resolved"].(bool); ok {
				resolved = r
			}
		}
		w.WriteHeader(http.StatusOK)
		body, _ = json.Marshal(map[string]any{
			"id":           "vc1",
			"comment_text": commentText,
			"resolved":     resolved,
			"date":         1234567890000,
		})
		_, _ = w.Write(body)
	})
	ts.Register("DELETE", "/v2/comment/vc1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	baseConfig := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
`, ts.URL)

	createConfig := baseConfig + `
resource "clickup_view_comment" "test" {
  view_id      = "123"
  comment_text = "hello"
  notify_all   = false
}
`
	updateConfig := baseConfig + `
resource "clickup_view_comment" "test" {
  view_id      = "123"
  comment_text = "hello2"
  notify_all   = false
  resolved     = true
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_view_comment.test", "comment_id", "vc1"),
					resource.TestCheckResourceAttr("clickup_view_comment.test", "comment_text", "hello"),
					resource.TestCheckResourceAttr("clickup_view_comment.test", "resolved", "false"),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_view_comment.test", "comment_text", "hello2"),
					resource.TestCheckResourceAttr("clickup_view_comment.test", "resolved", "true"),
				),
			},
		},
	})
}
