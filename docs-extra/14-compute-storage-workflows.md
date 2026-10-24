# Compute and storage workflows

## Immutable server pattern

The safest default is to treat `nobus_instance` as immutable application infrastructure. Put application configuration in image build, cloud-init/user data, or a deployment system; create a replacement instance for changes that alter boot inputs. This matches the provider’s lifecycle: image, flavor, keypair, network, boot-volume size, initial security groups, admin password, and user data are replacement-oriented.

```hcl
resource "nobus_instance" "api" {
  name       = "api-01"
  image      = var.image_id
  flavor     = var.flavor_name
  key_name   = nobus_keypair.ops.name
  network_id = nobus_network.application.id
  user_data  = file("${path.module}/cloud-init/api.yaml")
}
```

Changing an instance flavor via Terraform means replacement because the API’s resize operation needs a separate confirmation call. If in-place resize is an approved operational requirement, handle it through the Nobus process outside this steady-state resource and reconcile Terraform afterward; do not expect a `flavor` update to resize in place.

## Instance readiness and access

The provider waits for `ACTIVE`/`active` after create by default. It does not wait for SSH, cloud-init completion, an application health endpoint, or a floating IP to become externally routable. Build those checks into your deployment pipeline after `terraform apply`.

`addresses` is a computed JSON string. A safe output exposes only a known value if the returned API shape is understood:

```hcl
output "api_addresses_json" {
  value = nobus_instance.api.addresses
}
```

Avoid parsing undocumented address structure in reusable modules unless you test against the target API deployment. Prefer an explicit floating-IP resource for an external endpoint.

## Key management

`nobus_keypair` stores public-key material. Keep private keys outside Terraform state and outside the repository. Keypair changes are not a retroactive change to an existing instance’s authorized keys; changing the instance `key_name` triggers replacement. Rotate runtime SSH access through guest configuration or a controlled server replacement.

## Data-volume lifecycle

Separate durable data from server lifetime:

```hcl
resource "nobus_volume" "database" {
  name = "database-data"
  size = 500

  lifecycle {
    prevent_destroy = true
  }
}

resource "nobus_volume_attachment" "database" {
  instance_id = nobus_instance.database.id
  volume_id   = nobus_volume.database.id
}
```

`prevent_destroy` is a Terraform guardrail, not a backup. It blocks accidental destroy plans until deliberately removed. Establish backup, restore, and data-retention procedures separately. When replacing an instance, plan attachment detachment, guest unmount, and reattachment; Terraform can make the API calls but cannot confirm filesystem/application safety.

Volume expansion only grows. After Terraform extends a volume, extend the partition/LVM/filesystem inside the guest according to the operating system and storage layout. Test this sequence on a restore or non-production volume first.

## Snapshot and transfer workflow

Snapshots are point-in-time block-storage copies. Before taking a database snapshot, coordinate write quiescence, flushes, or database-native backup behavior; a provider cannot guarantee application consistency. Name snapshots with purpose and change reference, then test restoration.

Volume transfers are ownership handoffs. The transfer `auth_key` is sensitive, but it is still material in Terraform state. Share it only through approved secret channels, accept promptly, and rotate/revoke access according to the Nobus workflow. Do not place it in non-sensitive module outputs.

## Delete decisions

`delete_volume_on_destroy` on an instance affects the boot volume that Nobus created for that instance. It does not automatically govern separately managed data volumes. Be explicit about every volume’s owner and destroy behavior; an application’s recovery objective should not depend on Terraform’s implicit default.
