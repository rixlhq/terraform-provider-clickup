package providerdata

import "github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"

// Data holds the API client configured by the provider and passed to resources/data sources.
type Data struct {
	ClickUp *clickupclient.Client
}
