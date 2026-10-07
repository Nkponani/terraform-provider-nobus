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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

// Ensure the resource implements the expected interfaces.
var (
	_ resource.Resource                = &networkResource{}
	_ resource.ResourceWithConfigure   = &networkResource{}
	_ resource.ResourceWithImportState = &networkResource{}
)

// NewNetworkResource returns a new nobus_network resource.
func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

// networkResource manages a Nobus network.
//
// API mapping, verified against the OpenAPI specification:
//
//	Create: POST   /api/v3/network/  CreateNetworkRequestSchema -> CreateNetworkResponseSchema
//	Read:   GET    /api/v3/network/  ?project_id&network_id&availability_zone -> NetworkResponseSchema
//	Update: PUT    /api/v3/network/  NetworkUpdateRequestBody -> NetworkResponseSchema
//	Delete: DELETE /api/v3/network/  ?project_id&network_id&availability_zone -> GenericResponseSchema
type networkResource struct {
	config *providerConfig
}

// networkResourceModel maps the Terraform schema onto the Nobus API.
type networkResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	Name              types.String `tfsdk:"name"`
	SubnetCIDR        types.String `tfsdk:"subnet_cidr"`
	RouterName        types.String `tfsdk:"router_name"`
	ExternalNetworkID types.String `tfsdk:"external_network_id"`
	DNSNameservers    types.List   `tfsdk:"dns_nameservers"`
	EnableSNAT        types.Bool   `tfsdk:"enable_snat"`
	Description       types.String `tfsdk:"description"`
	AdminStateUp      types.Bool   `tfsdk:"admin_state_up"`
	Shared            types.Bool   `tfsdk:"shared"`
	MTU               types.Int64  `tfsdk:"mtu"`
	RouterExternal    types.Bool   `tfsdk:"router_external"`
	PortSecurity      types.Bool   `tfsdk:"port_security_enabled"`
	Tags              types.Set    `tfsdk:"tags"`
	Status            types.String `tfsdk:"status"`
	SubnetID          types.String `tfsdk:"subnet_id"`
	RouterID          types.String `tfsdk:"router_id"`
	SubnetIDs         types.List   `tfsdk:"subnet_ids"`
}

