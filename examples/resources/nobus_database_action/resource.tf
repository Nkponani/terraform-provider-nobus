resource "nobus_database_action" "restart" {
  database_id   = nobus_database.example.id
  database_type = nobus_database.example.database_type
  action        = "restart"
}
