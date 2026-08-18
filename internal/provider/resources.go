package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// resourceFactories returns all ClickUp resources implemented by this provider.
var resourceFactories = []func() resource.Resource{
	newChatChannelResource,
}
