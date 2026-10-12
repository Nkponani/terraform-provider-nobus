resource "nobus_lbaas_load_balancer" "example" {
  name        = "production-edge"
  flavor      = "small"
  router_id   = "ROUTER_ID"
  volume_size = 20
  tags        = ["environment=production"]
}
