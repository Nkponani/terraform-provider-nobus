# Managed-resource catalog

This is a map of every managed resource registered by the provider. It explains the ownership boundary and points to the generated schema for every exact field. Use the service workflow pages for composition guidance.

## Networking

| Resource | Owns | Lifecycle note |
| --- | --- | --- |
| [`nobus_network`](../docs/resources/network.md) | a network plus the initial subnet and router created by Nobus | initial CIDR/router topology is create-only; exposes subnet/router IDs |
| [`nobus_standalone_network`](../docs/resources/standalone_network.md) | a bare network | compose subnets and routers separately |
| [`nobus_subnet`](../docs/resources/subnet.md) | subnet addressing and DHCP/DNS/routes/pools | network, CIDR, and IP version establish identity |
| [`nobus_router`](../docs/resources/router.md) | a router | attach it with a separate interface resource |
| [`nobus_router_interface`](../docs/resources/router_interface.md) | a router-to-subnet or router-to-port connection | relationship is replace-only; import has a composite ID |
| [`nobus_port`](../docs/resources/port.md) | a network interface | port identity/placement changes can affect consumers |
| [`nobus_port_address_pairs`](../docs/resources/port_address_pairs.md) | one port’s allowed-address-pair collection | one owner per port’s pair set |
| [`nobus_floating_ip`](../docs/resources/floating_ip.md) | public IP allocation and optional port association | association follows port lifecycle |
| [`nobus_security_group`](../docs/resources/security_group.md) | a named network security policy group | create rules separately |
| [`nobus_security_group_rule`](../docs/resources/security_group_rule.md) | one ingress or egress rule | model each allowed flow explicitly |
| [`nobus_trunk`](../docs/resources/trunk.md) | parent port plus VLAN subports | use only with a compatible VLAN design |
| [`nobus_qos_policy`](../docs/resources/qos_policy.md) | Nobus QoS policy | confirm attachment/consumer semantics in API docs |
| [`nobus_firewall_group`](../docs/resources/firewall_group.md) | Nobus firewall group | coordinate policy ownership with network operations |

## Compute

| Resource | Owns | Lifecycle note |
| --- | --- | --- |
| [`nobus_instance`](../docs/resources/instance.md) | a VM created from image/flavor/key/network inputs | boot inputs are replacement choices; waits for configured ready state |
| [`nobus_instance_from_volume`](../docs/resources/instance_from_volume.md) | a VM booted from an existing volume | create-only boot/server inputs; no normal import implementation |
| [`nobus_keypair`](../docs/resources/keypair.md) | public SSH keypair | never put the private key in Terraform |
| [`nobus_instance_action`](../docs/resources/instance_action.md) | one imperative compute action | removal does not undo completed action |

## Block storage

| Resource | Owns | Lifecycle note |
| --- | --- | --- |
| [`nobus_volume`](../docs/resources/volume.md) | block volume | can grow; cannot shrink through this resource |
| [`nobus_volume_attachment`](../docs/resources/volume_attachment.md) | volume-to-instance attachment | waits for attachment/detachment; import `instance/volume` |
| [`nobus_volume_snapshot`](../docs/resources/volume_snapshot.md) | point-in-time snapshot | source/creation values are immutable |
| [`nobus_volume_type`](../docs/resources/volume_type.md) | volume type creation | API has no read/delete; state removal does not delete remote type |
| [`nobus_volume_transfer`](../docs/resources/volume_transfer.md) | transfer offer | sensitive authorization key; one-time handoff |
| [`nobus_volume_transfer_accept`](../docs/resources/volume_transfer_accept.md) | accepted transfer | model as an explicit ownership transition |

## File/Object Storage

| Resource | Owns | Lifecycle note |
| --- | --- | --- |
| [`nobus_fos_container`](../docs/resources/fos_container.md) | FOS container | visibility change replaces; empty it before delete |
| [`nobus_fos_object`](../docs/resources/fos_object.md) | local-file upload at a container/object path | use `source_hash` to detect content changes; import `container/object` |

## Managed Kubernetes and databases

| Resource | Owns | Lifecycle note |
| --- | --- | --- |
| [`nobus_kubernetes_cluster`](../docs/resources/kubernetes_cluster.md) | managed cluster | node counts/flavors/etcd size invoke service scale operations as supported |
| [`nobus_kubernetes_migration`](../docs/resources/kubernetes_migration.md) | registration request for existing cluster | sensitive JSON; deletion removes state, not the cluster |
| [`nobus_database`](../docs/resources/database.md) | managed database service | resize/flavor/password paths are service operations; topology changes can replace |
| [`nobus_database_action`](../docs/resources/database_action.md) | one database lifecycle action | state deletion does not reverse it |
| [`nobus_existing_databases`](../docs/resources/existing_databases.md) | registration of existing databases | sensitive JSON; does not manage customer database deletion |

## Load Balancing as a Service

| Resource | Owns | Lifecycle note |
| --- | --- | --- |
| [`nobus_lbaas_load_balancer`](../docs/resources/lbaas_load_balancer.md) | LB frontend and associated security-group relation | parent of listener hierarchy |
| [`nobus_lbaas_listener`](../docs/resources/lbaas_listener.md) | frontend protocol/port listener | reference the load-balancer ID |
| [`nobus_lbaas_pool`](../docs/resources/lbaas_pool.md) | backend pool and session policy | reference listener/pool relationship as schema requires |
| [`nobus_lbaas_member`](../docs/resources/lbaas_member.md) | one backend endpoint | address/port/pool identify membership |
| [`nobus_lbaas_health_monitor`](../docs/resources/lbaas_health_monitor.md) | health checking for a pool | tune against actual readiness behavior |
| [`nobus_lbaas_l7_policy`](../docs/resources/lbaas_l7_policy.md) | listener routing policy | policy order/action affects traffic |
| [`nobus_lbaas_l7_rule`](../docs/resources/lbaas_l7_rule.md) | a condition under an L7 policy | manage order and match semantics carefully |
| [`nobus_lbaas_security_group_rule`](../docs/resources/lbaas_security_group_rule.md) | LB-specific CIDR/port rule | separate from compute security rules |
| [`nobus_lbaas_failover`](../docs/resources/lbaas_failover.md) | one failover trigger | explicit irreversible operational side effect |
| [`nobus_lbaas_provision`](../docs/resources/lbaas_provision.md) | composite LBaaS provision request | do not overlap ownership with individual children |

## Reading the generated schemas

For each page, use the schema sections as a contract:

- **Required** values must be in configuration for creation.
- **Optional** values can be omitted; some are also computed to support imports/API defaults.
- **Read-only** values are returned from Nobus and are suitable for references/outputs.
- A “forces new resource” note means Terraform plans replacement; inspect dependent resources and data safety before applying.
- A **Sensitive** marker hides routine output only; it is not a substitute for state and secret controls.

## Choosing a resource boundary

Choose the smallest resource that owns the API object you actually intend to
control. For example, use a separate listener, pool, and member when they are
released on different schedules; use the composite LBaaS provision resource
only when the composite API request is the intended ownership boundary. Do not
create a resource merely to query an object—use the corresponding data source.

Where an API object has dependencies, express them with resource references
rather than copied IDs. References make the plan explain the relationship and
let Terraform derive creation and deletion ordering.
