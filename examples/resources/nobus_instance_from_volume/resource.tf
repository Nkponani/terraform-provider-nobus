resource "nobus_instance_from_volume" "web" {
  name      = "web-from-vol"
  image     = "ubuntu-22-04"
  flavor    = "small"
  key_name  = "operations"
  volume_id = "vol-123"
}
