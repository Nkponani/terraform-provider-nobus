# Smoke test for the DEPLOYED nobus-cloud/nobus provider (v0.1.0).
#
# It creates one 1 GiB volume, reads it back, then destroys it. That exercises
# the full create -> read -> delete lifecycle against the real Nobus API and
# proves the published binary can authenticate with your API key and project.
#
# A volume is the cheapest/cleanest thing to test: no image, flavor, network
# or keypair needed, and destroy removes it so nothing is left behind.
#
# NOTE: The deployed v0.1.0 binary only ships four resources
# (nobus_network, nobus_volume, nobus_instance, nobus_volume_attachment)
# and no data sources, so this test sticks to what the published build
# actually supports.
#
# Credentials come from the environment (see the README):
#   export NOBUS_API_KEY="..."
#   export NOBUS_PROJECT_ID="..."
#
# Run:
#   terraform init
#   terraform apply -auto-approve
#   terraform destroy -auto-approve   # cleans up the test volume

terraform {
  required_providers {
    nobus = {
      # Pulls the published artifact from the Terraform registry, not a local
      # build, so this exercises what users actually install.
      source = "nobus-cloud/nobus"
    }
  }
}

# api_key and project_id are read from NOBUS_API_KEY / NOBUS_PROJECT_ID.
provider "nobus" {}

resource "nobus_volume" "smoke" {
  name = "tf-smoke-test"
  size = 1

  metadata = {
    created_by = "terraform-smoke-test"
  }
}

output "volume_id" {
  description = "ID of the smoke-test volume created by the deployed provider."
  value       = nobus_volume.smoke.id
}

output "volume_status" {
  description = "Status reported by Nobus after creation (expect 'available')."
  value       = nobus_volume.smoke.status
}

output "volume_size" {
  description = "Size of the smoke-test volume in GiB."
  value       = nobus_volume.smoke.size
}

output "volume_created_at" {
  description = "Creation timestamp reported by Nobus."
  value       = nobus_volume.smoke.created_at
}
