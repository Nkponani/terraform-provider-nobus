# Networking patterns

This page expands the [networking overview](03-networking.md) into practical resource composition. Read every linked generated page for the accepted API values and field-level replacement behavior.

## Pattern: composite application network

`nobus_network` is the shortest path to a private application network because Nobus creates its first subnet and router in the same API call. The resource outputs the resulting IDs for dependent resources.

```hcl
resource "nobus_network" "application" {
  name                = "application"
  subnet_cidr         = "10.42.0.0/24"
  router_name         = "application-router"
  external_network_id = var.external_network_id
  dns_nameservers     = ["1.1.1.1", "8.8.8.8"]
  enable_snat         = true
  tags                = ["team=platform", "environment=production"]
}
```

The create API only accepts the initial topology fields. Fields such as description, tags, MTU, administrative state, sharing, router-external state, and port security are subsequently applied through the supported update operation when configured. Changes to initial subnet/router/external-network/DNS/SNAT choices require a new composite network; do not use that replacement casually in a live environment.

## Pattern: independently composed network

Choose `nobus_standalone_network` if one network needs several subnets, a separately managed router, or a topology that cannot be expressed by the composite create. Then model every relationship explicitly:

```hcl
resource "nobus_standalone_network" "service" {
  name = "service"
}

resource "nobus_subnet" "service" {
  network_id      = nobus_standalone_network.service.id
  cidr            = "10.43.0.0/24"
  ip_version      = 4
  enable_dhcp     = true
  dns_nameservers = ["10.43.0.2"]
}

resource "nobus_router" "service" {
  name                = "service-router"
  external_network_id = var.external_network_id
}

resource "nobus_router_interface" "service" {
  router_id = nobus_router.service.id
  subnet_id = nobus_subnet.service.id
}
```

`nobus_router_interface` requires one target, `subnet_id` or `port_id`. It has no rich read operation, so its Terraform ID encodes the selected router/subnet/port relationship. Do not configure both targets unless the upstream API explicitly supports that relationship for your use case.

## Ports, fixed addresses, and public exposure

Create an explicit `nobus_port` when an instance or appliance needs a predictable network interface, fixed-IP choice, security-group attachment, or special binding/address-pair configuration. An instance may instead receive its port implicitly through its creation API; choose one ownership model for each interface.

`nobus_port_address_pairs` manages the allowed address-pair set of one port. It is appropriate for virtual IPs, NAT appliances, or source/destination-check-like patterns where Nobus requires the extra addresses. Because it clears/sets that API collection, do not split pairs for the same port across competing Terraform resources or external tools.

`nobus_floating_ip` allocates a public address and can associate it with a port. The public address does not itself configure guest routing, firewall rules, or application TLS; those are separate concerns. Plan port replacement and floating-IP reassociation together.

## Security design

Use one `nobus_security_group` per policy boundary and one `nobus_security_group_rule` per permitted flow. This produces small, reviewable changes and avoids a broad group becoming an untraceable collection of exceptions.

```hcl
resource "nobus_security_group" "app" {
  name        = "app"
  description = "Ingress only from the load-balancer tier"
}

resource "nobus_security_group_rule" "app_http" {
  security_group_id = nobus_security_group.app.id
  direction         = "ingress"
  ether_type        = "IPv4"
  protocol          = "tcp"
  port_range_min    = 8080
  port_range_max    = 8080
  remote_ip_prefix  = var.load_balancer_subnet_cidr
}
```

Restrict CIDRs wherever possible. An allow rule with `0.0.0.0/0` is a deliberate public-service decision, not a default. Network security groups and LBaaS security-group rules are distinct resource models; configure the one attached to the component that must enforce the policy.

## Advanced networking resources

`nobus_trunk` models a parent port and VLAN subports. Use it for workloads that understand VLAN tagging and ensure the underlying network design permits it. `nobus_qos_policy` and `nobus_firewall_group` expose Nobus network-policy APIs; apply them where the service supports attachment and document their external operational owner.

For all three, avoid assuming generic OpenStack semantics beyond what the generated Nobus schema and API expose. The provider intentionally follows the Nobus API rather than borrowing fields or behavior from another provider.

## Teardown order

Terraform references normally calculate the right order. When resolving a failed destroy manually, remove or detach public IP associations and dependent instances/ports, then router interfaces, then subnets/routers/networks as the API requires. Verify there are no externally created dependents before retrying.
