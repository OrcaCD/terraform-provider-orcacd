terraform {
  required_providers {
    orcacd = {
      source = "orcacd/orcacd"
    }
  }
}

variable "repository_url" {
  type        = string
  description = "Git repository that contains compose.yaml on main."
}

variable "agent_name" {
  type        = string
  description = "Name of an existing OrcaCD agent."
}

variable "notification_webhook_url" {
  type        = string
  description = "URL of a webhook that can receive OrcaCD notifications."
  sensitive   = true
}

provider "orcacd" {}

data "orcacd_agent" "docker" {
  name = var.agent_name
}

resource "orcacd_repository" "stack" {
  url          = var.repository_url
  git_provider = "github"
  sync_type    = "manual"
}

resource "orcacd_application" "stack" {
  name          = "example-stack"
  repository_id = orcacd_repository.stack.id
  agent_id      = data.orcacd_agent.docker.id
  branch        = "main"
  path          = "compose.yaml"
}

resource "orcacd_notification" "ops" {
  name            = "example-stack-ops"
  type            = "webhook"
  config_json     = jsonencode({ webhookUrl = var.notification_webhook_url })
  application_ids = [orcacd_application.stack.id]
}
