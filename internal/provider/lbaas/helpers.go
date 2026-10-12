// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lbaas

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type apiResource struct {
	ID                 string   `json:"id"`
	Name               *string  `json:"name"`
	Description        *string  `json:"description"`
	AdminStateUp       bool     `json:"admin_state_up"`
	ProvisioningStatus string   `json:"provisioning_status"`
	OperatingStatus    string   `json:"operating_status"`
	Tags               []string `json:"tags"`
}

func stringsFromList(ctx context.Context, value types.List) ([]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return []string{}, nil
	}
	var result []string
	diagnostics := value.ElementsAs(ctx, &result, false)
	return result, diagnostics
}

func listFromStrings(ctx context.Context, value []string) (types.List, diag.Diagnostics) {
	if value == nil {
		value = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, value)
}

func decodeResource(raw json.RawMessage, target any) error { return json.Unmarshal(raw, target) }

func stringPointerValue(value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
func plainStringValue(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func optionalString(value types.String) any {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	return value.ValueString()
}

func persistenceValue(kind, cookie types.String) any {
	if kind.IsNull() || kind.IsUnknown() || kind.ValueString() == "" {
		return nil
	}
	result := map[string]any{"type": kind.ValueString()}
	if !cookie.IsNull() && !cookie.IsUnknown() && cookie.ValueString() != "" {
		result["cookie_name"] = cookie.ValueString()
	}
	return result
}
