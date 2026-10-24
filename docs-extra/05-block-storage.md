# Block storage

## Volumes

`nobus_volume` manages an independently provisioned block volume. The normal lifecycle is create volume → attach it → mount and initialize it through instance bootstrapping → snapshot/backup as needed.

```hcl
resource "nobus_volume" "postgres_data" {
  name        = "postgres-data"
  size        = 200
  description = "Persistent PostgreSQL data"
  metadata = {
    workload = "postgres"
  }
}
```

Increasing `size` calls the Nobus extend operation in place. Shrinking is not supported: a plan that asks for a smaller volume produces a clear diagnostic rather than silently recreating or destroying data. Plan filesystem growth separately—enlarging the cloud volume does not grow a partition or filesystem inside the guest. Full detail: [`nobus_volume`](../docs/resources/volume.md).

## Attachments and snapshots

Use [`nobus_volume_attachment`](../docs/resources/volume_attachment.md) for a data-volume attachment and [`nobus_volume_snapshot`](../docs/resources/volume_snapshot.md) for a point-in-time snapshot. Snapshot source and creation metadata are immutable. Consistency is workload-specific: quiesce or coordinate writes before a snapshot when the application needs a crash-consistent or application-consistent recovery point.

```hcl
resource "nobus_volume_snapshot" "before_upgrade" {
  volume_id = nobus_volume.postgres_data.id
  name      = "before-upgrade"
}
```

## Ownership transfers

`nobus_volume_transfer` creates a one-time transfer offer and exposes a sensitive `auth_key`. Pass it through a protected channel; do not output it to logs. `nobus_volume_transfer_accept` accepts an offered transfer. These are handoff operations, not a substitute for a shared state backend. Their exact input/output schemas are [`volume transfer`](../docs/resources/volume_transfer.md) and [`transfer accept`](../docs/resources/volume_transfer_accept.md).

## Volume types: an intentional exception

The live v3 API exposes creation of a volume type but no corresponding read or delete operation. Consequently [`nobus_volume_type`](../docs/resources/volume_type.md) creates the remote type, but removing the Terraform resource only removes it from state—it cannot delete the remote object. Use a restrictive lifecycle policy and record ownership before creating volume types. Do not expect `terraform destroy` to clean these remote types up.

## Backups and discovery

The provider reads existing backups with [`nobus_volume_backups`](../docs/data-sources/volume_backups.md) and a specific backup with [`nobus_volume_backup`](../docs/data-sources/volume_backup.md). It also exposes plural volumes, snapshots, transfers, and availability zones. These are read-only; schedule/retention creation APIs are not represented as a backup resource in this provider.

## Recovery practice

Test restores outside a production incident. Record the source volume/snapshot ID, expected availability zone, attachment destination, and in-guest recovery steps. Terraform can recreate infrastructure relationships, but it cannot infer application-level recovery correctness.
