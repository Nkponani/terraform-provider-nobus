# Operations, troubleshooting, and development

## Production operating practices

- Pin and upgrade provider versions intentionally. Run `terraform init -upgrade`, inspect the lock-file change, then plan against a non-production project first.
- Store state remotely with encryption, access control, versioning, and locking. Sensitive schema flags redact CLI output but do not make state harmless.
- Keep API keys in CI secret storage or environment variables. Rotate compromised keys and avoid `TF_LOG`/shell history leaks.
- Run `terraform fmt -recursive`, `terraform validate`, and a reviewed `terraform plan` in CI. Apply only the saved/reviewed plan where your workflow supports it.
- Use separate state per environment/project. Provider defaults are convenient but can direct a whole configuration to the wrong project if changed carelessly.
- Keep action resources (failover, lifecycle actions, registrations) in clearly named modules or separate applies. Their side effects are not reversed by deleting state.

## Reading provider diagnostics

The provider reports actionable Terraform diagnostics for missing API keys, missing project scope, and API failures. HTTP failures contain the Nobus status code and message when available. Common causes:

| Symptom | Likely cause | First checks |
| --- | --- | --- |
| Missing API key | neither `api_key` nor `NOBUS_API_KEY` resolves to a value | inspect secret injection; do not print the key |
| Scope/project error | project omitted at provider and resource levels, or key lacks access | set `project_id`; verify account permissions |
| Object not found during refresh | wrong project/AZ, object deleted out of band, or stale state | verify scope and ID; use `terraform import` or state repair only after checking ownership |
| Request rejected | unsupported field combination, quota, invalid API value, or permissions | use the returned API message and generated schema |
| Long-running apply | asynchronous Nobus operation remains incomplete | check the service task/status data source and service console; do not start a competing apply |
| Perpetual diff after import | API does not return a create-time/write-only field | configure the desired field or intentionally omit it; review the plan before accepting a replacement |

The underlying HTTP client respects Terraform context cancellation and adds no independent operation deadline. Cancel only after considering whether the remote operation may continue; refresh state before retrying a create or delete.

## Drift and recovery

Run a normal plan regularly to reveal drift. If someone changes an API object out of band, choose one owner:

1. Restore the Terraform-declared setting with an apply.
2. Update configuration to the approved live value and apply it.
3. If ownership moved elsewhere, remove the resource from state only after confirming Terraform must no longer manage it.

Do not blindly re-run a failed create: an API request may have succeeded before a network failure reached Terraform. Query with the appropriate data source or console, then import or remove the partially created object according to the approved recovery path.

## Imports and state moves

Use `terraform import` to adopt a real object, then immediately run `terraform plan`. Imports require current provider credentials and usually project/AZ defaults. Reorganizing configuration without changing an object should use Terraform `moved` blocks or `terraform state mv`, not destroy/recreate. State commands are powerful; take or verify a backend version before modifying state.

## Building and testing the provider

The provider is Go code built with the Terraform Plugin Framework. The project’s [development guide](../DEVELOPMENT.md) covers tools, local development, debugging, test strategy, documentation generation, and release-related workflows. [Architecture](../ARCHITECTURE.md) explains the client/provider split and service layout. Useful local checks are:

```bash
go test ./...
go vet ./...
terraform fmt -recursive
terraform validate
```

Run acceptance tests only against an explicitly designated, disposable Nobus project and with the environment variables they require. They create and delete real cloud resources.

## Documentation maintenance

Generated pages under `docs/` are maintained by `tfplugindocs`; do not hand-edit their schema sections. This `docs-extra/` tree is hand-written context. When changing provider behavior:

1. Update the schema description and examples near the code.
2. Regenerate the generated docs using the project’s documented workflow.
3. Update the relevant guide here—especially lifecycle, irreversible action, import, secret, and networking implications.
4. Add or update an example and automated test when practical.

The API source of truth is [`docs/api/openapi.json`](../docs/api/openapi.json). If the public API changes, update the client and provider deliberately rather than guessing fields from another cloud provider.
