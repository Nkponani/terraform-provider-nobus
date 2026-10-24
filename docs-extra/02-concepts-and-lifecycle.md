# Core concepts and lifecycle

## Ownership model

A `resource` declares an object Terraform owns: it creates it, reads it during refresh, updates it where the API supports updates, and deletes it when removed from configuration. A `data` block only reads existing API information. Do not define both for the same remote object unless the data source is deliberately querying a different object.

Terraform state is the link between an address such as `nobus_volume.data` and the Nobus object ID. Back up and protect state. It can include values marked sensitive; Terraform hides them in normal CLI output but the values can still exist in the state backend.

## Creation, refresh, and eventual completion

The provider sends authenticated HTTP requests with `Authorization: Bearer <api key>` and identifies itself as `terraform-provider-nobus`. Nobus wraps API responses in a status/data envelope. The provider converts unsuccessful HTTP responses and in-envelope failures into Terraform diagnostics without intentionally including credentials.

Nobus operations can be asynchronous. Where a resource needs to wait for a usable result, the provider polls roughly every two seconds and honours Terraform cancellation; it does not impose a separate provider timeout. A slow creation can therefore continue until the Terraform command timeout, cancellation, or API error. Keep automation job limits realistic and use the relevant status/task data source when an API exposes one.

## Change categories

Generated resource references label every field. In plain language:

| Schema behavior | Terraform result | What to do |
| --- | --- | --- |
| Updatable in place | API update; object ID remains | review the service impact, then apply |
| `RequiresReplace` / “forces a new resource” | create replacement then destroy old object according to lifecycle ordering | plan migrations and dependencies |
| Computed | returned by Nobus | use in outputs/references; do not configure it |
| Optional + computed | may be set or accepted from API | explicitly set it when you need drift control |
| Sensitive | redacted in CLI display | still protect input files and state |
| Write-only | Nobus does not return it | preserve it in configuration; importing may require special care |

Network placement (`project_id`, `availability_zone`) is replacement-only across normal scoped resources. Values the API needs only at creation may be optional/computed so imports can work; they must still be supplied for a new object.

## Imports

Most long-lived CRUD resources accept the remote ID:

```bash
terraform import nobus_volume.data VOLUME_ID
terraform import nobus_network.app NETWORK_ID
```

Import needs provider project and availability-zone configuration because Nobus read operations are scoped. Follow import with `terraform plan`; supply create-time and write-only fields in configuration if Terraform needs to manage future changes. In particular, an imported instance cannot have its original image, flavor, key name, boot-volume size, security-group names, admin password, or user data reconstructed if the API does not return them.

Special import formats:

| Resource | Import ID |
| --- | --- |
| `nobus_volume_attachment` | `INSTANCE_ID/VOLUME_ID` |
| `nobus_router_interface` | `ROUTER_ID/SUBNET_ID/PORT_ID` (leave the unused subnet or port component empty) |
| `nobus_port_address_pairs` | port ID |
| `nobus_fos_object` | `CONTAINER_NAME/OBJECT_NAME` |

One-shot/action resources and resources whose deletion has no remote API operation generally do not support meaningful import. See the service guides before using them in a reusable module.

## Dependencies and safe ordering

Terraform infers dependencies from references: `network_id = nobus_network.app.id` is better than a manually maintained `depends_on`. Use `depends_on` only for a real dependency without a data reference. For safe teardown, remember the natural order: detach volumes and floating IPs, remove LB children before their parent, empty FOS containers, and remove router interfaces before dependent networks/subnets where the API requires it.

Avoid broad `ignore_changes`. It can hide deliberate configuration changes and real drift. Use it only for an externally managed field, document its owner, and periodically audit the live object.

## JSON outputs and payloads

Some API fields are untyped or variable-shaped. The provider exposes them as JSON strings, commonly with a `_json` suffix. Decode them only where needed:

```hcl
locals {
  addresses = jsondecode(nobus_instance.web.addresses)
}
```

JSON action payloads are sensitive when they may include credentials. Prefer `jsonencode({...})` to hand-built JSON so Terraform validates structure and properly escapes strings.
