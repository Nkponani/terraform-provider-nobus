# The Nobus API creates a network, a subnet and a router in a single call, so a
# network requires a subnet CIDR and a router name.
resource "nobus_network" "private" {
  name        = "production-network"
  subnet_cidr = "10.20.0.0/24"
  router_name = "production-router"

  description    = "Primary private network"
  enable_snat    = true
  admin_state_up = true
  tags           = ["environment=production", "team=platform"]
}

output "network_id" {
  description = "ID of the network created by Nobus."
  value       = nobus_network.private.id
}

output "subnet_id" {
  description = "ID of the subnet created together with the network."
  value       = nobus_network.private.subnet_id
}

output "router_id" {
  description = "ID of the router created together with the network."
  value       = nobus_network.private.router_id
}
