---
page_title: "orcacd_application Data Source"
description: "Look up an existing OrcaCD application."
---

# orcacd_application (Data Source)

Set exactly one of `id` or `name`. Name matching is exact and uses the resources visible to the configured API token. Use the ID if multiple resources share the name.

```hcl
data "orcacd_application" "example" {
  id = var.application_id
}
```

The data source exports the resource's ID and name.
It also exports `repository_id`, `agent_id`, `branch`, `path`, `icon`, `health_status`, `sync_status`, `image_poll_enabled`, `image_poll_interval_seconds`, `delete_old_images`, and `rollback_enabled`.
