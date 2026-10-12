# Provider source layout

The provider is grouped by service:

```text
internal/provider/
├── networking/   # Subnets, routers, ports, floating IPs, security and QoS
├── storage/      # Snapshots, volume types, transfers and backup lookup
├── fos/          # Object-storage containers, objects, listings and downloads
├── kubernetes/   # Managed clusters, scaling, migration and access lookups
├── database/     # Managed databases, actions, credentials, status and logs
├── lbaas/        # Load balancers and their listener/pool/member/L7 hierarchy
├── datasources/  # Cross-service collection data sources
├── shared/       # Configuration, scope, schema and conversion helpers
└── *.go          # Provider wiring and the original core resources/tests
```

Inside a service folder, implementations use one Terraform type per file:

- `<name>_resource.go` contains the complete managed-resource lifecycle.
- `<name>_data_source.go` contains a singular data source.
- `<name>_resource_helpers.go` contains models or conversions shared only by
  that resource.

The root package is the provider assembly boundary. It owns provider
configuration and registration, plus the original composite network, volume,
attachment, instance, keypair and instance-action implementations whose
lifecycle tests are maintained together.

The matching API client is named after the plural API collection. For example,
`networking/port_resource.go` uses `internal/client/ports.go`, and
`storage/volume_snapshot_resource.go` uses
`internal/client/volume_snapshots.go`.

Do not add multiple managed resources to a generic `*_resources.go` file. Add
new implementations to the appropriate service folder and register their
constructor in `provider.go`.
