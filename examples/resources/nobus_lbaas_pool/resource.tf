resource "nobus_lbaas_pool" "web" {
  name         = "web"
  listener_id  = nobus_lbaas_listener.https.id
  protocol     = "HTTP"
  lb_algorithm = "ROUND_ROBIN"
}
