package providerdata

import (
	"github.com/rixlhq/terraform-provider-clickup/internal/clickupclient"
	"github.com/rixlhq/terraform-provider-clickup/internal/clickupv3"
)

// Data holds the API clients configured by the provider and passed to
// resources/data sources. ClickUp is the V2 dynamic client; ClickUpV3 is the
// generated typed client for the V3 API surface (chat, docs, audit logs).
type Data struct {
	ClickUp   *clickupclient.Client
	ClickUpV3 *clickupv3.Client
}
