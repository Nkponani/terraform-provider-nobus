# Managed Kubernetes and databases

## Managed Kubernetes

`nobus_kubernetes_cluster` provisions and maintains a Nobus managed cluster. Its creation inputs include the control-plane and worker counts/flavors, image, router, and etcd volume size. Reference the networking resource’s router ID instead of hard-coding it.

```hcl
resource "nobus_kubernetes_cluster" "platform" {
  name                = "platform"
  master_count        = 2
  worker_count        = 3
  master_flavor_id    = var.master_flavor_id
  worker_flavor_id    = var.worker_flavor_id
  image_id            = var.kubernetes_image_id
  router_id           = nobus_network.app.router_id
  etcd_volume_size_gb = 50
}
```

Changing node counts triggers horizontal scaling. Changing a flavor invokes the corresponding vertical-scale operation. Increasing `etcd_volume_size_gb` invokes etcd-volume resizing; design with headroom because decreases are not a normal resize workflow. Inputs that establish cluster identity or topology can require replacement—read the [cluster schema](../docs/resources/kubernetes_cluster.md) before changing a live cluster.

The provider exposes raw node/master/worker fields as sensitive JSON because the API shape may vary. Decode only what an integration needs and avoid outputting credentials. Use the Kubernetes data sources for clusters, task status, nodes, bastion access, load-balancer login, and monitoring; credential-bearing values are sensitive.

`nobus_kubernetes_migration` registers an existing cluster using a sensitive `payload_json` request. It is a registration/action resource: deleting it removes Terraform state rather than deprovisioning the pre-existing cluster. Keep the migration payload in a protected variable and do not treat a destroy plan as cluster deletion. Reference: [`kubernetes migration`](../docs/resources/kubernetes_migration.md).

## Managed databases

`nobus_database` manages a Nobus database service. It requires name, owner email, flavor, database type/version, size, install type, customer credentials, zones, and encryption choice according to its schema.

```hcl
resource "nobus_database" "orders" {
  name              = "orders"
  user_email        = "dba@example.com"
  flavor_name       = "standard"
  database_type     = "postgresql"
  database_version  = "16"
  size              = 100
  install_type      = "single"
  customer_username = "orders_admin"
  customer_password = var.orders_database_password
  zones             = ["nobus-wa-az1"]
  encrypted         = true
}
```

Increasing `size`, changing the flavor field supported by the resource, and changing `customer_password` call dedicated Nobus operations; a password change is credential rotation, not merely state redaction. Database type/version, install topology, zones, encryption, and other create-time identity fields may force replacement. Read the [database reference](../docs/resources/database.md), test the change path, and schedule it as production maintenance.

`instances_json` is sensitive computed output. Database credentials, Zabbix credentials, logs, provisioning status, and health/status are available through dedicated data sources. Outputs containing passwords or connection values should generally be passed directly to a secret-aware consumer, not printed in CLI outputs.

## Database action and registration resources

- [`nobus_database_action`](../docs/resources/database_action.md) sends a one-shot lifecycle action. Its operation/payload is replacement-based; removing it from configuration does not reverse an action already sent.
- [`nobus_existing_databases`](../docs/resources/existing_databases.md) registers existing customer-managed databases via sensitive JSON. Its delete removes Terraform’s registration state, not the customer-managed database itself.

Use an ordinary `nobus_database` for an API-managed database Terraform should own. Use these action/registration resources only when the operational intent is explicit and reviewable.

## Operational checklist

Before scaling or rotating either service: take a backup where relevant, verify quota/capacity, record an approved maintenance window, make sure clients can tolerate connection interruption, and use the task/status sources to verify actual completion. Terraform reports API completion paths; application health remains your responsibility.
