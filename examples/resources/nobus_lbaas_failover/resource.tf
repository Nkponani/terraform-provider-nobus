resource "nobus_lbaas_failover" "example" {
  load_balancer_id = nobus_lbaas_load_balancer.example.id
  trigger          = "maintenance-2026-10-12"
}
