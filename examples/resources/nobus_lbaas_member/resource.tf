resource "nobus_lbaas_member" "web_01" {
  pool_id       = nobus_lbaas_pool.web.id
  address       = "10.20.0.10"
  protocol_port = 8080
  weight        = 1
}
