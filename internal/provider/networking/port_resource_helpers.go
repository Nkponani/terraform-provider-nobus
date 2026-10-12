// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type fixedIPModel struct {
	SubnetID  types.String `tfsdk:"subnet_id"`
	IPAddress types.String `tfsdk:"ip_address"`
}
type extraDHCPOptionModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

var fixedIPType = types.ObjectType{AttrTypes: map[string]attr.Type{"subnet_id": types.StringType, "ip_address": types.StringType}}
var extraDHCPOptionType = types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType, "value": types.StringType}}

func fixedIPsFromList(ctx context.Context, v types.List) ([]client.FixedIP, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var in []fixedIPModel
	d := v.ElementsAs(ctx, &in, false)
	out := make([]client.FixedIP, 0, len(in))
	for _, x := range in {
		out = append(out, client.FixedIP{SubnetID: x.SubnetID.ValueString(), IPAddress: shared.OptionalString(x.IPAddress)})
	}
	return out, d
}
func extraDHCPOptionsFromList(ctx context.Context, v types.List) ([]client.ExtraDHCPOption, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var in []extraDHCPOptionModel
	d := v.ElementsAs(ctx, &in, false)
	out := make([]client.ExtraDHCPOption, 0, len(in))
	for _, x := range in {
		out = append(out, client.ExtraDHCPOption{Name: x.Name.ValueString(), Value: x.Value.ValueString()})
	}
	return out, d
}
func fixedIPsValue(ctx context.Context, v []client.FixedIP) (types.List, diag.Diagnostics) {
	in := make([]fixedIPModel, 0, len(v))
	for _, x := range v {
		in = append(in, fixedIPModel{SubnetID: types.StringValue(x.SubnetID), IPAddress: shared.StringOrNull(x.IPAddress)})
	}
	return types.ListValueFrom(ctx, fixedIPType, in)
}
func extraDHCPOptionsValue(ctx context.Context, v []client.ExtraDHCPOption) (types.List, diag.Diagnostics) {
	in := make([]extraDHCPOptionModel, 0, len(v))
	for _, x := range v {
		in = append(in, extraDHCPOptionModel{Name: types.StringValue(x.Name), Value: types.StringValue(x.Value)})
	}
	return types.ListValueFrom(ctx, extraDHCPOptionType, in)
}
