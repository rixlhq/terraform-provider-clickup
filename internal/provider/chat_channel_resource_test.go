package provider_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

func TestAccChatChannel_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	base := "/api/v3/workspaces/123/chat/channels"
	var mu sync.Mutex
	currentName := "my-channel"
	currentDesc := "test"

	channelJSON := func() string {
		mu.Lock()
		defer mu.Unlock()
		return fmt.Sprintf(`{"data":{"id":"ch-1","name":%q,"description":%q,"topic":null,"visibility":"PUBLIC","type":"CHANNEL","parent":{"id":"","type":0},"creator":"u1","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","workspace_id":"123","archived":false,"links":{"members":"","followers":""}}}`, currentName, currentDesc)
	}

	registerChatChannelHandlers(ts, base, channelJSON)

	providerConfig := fmt.Sprintf(`
provider "clickup" {
  api_token   = "pk_test"
  base_url    = "%s"
  v3_base_url = "%s"
}
`, ts.URL, ts.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "clickup_chat_channel" "test" {
  workspace_id = "123"
  name         = "my-channel"
  description  = "test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_chat_channel.test", "channel_id", "ch-1"),
					resource.TestCheckResourceAttr("clickup_chat_channel.test", "name", "my-channel"),
					resource.TestCheckResourceAttr("clickup_chat_channel.test", "visibility", "PUBLIC"),
					resource.TestCheckResourceAttr("clickup_chat_channel.test", "type", "CHANNEL"),
				),
			},
			{
				PreConfig: func() {
					mu.Lock()
					currentName = "updated-channel"
					currentDesc = "updated"
					mu.Unlock()
				},
				Config: providerConfig + `
resource "clickup_chat_channel" "test" {
  workspace_id = "123"
  name         = "updated-channel"
  description  = "updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("clickup_chat_channel.test", "channel_id", "ch-1"),
					resource.TestCheckResourceAttr("clickup_chat_channel.test", "name", "updated-channel"),
					resource.TestCheckResourceAttr("clickup_chat_channel.test", "description", "updated"),
				),
			},
		},
	})
}

func registerChatChannelHandlers(ts *clickupclient.TestServer, base string, channelJSON func() string) {
	ts.Register("POST", base, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(channelJSON()))
	})
	ts.Register("GET", base+"/ch-1", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(channelJSON()))
	})
	ts.Register("PUT", base+"/ch-1", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(channelJSON()))
	})
	ts.Register("DELETE", base+"/ch-1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}
