resource "nobus_lbaas_security_group_rule" "https" {
  load_balancer_id = nobus_lbaas_load_balancer.example.id
  port             = 443
  source           = "0.0.0.0/0"
}
