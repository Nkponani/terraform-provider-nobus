// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

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

const pollInterval = 2 * time.Second

// scope is the project and availability zone pair that every Nobus API
// operation requires.
type scope struct {
	ProjectID        string
	AvailabilityZone string
}

// resolveScope determines the project and availability zone for an operation,
// falling back to the provider defaults. Values already recorded in state win
// over provider defaults so that Read and Delete keep working if the provider
// configuration later changes.
func (p *providerConfig) resolveScope(projectID, availabilityZone types.String, diags *diag.Diagnostics) (scope, bool) {
	resolvedProject := firstNonEmpty(nullSafeString(projectID), p.ProjectID)
	resolvedZone := firstNonEmpty(
		nullSafeString(availabilityZone),
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
		return scope{}, false
	}

	return scope{ProjectID: resolvedProject, AvailabilityZone: resolvedZone}, true
}

// requireClient reports whether the provider was configured, adding a clear
// diagnostic when it was not.
func requireConfigured(config *providerConfig, diags *diag.Diagnostics) *client.Client {
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

// nullSafeString returns the value of a Terraform string, treating null and
// unknown as empty.
func nullSafeString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return strings.TrimSpace(value.ValueString())
}

// diagnosticDetail renders an API error as a diagnostic detail message while
// preserving the information Nobus returned. Credentials are never included.
func diagnosticDetail(err error) string {
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

// listToStringSlice converts a Terraform list of strings into a Go slice.
func listToStringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

// setToStringSlice converts a Terraform set of strings into a Go slice.
func setToStringSlice(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var out []string
	diags := set.ElementsAs(ctx, &out, false)
	return out, diags
}

// mapToStringMap converts a Terraform map of strings into a Go map.
func mapToStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}

	out := make(map[string]string, len(m.Elements()))
	diags := m.ElementsAs(ctx, &out, false)
	return out, diags
}

// requireCreateValue reports a clear diagnostic when a value that the Nobus API
// requires at creation time is absent.
//
// Such attributes are modelled as Optional because the API never returns them,
// so keeping them Required would force a replacement after an import. They are
// validated here instead, at the point where they are actually needed.
func requireCreateValue(ctx context.Context, diags *diag.Diagnostics, attribute string, value types.String, model networkResourceModel) error {
	if nullSafeString(value) != "" {
		return nil
	}

	diags.AddAttributeError(
		path.Root(attribute),
		fmt.Sprintf("Missing required %q", attribute),
		fmt.Sprintf(
			"The Nobus API requires %q when creating a network, but it is not set. Add it to the resource, "+
				"or remove it from configuration if this network was imported and you do not want to replace it.",
			attribute,
		),
	)
	return errMissingCreateValue
}

// errMissingCreateValue signals that a create-time validation failed.
var errMissingCreateValue = errors.New("missing create-time value")
