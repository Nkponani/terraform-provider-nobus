# Platform-service workflows

Managed Kubernetes and databases are services with their own control planes. Terraform supplies and changes their desired infrastructure configuration; it does not replace application deployment, backup validation, user lifecycle, monitoring ownership, or incident processes.

## Kubernetes cluster design

Before creating a cluster, decide its network/router relationship, control-plane availability requirements, worker capacity model, image policy, and etcd capacity. Use IDs from managed networking rather than manually copied values.

```hcl
resource "nobus_kubernetes_cluster" "production" {
  name                = "production"
  master_count        = 3
  worker_count        = 4
  master_flavor_id    = var.master_flavor_id
  worker_flavor_id    = var.worker_flavor_id
  image_id            = var.kubernetes_image_id
  router_id           = nobus_network.application.router_id
  etcd_volume_size_gb = 100
}
```

Node-count changes represent horizontal scaling. Treat them like a capacity event: ensure workloads have pod disruption budgets, account for autoscaler interaction, and monitor scheduling before/after the change. Flavor changes call vertical scaling; assess disruption and capacity headroom first. etcd-volume size is a growth operation—do not plan on shrinking it as an ordinary update.

The resource’s sensitive JSON outputs represent raw API node information. Prefer the dedicated node, monitoring, bastion, and load-balancer-login data sources for operational integrations. Do not emit those raw values wholesale in outputs.

## Registering an existing cluster

`nobus_kubernetes_migration` sends a sensitive JSON payload to register an existing cluster. It does not turn an arbitrary external cluster into a full Terraform-owned `nobus_kubernetes_cluster`, and delete does not deprovision the remote cluster. Use it in a one-time migration workflow with a preserved record of the original cluster owner, payload source, and rollback plan.

## Database provisioning workflow

Treat `nobus_database` creation as a service request with security and lifecycle consequences. Choose a database name, customer identity, engine/type/version, flavor, capacity, installation topology, zones, and encryption deliberately. Password is required and sensitive; store it in a secret manager/CI secret flow rather than a checked-in variable file.

```hcl
resource "nobus_database" "billing" {
  name              = "billing"
  user_email        = "dba@example.com"
  flavor_name       = var.database_flavor
  database_type     = "postgresql"
  database_version  = "16"
  size              = 200
  install_type      = "single"
  customer_username = "billing_admin"
  customer_password = var.billing_password
  zones             = ["nobus-wa-az1"]
  encrypted         = true
}
```

An update to `customer_password` is an API credential-rotation request. Coordinate it with every client, update dependent secret delivery, and validate connectivity with the new credential before retiring the old path. Increasing size and supported flavor changes similarly need capacity and maintenance review. Topology, engine, version, zones, and encryption choices can force replacement; a replacement is not a database migration strategy.

## Credentials and observability

Use the dedicated credential sources only at the narrow point of consumption. Mark downstream variables and outputs sensitive. A connection string placed in an unsafeguarded output is still exposed even when the source field was sensitive.

Creation status and database status answer different operational questions: use creation status while provisioning and health/status after the service exists. Database logs and Zabbix credentials are diagnostic/monitoring integrations; control log volume and destination access because they can include application or infrastructure context.

## Actions and existing databases

`nobus_database_action` is an intentional imperative action. It runs when Terraform creates/replaces that action resource; removing it does not reverse the remote operation. Use a meaningful resource address and trigger value, keep the plan/apply record, and avoid placing it in a module that applies routinely.

`nobus_existing_databases` registers customer-managed databases from a sensitive JSON array. It does not provision or delete those databases. Its purpose is controlled registration/inventory, so its state should be owned by the team responsible for that registration boundary.

## Service handoff checklist

- Confirm network reachability and security policy before client cutover.
- Store credentials in an approved secret delivery mechanism.
- Define owners for backup, restore tests, alerting, patching, and user management.
- Validate the reported task/status and a real client connection after changes.
- Record which operations are Terraform-managed versus performed in the service console.
