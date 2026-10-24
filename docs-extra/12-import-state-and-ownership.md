# Import, state, and ownership

Import records a real Nobus object in Terraform state. It does not change that object, create missing configuration, or discover every creation-time setting that Nobus does not return. Treat import as a three-step adoption process: inspect, import, then reconcile configuration and plan.

## Safe adoption process

1. Identify the exact remote object and its project and availability zone. Verify nobody else’s state owns it.
2. Write a minimal matching resource block at its intended Terraform address.
3. Run `terraform import ADDRESS REMOTE_ID` with the correct provider scope.
4. Run `terraform plan` immediately. Add fields that express your desired steady state; do not apply a replacement until it has been reviewed.
5. Commit the configuration and make the state backend the authoritative owner.

Example:

```bash
terraform import nobus_security_group.web SECURITY_GROUP_ID
terraform plan
```

Use the normal remote ID for most importable long-lived resources: networks, standalone networks, subnets, routers, ports, floating IPs, security groups/rules, trunks, QoS policies, firewall groups, volumes, snapshots, transfers, FOS containers/objects, instances, keypairs, Kubernetes clusters, databases, and the ordinary LBaaS hierarchy.

## Composite import identifiers

| Terraform address type | Required identifier | Example |
| --- | --- | --- |
| Volume attachment | `INSTANCE_ID/VOLUME_ID` | `server-1/volume-1` |
| Router interface | `ROUTER_ID/SUBNET_ID/PORT_ID` | `router-1/subnet-1/` or `router-1//port-1` |
| Port address pairs | port ID | `port-1` |
| FOS object | `CONTAINER_NAME/OBJECT_NAME` | `assets/releases/app.tar.gz` |

For a router interface, retain both slashes and leave the unused target component blank. FOS object parsing splits only on the first slash, so an object name may itself contain path separators.

## Why imported plans differ

Some API fields are needed for create but omitted from a later read. The provider models many of these as optional/computed to permit import, then requires them on a new creation. An imported object cannot reconstruct the original value for fields such as an instance’s image, flavor, key name, initial boot settings, or a composite network’s `subnet_cidr` and `router_name`.

Write-only values are more sensitive still: an instance `admin_pass` or `user_data`, database password, and JSON registration/action payload cannot be learned from the API. Add them only if Terraform should declare and manage future behavior. Never copy a guessed secret solely to clear a plan.

## State is not the cloud

State says what Terraform believes it owns. It can disagree with Nobus because an out-of-band user changed or deleted an object, a prior apply failed after sending a request, or another workspace manages the same object. Refresh/plan compares expected and actual state where the endpoint provides a read operation.

Do not use `terraform state rm` as a fix for a failed plan without checking the remote object. It merely stops Terraform tracking the object. Use it when ownership truly moves elsewhere; otherwise import, reconcile, or repair the resource configuration.

## Moving addresses without recreation

If only the Terraform address changes, use a `moved` block:

```hcl
moved {
  from = nobus_volume.data
  to   = module.storage.nobus_volume.data
}
```

This preserves the remote ID while changing the state address. For one-time advanced refactors, `terraform state mv` can do the same, but a committed `moved` block is reviewable and works for collaborators upgrading the configuration.

## Resources that are not normal adoption targets

One-shot action or registration resources have no durable remote object with a standard read lifecycle. This includes instance/database actions, Kubernetes migration registration, LB failover, and similar API-action resources. `nobus_volume_type` is also exceptional because the API lacks its read/delete lifecycle. Manage these with explicit change control rather than trying to use them as imported inventory.

## Removing ownership safely

Removing a resource from configuration ordinarily makes Terraform destroy the remote object. If you need to stop managing an object but preserve it, first apply a documented state-only ownership transfer approach (`terraform state rm` after verifying the target and successor owner); then remove configuration. Coordinate this with all state users. Never remove state first and assume a later `destroy` will still remove the remote object.
