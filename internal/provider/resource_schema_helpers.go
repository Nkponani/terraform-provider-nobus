// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
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
