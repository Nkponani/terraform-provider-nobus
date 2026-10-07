terraform {
  required_providers {
    nobus = {
      source = "nobus-cloud/nobus"
    }
  }
}

# Reads NOBUS_API_KEY, NOBUS_PROJECT_ID, and NOBUS_AVAILABILITY_ZONE from the
# environment.
provider "nobus" {}

resource "nobus_instance" "test" {
  name     = "abubak"
  image    = "8098febb-17a2-4c44-81dd-cfae9af1887f"
  flavor   = "0e5ac069-c0e5-44bd-8ab7-39858863fe89"
  key_name = "personal"

  volume_size     = 50
  network_id      = "15d305ae-29b0-4cb0-b475-67e70750fb5f"
  security_groups = ["default"]
  admin_pass      = "string"
  user_data       = "string"
}

output "instance_id" {
  value = nobus_instance.test.id
}

output "instance_addresses" {
  value = jsondecode(nobus_instance.test.addresses)
}
