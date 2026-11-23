---
page_title: "orcacd_notification Data Source"
description: "Look up an existing OrcaCD notification."
---

# orcacd_notification (Data Source)

Set exactly one of `id` or `name`. Name matching is exact and uses the resources visible to the configured API token. Use the ID if multiple resources share the name.

```hcl
data "orcacd_notification" "example" {
  id = var.notification_id
}
```

The data source exports the resource's ID and name.
It also exports `type`, `enabled`, `events`, the `all_applications`, `all_agents`, and `all_repositories` flags, and the visible `application_ids`, `agent_ids`, and `repository_ids` subscriptions.
