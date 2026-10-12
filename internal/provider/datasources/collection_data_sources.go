// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type collectionDataSource struct {
	config *shared.Config
	kind   string
}
type collectionDataSourceModel struct {
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	IDs              types.List   `tfsdk:"ids"`
	Names            types.Map    `tfsdk:"names"`
}

func NewCollectionDataSource(kind string) func() datasource.DataSource {
	return func() datasource.DataSource { return &collectionDataSource{kind: kind} }
}
func (r *collectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.kind
}
func (r *collectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *collectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Lists Nobus " + r.kind + " in a project and availability zone.", Attributes: map[string]schema.Attribute{"project_id": schema.StringAttribute{Optional: true, Computed: true}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "ids": schema.ListAttribute{Computed: true, ElementType: types.StringType}, "names": schema.MapAttribute{Computed: true, ElementType: types.StringType}}}
}
func (r *collectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var p collectionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &p)...)
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
	var ids []string
	names := map[string]string{}
	var err error
	add := func(id string, name *string) {
		ids = append(ids, id)
		if name != nil {
			names[id] = *name
		}
	}
	switch r.kind {
	case "networks":
		var v []client.Network
		v, err = c.ListNetworks(ctx, s.ProjectID, s.AvailabilityZone, 0, "")
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "subnets":
		var v []client.Subnet
		v, err = c.ListSubnets(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "ports":
		var v []client.Port
		v, err = c.ListPorts(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "routers":
		var v []client.Router
		v, err = c.ListRouters(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "floating_ips":
		var v []client.FloatingIP
		v, err = c.ListFloatingIPs(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].FloatingIPAddress)
		}
	case "security_groups":
		var v []client.SecurityGroup
		v, err = c.ListSecurityGroups(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "security_group_rules":
		var v []client.SecurityGroupRule
		v, err = c.ListSecurityGroupRules(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, nil)
		}
	case "trunks":
		var v []client.Trunk
		v, err = c.ListTrunks(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "qos_policies":
		var v []client.QosPolicy
		v, err = c.ListQosPolicies(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "firewall_groups":
		var v []client.FirewallGroup
		v, err = c.ListFirewallGroups(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "instances":
		var v []client.Instance
		v, err = c.ListInstances(ctx, s.ProjectID, s.AvailabilityZone, 0, "")
		for i := range v {
			name := v[i].Name
			add(v[i].ID, &name)
		}
	case "volumes":
	case "volume_snapshots":
		var v []client.Snapshot
		v, err = c.ListSnapshots(ctx, s.ProjectID, s.AvailabilityZone, nil)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "volume_backups":
		var v []client.Backup
		v, err = c.ListBackups(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "volume_transfers":
		var v []client.VolumeTransfer
		v, err = c.ListVolumeTransfers(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			add(v[i].ID, v[i].Name)
		}
	case "volume_availability_zones":
		var v []client.AvailabilityZone
		v, err = c.ListVolumeAvailabilityZones(ctx, s.ProjectID, s.AvailabilityZone)
		for i := range v {
			n := v[i].Name
			add(n, &n)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Nobus "+r.kind, shared.DiagnosticDetail(err))
		return
	}
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	x, d1 := types.ListValueFrom(ctx, types.StringType, ids)
	y, d2 := types.MapValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(d1...)
	resp.Diagnostics.Append(d2...)
	p.IDs = x
	p.Names = y
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
