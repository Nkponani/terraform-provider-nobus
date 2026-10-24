# Networking

Networking starts with a deliberate choice: `nobus_network` is Nobus’s composite create operation (network + one subnet + one router); `nobus_standalone_network` creates only a network. Use the composite resource for the usual isolated application network and standalone networking when you need to compose subnets and routers independently.

## Typical private network

```hcl
resource "nobus_network" "app" {
  name        = "app"
  subnet_cidr = "10.20.0.0/24"
  router_name = "app-router"
  description = "Application private network"
  tags        = ["environment=production"]
}

resource "nobus_security_group" "web" {
  name = "web"
}

resource "nobus_security_group_rule" "https" {
  security_group_id = nobus_security_group.web.id
  direction         = "ingress"
  protocol          = "tcp"
  port_range_min    = 443
  port_range_max    = 443
  remote_ip_prefix  = "0.0.0.0/0"
}
```

`nobus_network` requires `subnet_cidr` and `router_name` only when creating. Nobus does not return them on reads, so imports cannot verify them. Its `subnet_id` and `router_id` outputs are the canonical way to attach later resources.

## Building blocks

| Need | Resource | Notes |
| --- | --- | --- |
| Network plus initial subnet/router | [`nobus_network`](../docs/resources/network.md) | use its `subnet_id` and `router_id` outputs |
| Bare network | [`nobus_standalone_network`](../docs/resources/standalone_network.md) | pair with subnet/router resources |
| Subnet | [`nobus_subnet`](../docs/resources/subnet.md) | CIDR, IP version, DHCP, DNS, routes, allocation pools |
| Router and connection | [`nobus_router`](../docs/resources/router.md), [`nobus_router_interface`](../docs/resources/router_interface.md) | attach the interface to a subnet or port as supported by its schema |
| NIC | [`nobus_port`](../docs/resources/port.md) | network, fixed IPs, security groups, binding/profile fields |
| Extra allowed source/destination addresses | [`nobus_port_address_pairs`](../docs/resources/port_address_pairs.md) | manages address pairs for one port |
| Public address | [`nobus_floating_ip`](../docs/resources/floating_ip.md) | allocation and optional port association |
| Firewall policy | [`nobus_security_group`](../docs/resources/security_group.md), [`nobus_security_group_rule`](../docs/resources/security_group_rule.md) | model individual ingress/egress rules |
| VLAN trunk | [`nobus_trunk`](../docs/resources/trunk.md) | parent port with VLAN subports |
| Traffic policy | [`nobus_qos_policy`](../docs/resources/qos_policy.md), [`nobus_firewall_group`](../docs/resources/firewall_group.md) | use when the Nobus network policy model requires them |

## Design guidance

- Use CIDRs that do not overlap the networks you must route to. Terraform validates configuration syntax where possible, but routing feasibility is a cloud design decision.
- Prefer named security groups and separate rule resources. This makes ingress/egress intent reviewable and gives each rule a stable lifecycle.
- Attach a floating IP to a port only after deciding which instance interface is public. Replacing that port can change the association plan.
- Model router interfaces explicitly for standalone networks. A router is not useful to a subnet until the required interface exists.
- Treat `nobus_port_address_pairs` as exclusive management of the port’s allowed pairs. Avoid a second tool changing the same list.

## Discovery and diagnostics

Collection sources—[`nobus_networks`](../docs/data-sources/networks.md), [`nobus_subnets`](../docs/data-sources/subnets.md), [`nobus_ports`](../docs/data-sources/ports.md), [`nobus_routers`](../docs/data-sources/routers.md), [`nobus_floating_ips`](../docs/data-sources/floating_ips.md), security-group/rule, trunk, QoS, and firewall-group lists—are for discovery. Instance-focused sources expose network interfaces, security groups, tags, topology, and diagnostics; see [data sources](09-data-sources.md).

Read the generated reference for every field and its replacement behavior. Network, subnet, and port changes can interrupt connectivity even when Terraform calls them an in-place update.
