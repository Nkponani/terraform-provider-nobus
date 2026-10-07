# The Nobus API requires a project, an availability zone, an image, a flavor and
# a key name to create an instance.
resource "nobus_instance" "web" {
  name     = "web-01"
  image    = "ubuntu-22-04"
  flavor   = "small"
  key_name = "operations"

}

output "instance_id" {
  description = "ID of the instance."
  value       = nobus_instance.web.id
}

output "instance_status" {
  description = "Status reported by Nobus, for example `ACTIVE`."
  value       = nobus_instance.web.status
}

# InstanceSchema.addresses is documented only as an untyped object, so it is
# exposed as JSON. Use jsondecode() to read it.
output "instance_addresses" {
  description = "Addresses assigned to the instance, as reported by Nobus."
  value       = jsondecode(nobus_instance.web.addresses)
}
