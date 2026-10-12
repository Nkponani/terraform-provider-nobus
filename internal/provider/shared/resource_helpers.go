// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

const PollInterval = 2 * time.Second

// Scope is the project and availability zone pair that every Nobus API
// operation requires.
type Scope struct {
	ProjectID        string
	AvailabilityZone string
}

// ResolveScope determines the project and availability zone for an operation,
// falling back to the provider defaults. Values already recorded in state win
// over provider defaults so that Read and Delete keep working if the provider
// configuration later changes.
func (p *Config) ResolveScope(projectID, availabilityZone types.String, diags *diag.Diagnostics) (Scope, bool) {
	resolvedProject := firstNonEmpty(NullSafeString(projectID), p.ProjectID)
	resolvedZone := firstNonEmpty(
		NullSafeString(availabilityZone),
		p.AvailabilityZone,
		client.DefaultAvailabilityZone,
	)

	if resolvedProject == "" {
		diags.AddAttributeError(
			path.Root("project_id"),
			"Missing Nobus project ID",
			"Every Nobus API operation requires a project ID. Set the `project_id` attribute on the resource "+
				"or set `project_id` in the provider block.",
		)
		return Scope{}, false
	}

	return Scope{ProjectID: resolvedProject, AvailabilityZone: resolvedZone}, true
}

// requireClient reports whether the provider was configured, adding a clear
// diagnostic when it was not.
func RequireConfigured(config *Config, diags *diag.Diagnostics) *client.Client {
	if config == nil || config.Client == nil {
		diags.AddError(
			"Nobus provider not configured",
			"The Nobus provider was not configured, so no API client is available. This usually means the "+
				"provider block is missing or the provider configuration failed.",
		)
		return nil
	}
	return config.Client
}

// NullSafeString returns the value of a Terraform string, treating null and
// unknown as empty.
func NullSafeString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return strings.TrimSpace(value.ValueString())
}

// DiagnosticDetail renders an API error as a diagnostic detail message while
// preserving the information Nobus returned. Credentials are never included.
func DiagnosticDetail(err error) string {
	if err == nil {
		return ""
	}

	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.Message != "" {
		return fmt.Sprintf("The Nobus API returned HTTP %d: %s", apiErr.StatusCode, apiErr.Message)
	}

	var statusErr *client.StatusError
	if errors.As(err, &statusErr) && statusErr.Message != "" {
		return fmt.Sprintf("The Nobus API reported a failure: %s", statusErr.Message)
	}

	return err.Error()
}

// ListToStringSlice converts a Terraform list of strings into a Go slice.
func ListToStringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

// SetToStringSlice converts a Terraform set of strings into a Go slice.
func SetToStringSlice(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var out []string
	diags := set.ElementsAs(ctx, &out, false)
	return out, diags
}

// MapToStringMap converts a Terraform map of strings into a Go map.
func MapToStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}

	out := make(map[string]string, len(m.Elements()))
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}
