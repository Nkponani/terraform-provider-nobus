# Getting started

## Prerequisites

You need Terraform 1.0 or later, a Nobus Cloud account, an API key, and a Nobus project. Go 1.25 or later is only necessary when building this provider from source. Create the key through the Nobus API-key endpoint (`POST /api/v3/auth/api-key`) and keep it out of source control.

## Configure the provider

Use a pinned provider version in production. Put credentials in environment variables or a sensitive variable rather than in a checked-in `.tf` file.

```hcl
terraform {
  required_providers {
    nobus = {
      source = "nobus-cloud/nobus"
    }
  }
}

provider "nobus" {
  api_key           = var.nobus_api_key
  project_id        = var.nobus_project_id
  availability_zone = "nobus-wa-az1"
}
```

The exact provider fields are in the [provider schema](../docs/index.md). Each accepts an environment equivalent:

| Provider field | Environment variable | Default | Required when |
| --- | --- | --- | --- |
| `api_key` | `NOBUS_API_KEY` | none | always; it is sent as a bearer token |
| `endpoint` | `NOBUS_ENDPOINT` | `https://cloud-api.nobus.io` | only for a non-production API endpoint |
| `project_id` | `NOBUS_PROJECT_ID` | none | every resource operation unless set on the resource |
| `availability_zone` | `NOBUS_AVAILABILITY_ZONE` | `nobus-wa-az1` | every resource operation unless set on the resource |

Provider-block values take precedence over environment values. The API key is marked sensitive and is not written to state by the provider.

## First configuration

This small configuration creates the API’s composite network and an SSH keypair. It demonstrates references, rather than copying IDs into many files.

```hcl
resource "nobus_keypair" "operator" {
  name       = "operator"
  public_key = file("~/.ssh/id_ed25519.pub")
}

resource "nobus_network" "app" {
  name        = "app"
  subnet_cidr = "10.20.0.0/24"
  router_name = "app-router"
  tags        = ["environment=dev"]
}
```

Run the normal Terraform workflow:

```bash
terraform init
terraform fmt -recursive
terraform validate
terraform plan
terraform apply
```

Review the plan before applying it. It describes whether Terraform will create, update, replace, or destroy each remote object. On shared environments, use a remote state backend with locking and restrict access because state can contain sensitive resource inputs and service credentials.

## Project and availability-zone scope

Almost all managed resources have optional `project_id` and `availability_zone` fields. A resource-level value overrides the provider default. Scope becomes state, so moving an object to another project or availability zone is modelled as replacement, not an in-place move.

```hcl
resource "nobus_volume" "regional_data" {
  name              = "regional-data"
  size              = 100
  project_id        = var.special_project_id
  availability_zone = "nobus-wa-az1"
}
```

Data sources also need whatever scope their underlying Nobus endpoint requires. Set provider defaults first; then specify a local scope only when intentionally querying another project or zone.

## Layout that scales

Keep provider configuration and sensitive variables separate from service modules. A practical starting point is:

```text
providers.tf     provider and required_providers
variables.tf     input contracts; mark secrets sensitive
network.tf       network, subnet, router, security
compute.tf       keypairs and instances
storage.tf       volumes and snapshots
outputs.tf       stable, non-secret integration values
```

Prefer passing IDs between modules rather than repeating names. Use data sources for externally owned objects, not as a substitute for a resource Terraform should own.
