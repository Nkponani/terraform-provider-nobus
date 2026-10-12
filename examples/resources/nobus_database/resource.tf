variable "database_password" {
  type      = string
  sensitive = true
}

resource "nobus_database" "example" {
  name              = "orders"
  user_email        = "dba@example.com"
  flavor_name       = "standard"
  database_type     = "postgresql"
  database_version  = "16"
  size              = 100
  install_type      = "single"
  customer_username = "orders_admin"
  customer_password = var.database_password
  zones             = ["nobus-wa-az1"]
  encrypted         = true
}
