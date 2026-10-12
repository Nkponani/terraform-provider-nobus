resource "nobus_lbaas_listener" "https" {
  name             = "https"
  load_balancer_id = nobus_lbaas_load_balancer.example.id
  protocol         = "HTTPS"
  protocol_port    = 443
  allow_from       = "0.0.0.0/0"
}
