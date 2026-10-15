# Nobus provider examples

| Directory | Shows |
| --- | --- |
| `provider/` | Provider configuration and the available attributes |
| `resources/nobus_network/` | Creating a network and reading its subnet and router IDs |
| `resources/nobus_volume/` | Creating volumes and growing one in place |
| `resources/nobus_instance/` | Creating an instance and decoding its addresses |
| `resources/nobus_full_stack/` | A network, a volume and an instance wired together by references |
| `tests/instance/` | A minimal, standalone instance configuration for testing a Nobus connection |

Every attribute used in these examples exists in the Nobus OpenAPI
specification vendored at `docs/api/openapi.json`.

To try them locally, build the provider and add a development override as
described in the [README](../README.md#using-the-local-build-with-terraform),
then export your credentials:

```bash
export NOBUS_API_KEY="..."
export NOBUS_PROJECT_ID="..."
export NOBUS_AVAILABILTY_ZONE="..."
```
