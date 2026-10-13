// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

var (
	_ resource.Resource              = &instanceFromVolumeResource{}
	_ resource.ResourceWithConfigure = &instanceFromVolumeResource{}
)

func NewInstanceFromVolumeResource() resource.Resource {
	return &instanceFromVolumeResource{}
}

type instanceFromVolumeResource struct {
	config *providerConfig
}

type instanceFromVolumeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Image            types.String `tfsdk:"image"`
	Flavor           types.String `tfsdk:"flavor"`
	KeyName          types.String `tfsdk:"key_name"`
	VolumeID         types.String `tfsdk:"volume_id"`
	VolumeSize       types.Int64  `tfsdk:"volume_size"`
	NetworkID        types.String `tfsdk:"network_id"`
	SecurityGroups   types.List   `tfsdk:"security_groups"`
	UserData         types.String `tfsdk:"user_data"`
	DeleteVolume     types.Bool   `tfsdk:"delete_volume_on_destroy"`
}

func (r *instanceFromVolumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_from_volume"
}

func (r *instanceFromVolumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Create a Nobus instance from an existing volume.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Nobus instance ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"image": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"flavor": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"volume_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"volume_size": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64RequiresReplace()},
			},
			"network_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"security_groups": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.List{listRequiresReplace()},
			},
			"user_data": schema.StringAttribute{
				Optional:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"delete_volume_on_destroy": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (r *instanceFromVolumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}

func (r *instanceFromVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceFromVolumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiClient := r.config.Client
	if apiClient == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client is not configured.")
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	var securityGroups []string
	if !plan.SecurityGroups.IsNull() && !plan.SecurityGroups.IsUnknown() {
		resp.Diagnostics.Append(plan.SecurityGroups.ElementsAs(ctx, &securityGroups, false)...)
	}
	sgPtr := &securityGroups
	if len(securityGroups) == 0 {
		sgPtr = nil
	}

	createReq := client.CreateInstanceFromVolumeRequest{
		ProjectID:        scope.ProjectID,
		AvailabilityZone: scope.AvailabilityZone,
		Name:             plan.Name.ValueString(),
		Image:            plan.Image.ValueString(),
		Flavor:           plan.Flavor.ValueString(),
		KeyName:          plan.KeyName.ValueString(),
		VolumeID:         plan.VolumeID.ValueString(),
		NetworkID:        plan.NetworkID.ValueStringPointer(),
		SecurityGroups:   sgPtr,
		Userdata:         plan.UserData.ValueStringPointer(),
	}
	if !plan.VolumeSize.IsNull() && !plan.VolumeSize.IsUnknown() {
		v := plan.VolumeSize.ValueInt64()
		createReq.VolumeSize = &v
	}

	inst, err := apiClient.CreateInstanceFromVolume(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance from volume", err.Error())
		return
	}

	plan.ID = types.StringValue(inst.ID)
	plan.ProjectID = types.StringValue(scope.ProjectID)
	plan.AvailabilityZone = types.StringValue(scope.AvailabilityZone)

	if !plan.SecurityGroups.IsNull() || sgPtr != nil {
		if sgPtr == nil {
			plan.SecurityGroups = types.ListNull(types.StringType)
		} else {
			sgs, diags := types.ListValueFrom(ctx, types.StringType, *sgPtr)
			resp.Diagnostics.Append(diags...)
			plan.SecurityGroups = sgs
		}
	}
	if plan.VolumeSize.IsNull() || plan.VolumeSize.IsUnknown() {
		plan.VolumeSize = types.Int64Value(DefaultInstanceVolumeSize)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceFromVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceFromVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiClient := r.config.Client
	if apiClient == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client is not configured.")
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	inst, err := apiClient.GetInstance(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read instance", err.Error())
		return
	}

	state.ID = types.StringValue(inst.ID)
	state.ProjectID = types.StringValue(scope.ProjectID)
	state.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	state.Name = types.StringValue(inst.Name)
	if inst.Flavor != nil {
		state.Flavor = objectFieldAsString(inst.Flavor, "name")
	}
	if inst.Image != nil {
		state.Image = objectFieldAsString(inst.Image, "name")
	}
	state.KeyName = types.StringPointerValue(inst.KeyName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *instanceFromVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Updating nobus_instance_from_volume is not supported; replace the resource instead.")
}

func (r *instanceFromVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceFromVolumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiClient := r.config.Client
	if apiClient == nil {
		resp.Diagnostics.AddError("Client not configured", "The provider client is not configured.")
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	deleteVolumes := true
	if !state.DeleteVolume.IsNull() && !state.DeleteVolume.IsUnknown() {
		deleteVolumes = state.DeleteVolume.ValueBool()
	}

	if err := apiClient.DeleteInstance(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), deleteVolumes); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}

	if err := apiClient.WaitForInstanceDeleted(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), shared.PollInterval); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Failed to wait for instance deletion", err.Error())
			return
		}
	}
}
