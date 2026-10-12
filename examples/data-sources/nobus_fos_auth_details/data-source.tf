data "nobus_fos_auth_details" "current" {}

output "fos_management_endpoint" {
  value = data.nobus_fos_auth_details.current.management_endpoint
}
