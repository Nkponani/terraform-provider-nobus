resource "nobus_lbaas_health_monitor" "web" {
  pool_id     = nobus_lbaas_pool.web.id
  type        = "HTTP"
  delay       = 5
  timeout     = 3
  max_retries = 3
  url_path    = "/health"
}
