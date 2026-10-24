# Data-source catalog

This catalog lists every data source registered by the provider. The generated page linked for each source defines its exact arguments, filtering, result shape, and sensitivity. Sources read the Nobus API; they do not grant Terraform ownership.

## Networking inventory

| Data source | Returns |
| --- | --- |
| [`nobus_networks`](../docs/data-sources/networks.md) | networks visible in the requested scope |
| [`nobus_subnets`](../docs/data-sources/subnets.md) | subnets visible in scope |
| [`nobus_ports`](../docs/data-sources/ports.md) | ports/network interfaces |
| [`nobus_routers`](../docs/data-sources/routers.md) | routers |
| [`nobus_floating_ips`](../docs/data-sources/floating_ips.md) | floating-IP allocations/associations |
| [`nobus_security_groups`](../docs/data-sources/security_groups.md) | security groups |
| [`nobus_security_group_rules`](../docs/data-sources/security_group_rules.md) | security-group rules |
| [`nobus_trunks`](../docs/data-sources/trunks.md) | port trunks and subports |
| [`nobus_qos_policies`](../docs/data-sources/qos_policies.md) | QoS policies |
| [`nobus_firewall_groups`](../docs/data-sources/firewall_groups.md) | firewall groups |

## Instance inspection

| Data source | Returns |
| --- | --- |
| [`nobus_instance_network_interfaces`](../docs/data-sources/instance_network_interfaces.md) | interfaces for a given instance |
| [`nobus_instance_diagnostics`](../docs/data-sources/instance_diagnostics.md) | diagnostic information for an instance |
| [`nobus_instance_security_groups`](../docs/data-sources/instance_security_groups.md) | security groups applied to an instance |
| [`nobus_instance_tags`](../docs/data-sources/instance_tags.md) | instance tags |
| [`nobus_instance_topology`](../docs/data-sources/instance_topology.md) | instance topology information |

Use these for observation when the main instance resource does not expose the detail. They may change at refresh and should not be treated as a stable application API without validating the response contract.

## Block storage inventory

| Data source | Returns |
| --- | --- |
| [`nobus_volumes`](../docs/data-sources/volumes.md) | volume inventory |
| [`nobus_volume_snapshots`](../docs/data-sources/volume_snapshots.md) | snapshot inventory |
| [`nobus_volume_backups`](../docs/data-sources/volume_backups.md) | backup inventory |
| [`nobus_volume_backup`](../docs/data-sources/volume_backup.md) | full details for one backup ID |
| [`nobus_volume_transfers`](../docs/data-sources/volume_transfers.md) | transfer inventory |
| [`nobus_volume_availability_zones`](../docs/data-sources/volume_availability_zones.md) | available storage zones |

## FOS reads

| Data source | Returns |
| --- | --- |
| [`nobus_fos_account`](../docs/data-sources/fos_account.md) | account information |
| [`nobus_fos_auth_details`](../docs/data-sources/fos_auth_details.md) | FOS authentication details; handle as secret material |
| [`nobus_fos_containers`](../docs/data-sources/fos_containers.md) | container list/details |
| [`nobus_fos_objects`](../docs/data-sources/fos_objects.md) | object listing |
| [`nobus_fos_object_content`](../docs/data-sources/fos_object_content.md) | object body as sensitive base64, stored in state |

## Kubernetes reads

| Data source | Returns |
| --- | --- |
| [`nobus_kubernetes_clusters`](../docs/data-sources/kubernetes_clusters.md) | managed cluster inventory |
| [`nobus_kubernetes_bastion`](../docs/data-sources/kubernetes_bastion.md) | bastion access details |
| [`nobus_kubernetes_load_balancer_login`](../docs/data-sources/kubernetes_load_balancer_login.md) | load-balancer login/access details |
| [`nobus_kubernetes_monitoring`](../docs/data-sources/kubernetes_monitoring.md) | Grafana/monitoring details |
| [`nobus_kubernetes_nodes`](../docs/data-sources/kubernetes_nodes.md) | cluster node data |
| [`nobus_kubernetes_task_status`](../docs/data-sources/kubernetes_task_status.md) | status/details for a known task |

Access-related values may be sensitive. Pass them straight into a protected consumer when possible and avoid top-level outputs.

## Database reads

| Data source | Returns |
| --- | --- |
| [`nobus_databases`](../docs/data-sources/databases.md) | database inventory |
| [`nobus_database_credentials`](../docs/data-sources/database_credentials.md) | connection credentials; sensitive |
| [`nobus_database_creation_status`](../docs/data-sources/database_creation_status.md) | provisioning status |
| [`nobus_database_status`](../docs/data-sources/database_status.md) | database health/status |
| [`nobus_database_zabbix_credentials`](../docs/data-sources/database_zabbix_credentials.md) | monitoring credentials; sensitive |
| [`nobus_database_logs`](../docs/data-sources/database_logs.md) | database logs |

## LBaaS reads

| Data source | Returns |
| --- | --- |
| [`nobus_lbaas_load_balancers`](../docs/data-sources/lbaas_load_balancers.md) | load balancers |
| [`nobus_lbaas_listeners`](../docs/data-sources/lbaas_listeners.md) | listeners |
| [`nobus_lbaas_pools`](../docs/data-sources/lbaas_pools.md) | pools |
| [`nobus_lbaas_members`](../docs/data-sources/lbaas_members.md) | members |
| [`nobus_lbaas_health_monitors`](../docs/data-sources/lbaas_health_monitors.md) | health monitors |
| [`nobus_lbaas_l7_policies`](../docs/data-sources/lbaas_l7_policies.md) | L7 policies |
| [`nobus_lbaas_l7_rules`](../docs/data-sources/lbaas_l7_rules.md) | L7 rules |
| [`nobus_lbaas_task`](../docs/data-sources/lbaas_task.md) | status/details for one LBaaS task |
| [`nobus_lbaas_deployment_status`](../docs/data-sources/lbaas_deployment_status.md) | deployment progression/status |
| [`nobus_lbaas_load_balancer_status`](../docs/data-sources/lbaas_load_balancer_status.md) | status tree for a load balancer |
| [`nobus_lbaas_load_balancer_stats`](../docs/data-sources/lbaas_load_balancer_stats.md) | load-balancer metrics |
| [`nobus_lbaas_listener_stats`](../docs/data-sources/lbaas_listener_stats.md) | listener metrics |

## Source selection rules

Use a specific source when the provider has one and you know the target ID; use a collection source to discover/filter inventory. Keep selection deterministic: if a collection can return zero or many results, validate that assumption in Terraform expressions or module inputs. Do not use collection order as identity, and do not use sensitive source output in an ordinary output just because Terraform allowed the reference.
