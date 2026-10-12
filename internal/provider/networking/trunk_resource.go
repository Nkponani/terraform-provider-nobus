// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type trunkResource struct{ config *shared.Config }
type trunkModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	PortID           types.String `tfsdk:"port_id"`
	SubPorts         types.List   `tfsdk:"sub_ports"`
	Description      types.String `tfsdk:"description"`
	Tags             types.Set    `tfsdk:"tags"`
	Status           types.String `tfsdk:"status"`
}

func NewTrunkResource() resource.Resource { return &trunkResource{} }
func (r *trunkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trunk"
}
func (r *trunkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *trunkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus network trunk joining a parent port to VLAN subports.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true}, "port_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "sub_ports": schema.ListNestedAttribute{Optional: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"port_id": schema.StringAttribute{Required: true}, "segmentation_type": schema.StringAttribute{Optional: true, Computed: true}, "segmentation_id": schema.Int64Attribute{Required: true}}}}, "description": schema.StringAttribute{Optional: true}, "tags": schema.SetAttribute{Optional: true, ElementType: types.StringType}, "status": schema.StringAttribute{Computed: true}})}
}
func (r *trunkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p trunkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
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
	subs, d1 := trunkSubPorts(ctx, p.SubPorts)
	tags, d2 := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d1...)
	resp.Diagnostics.Append(d2...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := c.CreateTrunk(ctx, s.ProjectID, s.AvailabilityZone, client.TrunkCreate{Name: p.Name.ValueString(), PortID: p.PortID.ValueString(), SubPorts: subs, Description: shared.OptionalString(p.Description), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus network trunk", shared.DiagnosticDetail(e))
		return
	}
	applyTrunk(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *trunkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s trunkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.ResolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetTrunk(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus network trunk", shared.DiagnosticDetail(e))
		return
	}
	applyTrunk(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *trunkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p trunkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
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
	subs, d1 := trunkSubPorts(ctx, p.SubPorts)
	tags, d2 := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d1...)
	resp.Diagnostics.Append(d2...)
	v, e := c.UpdateTrunk(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString(), client.TrunkUpdate{Name: shared.OptionalString(p.Name), SubPorts: subs, Description: shared.OptionalString(p.Description), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update Nobus network trunk", shared.DiagnosticDetail(e))
		return
	}
	applyTrunk(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *trunkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s trunkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.ResolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.DeleteTrunk(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus network trunk", shared.DiagnosticDetail(e))
	}
}
func (r *trunkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyTrunk(ctx context.Context, v *client.Trunk, m *trunkModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.Name = shared.StringOrNull(v.Name)
	m.PortID = types.StringValue(v.PortID)
	m.Description = shared.StringOrNull(v.Description)
	m.Status = shared.StringOrNull(v.Status)
	if v.SubPorts != nil {
		x, ds := trunkSubPortsValue(ctx, v.SubPorts)
		d.Append(ds...)
		m.SubPorts = x
	}
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
}

var _ resource.ResourceWithConfigure = &trunkResource{}
var _ resource.ResourceWithImportState = &trunkResource{}
