output "instance_id" {
  value = nobus_instance.app.id
}

output "instance_status" {
  value = nobus_instance.app.status
}

output "network_id" {
  value = nobus_network.app.id
}

output "subnet_id" {
  value = nobus_subnet.app.id
}

output "floating_ip_id" {
  value = nobus_floating_ip.app.id
}

output "volume_id" {
  value = nobus_volume.data.id
}
