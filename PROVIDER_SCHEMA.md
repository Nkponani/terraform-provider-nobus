# Provider Configuration Schema Reference

Complete reference for the `nobus` provider configuration block.

## Provider Block

```hcl
terraform {
  required_providers {
    nobus = {
      source  = "nobus-cloud/nobus"
      version = "~> 0.1"
    }
  }
}

provider "nobus" {
  # Required
  api_key = var.nobus_api_key

  # Optional (with defaults)
  endpoint           = "https://cloud-api.nobus.io"
  project_id         = var.nobus_project_id
  availability_zone  = "nobus-wa-az1"
}
```

## Configuration Attributes

| Attribute | Type | Required | Default | Environment Variable | Description |
|-----------|------|----------|---------|---------------------|-------------|
| `api_key` | string | **Yes** | - | `NOBUS_API_KEY` | Nobus API key used as bearer token. **Marked sensitive** - never written to state or logs. |
| `endpoint` | string | No | `https://cloud-api.nobus.io` | `NOBUS_ENDPOINT` | Base URL of the Nobus API. Must include scheme (https://). |
| `project_id` | string | No* | - | `NOBUS_PROJECT_ID` | Default project ID for resources. Required on resources if not set here. |
| `availability_zone` | string | No | `nobus-wa-az1` | `NOBUS_AVAILABILITY_ZONE` | Default availability zone for resources. |

*Required on resources that don't set their own `project_id`/`availability_zone`.

## Attribute Details

### api_key

**Type:** `string` (Sensitive)

**Description:** The Nobus API key used for authentication. The provider sends this as a `Bearer` token in the `Authorization` header for all API requests.

**Security:**
- Marked `sensitive = true` in Terraform
- Never appears in plan output, state, or logs
- Should be stored in a secret manager or environment variable

**Generation:**
1. Log into Nobus Cloud Console
2. Navigate to API Keys
3. Create new API key with appropriate scopes
4. Copy the key (shown only once)

**Example:**
```hcl
variable "nobus_api_key" {
  type      = string
  sensitive = true
}

provider "nobus" {
  api_key = var.nobus_api_key
}
```

### endpoint

**Type:** `string`

**Default:** `https://cloud-api.nobus.io`

**Description:** Base URL for the Nobus API. The provider automatically appends `/api/v3` to this URL.

**Use cases:**
- Production: `https://cloud-api.nobus.io` (default)
- Testing against mock server: `http://localhost:8080`
- Regional endpoints (if available)

**Example:**
```hcl
provider "nobus" {
  endpoint = "https://cloud-api.nobus.io"
}
```

### project_id

**Type:** `string`

**Default:** None (must be set either here or on each resource)

**Description:** Default Nobus project ID. Every Nobus API operation requires a project ID. Setting it on the provider avoids repetition on each resource.

**Resource override:** Resources can set their own `project_id` which takes precedence:
```hcl
resource "nobus_volume" "example" {
  project_id = "different-project"  # Overrides provider default
  name       = "test"
  size       = 100
}
```

**Example:**
```hcl
provider "nobus" {
  project_id = "proj-abc123"
}
```

### availability_zone

**Type:** `string`

**Default:** `nobus-wa-az1`

**Description:** Default availability zone for resources. The Nobus API requires an availability zone for all operations. This default matches the Nobus API's own default.

**Available zones** (query via API):
- `nobus-wa-az1` (Washington DC)
- `nobus-ny-az1` (New York)
- `nobus-lon-az1` (London)
- `nobus-fra-az1` (Frankfurt)
- `nobus-sgp-az1` (Singapore)

**Resource override:** Resources can set their own `availability_zone`:
```hcl
resource "nobus_instance" "example" {
  availability_zone = "nobus-ny-az1"  # Overrides provider default
  name     = "test"
  image    = "ubuntu-22-04"
  flavor   = "small"
  key_name = "my-key"
}
```

**Example:**
```hcl
provider "nobus" {
  availability_zone = "nobus-wa-az1"
}
```

## Complete Example

```hcl
terraform {
  required_version = ">= 1.0"
  
  required_providers {
    nobus = {
      source  = "nobus-cloud/nobus"
      version = "~> 0.1"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "nobus" {
  # API key from environment variable (recommended)
  # api_key = var.nobus_api_key
  
  # Or explicitly from variable
  api_key           = var.nobus_api_key
  
  # Optional overrides
  endpoint           = var.nobus_endpoint
  project_id         = var.nobus_project_id
  availability_zone  = var.nobus_availability_zone
}

variable "nobus_api_key" {
  description = "Nobus API key"
  type        = string
  sensitive   = true
}

variable "nobus_endpoint" {
  description = "Nobus API endpoint"
  type        = string
  default     = "https://cloud-api.nobus.io"
}

variable "nobus_project_id" {
  description = "Default Nobus project ID"
  type        = string
  default     = "my-project"
}

variable "nobus_availability_zone" {
  description = "Default availability zone"
  type        = string
  default     = "nobus-wa-az1"
}

# Use random provider for unique names
resource "random_pet" "name" {
  length    = 2
  separator = "-"
}

# Example resources using provider defaults
resource "nobus_network" "main" {
  name        = "network-${random_pet.name.id}"
  subnet_cidr = "10.0.0.0/24"
  router_name = "router-${random_pet.name.id}"
}

resource "nobus_volume" "data" {
  name = "volume-${random_pet.name.id}"
  size = 50
}

resource "nobus_instance" "app" {
  name     = "instance-${random_pet.name.id}"
  image    = "ubuntu-22-04"
  flavor   = "small"
  key_name = "my-keypair"
  network_id = nobus_network.main.id
}

output "network_id" {
  value = nobus_network.main.id
}

output "volume_id" {
  value = nobus_volume.data.id
}

output "instance_id" {
  value = nobus_instance.app.id
}
```

## Environment Variable Reference

All provider attributes can be set via environment variables:

| Provider Attribute | Environment Variable | Required |
|-------------------|---------------------|----------|
| `api_key` | `NOBUS_API_KEY` | **Yes** |
| `endpoint` | `NOBUS_ENDPOINT` | No |
| `project_id` | `NOBUS_PROJECT_ID` | No* |
| `availability_zone` | `NOBUS_AVAILABILITY_ZONE` | No |

*Required if not set in provider block and not set on resource.

**Precedence Order:**
1. Provider block attribute (explicit)
2. Environment variable
3. Hard-coded default (endpoint, availability_zone only)

**Example with environment variables:**
```bash
export NOBUS_API_KEY="sk-..."
export NOBUS_PROJECT_ID="proj-abc123"
export NOBUS_AVAILABILITY_ZONE="nobus-ny-az1"

terraform init
terraform plan
terraform apply
```

## Resource-Level Configuration

Every resource supports optional `project_id` and `availability_zone` attributes that override the provider defaults:

```hcl
resource "nobus_volume" "example" {
  # These override provider defaults
  project_id         = "other-project"
  availability_zone  = "nobus-fra-az1"
  
  # Resource-specific attributes
  name = "my-volume"
  size = 100
}
```

## Data Source Configuration

Data sources also support `project_id` and `availability_zone` overrides:

```hcl
data "nobus_volumes" "all" {
  project_id        = "other-project"
  availability_zone = "nobus-fra-az1"
}
```

## Validation Rules

The provider validates configuration at startup:

1. **api_key** - Must be non-empty (checked in `Configure`)
2. **endpoint** - Must be valid URL with scheme
3. **project_id** - Required for resource operations (validated per-resource)
4. **availability_zone** - Must be valid zone (validated by API on first operation)

## Migration Notes

### From Environment Variables Only

If you previously used only environment variables, add explicit provider block:

```hcl
# Before (env vars only)
# export NOBUS_API_KEY="..."
# export NOBUS_PROJECT_ID="..."

# After (explicit provider block)
provider "nobus" {
  api_key   = var.nobus_api_key
  project_id = var.nobus_project_id
}
```

### Changing Defaults

To change default availability zone:
```hcl
provider "nobus" {
  availability_zone = "nobus-ny-az1"  # New default
}
```

Existing resources keep their zone; new resources use the new default.