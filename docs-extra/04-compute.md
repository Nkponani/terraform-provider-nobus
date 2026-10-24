# Compute

## Keypairs first

Create or manage the keypair that instances will use before creating the instance. Keep only a public key in Terraform configuration.

```hcl
resource "nobus_keypair" "ops" {
  name       = "ops"
  public_key = file("${path.module}/keys/ops.pub")
}
```

See [`nobus_keypair`](../docs/resources/keypair.md) for import and field details.

## Instances

An instance creation needs an image, flavor, and key name, in addition to project and availability-zone scope. `network_id` is recommended to place it on the intended network. The API permits image and flavor IDs or names as documented by Nobus.

```hcl
resource "nobus_instance" "web" {
  name            = "web-01"
  image           = "ubuntu-22-04"
  flavor          = "small"
  key_name        = nobus_keypair.ops.name
  network_id      = nobus_network.app.id
  security_groups = [nobus_security_group.web.name]

  # Values passed only on first boot should come from a sensitive variable/file.
  user_data = file("${path.module}/cloud-init.yaml")
}
```

After creation, the provider polls the reported instance status until it reaches `ACTIVE` or `active` by default. Set `instance_ready_states` only when your deployment uses different ready-state text. The [instance reference](../docs/resources/instance.md) contains all fields and outputs.

### Important instance lifecycle limits

Image, flavor, key name, network, boot-volume size, initial security groups, admin password, and user data are creation choices and force replacement when changed. The provider deliberately does not resize an instance in place: Nobus requires a separate resize-confirmation operation, and automatic Terraform confirmation could leave an instance in a risky intermediate state. Use a migration/replacement plan or the explicit action workflow your operation approves.

`description` and the supported `name` update are separate from initial boot settings. `hostname` is sent as an instance setting but is replacement-only. `addresses` is a JSON string returned by Nobus—use `jsondecode()` rather than assuming a fixed shape.

`delete_volume_on_destroy` controls whether Nobus deletes the boot volume during instance destruction. It maps to the API `delete_volume` parameter; decide explicitly before destroying production instances.

## Boot from an existing volume

Use [`nobus_instance_from_volume`](../docs/resources/instance_from_volume.md) when lifecycle ownership starts with a prepared boot volume. This is distinct from `nobus_volume_attachment`, which attaches a data volume to an already existing instance. Read its schema carefully: boot source and initial server attributes are immutable creation decisions.

## Attach data volumes

```hcl
resource "nobus_volume" "data" {
  name = "web-data"
  size = 100
}

resource "nobus_volume_attachment" "web_data" {
  instance_id = nobus_instance.web.id
  volume_id   = nobus_volume.data.id
}
```

The attachment waits until Nobus confirms it is attached, and it waits for detachment during destroy. It imports as `INSTANCE_ID/VOLUME_ID`. All attachment inputs force replacement because moving an attachment is detach/create work.

## One-shot instance actions

[`nobus_instance_action`](../docs/resources/instance_action.md) represents an imperative compute action as Terraform state. Its action and inputs are replacement-oriented: changing the configuration causes a new action to be invoked. Use it when an action belongs in a deliberate Terraform run, not for routine application deployment. Treat it as an audit trail, not a durable server configuration model.

## Read-only compute information

Use [`nobus_instance_network_interfaces`](../docs/data-sources/instance_network_interfaces.md), [`nobus_instance_security_groups`](../docs/data-sources/instance_security_groups.md), [`nobus_instance_tags`](../docs/data-sources/instance_tags.md), [`nobus_instance_topology`](../docs/data-sources/instance_topology.md), and [`nobus_instance_diagnostics`](../docs/data-sources/instance_diagnostics.md) for information the main resource does not normalize into state. These reads are useful in outputs and troubleshooting, but avoid making critical topology decisions dependent on unknown values at plan time.
