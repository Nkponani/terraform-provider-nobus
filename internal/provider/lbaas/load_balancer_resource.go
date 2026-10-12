// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type loadBalancerResource struct{ config *shared.Config }
type loadBalancerResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProjectID            types.String `tfsdk:"project_id"`
	AvailabilityZone     types.String `tfsdk:"availability_zone"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	RouterID             types.String `tfsdk:"router_id"`
	LoadBalancerProvider types.String `tfsdk:"load_balancer_provider"`
	Flavor               types.String `tfsdk:"flavor"`
	VolumeSize           types.Int64  `tfsdk:"volume_size"`
	AdminStateUp         types.Bool   `tfsdk:"admin_state_up"`
	Tags                 types.List   `tfsdk:"tags"`
	SecurityGroupNames   types.List   `tfsdk:"security_group_names"`
	TaskID               types.String `tfsdk:"task_id"`
	ProvisioningStatus   types.String `tfsdk:"provisioning_status"`
	OperatingStatus      types.String `tfsdk:"operating_status"`
	VIPAddress           types.String `tfsdk:"vip_address"`
	VIPSubnetID          types.String `tfsdk:"vip_subnet_id"`
	VIPNetworkID         types.String `tfsdk:"vip_network_id"`
	VIPPortID            types.String `tfsdk:"vip_port_id"`
	Hostname             types.String `tfsdk:"hostname"`
	ManagementIP         types.String `tfsdk:"management_ip"`
	InstanceID           types.String `tfsdk:"instance_id"`
	SecurityGroupID      types.String `tfsdk:"security_group_id"`
}
type loadBalancerAPI struct {
	apiResource
	Provider        string  `json:"provider"`
	Flavor          *string `json:"flavor"`
	VIPAddress      *string `json:"vip_address"`
	VIPSubnetID     *string `json:"vip_subnet_id"`
	VIPNetworkID    *string `json:"vip_network_id"`
	VIPPortID       *string `json:"vip_port_id"`
	Hostname        *string `json:"hostname"`
	ManagementIP    *string `json:"management_ip"`
	InstanceID      *string `json:"instance_id"`
	SecurityGroupID *string `json:"security_group_id"`
}

func NewLoadBalancerResource() resource.Resource { return &loadBalancerResource{} }
func (r *loadBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_load_balancer"
}
func (r *loadBalancerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}
func (r *loadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus LBaaS load balancer.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true},
		"router_id": schema.StringAttribute{Optional: true, PlanModifiers: replace}, "load_balancer_provider": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("haproxy"), PlanModifiers: replace, MarkdownDescription: "LBaaS backend provider sent as the API `provider` field."},
		"flavor": schema.StringAttribute{Required: true, PlanModifiers: replace}, "volume_size": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(20), PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
		"admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}, "tags": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		"security_group_names": schema.ListAttribute{Optional: true, ElementType: types.StringType}, "task_id": schema.StringAttribute{Computed: true},
		"provisioning_status": schema.StringAttribute{Computed: true}, "operating_status": schema.StringAttribute{Computed: true}, "vip_address": schema.StringAttribute{Computed: true},
		"vip_subnet_id": schema.StringAttribute{Computed: true}, "vip_network_id": schema.StringAttribute{Computed: true}, "vip_port_id": schema.StringAttribute{Computed: true},
		"hostname": schema.StringAttribute{Computed: true}, "management_ip": schema.StringAttribute{Computed: true}, "instance_id": schema.StringAttribute{Computed: true}, "security_group_id": schema.StringAttribute{Computed: true},
	})}
}
func (r *loadBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan loadBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	tags, ds := stringsFromList(ctx, plan.Tags)
	resp.Diagnostics.Append(ds...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{"name": plan.Name.ValueString(), "description": optionalString(plan.Description), "router_id": optionalString(plan.RouterID), "provider": plan.LoadBalancerProvider.ValueString(), "availability_zone": scope.AvailabilityZone, "flavor": plan.Flavor.ValueString(), "project_id": scope.ProjectID, "volume_size": plan.VolumeSize.ValueInt64(), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags}
	result, err := api.CreateLBaaSResource(ctx, client.LBaaSLoadBalancers, scope.ProjectID, scope.AvailabilityZone, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Nobus load balancer", shared.DiagnosticDetail(err))
		return
	}
	var value loadBalancerAPI
	if err := decodeResource(result.Resource, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode Nobus load balancer", err.Error())
		return
	}
	if value.ID == "" {
		resp.Diagnostics.AddError("Nobus LBaaS API returned no load balancer ID", "The create response cannot be tracked by Terraform.")
		return
	}
	plan.ProjectID, plan.AvailabilityZone, plan.TaskID = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone), plainStringValue(result.TaskID)
	applyLoadBalancer(ctx, &value, &plan, &resp.Diagnostics)
	if !plan.SecurityGroupNames.IsNull() && !plan.SecurityGroupNames.IsUnknown() {
		names, diags := stringsFromList(ctx, plan.SecurityGroupNames)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			if err := api.UpdateLBaaSSecurityGroups(ctx, scope.ProjectID, scope.AvailabilityZone, value.ID, names); err != nil {
				resp.Diagnostics.AddError("Unable to set load balancer security groups", shared.DiagnosticDetail(err))
			}
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *loadBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state loadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	raw, err := api.GetLBaaSResource(ctx, client.LBaaSLoadBalancers, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus load balancer", shared.DiagnosticDetail(err))
		return
	}
	var value loadBalancerAPI
	if err := decodeResource(raw, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode Nobus load balancer", err.Error())
		return
	}
	applyLoadBalancer(ctx, &value, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *loadBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state loadBalancerResourceModel
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
	tags, ds := stringsFromList(ctx, plan.Tags)
	resp.Diagnostics.Append(ds...)
	raw, err := api.UpdateLBaaSResource(ctx, client.LBaaSLoadBalancers, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil, map[string]any{"name": plan.Name.ValueString(), "description": optionalString(plan.Description), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Nobus load balancer", shared.DiagnosticDetail(err))
		return
	}
	if !plan.SecurityGroupNames.Equal(state.SecurityGroupNames) {
		names, diags := stringsFromList(ctx, plan.SecurityGroupNames)
		resp.Diagnostics.Append(diags...)
		if err := api.UpdateLBaaSSecurityGroups(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), names); err != nil {
			resp.Diagnostics.AddError("Unable to update load balancer security groups", shared.DiagnosticDetail(err))
			return
		}
	}
	var value loadBalancerAPI
	if decodeResource(raw, &value) == nil {
		applyLoadBalancer(ctx, &value, &plan, &resp.Diagnostics)
	} else {
		plan.ID = state.ID
	}
	plan.TaskID = state.TaskID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *loadBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state loadBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSLoadBalancers, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Nobus load balancer", shared.DiagnosticDetail(err))
	}
}
func (r *loadBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func applyLoadBalancer(ctx context.Context, value *loadBalancerAPI, state *loadBalancerResourceModel, diagnostics *diag.Diagnostics) {
	state.ID = types.StringValue(value.ID)
	state.Name = stringPointerValue(value.Name)
	state.Description = stringPointerValue(value.Description)
	state.AdminStateUp = types.BoolValue(value.AdminStateUp)
	state.ProvisioningStatus = plainStringValue(value.ProvisioningStatus)
	state.OperatingStatus = plainStringValue(value.OperatingStatus)
	tags, ds := listFromStrings(ctx, value.Tags)
	diagnostics.Append(ds...)
	state.Tags = tags
	state.VIPAddress = stringPointerValue(value.VIPAddress)
	state.VIPSubnetID = stringPointerValue(value.VIPSubnetID)
	state.VIPNetworkID = stringPointerValue(value.VIPNetworkID)
	state.VIPPortID = stringPointerValue(value.VIPPortID)
	state.Hostname = stringPointerValue(value.Hostname)
	state.ManagementIP = stringPointerValue(value.ManagementIP)
	state.InstanceID = stringPointerValue(value.InstanceID)
	state.SecurityGroupID = stringPointerValue(value.SecurityGroupID)
	state.LoadBalancerProvider = plainStringValue(value.Provider)
	if value.Flavor != nil {
		state.Flavor = stringPointerValue(value.Flavor)
	}
}

var _ resource.ResourceWithConfigure = &loadBalancerResource{}
var _ resource.ResourceWithImportState = &loadBalancerResource{}
