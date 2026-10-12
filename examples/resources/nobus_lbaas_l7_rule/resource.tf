resource "nobus_lbaas_l7_rule" "api_path" {
  l7_policy_id = nobus_lbaas_l7_policy.api.id
  type         = "PATH"
  compare_type = "STARTS_WITH"
  value        = "/api"
}
