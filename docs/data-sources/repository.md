---
page_title: "orcacd_repository Data Source"
description: "Look up an existing OrcaCD repository."
---

# orcacd_repository (Data Source)

Set exactly one of `id` or `name`. Name matching is exact and uses the resources visible to the configured API token. Use the ID if multiple resources share the name.

```hcl
data "orcacd_repository" "example" {
  id = var.repository_id
}
```

The data source exports the resource's ID and name.
It also exports `url`, `git_provider`, `auth_method`, `sync_type`, `polling_interval_seconds`, `require_signed_commits`, and the five GitHub Actions OIDC settings.
