resource "nobus_existing_databases" "example" {
  payload_json = jsonencode([{
    project_id        = var.nobus_project_id
    availability_zone = "nobus-wa-az1"
    user_email        = "dba@example.com"
    database_name     = "existing-orders"
    database_type     = "postgresql"
    database_version  = "16"
    installation_type = "single"
    customer_user     = "orders_admin"
    customer_password = var.database_password
    flavor_name       = "standard"
    image_name        = "ubuntu"
    number_of_masters = 1
    number_of_slaves  = 0
    number_of_azs     = 1
    database_size     = 100
    instances         = []
  }])
}
