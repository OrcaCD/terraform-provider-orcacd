---
page_title: "OrcaCD Provider"
description: "Manage resources on an OrcaCD hub."
---

# OrcaCD Provider

Use the OrcaCD provider to manage Git repositories, applications, notification subscriptions, and agents.

```hcl
terraform {
  required_providers {
    orcacd = {
      source = "orcacd/orcacd"
    }
  }
}

provider "orcacd" {
  url = "https://orca.example.com"
}
```

| Attribute | Description |
| --- | --- |
| `url` | Base URL of the OrcaCD hub. Defaults to `ORCACD_URL`. The provider adds `/api/v1`. |
| `token` | Sensitive bearer token created in OrcaCD. Defaults to `ORCACD_TOKEN`. |

Grant the token read access for refresh and lookup, and the applicable create, update, and delete permissions for each managed kind. Applications also need read access to their repository and agent when registering or changing sources. Source changes that redeploy the application may require deploy permission. Notification targets require read access to the referenced resources.

Repository credentials, webhook secrets, notification configuration, and one-time agent tokens are sensitive values in Terraform state. Keep state and saved plans in access-controlled storage.

Import uses the stable resource ID from the OrcaCD UI or a lookup data source. After import, manage only attributes you want Terraform to control. Secret values omitted by hub reads can remain omitted from an imported resource's Terraform configuration.
