// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringNotEmptyValidator rejects empty and whitespace-only strings.
type stringNotEmptyValidator struct{}

func (v stringNotEmptyValidator) Description(ctx context.Context) string {
	return "value must not be empty"
}

func (v stringNotEmptyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringNotEmptyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if strings.TrimSpace(req.ConfigValue.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			"The value must not be empty.",
		)
	}
}

// StringNotEmpty returns a validator rejecting empty strings.
func StringNotEmpty() validator.String { return stringNotEmptyValidator{} }

// stringOneOfValidator restricts a string to a fixed set of values.
type stringOneOfValidator struct {
	allowed []string
}

func (v stringOneOfValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be one of: %s", strings.Join(v.allowed, ", "))
}

func (v stringOneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v stringOneOfValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, allowed := range v.allowed {
		if req.ConfigValue.ValueString() == allowed {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid value",
		fmt.Sprintf("Value %q is not valid. %s", req.ConfigValue.ValueString(), v.Description(ctx)),
	)
}

// StringOneOf returns a validator accepting only the listed values.
func StringOneOf(allowed ...string) validator.String {
	return stringOneOfValidator{allowed: allowed}
}

// int64AtLeastValidator enforces a lower bound.
type int64AtLeastValidator struct {
	minimum int64
}

func (v int64AtLeastValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be at least %d", v.minimum)
}

func (v int64AtLeastValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int64AtLeastValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueInt64() < v.minimum {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			fmt.Sprintf("Value %d is not valid. %s.", req.ConfigValue.ValueInt64(), v.Description(ctx)),
		)
	}
}

// Int64AtLeast returns a validator enforcing an inclusive lower bound.
func Int64AtLeast(minimum int64) validator.Int64 { return int64AtLeastValidator{minimum: minimum} }

// optionalString converts an optional Terraform string into a Go pointer that
// is nil when the value is null or unknown. This maps directly onto the
// pointer semantics used by the Nobus request schemas, where an absent field
// means "leave unchanged".
func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	converted := value.ValueString()
	return &converted
}

// optionalStringDefault behaves like optionalString but substitutes a default
// for null or unknown values.
func optionalStringDefault(value types.String, fallback string) *string {
	if value.IsNull() || value.IsUnknown() {
		return &fallback
	}
	converted := value.ValueString()
	return &converted
}

// optionalBool converts an optional Terraform boolean into a Go pointer.
func optionalBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	converted := value.ValueBool()
	return &converted
}

// optionalInt64 converts an optional Terraform integer into a Go pointer.
func optionalInt64(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	converted := value.ValueInt64()
	return &converted
}

// stringOrNull converts a Go string pointer into a Terraform string, mapping nil
// onto a null value so that unset optional attributes do not produce diffs.
func stringOrNull(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// stringValueOrNull converts a Go string into a Terraform string, treating the
// empty string as absent.
func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

// boolOrNull converts a Go bool pointer into a Terraform boolean.
func boolOrNull(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

// int64OrNull converts a Go int64 pointer into a Terraform integer.
func int64OrNull(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*value)
}
