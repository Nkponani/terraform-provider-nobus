// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

type hostRouteModel struct {
	Destination types.String `tfsdk:"destination"`
	NextHop     types.String `tfsdk:"next_hop"`
}
type allocationPoolModel struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

var hostRouteType = types.ObjectType{AttrTypes: map[string]attr.Type{"destination": types.StringType, "next_hop": types.StringType}}
var allocationPoolType = types.ObjectType{AttrTypes: map[string]attr.Type{"start": types.StringType, "end": types.StringType}}

func hostRoutesFromList(ctx context.Context, v types.List) ([]client.HostRoute, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var in []hostRouteModel
	d := v.ElementsAs(ctx, &in, false)
	out := make([]client.HostRoute, 0, len(in))
	for _, x := range in {
		out = append(out, client.HostRoute{Destination: x.Destination.ValueString(), NextHop: x.NextHop.ValueString()})
	}
	return out, d
}
func allocationPoolsFromList(ctx context.Context, v types.List) ([]client.AllocationPool, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var in []allocationPoolModel
	d := v.ElementsAs(ctx, &in, false)
	out := make([]client.AllocationPool, 0, len(in))
	for _, x := range in {
		out = append(out, client.AllocationPool{Start: x.Start.ValueString(), End: x.End.ValueString()})
	}
	return out, d
}
func hostRoutesToList(ctx context.Context, v []client.HostRoute) (types.List, diag.Diagnostics) {
	in := make([]hostRouteModel, 0, len(v))
	for _, x := range v {
		in = append(in, hostRouteModel{Destination: types.StringValue(x.Destination), NextHop: types.StringValue(x.NextHop)})
	}
	return types.ListValueFrom(ctx, hostRouteType, in)
}
func allocationPoolsToList(ctx context.Context, v []client.AllocationPool) (types.List, diag.Diagnostics) {
	in := make([]allocationPoolModel, 0, len(v))
	for _, x := range v {
		in = append(in, allocationPoolModel{Start: types.StringValue(x.Start), End: types.StringValue(x.End)})
	}
	return types.ListValueFrom(ctx, allocationPoolType, in)
}
