# Nobus provider guide

This directory is the human-oriented documentation for the Nobus Terraform provider. The generated [provider reference](../docs/index.md), [resource reference](../docs/resources), and [data-source reference](../docs/data-sources) remain the source for the complete schema: every argument, type, default, sensitivity marker, and computed attribute. These guides explain how the pieces fit together, what Terraform will do over time, and the operational consequences of using them.

## Start here

1. [Getting started](01-getting-started.md) — install, authenticate, select a project and run a first plan.
2. [Core concepts and lifecycle](02-concepts-and-lifecycle.md) — scope, state, replacements, imports, asynchronous operations, and secrets.
3. Pick a service guide:
   - [Networking](03-networking.md)
   - [Compute](04-compute.md)
   - [Block storage](05-block-storage.md)
   - [File/Object Storage (FOS)](06-fos.md)
   - [Managed Kubernetes and databases](07-platform-services.md)
   - [Load Balancing as a Service](08-load-balancing.md)
4. [Data sources](09-data-sources.md) — discovery, status, credentials, logs, and metrics.
5. [Operations and troubleshooting](10-operations-and-troubleshooting.md) — safe changes, diagnostics, upgrades, testing, and contributing.

## Deep reference

The service guides above are the fastest way to get oriented. These pages go
deeper when you are designing a module, taking over existing infrastructure, or
automating an operational workflow.

6. [Provider configuration and credentials](11-provider-configuration.md)
7. [Import, state, and ownership](12-import-state-and-ownership.md)
8. [Networking patterns](13-networking-patterns.md)
9. [Compute and storage workflows](14-compute-storage-workflows.md)
10. [Platform-service workflows](15-platform-service-workflows.md)
11. [LBaaS traffic patterns](16-lbaas-traffic-patterns.md)
12. [Managed-resource catalog](17-managed-resource-catalog.md)
13. [Data-source catalog](18-data-source-catalog.md)

## Documentation contract

The provider talks only to the public Nobus API at `https://cloud-api.nobus.io/api/v3`; the vendored [OpenAPI specification](../docs/api/openapi.json) is the API contract used by the implementation. Documentation here describes the currently implemented provider behavior. When a guide conflicts with generated schema text or the code, use the generated schema for Terraform field semantics and open an issue or update this guide.

## Resource map

| Area | Managed resources | Guide |
| --- | --- | --- |
| Networking | network, standalone network, subnet, router, router interface, port, port address pairs, floating IP, security group/rule, trunk, QoS policy, firewall group | [Networking](03-networking.md) |
| Compute | instance, instance from volume, keypair, instance action | [Compute](04-compute.md) |
| Block storage | volume, attachment, snapshot, type, transfer, transfer accept | [Block storage](05-block-storage.md) |
| Object storage | FOS container, FOS object | [FOS](06-fos.md) |
| Platform | Kubernetes cluster/migration; database/action/existing databases | [Platform services](07-platform-services.md) |
| LBaaS | load balancer, listener, pool, member, health monitor, L7 policy/rule, security-group rule, failover, provision | [Load balancing](08-load-balancing.md) |

`nobus_volume_type`, `nobus_kubernetes_migration`, `nobus_database_action`, `nobus_existing_databases`, and `nobus_lbaas_failover` have intentionally unusual lifecycle behavior. Read their service sections before treating them like ordinary CRUD resources.
