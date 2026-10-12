data "nobus_fos_object_content" "manifest" {
  container_name = "terraform-assets"
  object_name    = "manifest.json"
}

output "manifest_sha256" {
  value = data.nobus_fos_object_content.manifest.sha256
}
