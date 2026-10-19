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
