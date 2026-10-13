# Complete Nobus Example

This example provisions a realistic set of Nobus resources: network, subnet, security group with rules, instance, floating IP, volume, and volume attachment.

## Prerequisites

Set the required environment variables:
- `NOBUS_API_KEY`
- `NOBUS_API_SECRET`
- `NOBUS_PROJECT_ID`
- `NOBUS_AVAILABILITY_ZONE`

Optional:
- `NOBUS_API_URL`

## Usage

```bash
terraform init
terraform plan -var="key_name=<your-ssh-key>"
terraform apply -var="key_name=<your-ssh-key>"
```

To clean up:
```bash
terraform destroy -var="key_name=<your-ssh-key>"
```
