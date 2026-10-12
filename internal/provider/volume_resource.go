// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

// Ensure the resource implements the expected interfaces.
var (
	_ resource.Resource                = &volumeResource{}
	_ resource.ResourceWithConfigure   = &volumeResource{}
	_ resource.ResourceWithImportState = &volumeResource{}
)

// NewVolumeResource returns a new nobus_volume resource.
func NewVolumeResource() resource.Resource {
	return &volumeResource{}
}

// volumeResource manages a Nobus volume.
//
// API mapping, verified against the OpenAPI specification:
//
//	Create: POST   /api/v3/volume/        VolumeCreateRequestSchema   -> VolumeResponseSchema
//	Read:   GET    /api/v3/volume/        ?project_id&volume_id&availability_zone -> GenericVolumeResponseSchema
//	Update: PUT    /api/v3/volume/        VolumeUpdateRequestSchema   -> GenericVolumeResponseSchema
//	Grow:   POST   /api/v3/volume/extend  ExtendVolumeRequestSchema   -> GenericVolumeResponseSchema
//	Delete: DELETE /api/v3/volume/        ?project_id&volume_id&availability_zone -> DeleteVolumeResponseSchema
type volumeResource struct {
	config *providerConfig
}

// volumeResourceModel maps the Terraform schema onto the Nobus API.
type volumeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Size             types.Int64  `tfsdk:"size"`
	VolumeType       types.String `tfsdk:"volume_type"`
	SnapshotID       types.String `tfsdk:"snapshot_id"`
	SourceVolumeID   types.String `tfsdk:"source_volume_id"`
	ImageRef         types.String `tfsdk:"image_ref"`
	Metadata         types.Map    `tfsdk:"metadata"`
	Status           types.String `tfsdk:"status"`
	Bootable         types.Bool   `tfsdk:"bootable"`
	Encrypted        types.Bool   `tfsdk:"encrypted"`
	Multiattach      types.Bool   `tfsdk:"multiattach"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
	AttachmentIDs    types.List   `tfsdk:"attachment_ids"`
	ServerIDs        types.List   `tfsdk:"server_ids"`
}

func (r *volumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *volumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Nobus volume.\n\n" +
			"Volumes are created asynchronously. After creation the provider polls the volume status " +
			"reported by Nobus until it becomes available, rather than sleeping for a fixed period.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Nobus volume ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Nobus project that owns the volume. Defaults to the provider `project_id`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"availability_zone": schema.StringAttribute{
				MarkdownDescription: "Availability zone hosting the volume. Defaults to the provider `availability_zone`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the volume. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the volume. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "Size of the volume in GiB. Required by the API. " +
					"Volumes can only grow: a larger value is applied through the Nobus extend operation, and a " +
					"smaller value forces a new volume.",
				Required:   true,
				Validators: []validator.Int64{Int64AtLeast(1)},
			},
			"volume_type": schema.StringAttribute{
				MarkdownDescription: "Volume type, for example `ssd` or `hdd`. Changing this forces a new volume.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"snapshot_id": schema.StringAttribute{
				MarkdownDescription: "Snapshot to create the volume from. Changing this forces a new volume.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_volume_id": schema.StringAttribute{
				MarkdownDescription: "Volume to clone. Changing this forces a new volume.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"image_ref": schema.StringAttribute{
				MarkdownDescription: "Image to create the volume from. Changing this forces a new volume.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Metadata key/value pairs attached to the volume. Updatable in place.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Volume status reported by Nobus, for example `available` or `in-use`.",
				Computed:            true,
			},
			"bootable": schema.BoolAttribute{
				MarkdownDescription: "Whether the volume is bootable.",
				Computed:            true,
			},
			"encrypted": schema.BoolAttribute{
				MarkdownDescription: "Whether the volume is encrypted. Changing this replaces the volume.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"multiattach": schema.BoolAttribute{
				MarkdownDescription: "Whether the volume may be attached to multiple instances, as reported by Nobus.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp reported by Nobus for the volume creation.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp reported by Nobus for the last volume modification.",
				Computed:            true,
			},
			"attachment_ids": schema.ListAttribute{
				MarkdownDescription: "Attachment IDs reported by Nobus for this volume.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"server_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of the instances the volume is currently attached to.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *volumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	metadata, metaDiags := mapToStringMap(ctx, plan.Metadata)
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if metadata == nil {
		// The API requires the metadata member to be present on create.
		metadata = map[string]string{}
	}

	created, err := apiClient.CreateVolume(ctx, client.CreateVolumeRequest{
		ProjectID:        resolved.ProjectID,
		AvailabilityZone: resolved.AvailabilityZone,
		Size:             plan.Size.ValueInt64(),
		Name:             optionalString(plan.Name),
		Description:      optionalString(plan.Description),
		VolumeType:       optionalString(plan.VolumeType),
		SnapshotID:       optionalString(plan.SnapshotID),
		SourceVolID:      optionalString(plan.SourceVolumeID),
		ImageRef:         optionalString(plan.ImageRef),
		Metadata:         metadata,
		Encrypted:        optionalBool(plan.Encrypted),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to create Nobus volume %q", plan.Name.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.ProjectID = types.StringValue(resolved.ProjectID)
	plan.AvailabilityZone = types.StringValue(resolved.AvailabilityZone)

	// Provisioning is asynchronous, so wait for the status Nobus reports rather
	// than sleeping for an arbitrary period.
	ready, err := apiClient.WaitForVolumeReady(ctx, resolved.ProjectID, resolved.AvailabilityZone, created.ID, nil, pollInterval)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Nobus volume %q did not become available", created.ID),
			diagnosticDetail(err),
		)
		return
	}

	applyVolumeToModel(ctx, ready, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	volume, err := apiClient.GetVolume(ctx, resolved.ProjectID, resolved.AvailabilityZone, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// The volume was removed outside Terraform; drop it from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Nobus volume %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	applyVolumeToModel(ctx, volume, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state volumeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	projectID, availabilityZone := resolved.ProjectID, resolved.AvailabilityZone
	newSize := plan.Size.ValueInt64()
	currentSize := state.Size.ValueInt64()

	switch {
	case newSize > currentSize:
		// Volumes can only grow, and growth is a dedicated API operation.
		if err := apiClient.ExtendVolume(ctx, client.ExtendVolumeRequest{
			ProjectID:        projectID,
			AvailabilityZone: availabilityZone,
			VolumeID:         state.ID.ValueString(),
			NewSize:          newSize,
		}); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to extend Nobus volume %q", state.ID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}

		if _, err := apiClient.WaitForVolumeReady(ctx, projectID, availabilityZone, state.ID.ValueString(), nil, pollInterval); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Nobus volume %q did not finish extending", state.ID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}

	case newSize < currentSize:
		resp.Diagnostics.AddAttributeError(
			path.Root("size"),
			"Cannot shrink a Nobus volume",
			fmt.Sprintf(
				"The volume is %d GiB and the configuration asks for %d GiB. The Nobus API only supports growing "+
					"a volume, so a smaller size forces a new volume. Remove this resource from state and "+
					"recreate it, or increase the size instead.",
				currentSize, newSize,
			),
		)
		return
	}

	// Name, description and metadata are the attributes the update model
	// supports. They are only sent when they actually changed.
	if volumeAttributesChanged(state, plan) {
		metadata, metaDiags := mapToStringMap(ctx, plan.Metadata)
		resp.Diagnostics.Append(metaDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if err := apiClient.UpdateVolume(ctx, client.UpdateVolumeRequest{
			ProjectID:        projectID,
			AvailabilityZone: availabilityZone,
			Name:             optionalString(plan.Name),
			Description:      optionalString(plan.Description),
			Metadata:         metadata,
		}); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to update Nobus volume %q", state.ID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
	}

	volume, err := apiClient.GetVolume(ctx, projectID, availabilityZone, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Nobus volume %q after update", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = state.ID
	plan.ProjectID = types.StringValue(projectID)
	plan.AvailabilityZone = types.StringValue(availabilityZone)
	applyVolumeToModel(ctx, volume, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiClient := requireConfigured(r.config, &resp.Diagnostics)
	if apiClient == nil {
		return
	}

	resolved, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}

	projectID, availabilityZone := resolved.ProjectID, resolved.AvailabilityZone
	volumeID := state.ID.ValueString()

	if err := apiClient.DeleteVolume(ctx, projectID, availabilityZone, volumeID); err != nil {
		if client.IsNotFound(err) {
			// Already gone; deleting is idempotent.
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Nobus volume %q", volumeID),
			diagnosticDetail(err),
		)
		return
	}

	// Deletion is asynchronous, so confirm the API no longer returns the volume.
	if err := apiClient.WaitForVolumeDeleted(ctx, projectID, availabilityZone, volumeID, pollInterval); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Nobus volume %q was not deleted", volumeID),
			diagnosticDetail(err),
		)
	}
}

// ImportState imports an existing volume by its Nobus volume ID.
func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.TrimSpace(req.ID) == "" {
		resp.Diagnostics.AddError(
			"Invalid import identifier",
			"Importing a Nobus volume requires the volume ID, for example:\n"+
				"  terraform import nobus_volume.data VOLUME_ID",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// volumeAttributesChanged reports whether any attribute covered by the update
// model differs between state and plan.
func volumeAttributesChanged(state, plan volumeResourceModel) bool {
	return !state.Name.Equal(plan.Name) ||
		!state.Description.Equal(plan.Description) ||
		!state.Metadata.Equal(plan.Metadata)
}

// applyVolumeToModel refreshes state from the remote volume.
//
// The create-only attributes (volume type, snapshot, source volume and image
// reference) are read back because the API returns them, but
// they are marked as requiring replacement in the schema.
func applyVolumeToModel(ctx context.Context, volume *client.Volume, model *volumeResourceModel, diags *diag.Diagnostics) {
	if volume == nil {
		return
	}

	model.ID = types.StringValue(volume.ID)
	model.Name = stringOrNull(volume.Name)
	model.Description = stringOrNull(volume.Description)
	model.Size = types.Int64Value(volume.Size)
	model.VolumeType = stringOrNull(volume.VolumeType)
	model.SnapshotID = stringOrNull(volume.SnapshotID)
	model.SourceVolumeID = stringOrNull(volume.SourceVolID)
	model.Bootable = boolOrNull(volume.Bootable)
	model.Encrypted = boolOrNull(volume.Encrypted)
	model.Multiattach = boolOrNull(volume.Multiattach)
	model.CreatedAt = stringOrNull(volume.CreatedAt)
	model.UpdatedAt = stringOrNull(volume.UpdatedAt)

	if volume.Status != nil {
		model.Status = types.StringValue(string(*volume.Status))
	} else {
		model.Status = types.StringNull()
	}

	if volume.ProjectID != nil {
		model.ProjectID = types.StringValue(*volume.ProjectID)
	}
	if volume.AvailabilityZone != nil {
		model.AvailabilityZone = types.StringValue(*volume.AvailabilityZone)
	}

	if volume.Metadata == nil {
		model.Metadata = types.MapNull(types.StringType)
	} else {
		metadata, metaDiags := types.MapValueFrom(ctx, types.StringType, volume.Metadata)
		diags.Append(metaDiags...)
		model.Metadata = metadata
	}

	attachmentIDs := make([]string, 0, len(volume.Attachments))
	serverIDs := make([]string, 0, len(volume.Attachments))
	for _, attachment := range volume.Attachments {
		if attachment.AttachmentID != nil && *attachment.AttachmentID != "" {
			attachmentIDs = append(attachmentIDs, *attachment.AttachmentID)
		}
		if attachment.ServerID != nil && *attachment.ServerID != "" {
			serverIDs = append(serverIDs, *attachment.ServerID)
		}
	}

	if len(attachmentIDs) == 0 {
		model.AttachmentIDs = types.ListNull(types.StringType)
	} else {
		ids, listDiags := types.ListValueFrom(ctx, types.StringType, attachmentIDs)
		diags.Append(listDiags...)
		model.AttachmentIDs = ids
	}

	if len(serverIDs) == 0 {
		model.ServerIDs = types.ListNull(types.StringType)
	} else {
		ids, listDiags := types.ListValueFrom(ctx, types.StringType, serverIDs)
		diags.Append(listDiags...)
		model.ServerIDs = ids
	}
}
