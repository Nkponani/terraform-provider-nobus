// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

// Ensure the resource implements the expected interfaces.
var (
	_ resource.Resource                = &instanceResource{}
	_ resource.ResourceWithConfigure   = &instanceResource{}
	_ resource.ResourceWithImportState = &instanceResource{}
)

// NewInstanceResource returns a new nobus_instance resource.
func NewInstanceResource() resource.Resource {
	return &instanceResource{}
}

// instanceResource manages a Nobus instance.
//
// API mapping, verified against the OpenAPI specification:
//
//	Create: POST   /api/v3/instance             CreateInstanceRequestSchema -> GetInstanceResponseSchema
//	Read:   GET    /api/v3/instance             ?project_id&availability_zone&server_id -> GetInstanceResponseSchema
//	Update: POST   /api/v3/instance/update-server UpdateInstanceRequestSchema -> ActionMessageResponseSchema
//	Delete: DELETE /api/v3/instance             ?project_id&availability_zone&server_id&delete_volume -> DeleteInstanceResponseSchema
//
// A resize is available through POST /api/v3/instance/resize, but the API also
// requires an explicit POST /api/v3/instance/confirm-resize to make it
// permanent. The provider therefore treats the flavor as immutable so that
// Terraform never leaves an instance pending confirmation.
type instanceResource struct {
	config *providerConfig
}

