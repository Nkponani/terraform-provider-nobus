# Development Guide

Comprehensive guide for developing, testing, and debugging the Nobus Terraform provider.

## Table of Contents

- [Environment Setup](#environment-setup)
- [Project Structure Deep Dive](#project-structure-deep-dive)
- [Common Development Tasks](#common-development-tasks)
- [Debugging](#debugging)
- [Testing Strategies](#testing-strategies)
- [Code Generation](#code-generation)
- [Performance Considerations](#performance-considerations)
- [Troubleshooting](#troubleshooting)

## Environment Setup

### Required Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25+ | Build and test |
| Terraform | 1.0+ | Integration testing |
| golangci-lint | latest | Linting |
| tfplugindocs | latest | Documentation generation |
| goreleaser | latest | Release builds |
| delve | latest | Debugging |

### VS Code / IDE Configuration

Recommended extensions:
- Go (Google)
- Terraform (HashiCorp)
- YAML (Red Hat)
- GitLens

Settings (`.vscode/settings.json`):
```json
{
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v"],
  "terraform.linting.enabled": true
}
```

### Pre-commit Hooks (Optional)

```bash
# Install pre-commit
brew install pre-commit  # macOS
# or
pip install pre-commit   # Python

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

`.pre-commit-config.yaml`:
```yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.60.0
    hooks:
      - id: golangci-lint
  - repo: https://github.com/terraform-linters/tflint
    rev: v0.50.0
    hooks:
      - id: tflint
```

## Project Structure Deep Dive

### internal/client/ - API Client (Zero Terraform Deps)

```
internal/client/
├── client.go                 # Core HTTP client, auth, JSON, polling
├── errors.go                 # Error types, not-found detection
├── networking_paths.go       # Networking endpoint constants
├── storage_paths.go          # Storage endpoint constants
├── networking_helpers.go     # Shared networking request/response logic
├── storage_helpers.go        # Shared storage request/response logic
├── networks.go               # Network API operations
├── subnets.go                # Subnet API operations
├── routers.go                # Router API operations
├── ports.go                  # Port API operations
├── security_groups.go        # Security group API operations
├── security_group_rules.go   # Security group rule API operations
├── trunks.go                 # Trunk API operations
├── qos_policies.go           # QoS policy API operations
├── firewall_groups.go        # Firewall group API operations
├── floating_ips.go           # Floating IP API operations
├── volumes.go                # Volume API operations
├── volume_snapshots.go       # Snapshot API operations
├── volume_backups.go         # Backup API operations
├── volume_transfers.go       # Volume transfer API operations
├── volume_types.go           # Volume type API operations
├── volume_availability_zones.go # AZ API operations
├── instances.go              # Instance API operations
├── keypairs.go               # Keypair API operations
├── standalone_networks.go    # Standalone network API operations
├── fos_account.go            # FOS account API operations
├── fos_containers.go         # FOS container API operations
├── fos_objects.go            # FOS object API operations
├── fos_helpers.go            # FOS shared logic
├── fos_paths.go              # FOS endpoint constants
├── databases.go              # Database API operations
├── kubernetes.go             # Kubernetes API operations
├── lbaas.go                  # LBaaS API operations
└── response.go               # Generic response envelope handling
```

**Key Patterns:**

1. **One file per API collection** - All operations for a resource type in one file
2. **Constants for paths** - `networking_paths.go`, `storage_paths.go`, `fos_paths.go`
3. **Typed request/response** - Structs matching OpenAPI spec exactly
4. **Error wrapping** - `APIError`, `TransportError`, `DecodeError` in `errors.go`
5. **Polling utility** - `WaitFor` in `client.go` for async operations

### internal/provider/ - Terraform Plugin Framework

```
internal/provider/
├── provider.go               # Provider registration, Resources(), DataSources()
├── configure.go              # Client construction, config resolution
├── shared/
│   ├── config.go             # ProviderConfig shared with resources
│   ├── schema.go             # Common schema patterns (ID, timestamps, etc.)
│   ├── converters.go         # Model ↔ API conversion functions
│   └── resource_helpers.go   # Shared resource utilities
├── networking/
│   ├── network_resource.go
│   ├── standalone_network_resource.go
│   ├── subnet_resource.go
│   ├── router_resource.go
│   ├── router_interface_resource.go
│   ├── port_resource.go
│   ├── port_resource_helpers.go
│   ├── port_address_pairs_resource.go
│   ├── floating_ip_resource.go
│   ├── security_group_resource.go
│   ├── security_group_rule_resource.go
│   ├── trunk_resource.go
│   ├── trunk_resource_helpers.go
│   ├── qos_policy_resource.go
│   └── firewall_group_resource.go
├── storage/
│   ├── volume_snapshot_resource.go
│   ├── volume_type_resource.go
│   ├── volume_transfer_resource.go
│   ├── volume_transfer_accept_resource.go
│   ├── volume_backup_data_source.go
│   └── storage_resource_helpers.go
├── fos/
│   ├── container_resource.go
│   ├── object_resource.go
│   ├── objects_data_source.go
│   ├── containers_data_source.go
│   ├── auth_details_data_source.go
│   ├── account_data_source.go
│   └── object_content_data_source.go
├── kubernetes/
│   ├── cluster_resource.go
│   ├── migration_resource.go
│   ├── data_sources.go
│   └── task_status_data_source.go
├── database/
│   ├── database_resource.go
│   ├── action_resources.go
│   ├── existing_resource.go
│   └── data_sources.go
├── lbaas/
│   ├── load_balancer_resource.go
│   ├── listener_resource.go
│   ├── pool_resource.go
│   ├── member_resource.go
│   ├── health_monitor_resource.go
│   ├── l7_policy_resource.go
│   ├── l7_rule_resource.go
│   ├── security_group_rule_resource.go
│   ├── failover_resource.go
│   ├── provision_resource.go
│   └── *_data_source.go (multiple)
├── datasources/
│   ├── collection_data_sources.go
│   ├── instance_security_groups_data_source.go
│   ├── instance_diagnostics_data_source.go
│   ├── instance_tags_data_source.go
│   ├── instance_network_interfaces_data_source.go
│   └── instance_topology_data_source.go
├── network_resource.go       # Core composite network
├── volume_resource.go        # Core volume
├── volume_attachment_resource.go
├── instance_resource.go      # Core instance
├── instance_from_volume_resource.go
├── keypair_resource.go
├── instance_action_resource.go
├── provider_test.go
├── mapping_test.go
├── lifecycle_test.go
└── acceptance_test.go
```

**Key Patterns:**

1. **Resource-per-file** - Complete lifecycle in one file
2. **Helper files** - Shared models/conversions in `_resource_helpers.go`
3. **Service packages** - Grouped by domain (networking, storage, etc.)
4. **Factory data sources** - Collection data sources via `NewCollectionDataSource()`

### docs/ - Generated Documentation

```
docs/
├── index.md                    # Provider overview (generated)
├── resources/                  # Resource docs (generated)
│   ├── network.md
│   ├── volume.md
│   └── ...
├── data-sources/               # Data source docs (generated)
│   ├── networks.md
│   ├── volumes.md
│   └── ...
└── api/
    ├── openapi.json           # Vendored OpenAPI spec (source of truth)
    └── coverage.md            # API operation → Terraform mapping
```

**Never edit generated files directly** - Update schema descriptions and regenerate.

### examples/ - Usage Examples

```
examples/
├── README.md                   # Examples index
├── provider/
│   └── provider.tf             # Provider configuration template
├── complete/                   # Full stack example
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   └── provider.tf
├── resources/                  # Per-resource examples
│   ├── nobus_network/
│   ├── nobus_volume/
│   └── ...
└── data-sources/               # Per-data-source examples
    └── ...
```

## Common Development Tasks

### Adding a New API Collection

1. **Create client file** (`internal/client/<collection>.go`):
```go
// Follow existing patterns in volumes.go, networks.go
type Create<Collection>Request struct { ... }
type <Collection>Response struct { ... }

func (c *Client) Create<Collection>(ctx context.Context, req Create<Collection>Request) (*<Collection>Response, error) {
    return doRequest[<Collection>Response](ctx, c, Options{
        Method: "POST",
        Path:   paths.<Collection>Create,
        Body:   req,
    })
}
```

2. **Add path constants** (`*_paths.go`):
```go
const (
    CollectionCreate = "/api/v3/collection/"
    CollectionRead   = "/api/v3/collection/%s/"
    // ...
)
```

3. **Add shared helpers** if needed (`*_helpers.go`)

4. **Write unit tests** (`<collection>_test.go`)

### Adding a New Resource

1. **Create resource file** (`internal/provider/<service>/<name>_resource.go`):
```go
func New<Name>Resource() resource.Resource {
    return &<Name>Resource{}
}

type <Name>Resource struct {
    client *client.Client
}

func (r *<Name>Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_<name>"
}

func (r *<Name>Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "...",
        Attributes: map[string]schema.Attribute{
            // Define all attributes with MarkdownDescription
        },
    }
}

func (r *<Name>Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil { return }
    r.client = req.ProviderData.(*client.Client)
}

func (r *<Name>Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { ... }
func (r *<Name>Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) { ... }
func (r *<Name>Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { ... }
func (r *<Name>Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) { ... }
func (r *<Name>Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

2. **Create model and converters** (in `_resource_helpers.go` or `shared/converters.go`):
```go
type <Name>Model struct {
    ID            types.String `tfsdk:"id"`
    Name          types.String `tfsdk:"name"`
    // ...
}

func (m *<Name>Model) toAPIRequest() client.Create<Name>Request { ... }
func <Name>ModelFromAPI(resp *client.<Name>Response) <Name>Model { ... }
```

3. **Register in provider.go** `Resources()` function

4. **Write unit tests** (`<name>_resource_test.go`)

5. **Write acceptance test** (if applicable)

6. **Generate docs**: `tfplugindocs generate`

### Adding a New Data Source

**Collection data source** (plural, list):
```go
// In datasources/collection_data_sources.go - just register in provider.go
providerdatasources.NewCollectionDataSource("collection_name")
```

**Singular data source** (specific lookup):
```go
// New file in appropriate service package
func New<Name>DataSource() datasource.DataSource {
    return &<Name>DataSource{}
}

// Implement: Metadata, Schema, Configure, Read
```

### Updating the OpenAPI Spec

The vendored spec at `docs/api/openapi.json` is the source of truth.

```bash
# Download latest
curl -o docs/api/openapi.json https://cloud-api.nobus.io/api/v3/openapi.json

# Verify changes
git diff docs/api/openapi.json

# Update client models to match new spec
# Run tests to verify
go test ./internal/client/...
```

## Debugging

### Debugging the Provider with Delve

```bash
# Build with debug info
go build -gcflags="all=-N -l" -o terraform-provider-nobus .

# Run provider in debug mode
dlv exec ./terraform-provider-nobus -- --debug
```

In another terminal:
```bash
# Configure Terraform to use debug provider
export TF_REATTACH_PROVIDERS="nobus-cloud/nobus"
terraform init
terraform plan
```

### Debugging Unit Tests

```bash
# Run specific test with delve
dlv test ./internal/client -run TestVolumeCreate

# VS Code launch.json
{
    "type": "go",
    "request": "launch",
    "name": "Debug Test",
    "mode": "test",
    "program": "./internal/client",
    "args": ["-test.run", "TestVolumeCreate"]
}
```

### Debugging Acceptance Tests

```bash
# Run single acceptance test with verbose output
TF_ACC=1 go test ./internal/provider -run TestAccVolumeBasic -v -timeout 30m

# With delve
TF_ACC=1 dlv test ./internal/provider -run TestAccVolumeBasic
```

### Common Debug Scenarios

**Provider not found:**
```bash
# Check terraformrc
cat ~/.terraformrc

# Verify binary exists
ls -la /path/to/terraform-provider-nobus
```

**API request/response inspection:**
```go
// In client.go Do() method, add temporary logging
log.Printf("REQUEST: %s %s\n%s", method, endpoint.String(), string(payload))
log.Printf("RESPONSE: %d\n%s", res.StatusCode, string(raw))
```

**State drift issues:**
```bash
# Refresh state
terraform refresh

# Check state
terraform state list
terraform state show nobus_volume.example
```

## Testing Strategies

### Unit Test Patterns

**Client Tests** (`internal/client/*_test.go`):
```go
func TestVolumeCreateRequest(t *testing.T) {
    // Create test server
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "/api/v3/volume/", r.URL.Path)
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
        
        // Return mock response
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]any{
            "status": true,
            "data": map[string]any{
                "id": "vol-123",
                "name": "test-volume",
                "size": 100,
                "status": "ACTIVE",
            },
        })
    }))
    defer ts.Close()
    
    // Create client pointing to test server
    c, _ := client.New(client.Config{
        APIKey:  "test-key",
        BaseURL: ts.URL,
    })
    
    // Call method
    resp, err := c.CreateVolume(context.Background(), client.CreateVolumeRequest{
        Name: "test-volume",
        Size: 100,
    })
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, "vol-123", resp.ID)
}
```

**Provider Tests** (`internal/provider/*_test.go`):
```go
func TestVolumeModelFromAPI(t *testing.T) {
    apiResp := &client.VolumeResponse{
        ID: "vol-123",
        Name: "test",
        Size: 100,
        Status: "ACTIVE",
    }
    
    model := VolumeModelFromAPI(apiResp)
    
    assert.Equal(t, "vol-123", model.ID.ValueString())
    assert.Equal(t, "test", model.Name.ValueString())
    assert.Equal(t, int64(100), model.Size.ValueInt64())
}
```

### Acceptance Test Patterns

**Basic lifecycle test:**
```go
func TestAccVolumeBasic(t *testing.T) {
    resourceName := "nobus_volume.test"
    
    resource.Test(t, resource.TestCase{
        PreCheck: func() { testAccPreCheck(t) },
        ProviderFactories: testAccProviderFactories,
        CheckDestroy: testAccCheckVolumeDestroy,
        Steps: []resource.TestStep{
            {
                Config: testAccVolumeConfig(),
                Check: resource.ComposeTestCheckFunc(
                    testAccCheckVolumeExists(resourceName),
                    resource.TestCheckResourceAttr(resourceName, "name", "test-volume"),
                    resource.TestCheckResourceAttr(resourceName, "size", "100"),
                ),
            },
            {
                Config: testAccVolumeConfigUpdated(),
                Check: resource.ComposeTestCheckFunc(
                    testAccCheckVolumeExists(resourceName),
                    resource.TestCheckResourceAttr(resourceName, "size", "200"),
                ),
            },
            {
                ResourceName:      resourceName,
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

**Helper functions** (in `acceptance_test.go`):
```go
func testAccPreCheck(t *testing.T) {
    if os.Getenv("NOBUS_API_KEY") == "" {
        t.Fatal("NOBUS_API_KEY must be set for acceptance tests")
    }
    // ... check other required env vars
}

func testAccProviderFactories() map[string]func() (*schema.Provider, error) {
    return map[string]func() (*schema.Provider, error){
        "nobus": func() (*schema.Provider, error) {
            return Provider(), nil
        },
    }
}
```

## Code Generation

### Documentation Generation

```bash
# Install
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Generate all docs
tfplugindocs generate

# Generate specific provider
tfplugindocs generate --provider-name nobus

# Validate (no write)
tfplugindocs generate --dry-run
```

**What triggers doc changes:**
- Schema attribute `MarkdownDescription` changes
- New/removed attributes
- New/removed resources or data sources

**Commit generated docs** with your changes.

### Mock Generation (if needed)

```bash
# Install mockery
go install github.com/vektra/mockery/v2@latest

# Generate mock for interface
mockery --name=Client --dir=internal/client --output=internal/client/mocks
```

### Protobuf/gRPC (Not Used)

This project uses REST + JSON, not gRPC. No protobuf generation needed.

## Performance Considerations

### Client-Side

- **Connection pooling**: `http.Client` reuses connections by default
- **Request timeouts**: Set via context, not client-level
- **Polling interval**: Default 2s, configurable per resource
- **Pagination**: Use cursor-based where available

### Provider-Side

- **Schema validation**: Runs on every plan/apply
- **State storage**: Terraform handles, keep models lean
- **Read operations**: Minimize API calls in `Read`
- **Batch operations**: Not currently supported by Nobus API

### Optimization Checklist

- [ ] Reuse `http.Client` (already done via provider config)
- [ ] Avoid N+1 reads in data sources
- [ ] Use `Computed` for server-generated values
- [ ] Mark sensitive attributes correctly
- [ ] Set `PlanModifiers` for default values

## Troubleshooting

### Build Issues

**"go: cannot find main module"**
```bash
# Ensure you're in project root
cd /path/to/terraform-provider-nobus
go mod tidy
```

**"package ... is not in GOROOT"**
```bash
# Use Go modules (default in 1.16+)
export GO111MODULE=on
go mod tidy
```

### Test Failures

**Acceptance test timeout**
```bash
# Increase timeout
TF_ACC=1 go test ./internal/provider -run TestAccVolumeBasic -timeout 180m
```

**Flaky tests due to API rate limits**
```go
// Add retry logic in test helpers
func createWithRetry(t *testing.T, config string) {
    for i := 0; i < 3; i++ {
        err := applyConfig(config)
        if err == nil { return }
        time.Sleep(5 * time.Second)
    }
    t.Fatal("Failed after retries")
}
```

### Provider Runtime Errors

**"provider doesn't support resource"**
- Check resource registered in `provider.go` `Resources()`
- Rebuild provider: `go install .`

**"invalid provider configuration"**
- Check `Configure` method in `configure.go`
- Verify environment variables set correctly

**"state doesn't match config"**
- Run `terraform refresh`
- Check `Read` implementation matches API response
- Verify converters handle all fields

### API Errors

**401 Unauthorized**
- Verify API key is valid
- Check `NOBUS_API_KEY` environment variable
- Ensure key has required permissions

**404 Not Found on Read**
- Implement `isNotFound` check in resource
- Verify resource ID format matches API expectation

**400 Bad Request**
- Check request payload matches OpenAPI spec
- Verify required fields are set
- Check enum values match API expectations

**500 Internal Server Error**
- Usually transient, retry with backoff
- Report to Nobus support if persistent

### Common Gotchas

1. **Pointer vs value receivers** - Use pointer receivers for methods that modify state
2. **Context cancellation** - Always pass `ctx` through call chain
3. **Sensitive values** - Use `types.String` with `Sensitive: true`, not plain strings
4. **Unknown values** - Handle `IsUnknown()` in `Configure`
5. **Plan vs Apply** - Don't make API calls in `Create` that should be in `Read`

## Useful Commands Reference

```bash
# Development
go fmt ./...
go vet ./...
golangci-lint run
go test ./...
go test -race ./...

# Documentation
tfplugindocs generate

# Building
go build -o terraform-provider-nobus .
go install .

# Testing
go test ./internal/client/... -v
go test ./internal/provider/... -v
TF_ACC=1 go test ./internal/provider -run TestAcc -v -timeout 120m

# Debugging
dlv debug .
dlv test ./internal/client -run TestVolume

# Release (maintainers only)
goreleaser release --snapshot --clean

# Clean
go clean -cache -testcache -modcache
rm terraform-provider-nobus
```

## Further Reading

- [Terraform Plugin Framework Docs](https://developer.hashicorp.com/terraform/plugin/framework)
- [Nobus Cloud API Docs](https://cloud-api.nobus.io/api/v3/openapi.json)
- [Go Testing Package](https://pkg.go.dev/testing)
- [golangci-lint Rules](https://golangci-lint.run/usage/linters/)