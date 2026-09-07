# Terraform Provider for ClickUp

A Terraform provider for managing resources in [ClickUp](https://clickup.com) via the ClickUp public APIs (V2 and V3).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.26
- [mise](https://mise.jdx.dev/) (recommended for running tasks and installing tools)

## Provider Configuration

```hcl
terraform {
  required_providers {
    clickup = {
      source  = "rixlhq/clickup"
      version = "~> 0.1"
    }
  }
}

provider "clickup" {
  api_token = var.clickup_api_token
}
```

The `api_token` can be set via the `CLICKUP_API_TOKEN` environment variable. The
`base_url` attribute (or `CLICKUP_BASE_URL`) overrides the V2 API endpoint,
and `v3_base_url` (or `CLICKUP_V3_BASE_URL`) overrides the V3 API endpoint.

## Development

This project uses [mise](https://mise.jdx.dev/) to manage tools and tasks.

Install tools and set up git hooks:

```sh
mise install
```

Run the default local checks:

```sh
mise run
```

Generate provider code from the OpenAPI specification:

```sh
mise run generate
```

Build the provider:

```sh
mise run build
```

Run unit tests (mock-server acceptance tests included with `TF_ACC=1`):

```sh
mise run test
TF_ACC=1 go test ./...
```

Check API coverage against the OpenAPI spec:

```sh
python3 tools/audit_coverage.py ClickUp_PUBLIC_API_V2.prepared.json
```

## License

[MIT](./LICENSE)
