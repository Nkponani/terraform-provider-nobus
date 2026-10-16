// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type clustersDataSource struct{ config *shared.Config }

type clustersDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	ResultCount      types.Int64  `tfsdk:"result_count"`
	IDs              types.List   `tfsdk:"ids"`
	Names            types.List   `tfsdk:"names"`
	ClustersJSON     types.String `tfsdk:"clusters_json"`
}

func NewClustersDataSource() datasource.DataSource { return &clustersDataSource{} }

func (d *clustersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_clusters"
}

func (d *clustersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}

func (d *clustersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Lists Nobus managed Kubernetes clusters.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "result_count": schema.Int64Attribute{Computed: true},
		"ids": schema.ListAttribute{Computed: true, ElementType: types.StringType}, "names": schema.ListAttribute{Computed: true, ElementType: types.StringType},
		"clusters_json": schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Complete cluster list encoded as JSON. Marked sensitive because node objects may contain access fields."},
	}}
}

func (d *clustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state clustersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	clusters, err := api.ListKubernetesClusters(ctx, scope.ProjectID, scope.AvailabilityZone)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Nobus Kubernetes clusters", shared.DiagnosticDetail(err))
		return
	}
	ids, names := make([]string, 0, len(clusters)), make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		ids = append(ids, cluster.ClusterID)
		names = append(names, cluster.ClusterName)
	}
	idValue, idDiags := types.ListValueFrom(ctx, types.StringType, ids)
	nameValue, nameDiags := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(idDiags...)
	resp.Diagnostics.Append(nameDiags...)
	raw, err := json.Marshal(clusters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to encode Kubernetes clusters", err.Error())
		return
	}
	state.ID = types.StringValue(scope.ProjectID + "/" + scope.AvailabilityZone)
	state.ProjectID = types.StringValue(scope.ProjectID)
	state.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	state.ResultCount = types.Int64Value(int64(len(clusters)))
	state.IDs, state.Names, state.ClustersJSON = idValue, nameValue, types.StringValue(string(raw))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type accessKind string

const (
	accessBastion    accessKind = "bastion"
	accessLB         accessKind = "load_balancer_login"
	accessMonitoring accessKind = "monitoring"
	accessNodes      accessKind = "nodes"
)

type accessDataSource struct {
	config *shared.Config
	kind   accessKind
}

func NewBastionDataSource() datasource.DataSource           { return &accessDataSource{kind: accessBastion} }
func NewLoadBalancerLoginDataSource() datasource.DataSource { return &accessDataSource{kind: accessLB} }
func NewMonitoringDataSource() datasource.DataSource {
	return &accessDataSource{kind: accessMonitoring}
}
func NewNodesDataSource() datasource.DataSource { return &accessDataSource{kind: accessNodes} }

func (d *accessDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_" + string(d.kind)
}

func (d *accessDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}

func (d *accessDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "cluster_id": schema.StringAttribute{Required: true},
	}
	switch d.kind {
	case accessBastion, accessLB:
		attributes["ip"] = schema.StringAttribute{Computed: true}
		attributes["username"] = schema.StringAttribute{Computed: true, Sensitive: true}
		attributes["password"] = schema.StringAttribute{Computed: true, Sensitive: true}
	case accessMonitoring:
		attributes["url"] = schema.StringAttribute{Computed: true}
		attributes["password"] = schema.StringAttribute{Computed: true, Sensitive: true}
	case accessNodes:
		attributes["cluster_name"] = schema.StringAttribute{Computed: true}
		attributes["masters_json"] = schema.StringAttribute{Computed: true, Sensitive: true}
		attributes["workers_json"] = schema.StringAttribute{Computed: true, Sensitive: true}
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Reads Kubernetes " + string(d.kind) + " details from Nobus.", Attributes: attributes}
}

func (d *accessDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var projectID, availabilityZone, clusterIDValue types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("availability_zone"), &availabilityZone)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("cluster_id"), &clusterIDValue)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(projectID, availabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	clusterID := clusterIDValue.ValueString()
	switch d.kind {
	case accessBastion:
		value, err := api.GetKubernetesBastion(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read Kubernetes bastion login", shared.DiagnosticDetail(err))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip"), stringValueOrNull(value.IP))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), stringValueOrNull(value.Username))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), stringValueOrNull(value.Password))...)
	case accessLB:
		value, err := api.GetKubernetesLoadBalancerLogin(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read Kubernetes load-balancer login", shared.DiagnosticDetail(err))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("ip"), stringValueOrNull(value.IP))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), stringValueOrNull(value.Username))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), stringValueOrNull(value.Password))...)
	case accessMonitoring:
		value, err := api.GetKubernetesMonitoring(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read Kubernetes monitoring details", shared.DiagnosticDetail(err))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("url"), stringValueOrNull(value.GrafanaURL))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), stringValueOrNull(value.GrafanaPassword))...)
	case accessNodes:
		value, err := api.GetKubernetesNodes(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read Kubernetes nodes", shared.DiagnosticDetail(err))
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), stringValueOrNull(value.ClusterName))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("masters_json"), rawJSONValue(value.Masters))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workers_json"), rawJSONValue(value.Workers))...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), clusterID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), clusterID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), scope.ProjectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("availability_zone"), scope.AvailabilityZone)...)
}

var _ datasource.DataSourceWithConfigure = &clustersDataSource{}
var _ datasource.DataSourceWithConfigure = &accessDataSource{}
