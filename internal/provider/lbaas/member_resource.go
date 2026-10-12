// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lbaas

import (
	"context"
	"strings"

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

type memberResource struct{ config *shared.Config }
type memberResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	AvailabilityZone   types.String `tfsdk:"availability_zone"`
	PoolID             types.String `tfsdk:"pool_id"`
	Name               types.String `tfsdk:"name"`
	Address            types.String `tfsdk:"address"`
	ProtocolPort       types.Int64  `tfsdk:"protocol_port"`
	SubnetID           types.String `tfsdk:"subnet_id"`
	Weight             types.Int64  `tfsdk:"weight"`
	Backup             types.Bool   `tfsdk:"backup"`
	MonitorAddress     types.String `tfsdk:"monitor_address"`
	MonitorPort        types.Int64  `tfsdk:"monitor_port"`
	AdminStateUp       types.Bool   `tfsdk:"admin_state_up"`
	Tags               types.List   `tfsdk:"tags"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	OperatingStatus    types.String `tfsdk:"operating_status"`
}
type memberAPI struct {
	apiResource
	Pool           string  `json:"pool"`
	Address        string  `json:"address"`
	ProtocolPort   int64   `json:"protocol_port"`
	SubnetID       *string `json:"subnet_id"`
	Weight         int64   `json:"weight"`
	Backup         bool    `json:"backup"`
	MonitorAddress *string `json:"monitor_address"`
	MonitorPort    *int64  `json:"monitor_port"`
}

func NewMemberResource() resource.Resource { return &memberResource{} }
func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_member"
}
func (r *memberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A backend member of a Nobus LBaaS pool.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "pool_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "name": schema.StringAttribute{Optional: true},
		"address": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "protocol_port": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "subnet_id": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()},
		"weight": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1)}, "backup": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"monitor_address": schema.StringAttribute{Optional: true}, "monitor_port": schema.Int64Attribute{Optional: true}, "admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}, "tags": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		"provisioning_status": schema.StringAttribute{Computed: true}, "operating_status": schema.StringAttribute{Computed: true},
	})}
}
func (r *memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan memberResourceModel
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
	body := map[string]any{"name": optionalString(plan.Name), "pool_id": plan.PoolID.ValueString(), "address": plan.Address.ValueString(), "protocol_port": plan.ProtocolPort.ValueInt64(), "subnet_id": optionalString(plan.SubnetID), "weight": plan.Weight.ValueInt64(), "backup": plan.Backup.ValueBool(), "monitor_address": optionalString(plan.MonitorAddress), "monitor_port": nullableInt(plan.MonitorPort), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags}
	result, err := api.CreateLBaaSResource(ctx, client.LBaaSMembers, scope.ProjectID, scope.AvailabilityZone, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LBaaS member", shared.DiagnosticDetail(err))
		return
	}
	var value memberAPI
	if err := decodeResource(result.Resource, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS member", err.Error())
		return
	}
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	applyMember(ctx, &value, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *memberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state memberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	parents := map[string]string{"pool_id": state.PoolID.ValueString()}
	raw, err := api.GetLBaaSResource(ctx, client.LBaaSMembers, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), parents)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read LBaaS member", shared.DiagnosticDetail(err))
		return
	}
	var value memberAPI
	if err := decodeResource(raw, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS member", err.Error())
		return
	}
	applyMember(ctx, &value, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *memberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state memberResourceModel
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
	parents := map[string]string{"pool_id": plan.PoolID.ValueString()}
	raw, err := api.UpdateLBaaSResource(ctx, client.LBaaSMembers, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), parents, map[string]any{"name": optionalString(plan.Name), "weight": plan.Weight.ValueInt64(), "backup": plan.Backup.ValueBool(), "monitor_address": optionalString(plan.MonitorAddress), "monitor_port": nullableInt(plan.MonitorPort), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update LBaaS member", shared.DiagnosticDetail(err))
		return
	}
	var value memberAPI
	if decodeResource(raw, &value) == nil {
		applyMember(ctx, &value, &plan, &resp.Diagnostics)
	} else {
		plan.ID = state.ID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *memberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state memberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSMembers, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), map[string]string{"pool_id": state.PoolID.ValueString()}); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LBaaS member", shared.DiagnosticDetail(err))
	}
}
func (r *memberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid LBaaS member import ID", "Use POOL_ID/MEMBER_ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pool_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
func applyMember(ctx context.Context, value *memberAPI, state *memberResourceModel, diagnostics *diag.Diagnostics) {
	state.ID = types.StringValue(value.ID)
	state.PoolID = plainStringValue(value.Pool)
	state.Name = stringPointerValue(value.Name)
	state.Address = plainStringValue(value.Address)
	state.ProtocolPort = types.Int64Value(value.ProtocolPort)
	state.SubnetID = stringPointerValue(value.SubnetID)
	state.Weight = types.Int64Value(value.Weight)
	state.Backup = types.BoolValue(value.Backup)
	state.MonitorAddress = stringPointerValue(value.MonitorAddress)
	if value.MonitorPort == nil {
		state.MonitorPort = types.Int64Null()
	} else {
		state.MonitorPort = types.Int64Value(*value.MonitorPort)
	}
	state.AdminStateUp = types.BoolValue(value.AdminStateUp)
	state.ProvisioningStatus = plainStringValue(value.ProvisioningStatus)
	state.OperatingStatus = plainStringValue(value.OperatingStatus)
	tags, ds := listFromStrings(ctx, value.Tags)
	diagnostics.Append(ds...)
	state.Tags = tags
}
func nullableInt(value types.Int64) any {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	return value.ValueInt64()
}

var _ resource.ResourceWithConfigure = &memberResource{}
var _ resource.ResourceWithImportState = &memberResource{}
