resource "nobus_kubernetes_migration" "example" {
  payload_json = jsonencode({
    cluster_name      = "existing-k8s"
    project_id        = var.nobus_project_id
    availability_zone = "nobus-wa-az1"
    user_email        = "operator@example.com"
    network_id        = "NETWORK_ID"
    nodes = [{
      instance_id = "INSTANCE_ID"
      role        = "master"
    }]
  })
}
