// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type routerInterfaceResource struct{ config *shared.Config }
type routerInterfaceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	RouterID         types.String `tfsdk:"router_id"`
	SubnetID         types.String `tfsdk:"subnet_id"`
	PortID           types.String `tfsdk:"port_id"`
}

func NewRouterInterfaceResource() resource.Resource { return &routerInterfaceResource{} }
func (r *routerInterfaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router_interface"
}
func (r *routerInterfaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *routerInterfaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Connects a Nobus router to a subnet or port.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "router_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "subnet_id": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()}, "port_id": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()}})}
}
func (r *routerInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p routerInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if p.SubnetID.IsNull() && p.PortID.IsNull() {
		resp.Diagnostics.AddError("Missing router interface target", "Set subnet_id or port_id.")
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
	if e := c.SetRouterInterface(ctx, s.ProjectID, s.AvailabilityZone, p.RouterID.ValueString(), p.SubnetID.ValueString(), p.PortID.ValueString(), true); e != nil {
		resp.Diagnostics.AddError("Unable to add Nobus router interface", shared.DiagnosticDetail(e))
		return
	}
	p.ID = types.StringValue(strings.Join([]string{p.RouterID.ValueString(), p.SubnetID.ValueString(), p.PortID.ValueString()}, "/"))
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *routerInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *routerInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *routerInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s routerInterfaceModel
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
	if e := c.SetRouterInterface(ctx, sc.ProjectID, sc.AvailabilityZone, s.RouterID.ValueString(), s.SubnetID.ValueString(), s.PortID.ValueString(), false); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to remove Nobus router interface", shared.DiagnosticDetail(e))
	}
}
func (r *routerInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid router interface import ID", "Use ROUTER_ID/SUBNET_ID/PORT_ID; leave the unused final component empty.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("router_id"), parts[0])...)
	if parts[1] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subnet_id"), parts[1])...)
	}
	if parts[2] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("port_id"), parts[2])...)
	}
}

var _ resource.ResourceWithConfigure = &routerInterfaceResource{}
var _ resource.ResourceWithImportState = &routerInterfaceResource{}
