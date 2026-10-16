// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

var (
	_ datasource.DataSource              = &InstanceNetworkInterfacesDataSource{}
	_ datasource.DataSourceWithConfigure = &InstanceNetworkInterfacesDataSource{}
)

// InstanceNetworkInterfacesDataSource defines the data source implementation.
type InstanceNetworkInterfacesDataSource struct {
	client *client.Client
}

// InstanceNetworkInterfacesDataSourceModel describes the data source data model.
type InstanceNetworkInterfacesDataSourceModel struct {
	InstanceID        types.String `tfsdk:"instance_id"`
	ProjectID         types.String `tfsdk:"project_id"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	NetworkInterfaces types.List   `tfsdk:"network_interfaces"`
}

type InstanceNetworkInterfaceModel struct {
	NetworkID types.String `tfsdk:"network_id"`
	PortID    types.String `tfsdk:"port_id"`
	MacAddr   types.String `tfsdk:"mac_addr"`
}

func (d *InstanceNetworkInterfacesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_network_interfaces"
}

func (d *InstanceNetworkInterfacesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List network interfaces attached to an instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{
				MarkdownDescription: "Instance ID (server_id).",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project ID.",
				Required:            true,
			},
			"availability_zone": schema.StringAttribute{
				MarkdownDescription: "Availability zone.",
				Required:            true,
			},
			"network_interfaces": schema.ListNestedAttribute{
				MarkdownDescription: "List of network interfaces.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"network_id": schema.StringAttribute{Computed: true},
						"port_id":    schema.StringAttribute{Computed: true},
						"mac_addr":   schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *InstanceNetworkInterfacesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *InstanceNetworkInterfacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceNetworkInterfacesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The client is not configured")
		return
	}
	nicsListRaw, err := d.client.GetInstanceNetworkInterfaces(ctx, data.ProjectID.ValueString(), data.AvailabilityZone.ValueString(), data.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance network interfaces", err.Error())
		return
	}
	nics := make([]InstanceNetworkInterfaceModel, 0, len(nicsListRaw))
	for _, ni := range nicsListRaw {
		n := InstanceNetworkInterfaceModel{
			NetworkID: types.StringValue(ni.NetworkID),
			PortID:    types.StringValue(ni.PortID),
			MacAddr:   types.StringValue(ni.MacAddr),
		}
		nics = append(nics, n)
	}
	nicsList, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{
		"network_id": types.StringType,
		"port_id":    types.StringType,
		"mac_addr":   types.StringType,
	}}, nics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.NetworkInterfaces = nicsList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
