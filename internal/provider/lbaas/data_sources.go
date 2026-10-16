// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lbaas

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type collectionDataSource struct {
	config     *shared.Config
	collection client.LBaaSCollection
	suffix     string
}

func NewLoadBalancersDataSource() datasource.DataSource {
	return &collectionDataSource{collection: client.LBaaSLoadBalancers, suffix: "load_balancers"}
}
func NewListenersDataSource() datasource.DataSource {
	return &collectionDataSource{collection: client.LBaaSListeners, suffix: "listeners"}
}
func NewPoolsDataSource() datasource.DataSource {
	return &collectionDataSource{collection: client.LBaaSPools, suffix: "pools"}
}
func NewMembersDataSource() datasource.DataSource {
	return &collectionDataSource{collection: client.LBaaSMembers, suffix: "members"}
}
func NewHealthMonitorsDataSource() datasource.DataSource {
	return &collectionDataSource{collection: client.LBaaSHealthMonitors, suffix: "health_monitors"}
}
func NewL7PoliciesDataSource() datasource.DataSource {
	return &collectionDataSource{collection: client.LBaaSL7Policies, suffix: "l7_policies"}
}
func NewL7RulesDataSource() datasource.DataSource {
	return &collectionDataSource{collection: client.LBaaSL7Rules, suffix: "l7_rules"}
}
func (d *collectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_" + d.suffix
}
func (d *collectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	d.config = config
}
func (d *collectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "page": schema.Int64Attribute{Optional: true}, "page_size": schema.Int64Attribute{Optional: true}, "result_count": schema.Int64Attribute{Computed: true}, "result_json": schema.StringAttribute{Computed: true}}
	switch d.collection {
	case client.LBaaSListeners:
		attributes["load_balancer_id"] = schema.StringAttribute{Optional: true}
	case client.LBaaSPools:
		attributes["load_balancer_id"] = schema.StringAttribute{Optional: true}
		attributes["listener_id"] = schema.StringAttribute{Optional: true}
	case client.LBaaSMembers:
		attributes["pool_id"] = schema.StringAttribute{Required: true}
	case client.LBaaSHealthMonitors:
		attributes["pool_id"] = schema.StringAttribute{Optional: true}
	case client.LBaaSL7Policies:
		attributes["listener_id"] = schema.StringAttribute{Optional: true}
	case client.LBaaSL7Rules:
		attributes["l7_policy_id"] = schema.StringAttribute{Required: true}
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Lists Nobus LBaaS " + d.suffix + ". The complete API array is exposed as JSON.", Attributes: attributes}
}
func (d *collectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var projectID, availabilityZone types.String
	var pageValue, pageSizeValue types.Int64
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("availability_zone"), &availabilityZone)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("page"), &pageValue)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("page_size"), &pageSizeValue)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(projectID, availabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	filters := map[string]string{}
	filterAttributes := map[string]string{}
	switch d.collection {
	case client.LBaaSListeners:
		filterAttributes["load_balancer_id"] = "loadbalancer_id"
	case client.LBaaSPools:
		filterAttributes["load_balancer_id"] = "loadbalancer_id"
		filterAttributes["listener_id"] = "listener_id"
	case client.LBaaSMembers, client.LBaaSHealthMonitors:
		filterAttributes["pool_id"] = "pool_id"
	case client.LBaaSL7Policies:
		filterAttributes["listener_id"] = "listener_id"
	case client.LBaaSL7Rules:
		filterAttributes["l7_policy_id"] = "l7policy_id"
	}
	for attribute, queryName := range filterAttributes {
		var value types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(attribute), &value)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attribute), value)...)
		if !value.IsNull() && !value.IsUnknown() {
			filters[queryName] = value.ValueString()
		}
	}
	if pageValue.ValueInt64() > 0 {
		filters["page"] = strconv.FormatInt(pageValue.ValueInt64(), 10)
	}
	if pageSizeValue.ValueInt64() > 0 {
		filters["page_size"] = strconv.FormatInt(pageSizeValue.ValueInt64(), 10)
	}
	raw, err := api.ListLBaaSResources(ctx, d.collection, scope.ProjectID, scope.AvailabilityZone, filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Nobus LBaaS "+d.suffix, shared.DiagnosticDetail(err))
		return
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		resp.Diagnostics.AddError("Unable to decode Nobus LBaaS list", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), scope.ProjectID+"/"+scope.AvailabilityZone+"/"+d.suffix)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), scope.ProjectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("availability_zone"), scope.AvailabilityZone)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("page"), pageValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("page_size"), pageSizeValue)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("result_count"), int64(len(values)))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("result_json"), string(raw))...)
}

type operationDataSource struct {
	config    *shared.Config
	operation string
	suffix    string
}

func NewTaskDataSource() datasource.DataSource {
	return &operationDataSource{operation: "task", suffix: "task"}
}
func NewDeploymentStatusDataSource() datasource.DataSource {
	return &operationDataSource{operation: "deployment_status", suffix: "deployment_status"}
}
func NewLoadBalancerStatusDataSource() datasource.DataSource {
	return &operationDataSource{operation: "loadbalancer_status", suffix: "load_balancer_status"}
}
func NewLoadBalancerStatsDataSource() datasource.DataSource {
	return &operationDataSource{operation: "loadbalancer_stats", suffix: "load_balancer_stats"}
}
func NewListenerStatsDataSource() datasource.DataSource {
	return &operationDataSource{operation: "listener_stats", suffix: "listener_stats"}
}
func (d *operationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_" + d.suffix
}
func (d *operationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	d.config = config
}
func (d *operationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "result_json": schema.StringAttribute{Computed: true}}
	switch d.operation {
	case "task":
		attributes["task_id"] = schema.StringAttribute{Required: true}
	case "listener_stats":
		attributes["listener_id"] = schema.StringAttribute{Required: true}
	default:
		attributes["load_balancer_id"] = schema.StringAttribute{Required: true}
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Reads Nobus LBaaS " + d.suffix + ".", Attributes: attributes}
}
func (d *operationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var projectID, availabilityZone types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("availability_zone"), &availabilityZone)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(projectID, availabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	query := map[string]string{}
	identifier := ""
	switch d.operation {
	case "task":
		var value types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("task_id"), &value)...)
		identifier = value.ValueString()
		query["task_id"] = identifier
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("task_id"), identifier)...)
	case "listener_stats":
		var value types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("listener_id"), &value)...)
		identifier = value.ValueString()
		query["listener_id"] = identifier
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("listener_id"), identifier)...)
	default:
		var value types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("load_balancer_id"), &value)...)
		identifier = value.ValueString()
		query["loadbalancer_id"] = identifier
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("load_balancer_id"), identifier)...)
	}
	raw, err := api.GetLBaaSOperation(ctx, d.operation, scope.ProjectID, scope.AvailabilityZone, query)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus LBaaS "+d.suffix, shared.DiagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identifier)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), scope.ProjectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("availability_zone"), scope.AvailabilityZone)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("result_json"), string(raw))...)
}

var _ datasource.DataSourceWithConfigure = &collectionDataSource{}
var _ datasource.DataSourceWithConfigure = &operationDataSource{}
