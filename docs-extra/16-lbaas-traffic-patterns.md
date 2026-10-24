# LBaaS traffic patterns

LBaaS is best modeled as a dependency tree with one Terraform resource for each independently managed API object. This page describes how to choose that tree and how to operate it safely.

## HTTP service pattern

Create the front door, listener, backend pool, members, and health check independently. References establish the correct lifecycle order.

```hcl
resource "nobus_lbaas_load_balancer" "web" {
  name      = "web"
  subnet_id = nobus_network.application.subnet_id
}

resource "nobus_lbaas_listener" "http" {
  load_balancer_id = nobus_lbaas_load_balancer.web.id
  protocol         = "HTTP"
  protocol_port    = 80
}

resource "nobus_lbaas_pool" "http" {
  listener_id = nobus_lbaas_listener.http.id
  protocol    = "HTTP"
  lb_method   = "ROUND_ROBIN"
}

resource "nobus_lbaas_member" "api_a" {
  pool_id       = nobus_lbaas_pool.http.id
  address       = "10.42.0.10"
  protocol_port = 8080
}
```

Member `address`, `protocol_port`, pool, and optional subnet relationship describe identity/placement and may be replacement-only. For an autoscaled application, create one member per stable backend address only if the address lifecycle is itself Terraform-managed. Otherwise use the Nobus/API integration appropriate for dynamic membership rather than leaving Terraform to fight an autoscaler.

## Health checks are rollout gates

A health monitor belongs to a pool. Configure its protocol, interval (`delay`), timeout, retry thresholds, URL path, method, and expected codes to match an endpoint that actually represents readiness. A TCP probe can prove that a port opens; it cannot prove that an HTTP application can serve correct traffic. Set the health endpoint before adding live members and monitor deployment status after applies.

Too-short timeouts or retry counts can create flapping; overly lenient settings delay removal of unhealthy backends. Choose values using measured response latency and failure tolerance, then validate with an intentional backend failure in a safe environment.

## TLS and L7 routing

Listeners own frontend protocol/port and related TLS settings exposed by the API. Keep certificate/private-key material out of source control and state wherever the schema/API supports an external reference rather than inline secret material.

L7 policies belong to listeners and L7 rules belong to policies. Model a routing behavior in this order: listener → policy (action/target/position) → one or more conditions. Rule order can change request routing, so make positions explicit where the schema supports them and review a plan like an application-routing change.

## Security rules

`nobus_lbaas_security_group_rule` creates an LB-specific rule from a source CIDR to a port. It is not the same as a compute port’s security-group rule. Limit the CIDR and port to the intended listener; a public HTTPS listener can be `0.0.0.0/0` only when that exposure is intentional and monitored.

## Composite provisioning versus individual resources

`nobus_lbaas_provision` accepts an API-shaped composite JSON payload. It is useful for an API-native provision request but should be an ownership boundary: do not also define individual listener/pool/member resources for children the composite resource owns. Choose one model per load-balancer topology.

Composite JSON may contain identifiers or secret-adjacent values. Generate it with `jsonencode`, mark inputs sensitive where appropriate, and review it as carefully as any infrastructure code. The individual-resource model is usually easier to review, import, and evolve in Terraform.

## Failover and verification

`nobus_lbaas_failover` is a one-shot side effect. Give its `trigger` a deliberate change-control value so a repeat operation is unmistakable in a plan. A state destroy cannot undo a completed failover.

After changing an LB, check task, deployment-status, status, and stats sources. Also make a real request from the expected network path. Control-plane success, a healthy member, and end-to-end application availability are three different signals.
