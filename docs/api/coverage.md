# Networking, block-storage, and FOS API coverage

This table tracks the live `https://cloud-api.nobus.io/api/v3/openapi.json`
networking, block-storage, and FOS operations exposed by the provider.

| API operation | Terraform surface |
| --- | --- |
| Network composite create/get/update/delete | `nobus_network` |
| Standalone network create | `nobus_standalone_network` |
| Network list | `nobus_networks` data source |
| Subnet create/get/update/delete/list | `nobus_subnet`, `nobus_subnets` |
| Port create/get/update/delete/list | `nobus_port`, `nobus_ports` |
| Set/clear port address pairs | `nobus_port_address_pairs` and `nobus_port.allowed_address_pairs` |
| Router create/get/update/delete/list | `nobus_router`, `nobus_routers` |
| Add/remove router interface | `nobus_router_interface` |
| Clear router gateway | `nobus_router.external_network_id = null` |
| Floating IP create/get/update/delete/list | `nobus_floating_ip`, `nobus_floating_ips` |
| Disassociate floating IP | `nobus_floating_ip.port_id = null` |
| Security group create/get/update/delete/list | `nobus_security_group`, `nobus_security_groups` |
| Security-group rule create/get/delete/list | `nobus_security_group_rule`, `nobus_security_group_rules` |
| Trunk create/get/update/delete/list | `nobus_trunk`, `nobus_trunks` |
| QoS policy create/get/update/delete/list | `nobus_qos_policy`, `nobus_qos_policies` |
| Firewall group create/get/update/delete/list | `nobus_firewall_group`, `nobus_firewall_groups` |
| Volume create/get/update/delete/list | `nobus_volume`, `nobus_volumes` |
| Volume attach/detach | `nobus_volume_attachment` |
| Volume extend | Increasing `nobus_volume.size` |
| Snapshot create/get/delete/list | `nobus_volume_snapshot`, `nobus_volume_snapshots` |
| Backup get/list | `nobus_volume_backup`, `nobus_volume_backups` data sources |
| Volume availability-zone list | `nobus_volume_availability_zones` data source |
| Volume type create | `nobus_volume_type` (the live API has no get/delete/list operation) |
| Volume transfer create/get/delete/list | `nobus_volume_transfer`, `nobus_volume_transfers` |
| Accept volume transfer | `nobus_volume_transfer_accept` |
| FOS account usage | `nobus_fos_account` data source |
| FOS authentication and Swift endpoint details | `nobus_fos_auth_details` data source |
| FOS container create/delete/list | `nobus_fos_container`, `nobus_fos_containers` |
| FOS object upload/delete/list | `nobus_fos_object`, `nobus_fos_objects` |
| FOS object download | `nobus_fos_object_content` data source (sensitive base64 state) |
| Kubernetes create/get/delete | `nobus_kubernetes_cluster` |
| Kubernetes horizontal/vertical/etcd-volume scaling | Updates to `nobus_kubernetes_cluster` |
| Kubernetes existing-cluster migration | `nobus_kubernetes_migration` |
| Kubernetes list/task status | `nobus_kubernetes_clusters`, `nobus_kubernetes_task_status` |
| Kubernetes bastion/LB/monitoring/node details | `nobus_kubernetes_bastion`, `nobus_kubernetes_load_balancer_login`, `nobus_kubernetes_monitoring`, `nobus_kubernetes_nodes` |
| Database create/get/delete | `nobus_database` |
| Database volume/flavor resize and password rotation | Updates to `nobus_database` |
| Database resume/general lifecycle actions | `nobus_database_action` |
| Register existing databases | `nobus_existing_databases` |
| Database list/status/creation status | `nobus_databases`, `nobus_database_status`, `nobus_database_creation_status` |
| Database credentials/monitoring credentials/logs | `nobus_database_credentials`, `nobus_database_zabbix_credentials`, `nobus_database_logs` |
| LBaaS load-balancer CRUD/security-group association | `nobus_lbaas_load_balancer` |
| LBaaS listener/pool/member/health-monitor CRUD | Corresponding `nobus_lbaas_*` resources and plural data sources |
| LBaaS L7 policy/rule CRUD | `nobus_lbaas_l7_policy`, `nobus_lbaas_l7_rule`, and plural data sources |
| LBaaS pool session persistence | Updates to `nobus_lbaas_pool` |
| LBaaS security-group rule create/delete | `nobus_lbaas_security_group_rule` |
| LBaaS failover and composite provision | `nobus_lbaas_failover`, `nobus_lbaas_provision` |
| LBaaS tasks/deployment/status/stats | `nobus_lbaas_task`, `nobus_lbaas_deployment_status`, `nobus_lbaas_load_balancer_status`, `nobus_lbaas_load_balancer_stats`, `nobus_lbaas_listener_stats` |

The live specification no longer exposes backup create/delete/restore or
consistency-group endpoints. They are therefore intentionally not registered
as Terraform resources.
