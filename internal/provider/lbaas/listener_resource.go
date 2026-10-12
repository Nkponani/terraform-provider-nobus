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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type listenerResource struct{ config *shared.Config }
type listenerResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	AvailabilityZone   types.String `tfsdk:"availability_zone"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	LoadBalancerID     types.String `tfsdk:"load_balancer_id"`
	Protocol           types.String `tfsdk:"protocol"`
	ProtocolPort       types.Int64  `tfsdk:"protocol_port"`
	DefaultPoolID      types.String `tfsdk:"default_pool_id"`
	ConnectionLimit    types.Int64  `tfsdk:"connection_limit"`
	AllowFrom          types.String `tfsdk:"allow_from"`
	AdminStateUp       types.Bool   `tfsdk:"admin_state_up"`
	Tags               types.List   `tfsdk:"tags"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	OperatingStatus    types.String `tfsdk:"operating_status"`
}
type listenerAPI struct {
	apiResource
	LoadBalancer    string  `json:"loadbalancer"`
	Protocol        string  `json:"protocol"`
	ProtocolPort    int64   `json:"protocol_port"`
	DefaultPool     *string `json:"default_pool"`
	ConnectionLimit int64   `json:"connection_limit"`
}

func NewListenerResource() resource.Resource { return &listenerResource{} }
func (r *listenerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_listener"
}
func (r *listenerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *listenerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A listener on a Nobus LBaaS load balancer.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true},
		"load_balancer_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "protocol": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()},
		"protocol_port": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "default_pool_id": schema.StringAttribute{Optional: true}, "connection_limit": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(-1)},
		"allow_from": schema.StringAttribute{Required: true}, "admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}, "tags": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		"provisioning_status": schema.StringAttribute{Computed: true}, "operating_status": schema.StringAttribute{Computed: true},
	})}
}
func (r *listenerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan listenerResourceModel
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
	body := map[string]any{"name": plan.Name.ValueString(), "description": optionalString(plan.Description), "loadbalancer_id": plan.LoadBalancerID.ValueString(), "protocol": plan.Protocol.ValueString(), "protocol_port": plan.ProtocolPort.ValueInt64(), "default_pool_id": optionalString(plan.DefaultPoolID), "connection_limit": plan.ConnectionLimit.ValueInt64(), "allow_from": plan.AllowFrom.ValueString(), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags}
	result, err := api.CreateLBaaSResource(ctx, client.LBaaSListeners, scope.ProjectID, scope.AvailabilityZone, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Nobus LBaaS listener", shared.DiagnosticDetail(err))
		return
	}
	var value listenerAPI
	if err := decodeResource(result.Resource, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS listener", err.Error())
		return
	}
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	applyListener(ctx, &value, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *listenerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state listenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	raw, err := api.GetLBaaSResource(ctx, client.LBaaSListeners, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read LBaaS listener", shared.DiagnosticDetail(err))
		return
	}
	var value listenerAPI
	if err := decodeResource(raw, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS listener", err.Error())
		return
	}
	applyListener(ctx, &value, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *listenerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state listenerResourceModel
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
	raw, err := api.UpdateLBaaSResource(ctx, client.LBaaSListeners, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil, map[string]any{"name": plan.Name.ValueString(), "description": optionalString(plan.Description), "default_pool_id": optionalString(plan.DefaultPoolID), "connection_limit": plan.ConnectionLimit.ValueInt64(), "allow_from": plan.AllowFrom.ValueString(), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update LBaaS listener", shared.DiagnosticDetail(err))
		return
	}
	var value listenerAPI
	if decodeResource(raw, &value) == nil {
		applyListener(ctx, &value, &plan, &resp.Diagnostics)
	} else {
		plan.ID = state.ID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *listenerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state listenerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSListeners, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LBaaS listener", shared.DiagnosticDetail(err))
	}
}
func (r *listenerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func applyListener(ctx context.Context, value *listenerAPI, state *listenerResourceModel, diagnostics *diag.Diagnostics) {
	state.ID = types.StringValue(value.ID)
	state.Name = stringPointerValue(value.Name)
	state.Description = stringPointerValue(value.Description)
	state.LoadBalancerID = plainStringValue(value.LoadBalancer)
	state.Protocol = plainStringValue(value.Protocol)
	state.ProtocolPort = types.Int64Value(value.ProtocolPort)
	state.DefaultPoolID = stringPointerValue(value.DefaultPool)
	state.ConnectionLimit = types.Int64Value(value.ConnectionLimit)
	state.AdminStateUp = types.BoolValue(value.AdminStateUp)
	state.ProvisioningStatus = plainStringValue(value.ProvisioningStatus)
	state.OperatingStatus = plainStringValue(value.OperatingStatus)
	tags, ds := listFromStrings(ctx, value.Tags)
	diagnostics.Append(ds...)
	state.Tags = tags
}

var _ resource.ResourceWithConfigure = &listenerResource{}
var _ resource.ResourceWithImportState = &listenerResource{}
