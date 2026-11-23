---
page_title: "orcacd_repository Resource"
description: "Register and configure a Git repository."
---

# orcacd_repository (Resource)

```hcl
resource "orcacd_repository" "stack" {
  url                      = "https://github.com/example/stack.git"
  git_provider             = "github"
  sync_type                = "polling"
  polling_interval_seconds = 120
}
```

- `url` and `git_provider` are required and changes replace the repository. Supported Git providers are `github`, `gitlab`, `gitea`, `bitbucket`, `azure_devops`, and `generic`.
- `auth_method` accepts `none`, `token`, `basic`, or `ssh`, subject to the selected Git provider. It defaults to `none` at creation. Supply credentials using the sensitive `auth_user` and `auth_token` attributes. An authentication change replaces the credentials as a pair. Set both values used by the new credentials when changing authentication on an imported repository.
- `sync_type` accepts `manual`, `polling`, or `webhook`. It defaults to `manual` at creation. The hub defaults to 60 seconds when polling and accepts `polling_interval_seconds` of at least 30.
- `github_actions_oidc_enabled` enables GitHub Actions OIDC. Use `github_actions_oidc_allow_repo_sync` and `github_actions_oidc_allow_image_sync` to control which syncs it can trigger. The `github_actions_oidc_allowed_branches` and `github_actions_oidc_allowed_workflows` sets restrict the permitted branches and workflow filenames. An empty set allows any branch or workflow, respectively.
- `id`, `name`, and `require_signed_commits` are read-only. `webhook_url` and the sensitive `webhook_secret` are captured when the hub creates or enables a webhook. Existing webhook secrets and Git credentials are unavailable on import.

```sh
terraform import orcacd_repository.stack <repository-id>
```
