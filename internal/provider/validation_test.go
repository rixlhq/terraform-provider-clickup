package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccValidation_WebhookInvalidEndpoint(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "clickup" {
  base_url  = "https://example.com/api"
  api_token = "pk_test"
}

resource "clickup_webhook" "test" {
  team_id  = 123
  endpoint = "ftp://invalid-url"
  events   = ["taskCreated"]
}
`,
				ExpectError: regexp.MustCompile(`must be a valid URL starting with http:// or https://`),
			},
		},
	})
}

func TestAccValidation_TaskDependencyConflict(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "clickup" {
  base_url  = "https://example.com/api"
  api_token = "pk_test"
}

resource "clickup_task_dependency" "test" {
  task_id       = "task-1"
  depends_on_id = "task-2"
  dependency_of = "task-3"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

func TestAccValidation_ChatChannelInvalidVisibility(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "clickup" {
  base_url  = "https://example.com/api"
  api_token = "pk_test"
}

resource "clickup_chat_channel" "test" {
  workspace_id = "ws-1"
  name         = "general"
  visibility   = "INVALID_VISIBILITY"
}
`,
				ExpectError: regexp.MustCompile(`Attribute visibility value must be one of: \["PUBLIC" "PRIVATE"\]`),
			},
		},
	})
}

func TestAccValidation_WebhookProtocolFreeEndpoint(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "clickup" {
  base_url  = "https://example.com/api"
  api_token = "pk_test"
}

resource "clickup_webhook" "test" {
  team_id  = 123
  endpoint = "hooks.example.com/clickup"
  events   = ["taskCreated"]
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
