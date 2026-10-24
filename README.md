# Terraform Provider for Nobus Cloud

A Terraform provider for [Nobus Cloud](https://cloud.nobus.io), built on the
public Nobus API (`https://cloud-api.nobus.io/api/v3`).

Every endpoint, request field, response field and validation in this provider is
derived from the Nobus OpenAPI specification, which is vendored at
[`docs/api/openapi.json`](docs/api/openapi.json). Nothing is inferred from other
cloud providers.

## Documentation

The generated Terraform reference in [`docs/`](docs/) is the complete schema
reference. For task-oriented setup, lifecycle behavior, service architecture,
operational guidance, imports, secrets, and troubleshooting, start with the
[full provider guide](docs-extra/README.md).

## Resources

| Terraform resource | Purpose |
| --- | --- |
| `nobus_network` | A network, created together with its subnet and router |
| `nobus_standalone_network` | A network without an automatically created subnet or router |
| `nobus_volume` | A volume that can be grown in place |
| `nobus_instance` | A virtual machine instance |
| `nobus_volume_attachment` | Attaches a volume to an instance |
| `nobus_keypair` | An SSH keypair for instances |
| `nobus_subnet` | A standalone network subnet |
| `nobus_router` | A standalone network router |
| `nobus_router_interface` | Connects a router to a subnet or port |
| `nobus_port` | A network port |
| `nobus_floating_ip` | A floating IP allocation and optional port association |
| `nobus_security_group` | A network security group |
| `nobus_security_group_rule` | A security-group ingress or egress rule |
| `nobus_trunk` | A parent port and its VLAN subports |
| `nobus_qos_policy` | A network QoS policy |
| `nobus_firewall_group` | A network firewall group |
| `nobus_port_address_pairs` | Allowed address pairs on a port |
| `nobus_volume_snapshot` | A point-in-time volume snapshot |
| `nobus_volume_type` | Creates a block-storage volume type (the API has no read/delete operation) |
| `nobus_volume_transfer` | Creates a volume ownership transfer |
| `nobus_volume_transfer_accept` | Accepts a volume ownership transfer |
| `nobus_instance_action` | Runs a one-shot compute operation |
| `nobus_fos_container` | A Swift-compatible FOS object-storage container |
| `nobus_fos_object` | A local file uploaded as an object, with in-place content replacement |
| `nobus_kubernetes_cluster` | A managed Kubernetes cluster with horizontal, vertical, and etcd-volume scaling |
| `nobus_kubernetes_migration` | Registers an existing Kubernetes cluster without provisioning it |
| `nobus_database` | A managed database with volume/flavor resize and password rotation |
| `nobus_database_action` | Runs a one-shot database lifecycle action |
| `nobus_existing_databases` | Registers existing customer-managed databases |
| `nobus_lbaas_load_balancer` | An LBaaS load balancer and its security-group associations |
| `nobus_lbaas_listener` | An LBaaS listener |
| `nobus_lbaas_pool` | An LBaaS backend pool and session-persistence configuration |
| `nobus_lbaas_member` | A backend pool member |
| `nobus_lbaas_health_monitor` | A pool health monitor |
| `nobus_lbaas_l7_policy` | A listener L7 policy |
| `nobus_lbaas_l7_rule` | An L7 policy rule |
| `nobus_lbaas_security_group_rule` | A load-balancer security-group rule |
| `nobus_lbaas_failover` | Triggers a one-shot load-balancer failover |
| `nobus_lbaas_provision` | Composite JSON provisioning for a load balancer and child resources |

## Data sources

Collection data sources are available for networks, subnets, ports, routers,
floating IPs, security groups and rules, trunks, QoS policies, firewall groups,
volumes, snapshots, backups, transfers, and volume availability zones. Use the
singular `nobus_volume_backup` data source to read full details for a backup ID.

FOS also provides `nobus_fos_account`, `nobus_fos_auth_details`,
`nobus_fos_containers`, `nobus_fos_objects`, and
`nobus_fos_object_content`. The last data source downloads an object as
sensitive base64; its content is therefore stored in Terraform state.

Managed Kubernetes data sources expose cluster lists, task status, nodes,
bastion access, HAProxy access, and Grafana monitoring details. Managed
database data sources expose lists, provisioning/health status, connection
credentials, monitoring credentials, and logs. LBaaS has collection data
sources for every child type plus task, deployment-status, status, and stats
lookups. Credential-bearing results are marked sensitive.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) 1.0 or later
- [Go](https://go.dev/doc/install) 1.25 or later (only for building the provider)
- A Nobus Cloud account, an API key and a project

## Authentication

The Nobus API authenticates with a bearer token. Generate an API key from the
Nobus API key endpoint (`POST /api/v3/auth/api-key`) and supply it either
through the provider block or through the environment.

```hcl
terraform {
  required_providers {
    nobus = {
      source = "nobus-cloud/nobus"
    }
  }
}

provider "nobus" {
  api_key = var.nobus_api_key
}
```

```bash
export NOBUS_API_KEY="..."
```

Every provider attribute can be supplied either way:

| Attribute | Environment variable | Default |
| --- | --- | --- |
| `api_key` | `NOBUS_API_KEY` | none, required |
| `endpoint` | `NOBUS_ENDPOINT` | `https://cloud-api.nobus.io` |
| `project_id` | `NOBUS_PROJECT_ID` | none, required on resources |
| `availability_zone` | `NOBUS_AVAILABILITY_ZONE` | `nobus-wa-az1` |

The API key is marked sensitive. It is never written to state and never appears
in diagnostics or error messages.

## Provider configuration

```hcl
provider "nobus" {
  api_key           = var.nobus_api_key
  project_id        = var.nobus_project_id # default for every resource
  availability_zone = "nobus-wa-az1"
}
```

`project_id` and `availability_zone` are required by every Nobus API operation.
Setting them once on the provider avoids repeating them on each resource.

## Examples

### Network

The Nobus API creates a network, a subnet and a router in a single call, so a
network needs a subnet CIDR and a router name.

```hcl
resource "nobus_network" "private" {
  name        = "production-network"
  subnet_cidr = "10.20.0.0/24"
  router_name = "production-router"

  description = "Primary private network"
  tags        = ["environment=production"]
}

output "subnet_id" {
  value = nobus_network.private.subnet_id
}
```

### Volume

```hcl
resource "nobus_volume" "data" {
  name = "production-data"
  size = 100

  metadata = {
    environment = "production"
  }
}
```

Increasing `size` grows the volume in place through the Nobus extend operation.
Volumes cannot be shrunk; the provider reports a clear error if the
configuration asks for a smaller size.

### Instance

The API requires an image, a flavor and a key name.

```hcl
resource "nobus_instance" "web" {
  name     = "web-01"
  image    = "ubuntu-22-04"
  flavor   = "small"
  key_name = "operations"
}
```

### FOS object storage

```hcl
resource "nobus_fos_container" "assets" {
  name      = "production-assets"
  is_public = false
}

resource "nobus_fos_object" "release" {
  container_name = nobus_fos_container.assets.name
  object_name    = "release.tar.gz"
  source         = "${path.module}/release.tar.gz"
  source_hash    = filesha256("${path.module}/release.tar.gz")
}
```

`source_hash` is optional, but setting it to `filesha256(source)` makes
Terraform upload the object again when the file content changes without its
path changing.

### Managed Kubernetes

```hcl
resource "nobus_kubernetes_cluster" "platform" {
  name                = "production-k8s"
  master_count        = 2
  worker_count        = 3
  master_flavor_id    = var.master_flavor_id
  worker_flavor_id    = var.worker_flavor_id
  image_id            = var.kubernetes_image_id
  router_id           = nobus_router.production.id
  etcd_volume_size_gb = 50
}
```

Changing either node count calls horizontal scaling, changing a flavor calls
vertical scaling, and increasing `etcd_volume_size_gb` calls the etcd-volume
resize endpoint.

### Managed database

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
  customer_password = var.database_password
  zones             = ["nobus-wa-az1"]
  encrypted         = true
}
```

Increasing `size`, changing `flavor_id`, or changing `customer_password`
invokes the corresponding dedicated API operation.

### LBaaS

```hcl
resource "nobus_lbaas_load_balancer" "edge" {
  name      = "production-edge"
  flavor    = "small"
  router_id = nobus_router.production.id
}

