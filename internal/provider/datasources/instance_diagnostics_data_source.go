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
	_ datasource.DataSource              = &InstanceDiagnosticsDataSource{}
	_ datasource.DataSourceWithConfigure = &InstanceDiagnosticsDataSource{}
)

// InstanceDiagnosticsDataSource defines the data source implementation.
type InstanceDiagnosticsDataSource struct {
	client *client.Client
}

// InstanceDiagnosticsDataSourceModel describes the data source data model.
type InstanceDiagnosticsDataSourceModel struct {
	InstanceID       types.String `tfsdk:"instance_id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Disk             types.Map    `tfsdk:"disk"`
	CPU              types.Map    `tfsdk:"cpu"`
	Memory           types.Map    `tfsdk:"memory"`
	Network          types.Map    `tfsdk:"network"`
	General          types.Map    `tfsdk:"general"`
	Platform         types.Map    `tfsdk:"platform"`
}

func (d *InstanceDiagnosticsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_diagnostics"
}

func (d *InstanceDiagnosticsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get instance diagnostics.",
		Attributes: map[string]schema.Attribute{
			"instance_id":       schema.StringAttribute{MarkdownDescription: "Instance ID (server_id).", Required: true},
			"project_id":        schema.StringAttribute{MarkdownDescription: "Project ID.", Required: true},
			"availability_zone": schema.StringAttribute{MarkdownDescription: "Availability zone.", Required: true},
			"disk":              schema.MapAttribute{ElementType: types.StringType, Computed: true},
			"cpu":               schema.MapAttribute{ElementType: types.StringType, Computed: true},
			"memory":            schema.MapAttribute{ElementType: types.StringType, Computed: true},
			"network":           schema.MapAttribute{ElementType: types.StringType, Computed: true},
			"general":           schema.MapAttribute{ElementType: types.StringType, Computed: true},
			"platform":          schema.MapAttribute{ElementType: types.StringType, Computed: true},
		},
	}
}

func (d *InstanceDiagnosticsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceDiagnosticsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceDiagnosticsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The client is not configured")
		return
	}
	res, err := d.client.GetInstanceDiagnostics(ctx, data.ProjectID.ValueString(), data.AvailabilityZone.ValueString(), data.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance diagnostics", err.Error())
		return
	}
	// convert map[string]any to map[string]string representation
	disk, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMap(res.Disk))
	resp.Diagnostics.Append(diags...)
	cpu, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMap(res.CPU))
	resp.Diagnostics.Append(diags...)
	mem, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMap(res.Memory))
	resp.Diagnostics.Append(diags...)
	net, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMap(res.Network))
	resp.Diagnostics.Append(diags...)
	gen, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMap(res.General))
	resp.Diagnostics.Append(diags...)
	plat, diags := types.MapValueFrom(ctx, types.StringType, flattenAnyMap(res.Platform))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Disk = disk
	data.CPU = cpu
	data.Memory = mem
	data.Network = net
	data.General = gen
	data.Platform = plat
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func flattenAnyMap(m map[string]any) map[string]string {
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
