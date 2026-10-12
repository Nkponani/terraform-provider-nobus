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

type trunkSubPortModel struct {
	PortID           types.String `tfsdk:"port_id"`
	SegmentationType types.String `tfsdk:"segmentation_type"`
	SegmentationID   types.Int64  `tfsdk:"segmentation_id"`
}

var trunkSubPortObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{"port_id": types.StringType, "segmentation_type": types.StringType, "segmentation_id": types.Int64Type}}

func trunkSubPorts(ctx context.Context, v types.List) ([]client.TrunkSubPort, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var values []trunkSubPortModel
	d := v.ElementsAs(ctx, &values, false)
	out := make([]client.TrunkSubPort, 0, len(values))
	for _, p := range values {
		kind := p.SegmentationType.ValueString()
		if kind == "" {
			kind = "vlan"
		}
		out = append(out, client.TrunkSubPort{PortID: p.PortID.ValueString(), SegmentationType: kind, SegmentationID: p.SegmentationID.ValueInt64()})
	}
	return out, d
}
func trunkSubPortsValue(ctx context.Context, v []client.TrunkSubPort) (types.List, diag.Diagnostics) {
	items := make([]trunkSubPortModel, 0, len(v))
	for _, p := range v {
		items = append(items, trunkSubPortModel{PortID: types.StringValue(p.PortID), SegmentationType: types.StringValue(p.SegmentationType), SegmentationID: types.Int64Value(p.SegmentationID)})
	}
	return types.ListValueFrom(ctx, trunkSubPortObjectType, items)
}
