resource "nobus_kubernetes_cluster" "example" {
  name                = "production-k8s"
  kubernetes_version  = "1.29.0"
  master_count        = 2
  worker_count        = 3
  etcd_volume_size_gb = 50
  master_flavor_id    = "MASTER_FLAVOR_ID"
  worker_flavor_id    = "WORKER_FLAVOR_ID"
  image_id            = "IMAGE_ID"
  router_id           = "ROUTER_ID"
}