resource "nobus_lbaas_listener" "https" {
  name             = "https"
  load_balancer_id = nobus_lbaas_load_balancer.edge.id
  protocol         = "HTTPS"
  protocol_port    = 443
  allow_from       = "0.0.0.0/0"
}
```

### Referencing a network from an instance

Terraform builds the dependency graph from the reference, so the instance is
created after the network exists and no explicit `depends_on` is needed.

```hcl
resource "nobus_network" "app" {
  name        = "app-network"
  subnet_cidr = "10.30.0.0/24"
  router_name = "app-router"
}

resource "nobus_instance" "api" {
  name    = "api-01"
  image   = "ubuntu-22-04"
  flavor  = "medium"
  key_name = "operations"

  network_id = nobus_network.app.id
}
```

More examples live in [`examples/`](examples).

## Asynchronous resources

Volume creation, instance creation, volume growth and all deletions are
asynchronous. The provider polls the status the API reports until the resource
reaches the expected state, an unrecoverable error state, or Terraform cancels
the operation.

`InstanceSchema.status` has no enum in the Nobus specification, so the accepted
ready states default to `ACTIVE` and `active`. If your deployment reports
something else, set `instance_ready_states`.

## Importing existing resources

Every resource supports `terraform import` using its Nobus ID. The project and
availability zone come from the provider configuration, because every Nobus read
requires them.

```bash
terraform import nobus_network.production NETWORK_ID
terraform import nobus_volume.data      VOLUME_ID
terraform import nobus_instance.web      INSTANCE_ID
terraform import nobus_volume_attachment.database INSTANCE_ID/VOLUME_ID
terraform import nobus_fos_container.assets CONTAINER_NAME
terraform import nobus_fos_object.release CONTAINER_NAME/OBJECT_NAME
terraform import nobus_kubernetes_cluster.platform CLUSTER_ID
terraform import nobus_database.orders DATABASE_ID
terraform import nobus_lbaas_load_balancer.edge LOAD_BALANCER_ID
```

## Error handling

API failures become Terraform diagnostics that preserve what Nobus returned:

```
Error: Unable to create Nobus instance "api-01"

  with nobus_instance.api,
  on main.tf line 2, in resource "nobus_instance" "api":
   2: resource "nobus_instance" "api" {

The Nobus API returned HTTP 400: requested image is unavailable in the selected
availability zone.
```

## Local development

```bash
go mod tidy
go fmt ./...
go vet ./...
go test ./...
go build
```

The unit tests use `httptest` servers, so they need no Nobus infrastructure and
no credentials. They cover request construction, authentication, response
decoding, error decoding, state mapping and the full create/read/update/delete
lifecycle of each resource against a fake API.

### Acceptance tests

Acceptance tests talk to the real API and only run when `TF_ACC` is set:

```bash
export NOBUS_API_KEY="..."
export NOBUS_PROJECT_ID="..."
export NOBUS_TEST_IMAGE="ubuntu-22-04"
export NOBUS_TEST_FLAVOR="small"
export NOBUS_TEST_KEY_NAME="operations"

TF_ACC=1 go test ./internal/provider/ -run TestAcc -v -timeout 120m
```

### Using the local build with Terraform

Build the provider and point Terraform at the binary with a development
override. Create `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "nobus-cloud/nobus" = "/absolute/path/to/your/GOBIN"
  }

  # Keep this so providers from the registry still work for other
  # configurations. Remove it to use only local builds.
  direct {}
}
```

Then build and run Terraform:

```bash
go install .

# In your Terraform configuration directory:
terraform init   # warns that the dev override is in effect
terraform plan
terraform apply
```

With a development override in place, `terraform init` prints a warning and
skips downloading the provider. Skip `init` on subsequent runs.

## Project layout

```text
terraform-provider-nobus/
├── main.go
├── internal/
│   ├── client/         # Nobus API client, no Terraform dependencies
│   │   ├── client.go   # transport, auth, JSON, polling
│   │   ├── errors.go   # error types and not-found detection
│   │   ├── networks.go
│   │   ├── volumes.go
│   │   └── instances.go
│   └── provider/       # Terraform Plugin Framework layer
│       ├── provider.go
│       ├── configure.go
│       ├── network_resource.go
│       ├── volume_resource.go
│       ├── instance_resource.go
│       └── volume_attachment_resource.go
├── examples/
├── docs/
└── docs/api/openapi.json
```

The API client contains no Terraform-specific code, so it can be reused and
tested on its own. The resources contain no raw HTTP logic.
