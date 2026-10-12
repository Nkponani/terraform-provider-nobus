// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type clusterResource struct{ config *shared.Config }

type clusterResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	AvailabilityZone    types.String `tfsdk:"availability_zone"`
	Name                types.String `tfsdk:"name"`
	KubernetesVersion   types.String `tfsdk:"kubernetes_version"`
	RouterID            types.String `tfsdk:"router_id"`
	MasterCount         types.Int64  `tfsdk:"master_count"`
	WorkerCount         types.Int64  `tfsdk:"worker_count"`
	EtcdVolumeSizeGB    types.Int64  `tfsdk:"etcd_volume_size_gb"`
	MasterFlavorID      types.String `tfsdk:"master_flavor_id"`
	WorkerFlavorID      types.String `tfsdk:"worker_flavor_id"`
	ImageID             types.String `tfsdk:"image_id"`
	TaskID              types.String `tfsdk:"task_id"`
	PodNetworkCIDR      types.String `tfsdk:"pod_network_cidr"`
	CNIPlugin           types.String `tfsdk:"cni_plugin"`
	Status              types.String `tfsdk:"status"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	HAProxyLBPublicIP   types.String `tfsdk:"haproxy_lb_public_ip"`
	HAProxyLBPrivateIP  types.String `tfsdk:"haproxy_lb_private_ip"`
	HAProxyLBInstanceID types.String `tfsdk:"haproxy_lb_instance_id"`
	BastionPublicIP     types.String `tfsdk:"bastion_public_ip"`
	BastionInstanceID   types.String `tfsdk:"bastion_instance_id"`
	HAProxyCPPrivateIP  types.String `tfsdk:"haproxy_cp_private_ip"`
	HAProxyCPInstanceID types.String `tfsdk:"haproxy_cp_instance_id"`
	KubeAPIVIP          types.String `tfsdk:"kube_api_vip"`
	NFSPrivateIP        types.String `tfsdk:"nfs_private_ip"`
	NFSInstanceID       types.String `tfsdk:"nfs_instance_id"`
	NetworkID           types.String `tfsdk:"network_id"`
	SubnetID            types.String `tfsdk:"subnet_id"`
	NodesJSON           types.String `tfsdk:"nodes_json"`
	MastersJSON         types.String `tfsdk:"masters_json"`
	WorkersJSON         types.String `tfsdk:"workers_json"`
	GrafanaURL          types.String `tfsdk:"grafana_url"`
}

func NewClusterResource() resource.Resource { return &clusterResource{} }

func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_cluster"
}

func (r *clusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *clusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus managed Kubernetes cluster. Worker/master counts, flavors, and etcd volume size are updated through the dedicated scaling APIs.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id":                     schema.StringAttribute{Computed: true},
		"name":                   schema.StringAttribute{Required: true, PlanModifiers: replace},
		"kubernetes_version":     schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("1.29.0"), PlanModifiers: replace},
		"router_id":              schema.StringAttribute{Optional: true, PlanModifiers: replace},
		"master_count":           schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(2)},
		"worker_count":           schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(2)},
		"etcd_volume_size_gb":    schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(50)},
		"master_flavor_id":       schema.StringAttribute{Optional: true, Computed: true},
		"worker_flavor_id":       schema.StringAttribute{Optional: true, Computed: true},
		"image_id":               schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace},
		"task_id":                schema.StringAttribute{Computed: true},
		"pod_network_cidr":       schema.StringAttribute{Computed: true},
		"cni_plugin":             schema.StringAttribute{Computed: true},
		"status":                 schema.StringAttribute{Computed: true},
		"created_at":             schema.StringAttribute{Computed: true},
		"updated_at":             schema.StringAttribute{Computed: true},
		"haproxy_lb_public_ip":   schema.StringAttribute{Computed: true},
		"haproxy_lb_private_ip":  schema.StringAttribute{Computed: true},
		"haproxy_lb_instance_id": schema.StringAttribute{Computed: true},
		"bastion_public_ip":      schema.StringAttribute{Computed: true},
		"bastion_instance_id":    schema.StringAttribute{Computed: true},
		"haproxy_cp_private_ip":  schema.StringAttribute{Computed: true},
		"haproxy_cp_instance_id": schema.StringAttribute{Computed: true},
		"kube_api_vip":           schema.StringAttribute{Computed: true},
		"nfs_private_ip":         schema.StringAttribute{Computed: true},
		"nfs_instance_id":        schema.StringAttribute{Computed: true},
		"network_id":             schema.StringAttribute{Computed: true},
		"subnet_id":              schema.StringAttribute{Computed: true},
		"nodes_json":             schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Raw node objects returned by the API."},
		"masters_json":           schema.StringAttribute{Computed: true, Sensitive: true},
		"workers_json":           schema.StringAttribute{Computed: true, Sensitive: true},
		"grafana_url":            schema.StringAttribute{Computed: true},
	})}
}

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	result, err := api.CreateKubernetesCluster(ctx, client.KubernetesClusterCreateRequest{
		ClusterName: plan.Name.ValueString(), AvailabilityZone: scope.AvailabilityZone, ProjectID: scope.ProjectID,
		KubernetesVersion: plan.KubernetesVersion.ValueString(), RouterID: plan.RouterID.ValueString(),
		MasterCount: plan.MasterCount.ValueInt64(), WorkerCount: plan.WorkerCount.ValueInt64(), EtcdVolumeSizeGB: plan.EtcdVolumeSizeGB.ValueInt64(),
		MasterFlavorID: plan.MasterFlavorID.ValueString(), WorkerFlavorID: plan.WorkerFlavorID.ValueString(), ImageID: plan.ImageID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Nobus Kubernetes cluster", shared.DiagnosticDetail(err))
		return
	}
	if result.Cluster.ClusterID == "" {
		resp.Diagnostics.AddError("Nobus Kubernetes API returned no cluster ID", "The create request succeeded but its response did not contain data.cluster_id, so Terraform cannot track the cluster.")
		return
	}
	plan.ID = types.StringValue(result.Cluster.ClusterID)
	plan.ProjectID = types.StringValue(scope.ProjectID)
	plan.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	plan.TaskID = stringValueOrNull(result.TaskID)
	applyCluster(&result.Cluster, &plan)
	if plan.MasterFlavorID.IsUnknown() {
		plan.MasterFlavorID = types.StringNull()
	}
	if plan.WorkerFlavorID.IsUnknown() {
		plan.WorkerFlavorID = types.StringNull()
	}
	if plan.ImageID.IsUnknown() {
		plan.ImageID = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	cluster, err := api.GetKubernetesCluster(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString())
	if client.IsNotFound(err) || cluster == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus Kubernetes cluster", shared.DiagnosticDetail(err))
		return
	}
	applyCluster(cluster, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	clusterID := state.ID.ValueString()
	mutations := []struct {
		changed bool
		name    string
		run     func() error
	}{
		{!plan.MasterCount.Equal(state.MasterCount), "master count", func() error {
			return api.ScaleKubernetesCluster(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID, "master", plan.MasterCount.ValueInt64())
		}},
		{!plan.WorkerCount.Equal(state.WorkerCount), "worker count", func() error {
			return api.ScaleKubernetesCluster(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID, "worker", plan.WorkerCount.ValueInt64())
		}},
		{!plan.MasterFlavorID.Equal(state.MasterFlavorID), "master flavor", func() error {
			return api.VerticallyScaleKubernetesCluster(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID, "master", plan.MasterFlavorID.ValueString(), nil)
		}},
		{!plan.WorkerFlavorID.Equal(state.WorkerFlavorID), "worker flavor", func() error {
			return api.VerticallyScaleKubernetesCluster(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID, "worker", plan.WorkerFlavorID.ValueString(), nil)
		}},
		{!plan.EtcdVolumeSizeGB.Equal(state.EtcdVolumeSizeGB), "etcd volume size", func() error {
			return api.ResizeKubernetesEtcdVolume(ctx, scope.ProjectID, scope.AvailabilityZone, clusterID, plan.EtcdVolumeSizeGB.ValueInt64())
		}},
	}
	for _, mutation := range mutations {
		if mutation.changed {
			if err := mutation.run(); err != nil {
				resp.Diagnostics.AddError("Unable to update Kubernetes "+mutation.name, shared.DiagnosticDetail(err))
				return
			}
		}
	}
	plan.ID = state.ID
	plan.TaskID = state.TaskID
	plan.PodNetworkCIDR = state.PodNetworkCIDR
	plan.CNIPlugin = state.CNIPlugin
	plan.Status = state.Status
	plan.CreatedAt = state.CreatedAt
	plan.UpdatedAt = state.UpdatedAt
	plan.HAProxyLBPublicIP = state.HAProxyLBPublicIP
	plan.HAProxyLBPrivateIP = state.HAProxyLBPrivateIP
	plan.HAProxyLBInstanceID = state.HAProxyLBInstanceID
	plan.BastionPublicIP = state.BastionPublicIP
	plan.BastionInstanceID = state.BastionInstanceID
	plan.HAProxyCPPrivateIP = state.HAProxyCPPrivateIP
	plan.HAProxyCPInstanceID = state.HAProxyCPInstanceID
	plan.KubeAPIVIP = state.KubeAPIVIP
	plan.NFSPrivateIP = state.NFSPrivateIP
	plan.NFSInstanceID = state.NFSInstanceID
	plan.NetworkID = state.NetworkID
	plan.SubnetID = state.SubnetID
	plan.NodesJSON = state.NodesJSON
	plan.MastersJSON = state.MastersJSON
	plan.WorkersJSON = state.WorkersJSON
	plan.GrafanaURL = state.GrafanaURL
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteKubernetesCluster(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Nobus Kubernetes cluster", shared.DiagnosticDetail(err))
	}
}

func (r *clusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func applyCluster(cluster *client.KubernetesCluster, model *clusterResourceModel) {
	model.ID = types.StringValue(cluster.ClusterID)
	if cluster.ClusterName != "" {
		model.Name = types.StringValue(cluster.ClusterName)
	}
	if cluster.KubernetesVersion != "" {
		model.KubernetesVersion = types.StringValue(cluster.KubernetesVersion)
	}
	if cluster.MasterCount != 0 {
		model.MasterCount = types.Int64Value(cluster.MasterCount)
	}
	if cluster.WorkerCount != 0 {
		model.WorkerCount = types.Int64Value(cluster.WorkerCount)
	}
	if cluster.EtcdVolumeSizeGB != 0 {
		model.EtcdVolumeSizeGB = types.Int64Value(cluster.EtcdVolumeSizeGB)
	}
	if cluster.MasterFlavorID != "" {
		model.MasterFlavorID = types.StringValue(cluster.MasterFlavorID)
	}
	if cluster.WorkerFlavorID != "" {
		model.WorkerFlavorID = types.StringValue(cluster.WorkerFlavorID)
	}
	if cluster.ImageID != "" {
		model.ImageID = types.StringValue(cluster.ImageID)
	}
	model.PodNetworkCIDR = stringValueOrNull(cluster.PodNetworkCIDR)
	model.CNIPlugin = stringValueOrNull(cluster.CNIPlugin)
	model.Status = stringValueOrNull(cluster.Status)
	model.CreatedAt = stringValueOrNull(cluster.CreatedAt)
	model.UpdatedAt = stringValueOrNull(cluster.UpdatedAt)
	model.HAProxyLBPublicIP = stringValueOrNull(cluster.HAProxyLBPublicIP)
	model.HAProxyLBPrivateIP = stringValueOrNull(cluster.HAProxyLBPrivateIP)
	model.HAProxyLBInstanceID = stringValueOrNull(cluster.HAProxyLBInstanceID)
	model.BastionPublicIP = stringValueOrNull(cluster.BastionPublicIP)
	model.BastionInstanceID = stringValueOrNull(cluster.BastionInstanceID)
	model.HAProxyCPPrivateIP = stringValueOrNull(cluster.HAProxyCPPrivateIP)
	model.HAProxyCPInstanceID = stringValueOrNull(cluster.HAProxyCPInstanceID)
	model.KubeAPIVIP = stringValueOrNull(cluster.KubeAPIVIP)
	model.NFSPrivateIP = stringValueOrNull(cluster.NFSPrivateIP)
	model.NFSInstanceID = stringValueOrNull(cluster.NFSInstanceID)
	model.NetworkID = stringValueOrNull(cluster.NetworkID)
	model.SubnetID = stringValueOrNull(cluster.SubnetID)
	model.NodesJSON = rawJSONValue(cluster.Nodes)
	model.MastersJSON = rawJSONValue(cluster.Masters)
	model.WorkersJSON = rawJSONValue(cluster.Workers)
	model.GrafanaURL = stringValueOrNull(cluster.GrafanaURL)
}

func stringValueOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func rawJSONValue(raw json.RawMessage) types.String {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return types.StringNull()
	}
	var compact bytes.Buffer
	if json.Compact(&compact, raw) == nil {
		return types.StringValue(compact.String())
	}
	return types.StringValue(string(raw))
}

var _ resource.ResourceWithConfigure = &clusterResource{}
var _ resource.ResourceWithImportState = &clusterResource{}
