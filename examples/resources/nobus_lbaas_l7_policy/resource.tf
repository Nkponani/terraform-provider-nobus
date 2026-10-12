resource "nobus_lbaas_l7_policy" "api" {
  listener_id      = nobus_lbaas_listener.https.id
  name             = "api"
  action           = "REDIRECT_TO_POOL"
  redirect_pool_id = nobus_lbaas_pool.web.id
}
