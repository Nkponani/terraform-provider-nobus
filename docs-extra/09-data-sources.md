# Data sources

Data sources fetch remote data into a Terraform plan. They do not establish ownership and do not make a remote object safe to destroy. Use them for discovery, integration values, status checks, and objects managed outside the current state.

## Collection sources

The provider has plural list sources for these service inventories:

| Service | Sources |
| --- | --- |
| Networking | `nobus_networks`, `nobus_subnets`, `nobus_ports`, `nobus_routers`, `nobus_floating_ips`, `nobus_security_groups`, `nobus_security_group_rules`, `nobus_trunks`, `nobus_qos_policies`, `nobus_firewall_groups` |
| Storage | `nobus_volumes`, `nobus_volume_snapshots`, `nobus_volume_backups`, `nobus_volume_transfers`, `nobus_volume_availability_zones` |
| FOS | `nobus_fos_containers`, `nobus_fos_objects` |
| Kubernetes | `nobus_kubernetes_clusters` |
| Databases | `nobus_databases` |
| LBaaS | `nobus_lbaas_load_balancers`, `nobus_lbaas_listeners`, `nobus_lbaas_pools`, `nobus_lbaas_members`, `nobus_lbaas_health_monitors`, `nobus_lbaas_l7_policies`, `nobus_lbaas_l7_rules` |

Each source page in [`docs/data-sources`](../docs/data-sources) defines filtering and returned lists. Lists can be empty and can change after plan; write expressions defensively and avoid indexing `[0]` unless an empty result is an intentional hard failure.

## Focused reads

| Area | Focused sources | Sensitivity/usage |
| --- | --- | --- |
| Instance | network interfaces, diagnostics, security groups, tags, topology | operational inspection |
| Storage | `nobus_volume_backup` | full details for a known backup ID |
| FOS | account, auth details, object content | auth/content results require state caution |
| Kubernetes | bastion, load-balancer login, monitoring, nodes, task status | access/monitoring values can be secret |
| Database | credentials, creation status, status, Zabbix credentials, logs | credentials are sensitive; logs can be large/sensitive |
| LBaaS | task, deployment status, load-balancer status/stats, listener stats | verification and metrics, not steady-state ownership |

Example: pass a discovered volume ID to a consumer only after deciding it is externally owned.

```hcl
data "nobus_volumes" "existing" {
  # Use the schema’s supported scope/filter fields.
}

# Prefer a clear selection strategy over assuming list order.
```

## Plan-time behavior and secrets

Terraform usually reads data sources during planning. A source depending on a resource that has not yet been created may be deferred to apply, leaving values unknown in the plan. Use direct resource outputs when you control the object; use data source references when the object must be looked up.

Values marked sensitive are masked in Terraform’s normal output but are not automatically removed from state. Avoid feeding credentials into non-sensitive outputs, resource names, tags, or error-prone external provisioners. For raw JSON result fields, decode only known fields and do not log the full payload.
