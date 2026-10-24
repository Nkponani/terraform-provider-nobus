# File/Object Storage (FOS)

FOS is Nobus’s Swift-compatible object-storage service. The provider manages containers and uploads local files as objects; the read-only sources list accounts, auth details, containers, objects, and object content.

## Containers and objects

```hcl
resource "nobus_fos_container" "artifacts" {
  name      = "release-artifacts"
  is_public = false
}

resource "nobus_fos_object" "release" {
  container_name = nobus_fos_container.artifacts.name
  object_name    = "releases/app.tar.gz"
  source         = "${path.module}/dist/app.tar.gz"
  source_hash    = filesha256("${path.module}/dist/app.tar.gz")
  content_type   = "application/gzip"
}
```

`source_hash` is optional but strongly recommended. Without it, a file changed in place at the same path may not appear as a configuration change. With `filesha256(source)`, Terraform plans an upload when bytes change. The object resource supports replacement of content without needing a different object name; consult [`nobus_fos_object`](../docs/resources/fos_object.md) for metadata and headers.

Container names and object identity shape the remote path. A container’s visibility is a creation choice and requires replacement when changed. Before removing a container, remove the objects Terraform manages and make sure no externally managed objects remain; object-storage APIs often reject deletion of a non-empty container. See [`nobus_fos_container`](../docs/resources/fos_container.md).

## State and content safety

`nobus_fos_object_content` downloads an object as sensitive base64 content. Sensitive suppresses routine display; it does **not** prevent the value being stored in state. Do not use it for large artifacts or secrets unless your state backend is designed for that exposure. Prefer passing object locations, short-lived access mechanisms, or an external secret system to workloads.

Likewise, setting content-related resource fields can make the object’s source and metadata part of your configuration history. Keep build outputs out of source control and use a remote encrypted state backend with least-privilege access.

## Authentication and discovery sources

| Source | Use |
| --- | --- |
| [`nobus_fos_account`](../docs/data-sources/fos_account.md) | account-level information |
| [`nobus_fos_auth_details`](../docs/data-sources/fos_auth_details.md) | FOS authentication details; treat results as secret |
| [`nobus_fos_containers`](../docs/data-sources/fos_containers.md) | container inventory |
| [`nobus_fos_objects`](../docs/data-sources/fos_objects.md) | object inventory for a container/prefix as supported |
| [`nobus_fos_object_content`](../docs/data-sources/fos_object_content.md) | base64 object body; sensitive and stateful |

Use FOS sources for discovery and integration, not to copy an entire bucket/container into Terraform state.
