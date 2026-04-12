# terraform-provider-oauth4os

Terraform provider for managing oauth4os resources: clients, scope mappings, and Cedar policies.

## Usage

```hcl
terraform {
  required_providers {
    oauth4os = {
      source = "seraphjiang/oauth4os"
    }
  }
}

provider "oauth4os" {
  url      = "http://localhost:8443"
  admin_token = var.admin_token  # optional, for authenticated Admin API
}

# Register a client
resource "oauth4os_client" "ci_agent" {
  client_name = "ci-agent"
  scope       = "read:logs-* write:logs-*"
}

# Scope mapping
resource "oauth4os_scope_mapping" "read_logs" {
  scope         = "read:logs-*"
  backend_roles = ["readall"]
}

# Cedar policy
resource "oauth4os_cedar_policy" "deny_security" {
  policy_id = "deny-security-index"
  policy    = <<-CEDAR
    forbid(*, *, .opendistro_security);
  CEDAR
}

# OIDC provider
resource "oauth4os_provider" "keycloak" {
  name     = "keycloak"
  issuer   = "https://keycloak.example.com/realms/main"
  jwks_uri = "https://keycloak.example.com/realms/main/protocol/openid-connect/certs"
}
```

## Resources

| Resource | Description | CRUD |
|----------|-------------|------|
| `oauth4os_client` | Registered OAuth client | Create, Read, Delete |
| `oauth4os_scope_mapping` | Scope-to-role mapping | Create, Read, Update, Delete |
| `oauth4os_cedar_policy` | Cedar access policy | Create, Read, Delete |
| `oauth4os_provider` | OIDC provider | Create, Read, Delete |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `oauth4os_config` | Current runtime configuration |

## Building

```bash
go build -o terraform-provider-oauth4os
```
