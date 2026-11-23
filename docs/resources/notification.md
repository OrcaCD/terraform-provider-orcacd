---
page_title: "orcacd_notification Resource"
description: "Manage an OrcaCD notification and its subscriptions."
---

# orcacd_notification (Resource)

```hcl
resource "orcacd_notification" "ops" {
  name            = "ops-webhook"
  type            = "webhook"
  config_json     = jsonencode({ webhookUrl = var.notification_webhook_url })
  application_ids = [orcacd_application.stack.id]
}
```

- `name` and `type` are required. The sensitive `config_json` accepts a JSON object or a JSON-encoded string and is required for creation. The hub tests delivery before saving a new notification. Changes to the name, type, or configuration replace the notification.
- `enabled` defaults to true at creation. `events` is a set of OrcaCD event names and defaults to all events at creation.
- Use `application_ids`, `agent_ids`, or `repository_ids` to subscribe to specific resources, or `all_applications`, `all_agents`, or `all_repositories` to subscribe to every resource of that kind. The all flags default to false. Set a target set to `[]` to clear the corresponding subscription.
- `id` is read-only. The hub keeps `config_json` private. Leave it omitted when importing and managing an existing subscription. Adding it later plans a replacement.

```sh
terraform import orcacd_notification.ops <notification-id>
```
