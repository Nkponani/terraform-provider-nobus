variable "object_source" {
  type        = string
  description = "Path to the local file to upload."
}

resource "nobus_fos_container" "example" {
  name = "terraform-assets"
}

resource "nobus_fos_object" "example" {
  container_name = nobus_fos_container.example.name
  object_name    = "release.tar.gz"
  source         = var.object_source
  source_hash    = filesha256(var.object_source)
}
