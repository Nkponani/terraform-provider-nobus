# Contributing to terraform-provider-nobus

Thank you for your interest in contributing! This document outlines the process and standards for contributing to the Nobus Terraform provider.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Pull Request Process](#pull-request-process)
- [Release Process](#release-process)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- **Go 1.25+** - Required for building the provider
- **Terraform 1.0+** - For testing provider behavior
- **Git** - Version control
- **Make** (optional) - For common tasks

### Local Development Setup

```bash
# Clone the repository
git clone https://github.com/nobus-cloud/terraform-provider-nobus.git
cd terraform-provider-nobus

# Install dependencies
go mod tidy

# Build the provider
go build -o terraform-provider-nobus .

# Run unit tests
go test ./...

# Run linter
golangci-lint run
```

### Using Local Build with Terraform

Create `~/.terraformrc`:
```hcl
provider_installation {
  dev_overrides {
    "nobus-cloud/nobus" = "/absolute/path/to/terraform-provider-nobus"
  }
  direct {}
}
```

Then in your Terraform configuration:
```bash
terraform init   # Will use local build
terraform plan
terraform apply
```

## Development Workflow

### 1. Create an Issue First

Before starting work, **create or claim an issue** describing:
- What you're adding/fixing
- Why it's needed
- Any design considerations

This avoids duplicate work and allows maintainers to provide early feedback.

### 2. Branch Naming

Use descriptive branch names:
```
feature/add-database-read-replica
fix/volume-snapshot-delete-timeout
docs/update-kubernetes-examples
refactor/client-error-handling
```

### 3. Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`

Examples:
```
feat(storage): add volume backup restore resource
fix(networking): handle router gateway clearing on update
docs: add architecture decision records
refactor(client): extract common pagination logic
test(lbaas): add acceptance test for listener stats
```

### 4. Keep Changes Focused

- One logical change per PR
- Separate refactoring from functional changes
- If a PR grows large, consider splitting it

## Code Standards

### Go Style

- **Follow standard Go conventions** - `gofmt`, `go vet`, `golangci-lint`
- **No `golint`** - Use `golangci-lint` with default config (`.golangci.yml`)
- **Maximum file size**: ~250 lines (prefer splitting)
- **Exhaustive error handling** - No ignored errors

### Project-Specific Patterns

#### 1. API Client (`internal/client/`)

```go
// One file per API collection
// Centralized paths in *_paths.go
// Shared helpers in *_helpers.go
// Typed errors in errors.go

// Request/Response models match OpenAPI spec exactly
// No Terraform types in client package
```

#### 2. Provider Resources (`internal/provider/`)

```go
// One resource per file: <name>_resource.go
// State model with tfsdk tags
// Conversion functions in _resource_helpers.go or shared/converters.go
// Full CRUD + ImportState implementation
```

#### 3. Schema Documentation

Every schema attribute **must** have `MarkdownDescription`:
```go
"size": schema.Int64Attribute{
    MarkdownDescription: "Size of the volume in gigabytes. " +
        "Increasing this value grows the volume in place. " +
        "Volumes cannot be shrunk.",
    Required: true,
},
```

#### 4. Error Messages

Preserve API error messages in diagnostics:
```go
resp.Diagnostics.AddError(
    "Unable to create Nobus volume",
    fmt.Sprintf("The Nobus API returned: %s", err.Error()),
)
```

#### 5. Sensitive Data

Mark secrets as sensitive:
```go
"api_key": schema.StringAttribute{
    Sensitive: true,
    Optional:  true,
},
```

### File Organization

```
internal/client/
├── client.go              # Core HTTP client
├── errors.go              # Error types
├── networking_paths.go    # Networking endpoints
├── storage_paths.go       # Storage endpoints
├── networking_helpers.go  # Networking shared logic
├── storage_helpers.go     # Storage shared logic
├── networks.go            # Network operations
├── subnets.go             # Subnet operations
├── volumes.go             # Volume operations
└── ...                    # One file per collection

internal/provider/
├── provider.go            # Provider registration
├── configure.go           # Client construction
├── shared/                # Cross-cutting helpers
│   ├── config.go         # Provider config
│   ├── schema.go         # Common schema patterns
│   ├── converters.go     # Model ↔ API conversions
│   └── resource_helpers.go
├── networking/           # Networking resources
│   ├── network_resource.go
│   ├── subnet_resource.go
│   └── ...
├── storage/              # Storage resources
├── fos/                  # FOS resources
├── kubernetes/           # K8s resources
├── database/             # Database resources
├── lbaas/                # LBaaS resources
└── datasources/          # Collection data sources
```

## Testing

### Unit Tests (Required for All Changes)

```bash
# Run all unit tests
go test ./...

# Run specific package
go test ./internal/client/...
go test ./internal/provider/...

# With coverage
go test -cover ./...
```

**Requirements for unit tests:**
- Test request construction
- Test response decoding
- Test error handling paths
- Test state mapping (Terraform ↔ API)
- Use `httptest` for client tests
- No external dependencies

### Acceptance Tests (For New Resources)

Required for new resources and significant changes:

```bash
# Set required environment variables
export NOBUS_API_KEY="..."
export NOBUS_PROJECT_ID="..."
export NOBUS_TEST_IMAGE="ubuntu-22-04"
export NOBUS_TEST_FLAVOR="small"
export NOBUS_TEST_KEY_NAME="operations"

# Run acceptance tests
TF_ACC=1 go test ./internal/provider/ -run TestAcc<ResourceName> -v -timeout 120m
```

**Acceptance test patterns:**
- Create → Read → Update → Delete lifecycle
- Test import functionality
- Test edge cases (conflicts, not found, validation errors)
- Clean up resources in `defer` or `tearDown`

### Test Naming

```go
// Unit tests
func TestVolumeCreateRequest(t *testing.T) { ... }
func TestVolumeStateFromAPI(t *testing.T) { ... }

// Acceptance tests
func TestAccVolumeBasic(t *testing.T) { ... }
func TestAccVolumeUpdate(t *testing.T) { ... }
func TestAccVolumeImport(t *testing.T) { ... }
```

## Documentation

### Auto-Generated Documentation

Resource and data source docs are generated from code:

```bash
# Install generator
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Generate
tfplugindocs generate
```

**What gets generated:**
- `docs/resources/<name>.md` - From resource schema
- `docs/data-sources/<name>.md` - From data source schema
- `docs/index.md` - Provider overview

**You must:**
- Write clear `MarkdownDescription` on all schema attributes
- Run `tfplugindocs generate` after schema changes
- Commit generated docs with your changes

### Manual Documentation

Update these files when relevant:
- `README.md` - Overview, examples, authentication
- `ARCHITECTURE.md` - Architecture changes
- `CHANGELOG.md` - User-facing changes
- `examples/` - New usage examples

### Example Guidelines

Examples in `examples/` should:
- Be complete, runnable configurations
- Use variables for secrets (`var.api_key`)
- Include `provider.tf` with required configuration
- Have descriptive comments

## Pull Request Process

### Before Submitting

Run the full check suite:
```bash
# Format
go fmt ./...

# Vet
go vet ./...

# Lint
golangci-lint run

# Test
go test ./...

# Generate docs
tfplugindocs generate

# Check for generated changes
git diff --stat docs/
```

### PR Checklist

- [ ] Issue referenced in PR description
- [ ] All checks pass (fmt, vet, lint, test)
- [ ] Documentation generated (`tfplugindocs generate`)
- [ ] Unit tests added/updated
- [ ] Acceptance tests added (for new resources)
- [ ] CHANGELOG.md updated (user-facing changes)
- [ ] No unrelated changes (whitespace, reformatting)

### PR Review

- Maintainers will review within 3 business days
- Address feedback promptly
- Squash commits before merge if requested
- PRs require at least one maintainer approval

### Merge Requirements

- All CI checks pass
- At least one maintainer approval
- No merge conflicts
- Linear history (rebase preferred)

## Release Process

Releases are automated via **goreleaser** on tag push:

```bash
# Create and push tag
git tag v1.2.3
git push origin v1.2.3
```

Goreleaser will:
1. Build binaries for all platforms
2. Generate checksums
3. Create GitHub release
4. Publish to Terraform Registry

### Versioning

Follow [Semantic Versioning](https://semver.org/):
- **Major** (v2.0.0) - Breaking changes
- **Minor** (v1.3.0) - New features, backward compatible
- **Patch** (v1.2.1) - Bug fixes, backward compatible

### Changelog

Update `CHANGELOG.md` with each PR:
```markdown
## [Unreleased]

### Added
- New `nobus_database_read_replica` resource

### Fixed
- Volume attachment timeout handling

### Changed
- Updated minimum Terraform version to 1.5
```

## Getting Help

- **Questions**: Open a GitHub Discussion
- **Bugs**: Open a GitHub Issue with reproduction steps
- **Security**: Email security@nobus.cloud (not public issues)

## Recognition

Contributors are recognized in:
- Release notes
- `CONTRIBUTORS.md` (auto-generated from git history)

Thank you for contributing! 🎉