func (r *networkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Nobus network.\n\n" +
			"The Nobus API creates a network, a subnet and a router in a single call, so this resource " +
			"requires a subnet CIDR and a router name and exposes the resulting subnet and router IDs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Nobus network ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Nobus project that owns the network. Defaults to the provider `project_id`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"availability_zone": schema.StringAttribute{
				MarkdownDescription: "Availability zone hosting the network. Defaults to the provider `availability_zone`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the network.",
				Required:            true,
				Validators:          []validator.String{StringNotEmpty()},
			},
			"subnet_cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR block of the subnet created together with the network. " +
					"Required by the Nobus API when creating a network, and never returned by the API, so it " +
					"cannot be verified after an import. Changing this forces a new network.",
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{StringNotEmpty()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"router_name": schema.StringAttribute{
				MarkdownDescription: "Name of the router created together with the network. " +
					"Required by the Nobus API when creating a network, and never returned by the API, so it " +
					"cannot be verified after an import. Changing this forces a new network.",
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{StringNotEmpty()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"external_network_id": schema.StringAttribute{
				MarkdownDescription: "ID of an external network to attach the router to. Changing this forces a new network.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns_nameservers": schema.ListAttribute{
				MarkdownDescription: "DNS nameservers for the subnet. Changing this forces a new network.",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"enable_snat": schema.BoolAttribute{
				MarkdownDescription: "Whether the router performs source NAT. Defaults to `true`, which is the " +
					"documented API default. Never returned by the API, so it cannot be verified after an " +
					"import. Changing this forces a new network.",
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the network. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"admin_state_up": schema.BoolAttribute{
				MarkdownDescription: "Administrative state of the network. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"shared": schema.BoolAttribute{
				MarkdownDescription: "Whether the network is shared. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"mtu": schema.Int64Attribute{
				MarkdownDescription: "MTU of the network. The API accepts values between 68 and 9216. Updatable in place.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{Int64AtLeast(68)},
			},
			"router_external": schema.BoolAttribute{
				MarkdownDescription: "Whether the router attached to this network is an external router. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"port_security_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether port security is enabled on the network. Updatable in place.",
				Optional:            true,
				Computed:            true,
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags assigned to the network. Updatable in place.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status reported by Nobus.",
				Computed:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "ID of the subnet created together with the network.",
				Computed:            true,
			},
			"router_id": schema.StringAttribute{
				MarkdownDescription: "ID of the router created together with the network.",
				Computed:            true,
			},
			"subnet_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of all subnets associated with the network, as reported by Nobus.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *networkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkResourceModel

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

	// subnet_cidr and router_name are required by the API but are not returned on
	// read, so they are only validated when the resource is actually created.
	if err := requireCreateValue(ctx, &resp.Diagnostics, "subnet_cidr", plan.SubnetCIDR, plan); err != nil {
		return
	}
	if err := requireCreateValue(ctx, &resp.Diagnostics, "router_name", plan.RouterName, plan); err != nil {
		return
	}

	enableSNAT := plan.EnableSNAT
	if enableSNAT.IsNull() || enableSNAT.IsUnknown() {
		// Documented API default.
		enableSNAT = types.BoolValue(true)
	}

	dnsNameservers, dnsDiags := listToStringSlice(ctx, plan.DNSNameservers)
	resp.Diagnostics.Append(dnsDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := apiClient.CreateNetwork(ctx, client.CreateNetworkRequest{
		ProjectID:         projectID,
		AvailabilityZone:  availabilityZone,
		NetworkName:       plan.Name.ValueString(),
		SubnetCIDR:        plan.SubnetCIDR.ValueString(),
		RouterName:        plan.RouterName.ValueString(),
		ExternalNetworkID: optionalString(plan.ExternalNetworkID),
		DNSNameservers:    dnsNameservers,
		EnableSNAT:        optionalBool(enableSNAT),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to create Nobus network %q", plan.Name.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = types.StringValue(created.NetworkID)
	plan.SubnetID = types.StringValue(created.SubnetID)
	plan.RouterID = types.StringValue(created.RouterID)
	plan.ProjectID = types.StringValue(projectID)
	plan.AvailabilityZone = types.StringValue(availabilityZone)

	// CreateNetworkRequestSchema has no members for description, tags,
	// admin_state_up, shared, mtu, router_external or port_security_enabled.
	// When any of them are configured they are applied through the documented
	// update operation so that the following plan stays empty.
	if networkUpdateRequired(plan) {
		if _, err := r.updateNetwork(ctx, projectID, availabilityZone, created.NetworkID, plan); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to configure Nobus network %q", plan.Name.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
	}

	// Read back the authoritative object so server generated values are stored
	// exactly as Nobus reports them.
	network, err := apiClient.GetNetwork(ctx, projectID, availabilityZone, created.NetworkID)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Nobus network %q after creation", plan.Name.ValueString()),
			diagnosticDetail(err),
		)
		return
	}
	applyNetworkToModel(ctx, network, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkResourceModel

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

	network, err := apiClient.GetNetwork(ctx, projectID, availabilityZone, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// The network was removed outside Terraform; drop it from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Nobus network %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	state.ID = types.StringValue(network.ID)
	applyNetworkToModel(ctx, network, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state networkResourceModel

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

	if networkUpdateRequired(plan) {
		if _, err := r.updateNetwork(ctx, projectID, availabilityZone, state.ID.ValueString(), plan); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Unable to update Nobus network %q", state.ID.ValueString()),
				diagnosticDetail(err),
			)
			return
		}
	}

	network, err := apiClient.GetNetwork(ctx, projectID, availabilityZone, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to read Nobus network %q after update", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}

	plan.ID = state.ID
	plan.SubnetID = state.SubnetID
	plan.RouterID = state.RouterID
	plan.ProjectID = types.StringValue(projectID)
	plan.AvailabilityZone = types.StringValue(availabilityZone)
	applyNetworkToModel(ctx, network, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkResourceModel

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

	err := apiClient.DeleteNetwork(ctx, projectID, availabilityZone, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to delete Nobus network %q", state.ID.ValueString()),
			diagnosticDetail(err),
		)
		return
	}
}

// ImportState imports an existing network by its Nobus network ID. The
// availability zone and project are taken from the provider configuration,
// because every Nobus read requires them.
func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.TrimSpace(req.ID) == "" {
		resp.Diagnostics.AddError(
			"Invalid import identifier",
			"Importing a Nobus network requires the network ID, for example:\n"+
				"  terraform import nobus_network.production NETWORK_ID",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// updateNetwork applies the mutable attributes of the model through the
// documented update operation.
func (r *networkResource) updateNetwork(
	ctx context.Context,
	projectID, availabilityZone, networkID string,
	plan networkResourceModel,
) (*client.Network, error) {
	update := client.NetworkUpdate{
		Name:                optionalString(plan.Name),
		Description:         optionalString(plan.Description),
		AdminStateUp:        optionalBool(plan.AdminStateUp),
		Shared:              optionalBool(plan.Shared),
		MTU:                 optionalInt64(plan.MTU),
		RouterExternal:      optionalBool(plan.RouterExternal),
		PortSecurityEnabled: optionalBool(plan.PortSecurity),
	}

	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags, diags := setToStringSlice(ctx, plan.Tags)
		if diags.HasError() {
			return nil, fmt.Errorf("reading tags: %s", diags.Errors()[0].Detail())
		}
		update.Tags = &tags
	}

	return r.config.Client.UpdateNetwork(ctx, client.NetworkUpdateRequest{
		ProjectID:        projectID,
		AvailabilityZone: availabilityZone,
		NetworkID:        networkID,
		Network:          update,
	})
}

// networkUpdateRequired reports whether any in-place updatable attribute is
// configured.
func networkUpdateRequired(plan networkResourceModel) bool {
	return !plan.Description.IsNull() && !plan.Description.IsUnknown() ||
		!plan.AdminStateUp.IsNull() && !plan.AdminStateUp.IsUnknown() ||
		!plan.Shared.IsNull() && !plan.Shared.IsUnknown() ||
		!plan.MTU.IsNull() && !plan.MTU.IsUnknown() ||
		!plan.RouterExternal.IsNull() && !plan.RouterExternal.IsUnknown() ||
		!plan.PortSecurity.IsNull() && !plan.PortSecurity.IsUnknown() ||
		!plan.Tags.IsNull() && !plan.Tags.IsUnknown()
}

// applyNetworkToModel refreshes every state property from the remote object.
//
// The immutable create-only attributes (subnet CIDR, router name, external
// network ID, DNS nameservers and SNAT) are deliberately left untouched because
// the API does not expose them on read and cannot change them.
func applyNetworkToModel(ctx context.Context, network *client.Network, model *networkResourceModel, diags *diag.Diagnostics) {
	if network == nil {
		return
	}

	if network.ID != "" {
		model.ID = types.StringValue(network.ID)
	}
	model.Name = stringOrNull(network.Name)
	model.Description = stringOrNull(network.Description)
	model.AdminStateUp = boolOrNull(network.AdminStateUp)
	model.Shared = boolOrNull(network.Shared)
	model.MTU = int64OrNull(network.MTU)
	model.RouterExternal = boolOrNull(network.RouterExternal)
	model.PortSecurity = boolOrNull(network.PortSecurityEnabled)
	model.Status = stringOrNull(network.Status)

	if network.ProjectID != nil {
		model.ProjectID = types.StringValue(*network.ProjectID)
	}

	if network.Tags == nil {
		model.Tags = types.SetNull(types.StringType)
	} else {
		tags, tagDiags := types.SetValueFrom(ctx, types.StringType, network.Tags)
		diags.Append(tagDiags...)
		model.Tags = tags
	}

	if network.Subnets == nil {
		model.SubnetIDs = types.ListNull(types.StringType)
	} else {
		subnets, subnetDiags := types.ListValueFrom(ctx, types.StringType, network.Subnets)
		diags.Append(subnetDiags...)
		model.SubnetIDs = subnets
	}
}