// instanceResourceModel maps the Terraform schema onto the Nobus API.
type instanceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	AvailabilityZone    types.String `tfsdk:"availability_zone"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Hostname            types.String `tfsdk:"hostname"`
	Image               types.String `tfsdk:"image"`
	Flavor              types.String `tfsdk:"flavor"`
	KeyName             types.String `tfsdk:"key_name"`
	NetworkID           types.String `tfsdk:"network_id"`
	VolumeSize          types.Int64  `tfsdk:"volume_size"`
	SecurityGroups      types.List   `tfsdk:"security_groups"`
	AdminPass           types.String `tfsdk:"admin_pass"`
	UserData            types.String `tfsdk:"user_data"`
	DeleteVolume        types.Bool   `tfsdk:"delete_volume_on_destroy"`
	InstanceReadyStates types.List   `tfsdk:"instance_ready_states"`

	Status     types.String `tfsdk:"status"`
	Addresses  types.String `tfsdk:"addresses"`
	VolumeIDs  types.List   `tfsdk:"volume_ids"`
	FlavorID   types.String `tfsdk:"flavor_id"`
	FlavorName types.String `tfsdk:"flavor_name"`
	ImageID    types.String `tfsdk:"image_id"`
	ImageName  types.String `tfsdk:"image_name"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (r *instanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *instanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Nobus instance.\n\n" +
			"The Nobus API requires a project, an availability zone, an image, a flavor and a key name to " +
			"create an instance. After creation the provider polls the instance status reported by Nobus " +
			"until it reaches a running state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Nobus instance ID (`server_id`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Nobus project that owns the instance. Defaults to the provider `project_id`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"availability_zone": schema.StringAttribute{
				MarkdownDescription: "Availability zone hosting the instance. Defaults to the provider `availability_zone`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the instance. Updatable in place.",
				Required:            true,
				Validators:          []validator.String{StringNotEmpty()},
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Image to boot the instance from, as an image ID or name. " +
					"Required by the Nobus API when creating an instance and never returned by it, so it " +
					"cannot be verified after an import. Changing this forces a new instance.",
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{StringNotEmpty()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.StringAttribute{
				MarkdownDescription: "Flavor (instance size) to create the instance with, as a flavor ID or name. " +
					"Required by the Nobus API when creating an instance and never returned by it, so it cannot " +
					"be verified after an import. Changing this forces a new instance, because the Nobus resize " +
					"operation requires a separate confirmation step that Terraform does not perform automatically.",
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{StringNotEmpty()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_name": schema.StringAttribute{
				MarkdownDescription: "Name of the keypair to install on the instance. Required by the Nobus API " +
					"when creating an instance and never returned by it, so it cannot be verified after an " +
					"import. Changing this forces a new instance.",
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{StringNotEmpty()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "ID of the network to attach the instance to, for example " +
					"`nobus_network.app.id`. Changing this forces a new instance.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"volume_size": schema.Int64Attribute{
				MarkdownDescription: "Size in GiB of the boot volume created with the instance. " +
					"Defaults to `50`, which is the documented API default. Never returned by the API, so it " +
					"cannot be verified after an import. Changing this forces a new instance.",
				Optional:   true,
				Validators: []validator.Int64{Int64AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{
					int64RequiresReplace(),
				},
			},
			"security_groups": schema.ListAttribute{
				MarkdownDescription: "Names of the security groups to assign to the instance. " +
					"Defaults to `[\"default\"]`, the documented API default. The API returns security groups " +
					"as objects rather than names, so this cannot be verified after an import. Changing this " +
					"forces a new instance.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listRequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the instance. Updatable in place. " +
					"Not returned by the instance read operation, so it is only sent to the API.",
				Optional: true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname of the instance. Updatable in place. " +
					"Not returned by the instance read operation, so it is only sent to the API.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"admin_pass": schema.StringAttribute{
				MarkdownDescription: "Administrative password for the instance. Write-only: the Nobus API never " +
					"returns it. Changing this forces a new instance.",
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_data": schema.StringAttribute{
				MarkdownDescription: "User data passed to the instance on first boot. Write-only: the Nobus API " +
					"never returns it. Changing this forces a new instance.",
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"delete_volume_on_destroy": schema.BoolAttribute{
				MarkdownDescription: "Whether the boot volume created with the instance is deleted when the " +
					"instance is destroyed. This maps to the `delete_volume` query parameter, which defaults to " +
					"`true` in the Nobus API.",
				Optional: true,
			},
			"instance_ready_states": schema.ListAttribute{
				MarkdownDescription: "Instance states accepted as ready when waiting after a create. " +
					"`InstanceSchema.status` is documented only as a free-form string with no enum, so the " +
					"defaults to `[\"ACTIVE\", \"active\"]`. Override this if your Nobus deployment reports " +
					"different values.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Instance status reported by Nobus.",
				Computed:            true,
			},
			"addresses": schema.StringAttribute{
				MarkdownDescription: "Addresses assigned to the instance as a JSON object, exactly as reported by " +
					"Nobus. The API documents this member only as an untyped object, so it is exposed as JSON " +
					"rather than assuming a nested structure. Use `jsondecode()` to read it.",
				Computed: true,
			},
			"volume_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of the volumes attached to the instance.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"flavor_id": schema.StringAttribute{
				MarkdownDescription: "Flavor ID reported by Nobus for the instance.",
				Computed:            true,
			},
			"flavor_name": schema.StringAttribute{
				MarkdownDescription: "Flavor name reported by Nobus for the instance.",
				Computed:            true,
			},
			"image_id": schema.StringAttribute{
				MarkdownDescription: "Image ID reported by Nobus for the instance.",
				Computed:            true,
			},
			"image_name": schema.StringAttribute{
				MarkdownDescription: "Image name reported by Nobus for the instance.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp reported by Nobus.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Last modification timestamp reported by Nobus.",
				Computed:            true,
			},
		},
	}
}

func (r *instanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan instanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.resolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	// The API requires an image, a flavor and a key name to create an instance.
	// They are never returned on read, so they are validated here rather than
	// being modelled as Required, which would force a replacement after an import.
	for attribute, value := range map[string]types.String{
		"image":    plan.Image,
		"flavor":   plan.Flavor,
		"key_name": plan.KeyName,
	} {
		if nullSafeString(value) == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root(attribute),
				fmt.Sprintf("Missing required %q", attribute),
				fmt.Sprintf(
					"The Nobus API requires %q when creating an instance, but it is not set. Add it to the "+
						"resource, or remove it from configuration if this instance was imported and you do "+
						"not want to replace it.",
					attribute,
				),
			)
			return
		}
	}

	securityGroups, sgDiags := listToStringSlice(ctx, plan.SecurityGroups)
	resp.Diagnostics.Append(sgDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readyStates, stateDiags := listToStringSlice(ctx, plan.InstanceReadyStates)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateInstanceRequest{
		ProjectID:        resolved.ProjectID,
		AvailabilityZone: resolved.AvailabilityZone,
		Name:             plan.Name.ValueString(),
		Image:            plan.Image.ValueString(),
		Flavor:           plan.Flavor.ValueString(),
		KeyName:          plan.KeyName.ValueString(),
		Nics:             "auto",
		NetworkID:        optionalString(plan.NetworkID),
		VolumeSize:       optionalInt64(plan.VolumeSize),
		AdminPass:        optionalString(plan.AdminPass),
		Userdata:         optionalString(plan.UserData),
	}
	if len(securityGroups) > 0 {
		createReq.SecurityGroups = securityGroups
	}

	created, err := apiClient.CreateInstance(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to create Nobus instance %q", plan.Name.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.ProjectID = types.StringValue(resolved.ProjectID)
	plan.AvailabilityZone = types.StringValue(resolved.AvailabilityZone)

	// Description and hostname are updatable but are not members of the create
	// model, so they are applied through the documented update operation.
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		if err := apiClient.UpdateInstance(ctx, client.UpdateInstanceRequest{
			ProjectID:        resolved.ProjectID,
			AvailabilityZone: resolved.AvailabilityZone,
			ServerID:         created.ID,
			Description:      optionalString(plan.Description),
			Hostname:         optionalString(plan.Hostname),
		}); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to configure Nobus instance %q", plan.Name.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
	}

	// security_groups and volume_size are Optional+Computed. When the
	// configuration omits them the plan value is unknown, and Terraform requires
	// every value to be known after apply, so the values actually sent to the API
	// are recorded here.
	if len(securityGroups) > 0 {
		var sgValueDiags diag.Diagnostics
		plan.SecurityGroups, sgValueDiags = types.ListValueFrom(ctx, types.StringType, securityGroups)
		resp.Diagnostics.Append(sgValueDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	appliedVolumeSize := int64(DefaultInstanceVolumeSize)
	if !plan.VolumeSize.IsNull() && !plan.VolumeSize.IsUnknown() {
		appliedVolumeSize = plan.VolumeSize.ValueInt64()
	}
	createReq.VolumeSize = &appliedVolumeSize
	if plan.VolumeSize.IsNull() || plan.VolumeSize.IsUnknown() {
		// Leave it null in state: the API never reports the boot volume size.
		plan.VolumeSize = types.Int64Null()
	}

	// Creation is asynchronous, so wait for the status Nobus reports.
	ready, err := apiClient.WaitForInstanceReady(
		ctx, resolved.ProjectID, resolved.AvailabilityZone, created.ID,
		readyStates, pollInterval,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Nobus instance %q did not reach a running state", created.ID),
			diagnosticDetail(err),
		)
		return
	}

	applyInstanceToModel(ctx, ready, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state instanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.resolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	instance, err := apiClient.GetInstance(ctx, resolved.ProjectID, resolved.AvailabilityZone, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// The instance was removed outside Terraform; drop it from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Nobus instance %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	applyInstanceToModel(ctx, instance, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state instanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.resolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	projectID, availabilityZone := resolved.ProjectID, resolved.AvailabilityZone

	// The update model supports name, description and hostname only.
	updateReq := client.UpdateInstanceRequest{
		ProjectID:        projectID,
		AvailabilityZone: availabilityZone,
		ServerID:         state.ID.ValueString(),
	}
	if !plan.Name.Equal(state.Name) {
		updateReq.Name = optionalString(plan.Name)
	}
	// description is write-only, so the prior value cannot be compared. Send it
	// whenever it is present in the configuration.
	updateReq.Description = optionalString(plan.Description)
	if !plan.Hostname.Equal(state.Hostname) {
		updateReq.Hostname = optionalString(plan.Hostname)
	}

	if updateReq.Name != nil || updateReq.Description != nil || updateReq.Hostname != nil {
		if err := apiClient.UpdateInstance(ctx, updateReq); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to update Nobus instance %q", state.ID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
	}

	readyStates, stateDiags := listToStringSlice(ctx, state.InstanceReadyStates)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Confirm the instance settles in a running state before recording it.
	instance, err := apiClient.WaitForInstanceReady(
		ctx, projectID, availabilityZone, state.ID.ValueString(),
		readyStates, pollInterval,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Nobus instance %q did not settle after the update", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	// Write-only attributes must be carried over from state because the API
	// never returns them.
	plan.ID = state.ID
	plan.AdminPass = state.AdminPass
	plan.UserData = state.UserData
	plan.DeleteVolume = state.DeleteVolume
	plan.InstanceReadyStates = state.InstanceReadyStates
	plan.ProjectID = types.StringValue(projectID)
	plan.AvailabilityZone = types.StringValue(availabilityZone)

	applyInstanceToModel(ctx, instance, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state instanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.resolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	projectID, availabilityZone := resolved.ProjectID, resolved.AvailabilityZone
	serverID := state.ID.ValueString()

	deleteVolumes := true
	if !state.DeleteVolume.IsNull() && !state.DeleteVolume.IsUnknown() {
		deleteVolumes = state.DeleteVolume.ValueBool()
	}

	if err := apiClient.DeleteInstance(ctx, projectID, availabilityZone, serverID, deleteVolumes); err != nil {
		if client.IsNotFound(err) {
			// Already gone; deleting is idempotent.
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Nobus instance %q", serverID),
			diagnosticDetail(err),
		)
		return
	}

	if err := apiClient.WaitForInstanceDeleted(ctx, projectID, availabilityZone, serverID, pollInterval); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Nobus instance %q was not deleted", serverID),
			diagnosticDetail(err),
		)
	}
}

// ImportState imports an existing instance by its Nobus instance ID.
func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.TrimSpace(req.ID) == "" {
		resp.Diagnostics.AddError(
			"Invalid import identifier",
			"Importing a Nobus instance requires the instance ID, for example:\n"+
				"  terraform import nobus_instance.web INSTANCE_ID",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// applyInstanceToModel refreshes state from the remote instance.
//
// Write-only attributes (admin_pass, user_data) are never returned by the API
// and are therefore left untouched.
func applyInstanceToModel(ctx context.Context, instance *client.Instance, model *instanceResourceModel, diags *diag.Diagnostics) {
	if instance == nil {
		return
	}

	model.ID = types.StringValue(instance.ID)
	if instance.Name != "" {
		model.Name = types.StringValue(instance.Name)
	}
	model.Status = stringOrNull(instance.Status)
	model.KeyName = stringOrNull(instance.KeyName)
	model.CreatedAt = stringOrNull(instance.Created)
	model.UpdatedAt = stringOrNull(instance.Updated)

	model.FlavorID = objectFieldAsString(instance.Flavor, "id")
	model.FlavorName = objectFieldAsString(instance.Flavor, "name")
	model.ImageID = objectFieldAsString(instance.Image, "id")
	model.ImageName = objectFieldAsString(instance.Image, "name")

	// InstanceSchema.addresses is documented only as an untyped object, so it is
	// stored verbatim as JSON rather than being reshaped into an assumed layout.
	if instance.Addresses == nil {
		model.Addresses = types.StringNull()
	} else {
		encoded, err := json.Marshal(instance.Addresses)
		if err != nil {
			diags.AddAttributeError(
				path.Root("addresses"),
				"Unable to read instance addresses",
				"Nobus returned addresses that could not be encoded as JSON: "+err.Error(),
			)
			return
		}
		model.Addresses = types.StringValue(string(encoded))
	}

	volumeIDs := make([]string, 0, len(instance.Volumes))
	for _, volume := range instance.Volumes {
		if volume.ID != "" {
			volumeIDs = append(volumeIDs, volume.ID)
		}
	}
	if len(volumeIDs) == 0 {
		model.VolumeIDs = types.ListNull(types.StringType)
	} else {
		ids, listDiags := types.ListValueFrom(ctx, types.StringType, volumeIDs)
		diags.Append(listDiags...)
		model.VolumeIDs = ids
	}
}

// objectFieldAsString reads a string field out of an opaque API object such as
// the flavor or image returned by the instance read operation.
func objectFieldAsString(source map[string]any, key string) types.String {
	if source == nil {
		return types.StringNull()
	}
	value, ok := source[key]
	if !ok {
		return types.StringNull()
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return types.StringNull()
	}
	return types.StringValue(text)
}

// int64RequiresReplace returns a plan modifier forcing replacement when the
// boot volume size changes.
func int64RequiresReplace() planmodifier.Int64 {
	return int64planmodifier.RequiresReplace()
}

// listRequiresReplace returns a plan modifier forcing replacement when a list
// of strings changes.
func listRequiresReplace() planmodifier.List {
	return listplanmodifier.RequiresReplace()
}

// DefaultInstanceVolumeSize is the boot volume size the Nobus API uses when
// volume_size is omitted, as documented on CreateInstanceRequestSchema.
const DefaultInstanceVolumeSize = 50
