data "nobus_fos_account" "current" {}

output "fos_bytes_used" {
  value = data.nobus_fos_account.current.bytes_used
}
