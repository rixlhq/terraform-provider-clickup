terraform {
  required_providers {
    clickup = {
      source  = "rixlhq/clickup"
      version = "~> 1.0"
    }
  }
}

provider "clickup" {
  api_token = var.clickup_api_token
}
