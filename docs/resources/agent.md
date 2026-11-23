---
page_title: "orcacd_agent Resource"
description: "Register an agent and obtain its connection token."
---

# orcacd_agent (Resource)

```hcl
resource "orcacd_agent" "docker" {
  name        = "docker"
  allowed_ips = ["203.0.113.0/24"]
}

output "agent_token" {
  value     = orcacd_agent.docker.token
  sensitive = true
}
```

`name` is required. `icon` defaults to `server` at creation. `allowed_ips` accepts a set of IP addresses and CIDR ranges. An empty set allows any address.

The hub assigns `id` and returns the sensitive `token` at creation. Use the token as the agent's `AUTH_TOKEN`. Import leaves `token` empty. If a token is rotated in OrcaCD, use the new value from the hub to reconnect the agent.

```sh
terraform import orcacd_agent.docker <agent-id>
```
