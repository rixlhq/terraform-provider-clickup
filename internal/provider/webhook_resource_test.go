package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

//nolint:funlen
func TestAccWebhook_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	endpoint := "https://example.com/webhook"
	events := []any{"taskCreated"}
	status := "active"

	ts.Register("POST", "/v2/team/123/webhook", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"wh-1"}`))
	})
	ts.Register("GET", "/v2/team/123/webhook", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{
			"webhooks": []any{
				map[string]any{
					"id":        "wh-1",
					"endpoint":  endpoint,
					"events":    events,
					"status":    status,
					"health":    map[string]any{"status": status, "fail_count": 0},
					"team_id":   123,
					"userid":    1,
					"client_id": "cid",
					"secret":    "shh",
					"space_id":  "123",
				},
			},
		})
		_, _ = w.Write(body)
	})
	ts.Register("PUT", "/v2/webhook/wh-1", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err == nil {
			if e, ok := payload["endpoint"].(string); ok {
				endpoint = e
			}
			if s, ok := payload["status"].(string); ok {
				status = s
			}
			if ev, ok := payload["events"].(string); ok {
				if ev == "*" {
					events = []any{"*"}
				} else {
					events = make([]any, 0)
					for part := range strings.SplitSeq(ev, ",") {
						if part != "" {
							events = append(events, part)
						}
					}
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	ts.Register("DELETE", "/v2/webhook/wh-1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	baseConfig := fmt.Sprintf(`
provider "clickup" {
  base_url  = "%s"
  api_token = "pk_test"
}
`, ts.URL)

	createConfig := baseConfig + `
resource "clickup_webhook" "test" {
  team_id  = 123
  endpoint = "https://example.com/webhook"
  events   = ["taskCreated"]
  space_id = "123"
}
`
	updateConfig := baseConfig + `
resource "clickup_webhook" "test" {
  team_id  = 123
  endpoint = "https://example.com/webhook2"
  events   = ["taskUpdated"]
  status   = "paused"
  space_id = "123"
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_webhook.test", "webhook_id", "wh-1"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "endpoint", "https://example.com/webhook"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "events.0", "taskCreated"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "status", "active"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "user_id", "1"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "secret", "shh"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "space_id", "123"),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_webhook.test", "webhook_id", "wh-1"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "endpoint", "https://example.com/webhook2"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "events.0", "taskUpdated"),
					resource.TestCheckResourceAttr("clickup_webhook.test", "status", "paused"),
				),
			},
		},
	})
}
