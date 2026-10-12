// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func ReplaceStrings() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}
}
func ScopeAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"project_id":        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: ReplaceStrings()},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: ReplaceStrings()},
	}
}
func MergeAttributes(base map[string]schema.Attribute, more map[string]schema.Attribute) map[string]schema.Attribute {
	for k, v := range more {
		base[k] = v
	}
	return base
}
func ImportID(ctx context.Context, id string, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
