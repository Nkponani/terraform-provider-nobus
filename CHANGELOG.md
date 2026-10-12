## 0.1.0 (Unreleased)

FEATURES:

* Add complete Nobus v3 networking coverage, including standalone networks,
  mutable subnets/ports/routers/floating IPs/security groups, trunks, QoS
  policies, firewall groups, router gateways, and port address pairs.
* Add volume snapshots, volume types, ownership transfers and transfer
  acceptance, plus read-only backup and availability-zone data sources.
* Add collection data sources for every networking and block-storage list
  endpoint exposed by the live OpenAPI specification.
* Add encrypted-volume creation and expose the read-only multiattach status.
* Organize provider and API-client implementations by resource, with one
  lifecycle/client collection per clearly named file, and remove obsolete
  unregistered storage implementations.
* Group networking, storage, data-source, and shared provider implementations
  into dedicated Go subpackages for easier navigation and ownership.
* Add complete FOS object-storage support: managed containers and uploaded
  objects, account usage and authentication metadata, container/object
  listings, and sensitive base64 object downloads.
* Add managed Kubernetes clusters, scaling, migration, task, node, bastion,
  load-balancer-login, and monitoring support from the live v3 API.
* Add managed databases with resize, credential rotation, lifecycle actions,
  existing-database registration, status, credential, monitoring, and log
  lookups.
* Add complete LBaaS coverage for load balancers, listeners, pools, members,
  health monitors, L7 policies/rules, security rules, failover, composite
  provisioning, lists, task/status, and statistics operations.
