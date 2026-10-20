# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive architecture documentation (`ARCHITECTURE.md`)
- Contribution guidelines (`CONTRIBUTING.md`)
- This changelog restructured to Keep a Changelog format

### Changed
- Updated documentation structure for better maintainability
- CHANGELOG.md reformatted to standard Keep a Changelog format

## [v0.1.0] - 2026-10-20

### Added
- **Networking Resources:**
  - `nobus_network` - Composite network with subnet and router
  - `nobus_standalone_network` - Network without auto-created subnet/router
  - `nobus_subnet` - Standalone subnet
  - `nobus_router` - Standalone router
  - `nobus_router_interface` - Router-subnet/port connections
  - `nobus_port` - Network port
  - `nobus_floating_ip` - Floating IP allocation and association
  - `nobus_security_group` - Security group
  - `nobus_security_group_rule` - Security group rules
  - `nobus_trunk` - Trunk ports with VLAN subports
  - `nobus_qos_policy` - Quality of Service policies
  - `nobus_firewall_group` - Firewall groups
  - `nobus_port_address_pairs` - Allowed address pairs

- **Block Storage Resources:**
  - `nobus_volume` - Volumes with in-place growth
  - `nobus_volume_attachment` - Volume-instance attachments
  - `nobus_volume_snapshot` - Point-in-time snapshots
  - `nobus_volume_type` - Volume types (create-only)
  - `nobus_volume_transfer` - Volume ownership transfers
  - `nobus_volume_transfer_accept` - Transfer acceptance

- **File/Object Storage (FOS) Resources:**
  - `nobus_fos_container` - Swift-compatible containers
  - `nobus_fos_object` - Object upload with content replacement

- **Managed Kubernetes Resources:**
  - `nobus_kubernetes_cluster` - Managed K8s with H/V/etcd scaling
  - `nobus_kubernetes_migration` - Existing cluster registration

- **Managed Database Resources:**
  - `nobus_database` - Managed databases with resize/password rotation
  - `nobus_database_action` - Database lifecycle actions
  - `nobus_existing_databases` - Customer-managed database registration

- **LBaaS Resources:**
  - `nobus_lbaas_load_balancer` - Load balancers with SG associations
  - `nobus_lbaas_listener` - Listeners
  - `nobus_lbaas_pool` - Backend pools with session persistence
  - `nobus_lbaas_member` - Pool members
  - `nobus_lbaas_health_monitor` - Health monitors
  - `nobus_lbaas_l7_policy` - Layer 7 policies
  - `nobus_lbaas_l7_rule` - L7 policy rules
  - `nobus_lbaas_security_group_rule` - LB security group rules
  - `nobus_lbaas_failover` - Failover trigger
  - `nobus_lbaas_provision` - Composite JSON provisioning

- **Compute Resources:**
  - `nobus_instance` - Virtual machine instances
  - `nobus_instance_from_volume` - Boot from volume instances
  - `nobus_keypair` - SSH keypairs
  - `nobus_instance_action` - One-shot compute operations

- **Data Sources (Collection):**
  - Networking: networks, subnets, ports, routers, floating IPs, security groups/rules, trunks, QoS policies, firewall groups
  - Storage: volumes, snapshots, backups, transfers, availability zones
  - FOS: account, auth details, containers, objects, object content
  - Kubernetes: clusters, bastion, load balancer login, monitoring, nodes, task status
  - Database: databases, credentials, creation status, status, Zabbix credentials, logs
  - LBaaS: load balancers, listeners, pools, members, health monitors, L7 policies/rules, tasks, deployment status, load balancer status/stats, listener stats

- **Data Sources (Singular):**
  - `nobus_volume_backup` - Full backup details
  - Instance: diagnostics, network interfaces, security groups, tags, topology

- **Provider Features:**
  - Bearer token authentication via API key
  - Environment variable fallback for all config
  - Sensitive API key handling
  - Asynchronous operation polling
  - Full import support for all resources
  - Comprehensive error diagnostics preserving API messages

### Security
- API key marked sensitive (never in state, logs, or diagnostics)
- Credential-bearing data source results marked sensitive
- HTTPS enforced for all API communication

### Documentation
- Complete README with authentication, configuration, examples
- Auto-generated resource/data source reference docs
- API coverage mapping (OpenAPI operation → Terraform surface)
- Project layout and development guide

---

## Release Notes Format

Each release includes:

### Added
- New resources, data sources, features

### Changed
- Modifications to existing behavior (backward compatible)

### Deprecated
- Features marked for removal

### Removed
- Removed features (breaking changes)

### Fixed
- Bug fixes

### Security
- Security-related changes

---

## Migration Guides

### v0.x → v1.0 (Future)

When v1.0 is released, breaking changes will be documented here with:
- Specific resource attribute changes
- Required configuration updates
- Automated migration tooling (if applicable)

---

## Links

- [GitHub Releases](https://github.com/nobus-cloud/terraform-provider-nobus/releases)
- [Terraform Registry](https://registry.terraform.io/providers/nobus-cloud/nobus)
- [Nobus Cloud API Docs](https://cloud-api.nobus.io/api/v3/openapi.json)