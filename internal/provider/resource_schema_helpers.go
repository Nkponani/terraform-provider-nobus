// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func replaceStrings() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}
}
func scopeAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"project_id":        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replaceStrings()},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replaceStrings()},
	}
}
func mergeAttributes(base map[string]schema.Attribute, more map[string]schema.Attribute) map[string]schema.Attribute {
	for k, v := range more {
		base[k] = v
	}
	return base
}
func importID(ctx context.Context, id string, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
