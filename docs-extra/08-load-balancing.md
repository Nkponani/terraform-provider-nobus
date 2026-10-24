# Load Balancing as a Service (LBaaS)

LBaaS models a hierarchy. A load balancer has listeners; listeners route to pools; pools have members and optionally a health monitor; listeners may also have L7 policies, which have L7 rules. Terraform references make this order explicit and give safe creation/deletion ordering.

```hcl
resource "nobus_lbaas_load_balancer" "public" {
  name      = "public-web"
  subnet_id = nobus_network.app.subnet_id
}

resource "nobus_lbaas_listener" "https" {
  load_balancer_id = nobus_lbaas_load_balancer.public.id
  protocol         = "TERMINATED_HTTPS"
  protocol_port    = 443
}

resource "nobus_lbaas_pool" "web" {
  listener_id = nobus_lbaas_listener.https.id
  protocol    = "HTTP"
  lb_method   = "ROUND_ROBIN"
}

resource "nobus_lbaas_member" "web_1" {
  pool_id       = nobus_lbaas_pool.web.id
  address       = "10.20.0.10"
  protocol_port = 8080
}

resource "nobus_lbaas_health_monitor" "web" {
  pool_id            = nobus_lbaas_pool.web.id
  type               = "HTTP"
  delay              = 5
  timeout            = 3
  max_retries        = 3
  max_retries_down   = 3
  url_path           = "/healthz"
  expected_codes     = "200"
  http_method        = "GET"
}
```

Protocol choices and API-supported names must match the generated schemas; examples are not a substitute for the available enum/validation rules. TLS certificates, SNI, session persistence, tags, and admin state are configured on the resource that owns them where the API exposes them.

## Resource roles

| Resource | Role |
| --- | --- |
| [`nobus_lbaas_load_balancer`](../docs/resources/lbaas_load_balancer.md) | frontend VIP/load-balancer object and its security-group associations |
| [`nobus_lbaas_listener`](../docs/resources/lbaas_listener.md) | protocol/port frontend and default routing point |
| [`nobus_lbaas_pool`](../docs/resources/lbaas_pool.md) | backend policy, algorithm, and session persistence |
| [`nobus_lbaas_member`](../docs/resources/lbaas_member.md) | one backend address and port |
| [`nobus_lbaas_health_monitor`](../docs/resources/lbaas_health_monitor.md) | probe settings for a pool |
| [`nobus_lbaas_l7_policy`](../docs/resources/lbaas_l7_policy.md) | ordered listener routing policy |
| [`nobus_lbaas_l7_rule`](../docs/resources/lbaas_l7_rule.md) | match condition belonging to an L7 policy |
| [`nobus_lbaas_security_group_rule`](../docs/resources/lbaas_security_group_rule.md) | LB-specific source CIDR/port rule |

## Composite and action resources

`nobus_lbaas_provision` submits a composite JSON request that can create a load balancer and child objects together. Use it when you already have a validated API-shaped payload and want that operation to be Terraform-managed as one unit. It is not interchangeable with the individual resources: avoid managing the same LB children through both models, which would create competing ownership.

`nobus_lbaas_failover` triggers an explicit one-shot failover. Destroying its Terraform state does not undo a completed failover. Put it in a separately reviewed, intentionally applied change rather than a steady-state service module.

## Status and rollout checks

LBaaS data sources list each collection and expose task, deployment status, LB status, LB stats, and listener stats. Use them after provisioning and during incident diagnosis; see [data sources](09-data-sources.md). A Terraform apply does not prove that application backends are serving correct responses—health-monitor and application telemetry should be part of the rollout gate.
