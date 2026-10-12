data "nobus_fos_objects" "assets" {
  container_name = "terraform-assets"
  prefix         = "releases/"
  full_listing   = true
}
