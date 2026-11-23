---
page_title: "orcacd_agent Data Source"
description: "Look up an existing OrcaCD agent."
---

# orcacd_agent (Data Source)

Set exactly one of `id` or `name`. Name matching is exact and uses the resources visible to the configured API token. Use the ID if multiple resources share the name.

```hcl
data "orcacd_agent" "example" {
  id = var.agent_id
}
```

The data source exports the resource's ID and name.
It also exports `icon`, `allowed_ips`, `status`, and `version`. The connection token is available from the managed resource on creation.
