# Terraform Provider for ClickUp

A Terraform provider for managing resources in [ClickUp](https://clickup.com) via the public API v3.

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
`base_url` attribute (or `CLICKUP_BASE_URL`) can be used to point at a different
ClickUp API endpoint.

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

## License

[MIT](./LICENSE)
