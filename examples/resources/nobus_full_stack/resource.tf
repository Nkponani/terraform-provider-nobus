# A small combined example: a network, a volume and an instance that depends on
# both. Terraform builds the dependency graph from the references below, so no
# explicit depends_on is needed.

resource "nobus_network" "app" {
  name        = "app-network"
  subnet_cidr = "10.30.0.0/24"
  router_name = "app-router"
}

resource "nobus_volume" "database" {
  name        = "app-database"
  description = "Database volume for the API instance"
  size        = 100
}

resource "nobus_instance" "api" {
  name     = "api-01"
  image    = "ubuntu-22-04"
  flavor   = "medium"
  key_name = "operations"

  # The instance is created after the network exists, because the reference
  # below becomes part of the create payload.
  network_id = nobus_network.app.id

  volume_size     = 80
  security_groups = ["default"]
}

# Attach the data volume to the instance. Both resources must exist first, so the
# volume and the instance are created before this runs.
resource "nobus_volume_attachment" "database" {
  instance_id = nobus_instance.api.id
  volume_id   = nobus_volume.database.id
}

output "network_id" {
  value = nobus_network.app.id
}

output "instance_id" {
  value = nobus_instance.api.id
}

output "instance_addresses" {
  value = jsondecode(nobus_instance.api.addresses)
}

output "attached_volume_id" {
  value = nobus_volume_attachment.database.volume_id
}
