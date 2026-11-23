---
page_title: "orcacd_application Resource"
description: "Register a Compose application on an OrcaCD agent."
---

# orcacd_application (Resource)

```hcl
resource "orcacd_application" "stack" {
  name          = "stack"
  repository_id = orcacd_repository.stack.id
  agent_id      = data.orcacd_agent.docker.id
  branch        = "main"
  path          = "compose.yaml"
}
```

All five arguments are required. `path` is relative to the repository root. The hub assigns the read-only `id`. Keep the agent connected when destroying or moving an application so the hub can remove its previous deployment.

Terraform manages the application's registration. Use OrcaCD to manage its icon, image polling, old-image deletion, and rollback settings. The application data source exposes effective settings from the hub.

```sh
terraform import orcacd_application.stack <application-id>
```
