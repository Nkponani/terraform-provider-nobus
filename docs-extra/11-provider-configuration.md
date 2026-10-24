# Provider configuration and credentials

This page is the operational companion to the generated [provider schema](../docs/index.md). Configure the provider once per Nobus project and availability zone unless a configuration intentionally spans scopes.

## Resolution rules

The provider resolves each setting independently. A non-empty provider-block setting wins; otherwise it reads the matching environment variable; then the endpoint and availability zone fall back to their documented defaults.

| Setting | Provider attribute | Environment | Default | Used for |
| --- | --- | --- | --- | --- |
| API key | `api_key` | `NOBUS_API_KEY` | none | every API call |
| API base URL | `endpoint` | `NOBUS_ENDPOINT` | `https://cloud-api.nobus.io` | every API call |
| Project | `project_id` | `NOBUS_PROJECT_ID` | none | resource/data-source scope when required |
| Availability zone | `availability_zone` | `NOBUS_AVAILABILITY_ZONE` | `nobus-wa-az1` | resource/data-source scope when required |

An unknown provider value during Terraform’s initial planning phase is treated as unset. This allows partially known configurations to plan, but an actual provider operation still fails clearly if no API key or required project is available.

## Recommended credential flow

For developer shells, export a short-lived key outside the repository:

```bash
export NOBUS_API_KEY='…'
export NOBUS_PROJECT_ID='project-id'
export NOBUS_AVAILABILITY_ZONE='nobus-wa-az1'
```

For CI, inject those values from the CI platform’s secret store. Do not pass them in command-line `-var` arguments: command lines are often retained in logs or process lists. If a module must accept a key, mark the variable sensitive and pass it from a parent configuration that retrieves it safely.

```hcl
variable "nobus_api_key" {
  type      = string
  sensitive = true
}

provider "nobus" {
  api_key    = var.nobus_api_key
  project_id = var.nobus_project_id
}
```

The provider never deliberately writes `api_key` to resource state or includes it in its own diagnostic text. That does not protect keys accidentally placed in user-defined outputs, provisioner commands, filenames, or debug logs.

## Endpoint overrides

`endpoint` is primarily for a Nobus deployment or test endpoint. The client accepts either a complete URL or a host-like value and adds `https://` if a scheme is omitted. It preserves an endpoint path prefix and targets relative API paths beneath it. Do not point a production state at a test endpoint: Terraform will see a different inventory and may propose creation or deletion unexpectedly.

Use a separate Terraform backend/state and a separate API key for every environment. An endpoint override should be explicit in CI configuration, not an unnoticed developer-shell export.

## Multiple projects or zones

Provider aliases make intent clear when a root module genuinely owns multiple scopes:

```hcl
provider "nobus" {
  alias             = "shared"
  api_key           = var.nobus_api_key
  project_id        = var.shared_project_id
  availability_zone = "nobus-wa-az1"
}

resource "nobus_volume" "shared" {
  provider = nobus.shared
  name     = "shared-data"
  size     = 100
}
```

Alternatively, a scoped resource can set `project_id` and `availability_zone` itself. That value overrides the provider defaults for that resource only. Both approaches are valid; aliases are usually more readable when many resources share a non-default scope.

Changing an object’s project or availability zone is replacement-only. A Terraform plan cannot migrate a remote object across those boundaries just because its scope changes. Plan the service-specific migration, import the destination object if appropriate, and cut consumers over deliberately.

## Permission design

The API key needs access to the project and service operations in the plan. Prefer a dedicated automation identity with least privilege rather than a personal administrator key. Permission failures can appear as authorization diagnostics or not-found responses depending on the API; verify project ID, zone, and account permissions before assuming an object was deleted.

Rotate keys on personnel changes, compromise, or normal secret-rotation schedules. Rotation normally changes only runtime authentication, not resource configuration, so it should not cause a resource diff. Test the replacement key with a read-only plan before revoking the old one.

## Configuration review checklist

- Is the provider source pinned in `.terraform.lock.hcl` and upgraded intentionally?
- Is the target project correct for this workspace/environment?
- Is the availability zone explicitly chosen where it matters?
- Is an endpoint override expected and isolated from production state?
- Does the CI identity have only the permissions this apply needs?
- Are state and plan artifacts protected as if they may contain service credentials?
