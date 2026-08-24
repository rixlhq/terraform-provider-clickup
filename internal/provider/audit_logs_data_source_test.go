package provider_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
)

func TestAccAuditLogs_basic(t *testing.T) {
	ts := clickupclient.NewTestServer()
	defer ts.Close()

	// ogen serializes the float64 workspace ID as "123.0000000000" in the path.
	ts.Register("POST", "/api/v3/workspaces/123.0000000000/auditlogs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"data":[{"id":"log-1","action":"login","user_id":"u1"}],"next_cursor":""}`)
	})

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
data "clickup_audit_logs" "test" {
  workspace_id = "123"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.clickup_audit_logs.test", "workspace_id", "123"),
					resource.TestCheckResourceAttrSet("data.clickup_audit_logs.test", "result"),
				),
			},
		},
	})
}
