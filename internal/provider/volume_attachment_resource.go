// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

// Ensure the resource implements the expected interfaces.
var (
	_ resource.Resource                = &volumeAttachmentResource{}
	_ resource.ResourceWithConfigure   = &volumeAttachmentResource{}
	_ resource.ResourceWithImportState = &volumeAttachmentResource{}
)

// NewVolumeAttachmentResource returns a new nobus_volume_attachment resource.
func NewVolumeAttachmentResource() resource.Resource {
	return &volumeAttachmentResource{}
}

// volumeAttachmentResource attaches a volume to an instance.
//
// API mapping, verified against the OpenAPI specification:
//
//	Attach: POST /api/v3/volume/attach AttachVolumeRequestSchema
//	Detach: POST /api/v3/volume/detach DetachVolumeRequestSchema
//
// Both operations take the same members, so a single request type is used.
type volumeAttachmentResource struct {
	config *providerConfig
}

// volumeAttachmentResourceModel maps the Terraform schema onto the Nobus API.
type volumeAttachmentResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	InstanceID       types.String `tfsdk:"instance_id"`
	VolumeID         types.String `tfsdk:"volume_id"`
	VolumeStatus     types.String `tfsdk:"volume_status"`
	Attached         types.Bool   `tfsdk:"attached"`
}

func (r *volumeAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_attachment"
}

func (r *volumeAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Attaches a Nobus volume to an instance.\n\n" +
			"After attaching, the provider waits until the volume reports the `in-use` status. On destroy it " +
			"waits until the volume leaves that status again.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier of this attachment, formed from the instance and volume IDs.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Nobus project that owns the volume and the instance. " +
					"Defaults to the provider `project_id`.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"availability_zone": schema.StringAttribute{
				MarkdownDescription: "Availability zone hosting the volume and the instance. " +
					"Defaults to the provider `availability_zone`.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "ID of the instance to attach the volume to.",
				Required:            true,
				Validators:          []validator.String{StringNotEmpty()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"volume_id": schema.StringAttribute{
				MarkdownDescription: "ID of the volume to attach.",
				Required:            true,
				Validators:          []validator.String{StringNotEmpty()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"volume_status": schema.StringAttribute{
				MarkdownDescription: "Status of the attached volume, as reported by Nobus.",
				Computed:            true,
			},
			"attached": schema.BoolAttribute{
				MarkdownDescription: "Whether the volume is currently attached.",
				Computed:            true,
			},
		},
	}
}

func (r *volumeAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *volumeAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeAttachmentResourceModel

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

	projectID, availabilityZone := resolved.ProjectID, resolved.AvailabilityZone
	volumeID := plan.VolumeID.ValueString()

	if err := apiClient.AttachVolume(ctx, client.AttachVolumeRequest{
		ProjectID:        projectID,
		AvailabilityZone: availabilityZone,
		ServerID:         plan.InstanceID.ValueString(),
		VolumeID:         volumeID,
	}); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to attach Nobus volume %q to instance %q", volumeID, plan.InstanceID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	volume, err := apiClient.WaitForVolumeAttached(ctx, projectID, availabilityZone, volumeID, true, pollInterval)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Nobus volume %q was not attached", volumeID),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = types.StringValue(attachmentID(plan.InstanceID.ValueString(), volumeID))
	plan.ProjectID = types.StringValue(projectID)
	plan.AvailabilityZone = types.StringValue(availabilityZone)
	plan.Attached = types.BoolValue(true)
	plan.VolumeStatus = volumeStatusOrNull(volume)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeAttachmentResourceModel

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

	volume, err := apiClient.GetVolume(ctx, resolved.ProjectID, resolved.AvailabilityZone, state.VolumeID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Either the volume or the instance is gone, so the attachment no
			// longer exists.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Nobus volume %q", state.VolumeID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	state.VolumeStatus = volumeStatusOrNull(volume)
	state.Attached = types.BoolValue(volume.Status != nil && *volume.Status == client.VolumeStateInUse)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *volumeAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every configurable attribute requires replacement, so there is nothing to
	// update in place.
	var plan volumeAttachmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeAttachmentResourceModel

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
	volumeID := state.VolumeID.ValueString()

	if err := apiClient.DetachVolume(ctx, client.AttachVolumeRequest{
		ProjectID:        projectID,
		AvailabilityZone: availabilityZone,
		ServerID:         state.InstanceID.ValueString(),
		VolumeID:         volumeID,
	}); err != nil {
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to detach Nobus volume %q from instance %q", volumeID, state.InstanceID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
		// The volume or instance is already gone; nothing left to detach.
		return
	}

	if _, err := apiClient.WaitForVolumeAttached(ctx, projectID, availabilityZone, volumeID, false, pollInterval); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Nobus volume %q was not detached", volumeID),
			diagnosticDetail(err),
		)
	}
}

// ImportState imports an attachment using "<instance_id>/<volume_id>".
func (r *volumeAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	instanceID, volumeID, found := strings.Cut(strings.TrimSpace(req.ID), "/")
	if !found || instanceID == "" || volumeID == "" {
		resp.Diagnostics.AddError(
			"Invalid import identifier",
			"Importing a Nobus volume attachment requires INSTANCE_ID and VOLUME_ID separated by a slash, "+
				"for example:\n  terraform import nobus_volume_attachment.database INSTANCE_ID/VOLUME_ID",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), attachmentID(instanceID, volumeID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), instanceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_id"), volumeID)...)
}

// attachmentID builds the stable Terraform identifier of an attachment.
func attachmentID(instanceID, volumeID string) string {
	return instanceID + "/" + volumeID
}

// volumeStatusOrNull converts an API volume status into a Terraform string.
func volumeStatusOrNull(volume *client.Volume) types.String {
	if volume == nil || volume.Status == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*volume.Status))
}
