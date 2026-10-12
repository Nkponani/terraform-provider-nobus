# API client source layout

Each API collection has one file containing its request/response models and all
of its operations. For example, `ports.go` contains create, read, update,
delete, list, and address-pair operations for ports.

Shared HTTP envelope and query behavior lives in `networking_helpers.go` or
`storage_helpers.go`. Endpoint constants are centralized in
`networking_paths.go` and `storage_paths.go` so path changes remain easy to
audit.

Do not split one API collection across `*_extended.go` or generic
`*_resources.go` files.
