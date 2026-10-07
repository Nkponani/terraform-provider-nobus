resource "nobus_volume" "data" {
  name        = "production-data"
  description = "Database data volume"
  size        = 100

  metadata = {
    environment = "production"
    role        = "database"
  }
}

# A volume can grow in place. The provider calls the Nobus extend operation and
# waits for the volume to become available again.
resource "nobus_volume" "logs" {
  name = "production-logs"
  size = 50
}

output "volume_id" {
  description = "ID of the data volume."
  value       = nobus_volume.data.id
}

output "volume_status" {
  description = "Status reported by Nobus, for example `available` or `in-use`."
  value       = nobus_volume.data.status
}
