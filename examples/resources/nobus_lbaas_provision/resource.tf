resource "nobus_lbaas_provision" "example" {
  payload_json = jsonencode({
    name              = "production-edge"
    project_id        = var.nobus_project_id
    availability_zone = "nobus-wa-az1"
    flavor            = "small"
    listeners = [{
      name          = "http"
      protocol      = "HTTP"
      protocol_port = 80
    }]
    pools = []
  })
}
