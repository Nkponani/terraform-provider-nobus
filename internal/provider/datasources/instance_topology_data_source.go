package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

var (
	_ datasource.DataSource              = &InstanceTopologyDataSource{}
	_ datasource.DataSourceWithConfigure = &InstanceTopologyDataSource{}
)

// InstanceTopologyDataSource defines the data source implementation.
type InstanceTopologyDataSource struct {
	client *client.Client
}

// InstanceTopologyDataSourceModel describes the data source data model.
type InstanceTopologyDataSourceModel struct {
	InstanceID       types.String `tfsdk:"instance_id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Devices          types.Map    `tfsdk:"devices"`
	Links            types.List   `tfsdk:"links"`
}

func (d *InstanceTopologyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_topology"
}

func (d *InstanceTopologyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get instance topology.",
		Attributes: map[string]schema.Attribute{
			"instance_id":       schema.StringAttribute{MarkdownDescription: "Instance ID (server_id).", Required: true},
			"project_id":        schema.StringAttribute{MarkdownDescription: "Project ID.", Required: true},
			"availability_zone": schema.StringAttribute{MarkdownDescription: "Availability zone.", Required: true},
			"id":                schema.StringAttribute{Computed: true},
			"name":              schema.StringAttribute{Computed: true},
			"devices":           schema.MapAttribute{ElementType: types.StringType, Computed: true},
			"links":             schema.ListAttribute{ElementType: types.MapType{ElemType: types.StringType}, Computed: true},
		},
	}
}

func (d *InstanceTopologyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceTopologyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceTopologyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The client is not configured")
		return
	}
	res, err := d.client.GetInstanceTopology(ctx, data.ProjectID.ValueString(), data.AvailabilityZone.ValueString(), data.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance topology", err.Error())
		return
	}
	data.ID = types.StringValue(res.ID)
	data.Name = types.StringValue(res.Name)
	devices, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMapTopo(res.Devices))
	resp.Diagnostics.Append(diags...)
	links := make([]types.Map, 0, len(res.Links))
	for _, l := range res.Links {
		lm, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMapTopo(l))
		resp.Diagnostics.Append(diags...)
		links = append(links, lm)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	data.Devices = devices
	linksList, diags := types.ListValueFrom(ctx, types.MapType{ElemType: types.StringType}, links)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Links = linksList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func flattenAnyMapTopo(m map[string]any) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		switch t := v.(type) {
		case string:
			out[k] = t
		default:
			out[k] = fmt.Sprintf("%v", v)
		}
	}
	return out
}
