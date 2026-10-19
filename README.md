<div align="center">

# Terraform Provider for ClickUp

[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-rixlhq%2Fclickup-623CE4?logo=terraform&logoColor=white)](https://registry.terraform.io/providers/rixlhq/clickup/latest)
[![Release](https://img.shields.io/github/v/release/rixlhq/terraform-provider-clickup?sort=semver)](https://github.com/rixlhq/terraform-provider-clickup/releases)
[![CI](https://github.com/rixlhq/terraform-provider-clickup/actions/workflows/ci.yml/badge.svg)](https://github.com/rixlhq/terraform-provider-clickup/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Manage ClickUp Workspaces, Spaces, Lists, Tasks, Webhooks, Goals, and Chat Channels as Code.**

[Documentation](https://registry.terraform.io/providers/rixlhq/clickup/latest/docs) | [Terraform Registry](https://registry.terraform.io/providers/rixlhq/clickup/latest) | [ClickUp API Docs](https://developer.clickup.com/) | [Report an Issue](https://github.com/rixlhq/terraform-provider-clickup/issues)

</div>

---

## Overview

The **ClickUp Terraform Provider** allows DevOps, platform engineers, and workspace administrators to manage and automate [ClickUp](https://clickup.com) organizations declaratively using **HashiCorp Terraform** and **OpenTofu**.

Define your workspace hierarchy, project templates, permission assignments, event-driven webhooks, and team communication channels in version-controlled Infrastructure as Code (IaC).

### Key Features

- **Protocol v6 & Plugin Framework Native**: Built on HashiCorp's `terraform-plugin-framework` for robust state handling without tri-state zero-value ambiguities.
- **Dual API Architecture**: Fast Go REST client for [ClickUp API V2](https://developer.clickup.com/) combined with a strictly typed client for ClickUp API V3 (e.g. Chat Channels).
- **Plan-Time Validation**: Early validation of attribute constraints, regex formats (endpoints, emails), mutual exclusivity, and allowed enum values before `apply`.
- **Structured Error Reporting**: Direct visibility into ClickUp validation errors and `ECODE` failure codes without raw JSON dumps.
- **Full Hierarchy Support**: Spaces, Folders, Lists, Tasks, Views, Goals, Key Results, Checklists, Time Tracking, and Webhooks.

---

## Getting Started

### Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 (or OpenTofu >= 1.6)
- ClickUp Personal API Token (or OAuth token)

### Provider Configuration

Add the provider to your Terraform configuration:

```terraform
terraform {
  required_version = ">= 1.0"
  required_providers {
    clickup = {
      source  = "rixlhq/clickup"
      version = "~> 1.0"
    }
  }
}

# Configure the provider with your ClickUp API token
provider "clickup" {
  api_token = var.clickup_api_token
}
```

### Authentication

Set your ClickUp Personal API Token via the `CLICKUP_API_TOKEN` environment variable:

```sh
export CLICKUP_API_TOKEN="pk_1234567_abcdef..."
```

To generate a personal API token in ClickUp:
1. Log into ClickUp and open **Settings**.
2. Navigate to **Apps** > **API Token**.
3. Click **Generate** and copy your token.

#### Optional Endpoint Overrides

For enterprise environments or testing:

| Argument | Environment Variable | Default | Description |
|:---|:---|:---|:---|
| `api_token` | `CLICKUP_API_TOKEN` | — | ClickUp API token (Sensitive) |
| `base_url` | `CLICKUP_BASE_URL` | `https://api.clickup.com/api` | ClickUp V2 REST API endpoint |
| `v3_base_url` | `CLICKUP_V3_BASE_URL` | `https://api.clickup.com/` | ClickUp V3 REST API endpoint |

---

## Example Usage

### Provisioning a Project Workspace

```terraform
# Fetch current authorized team / workspace
data "clickup_authorized_teams" "current" {}

locals {
  team_id = data.clickup_authorized_teams.current.teams[0].id
}

# 1. Create a Space with project features
resource "clickup_space" "engineering" {
  team_id  = local.team_id
  name     = "Engineering"
  private  = false
  features = {
    due_dates = {
      enabled    = true
      start_date = true
      reminders  = true
    }
    time_tracking = {
      enabled = true
    }
    checklists = {
      enabled = true
    }
    custom_fields = {
      enabled = true
    }
  }
}

# 2. Create a Folder
resource "clickup_folder" "sprints" {
  space_id = clickup_space.engineering.id
  name     = "Sprint Management"
}

# 3. Create a List inside the Folder
resource "clickup_list" "sprint_active" {
  folder_id = clickup_folder.sprints.id
  name      = "Sprint 42"
  content   = "Active sprint items and deliverables"
}

# 4. Create a Task
resource "clickup_task" "infra_task" {
  list_id     = clickup_list.sprint_active.id
  name        = "Automate Infrastructure"
  description = "Provision cloud infrastructure and CI/CD pipelines"
  priority    = 1 # 1: Urgent, 2: High, 3: Normal, 4: Low
  status      = "to do"
}

# 5. Create a Webhook for automated event notifications
resource "clickup_webhook" "task_alerts" {
  team_id  = local.team_id
  endpoint = "https://hooks.example.com/clickup"
  events   = [
    "taskCreated",
    "taskUpdated",
    "taskDeleted"
  ]
  space_id = clickup_space.engineering.id
}

# 6. Create a Chat Channel (V3 API)
resource "clickup_chat_channel" "devops" {
  workspace_id = tostring(local.team_id)
  name         = "devops-alerts"
  visibility   = "PUBLIC"
}
```

---

## Supported Resources & Data Sources

| Domain | Resources | Data Sources |
|:---|:---|:---|
| **Workspace & Teams** | — | `clickup_authorized_teams`, `clickup_authorized_user`, `clickup_team_views`, `clickup_team_available_fields`, `clickup_workspace_plan`, `clickup_workspace_seats` |
| **Spaces** | `clickup_space`, `clickup_space_tag`, `clickup_space_view` | `clickup_space`, `clickup_spaces`, `clickup_space_tags`, `clickup_space_views`, `clickup_space_available_fields` |
| **Folders & Lists** | `clickup_folder`, `clickup_folder_view`, `clickup_list`, `clickup_folderless_list`, `clickup_list_view` | `clickup_folder`, `clickup_folders`, `clickup_folder_views`, `clickup_folder_templates`, `clickup_folderless_lists`, `clickup_list`, `clickup_lists`, `clickup_list_views`, `clickup_list_templates`, `clickup_list_members` |
| **Tasks** | `clickup_task`, `clickup_task_checklist`, `clickup_checklist_item`, `clickup_task_tag`, `clickup_task_dependency`, `clickup_task_custom_field`, `clickup_task_link` | `clickup_task`, `clickup_tasks`, `clickup_task_templates`, `clickup_task_members`, `clickup_filtered_team_tasks`, `clickup_task_time_in_status` |
| **Communication** | `clickup_chat_channel`, `clickup_task_comment`, `clickup_list_comment`, `clickup_view_comment`, `clickup_comment_reply` | `clickup_chat_view_comments`, `clickup_task_comments`, `clickup_list_comments`, `clickup_threaded_comments` |
| **Webhooks** | `clickup_webhook` | `clickup_webhooks` |
| **Goals & OKRs** | `clickup_goal`, `clickup_key_result` | `clickup_goal`, `clickup_goals` |
| **Time Tracking** | `clickup_time_entry`, `clickup_task_time` | `clickup_time_entry`, `clickup_time_entries`, `clickup_running_time_entry`, `clickup_time_entry_tags`, `clickup_time_entry_history`, `clickup_tracked_time`, `clickup_bulk_time_in_status` |
| **Access & Governance** | `clickup_task_guest`, `clickup_folder_guest`, `clickup_list_guest`, `clickup_team_user`, `clickup_team_guest`, `clickup_user_group` | `clickup_guest`, `clickup_user`, `clickup_user_groups`, `clickup_custom_roles`, `clickup_custom_items`, `clickup_audit_logs`, `clickup_shared_hierarchy`, `clickup_accessible_custom_fields` |

---

## Development

This repository uses [mise](https://mise.jdx.dev/) to manage tool versions and build tasks.

### Prerequisites

- [Go](https://go.dev/doc/install) >= 1.26
- [mise](https://mise.jdx.dev/)
- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0

### Setup

```sh
# Install development toolchain & configure git hooks
mise install
```

### Local Commands

```sh
# Run format, lint, build, and tests
mise run

# Run unit tests
mise run test

# Run acceptance tests with mock ClickUp API server
mise run testacc

# Run linter
mise run lint

# Check formatting
mise run fmt-check

# Generate documentation for Terraform Registry
mise run docs

# Re-generate provider code from ClickUp OpenAPI spec
mise run generate
```

---

## Documentation

Full resource schemas, arguments, and attribute references are available on the [Terraform Registry Documentation](https://registry.terraform.io/providers/rixlhq/clickup/latest/docs).

---

## Contributing

Contributions, bug reports, and feature requests are welcome! Please check out the [GitHub Issues](https://github.com/rixlhq/terraform-provider-clickup/issues) to report bugs or submit pull requests.

---

## License

This project is licensed under the [MIT License](LICENSE).
