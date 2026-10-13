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
	_ datasource.DataSource              = &InstanceTagsDataSource{}
	_ datasource.DataSourceWithConfigure = &InstanceTagsDataSource{}
)

// InstanceTagsDataSource defines the data source implementation.
type InstanceTagsDataSource struct {
	client *client.Client
}

// InstanceTagsDataSourceModel describes the data source data model.
type InstanceTagsDataSourceModel struct {
	InstanceID       types.String `tfsdk:"instance_id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Tags             types.List   `tfsdk:"tags"`
}

func (d *InstanceTagsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_tags"
}

func (d *InstanceTagsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List tags attached to an instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id":       schema.StringAttribute{MarkdownDescription: "Instance ID (server_id).", Required: true},
			"project_id":        schema.StringAttribute{MarkdownDescription: "Project ID.", Required: true},
			"availability_zone": schema.StringAttribute{MarkdownDescription: "Availability zone.", Required: true},
			"tags":              schema.ListAttribute{ElementType: types.StringType, Computed: true},
		},
	}
}

func (d *InstanceTagsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceTagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The client is not configured")
		return
	}
	tags, err := d.client.GetInstanceTags(ctx, data.ProjectID.ValueString(), data.AvailabilityZone.ValueString(), data.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance tags", err.Error())
		return
	}
	tagsList, diags := types.ListValueFrom(ctx, types.StringType, tags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Tags = tagsList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
