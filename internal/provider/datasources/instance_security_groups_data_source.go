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
	_ datasource.DataSource              = &InstanceSecurityGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &InstanceSecurityGroupsDataSource{}
)

// InstanceSecurityGroupsDataSource defines the data source implementation.
type InstanceSecurityGroupsDataSource struct {
	client *client.Client
}

// InstanceSecurityGroupsDataSourceModel describes the data source data model.
type InstanceSecurityGroupsDataSourceModel struct {
	InstanceID       types.String `tfsdk:"instance_id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	SecurityGroups   types.List   `tfsdk:"security_groups"`
}

type SecurityGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ProjectID   types.String `tfsdk:"project_id"`
	Tags        types.List   `tfsdk:"tags"`
}

func (d *InstanceSecurityGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_security_groups"
}

func (d *InstanceSecurityGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List security groups attached to an instance.",
		Attributes: map[string]schema.Attribute{
			"instance_id":       schema.StringAttribute{MarkdownDescription: "Instance ID (server_id).", Required: true},
			"project_id":        schema.StringAttribute{MarkdownDescription: "Project ID.", Required: true},
			"availability_zone": schema.StringAttribute{MarkdownDescription: "Availability zone.", Required: true},
			"security_groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of security groups.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"project_id":  schema.StringAttribute{Computed: true},
						"tags":        schema.ListAttribute{ElementType: types.StringType, Computed: true},
					},
				},
			},
		},
	}
}

func (d *InstanceSecurityGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *InstanceSecurityGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data InstanceSecurityGroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d.client == nil {
		resp.Diagnostics.AddError("Client not configured", "The client is not configured")
		return
	}
	sgs, err := d.client.GetInstanceSecurityGroups(ctx, data.ProjectID.ValueString(), data.AvailabilityZone.ValueString(), data.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance security groups", err.Error())
		return
	}
	items := make([]SecurityGroupModel, 0, len(sgs))
	for _, sg := range sgs {
		tags, diags := types.ListValueFrom(ctx, types.StringType, sg.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		name := types.StringNull()
		if sg.Name != nil {
			name = types.StringValue(*sg.Name)
		}
		desc := types.StringNull()
		if sg.Description != nil {
			desc = types.StringValue(*sg.Description)
		}
		proj := types.StringNull()
		if sg.ProjectID != nil {
			proj = types.StringValue(*sg.ProjectID)
		} else if sg.TenantID != nil {
			proj = types.StringValue(*sg.TenantID)
		}
		items = append(items, SecurityGroupModel{
			ID:          types.StringValue(sg.ID),
			Name:        name,
			Description: desc,
			ProjectID:   proj,
			Tags:        tags,
		})
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"description": types.StringType,
		"project_id":  types.StringType,
		"tags":        types.ListType{ElemType: types.StringType},
	}}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SecurityGroups = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
