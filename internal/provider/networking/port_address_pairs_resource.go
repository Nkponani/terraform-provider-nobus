// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type portAddressPairModel struct {
	IPAddress  types.String `tfsdk:"ip_address"`
	MACAddress types.String `tfsdk:"mac_address"`
}

var portAddressPairType = types.ObjectType{AttrTypes: map[string]attr.Type{"ip_address": types.StringType, "mac_address": types.StringType}}

func portAddressPairs(ctx context.Context, v types.List) ([]client.PortAddressPair, diag.Diagnostics) {
	if v.IsNull() || v.IsUnknown() {
		return nil, nil
	}
	var in []portAddressPairModel
	d := v.ElementsAs(ctx, &in, false)
	out := make([]client.PortAddressPair, 0, len(in))
	for _, p := range in {
		out = append(out, client.PortAddressPair{IPAddress: p.IPAddress.ValueString(), MACAddress: shared.OptionalString(p.MACAddress)})
	}
	return out, d
}
func portAddressPairsValue(ctx context.Context, v []client.PortAddressPair) (types.List, diag.Diagnostics) {
	in := make([]portAddressPairModel, 0, len(v))
	for _, p := range v {
		in = append(in, portAddressPairModel{IPAddress: types.StringValue(p.IPAddress), MACAddress: shared.StringOrNull(p.MACAddress)})
	}
	return types.ListValueFrom(ctx, portAddressPairType, in)
}

type portAddressPairsResource struct{ config *shared.Config }
type portAddressPairsModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	PortID           types.String `tfsdk:"port_id"`
	AddressPairs     types.List   `tfsdk:"address_pairs"`
}

func NewPortAddressPairsResource() resource.Resource { return &portAddressPairsResource{} }
func (r *portAddressPairsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_address_pairs"
}
func (r *portAddressPairsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *portAddressPairsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages the allowed address pairs on a Nobus network port.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "port_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "address_pairs": schema.ListNestedAttribute{Required: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"ip_address": schema.StringAttribute{Required: true}, "mac_address": schema.StringAttribute{Optional: true}}}}})}
}
func (r *portAddressPairsResource) apply(ctx context.Context, p *portAddressPairsModel, d *diag.Diagnostics) bool {
	c := shared.RequireConfigured(r.config, d)
	if c == nil {
		return false
	}
	s, ok := r.config.ResolveScope(p.ProjectID, p.AvailabilityZone, d)
	if !ok {
		return false
	}
	pairs, pd := portAddressPairs(ctx, p.AddressPairs)
	d.Append(pd...)
	if d.HasError() {
		return false
	}
	var e error
	if len(pairs) == 0 {
		e = c.ClearPortAddressPairs(ctx, s.ProjectID, s.AvailabilityZone, p.PortID.ValueString())
	} else {
		e = c.SetPortAddressPairs(ctx, s.ProjectID, s.AvailabilityZone, p.PortID.ValueString(), pairs)
	}
	if e != nil {
		d.AddError("Unable to configure Nobus port address pairs", shared.DiagnosticDetail(e))
		return false
	}
	p.ID = types.StringValue(p.PortID.ValueString())
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	return true
}
func (r *portAddressPairsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p portAddressPairsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if r.apply(ctx, &p, &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	}
}
func (r *portAddressPairsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var p portAddressPairsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.ResolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	port, err := c.GetPort(ctx, s.ProjectID, s.AvailabilityZone, p.PortID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus port address pairs", shared.DiagnosticDetail(err))
		return
	}
	pairs, d := portAddressPairsValue(ctx, port.AllowedAddressPairs)
	resp.Diagnostics.Append(d...)
	p.AddressPairs = pairs
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *portAddressPairsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p portAddressPairsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if r.apply(ctx, &p, &resp.Diagnostics) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	}
}
func (r *portAddressPairsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var p portAddressPairsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.ResolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.ClearPortAddressPairs(ctx, s.ProjectID, s.AvailabilityZone, p.PortID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to clear Nobus port address pairs", shared.DiagnosticDetail(e))
	}
}

func (r *portAddressPairsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("port_id"), req.ID)...)
}

var _ resource.ResourceWithConfigure = &portAddressPairsResource{}
var _ resource.ResourceWithImportState = &portAddressPairsResource{}
