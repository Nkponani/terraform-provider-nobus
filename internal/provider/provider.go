// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	providerdatabase "github.com/nobus-cloud/terraform-provider-nobus/internal/provider/database"
	providerdatasources "github.com/nobus-cloud/terraform-provider-nobus/internal/provider/datasources"
	providerfos "github.com/nobus-cloud/terraform-provider-nobus/internal/provider/fos"
	providerkubernetes "github.com/nobus-cloud/terraform-provider-nobus/internal/provider/kubernetes"
	providerlbaas "github.com/nobus-cloud/terraform-provider-nobus/internal/provider/lbaas"
	providernetworking "github.com/nobus-cloud/terraform-provider-nobus/internal/provider/networking"
	providerstorage "github.com/nobus-cloud/terraform-provider-nobus/internal/provider/storage"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure NobusProvider satisfies the provider interfaces.
var _ provider.Provider = &NobusProvider{}

// NobusProvider defines the Nobus Cloud provider implementation.
type NobusProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NobusProviderModel describes the provider data model.
type NobusProviderModel struct {
	APIKey           types.String `tfsdk:"api_key"`
	Endpoint         types.String `tfsdk:"endpoint"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
}

// Metadata returns the provider type name and version.
func (p *NobusProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nobus"
	resp.Version = p.version
}

// Schema defines the provider level configuration.
func (p *NobusProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage infrastructure on [Nobus Cloud](https://cloud.nobus.io) through the public Nobus API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Nobus API key, sent as a bearer token. Can also be set with the `NOBUS_API_KEY` " +
					"environment variable. Keep the key in a sensitive variable rather than in version control.",
				Optional:  true,
				Sensitive: true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Nobus API. Defaults to `" + client.DefaultBaseURL + "`. " +
					"Can also be set with the `NOBUS_ENDPOINT` environment variable.",
				Optional: true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Default Nobus project ID applied to resources that do not set their own " +
					"`project_id`. Can also be set with the `NOBUS_PROJECT_ID` environment variable.",
				Optional: true,
			},
			"availability_zone": schema.StringAttribute{
				MarkdownDescription: "Default availability zone applied to resources that do not set their own " +
					"`availability_zone`. Defaults to `" + client.DefaultAvailabilityZone + "`, the value the Nobus API " +
					"itself defaults to. Can also be set with the `NOBUS_AVAILABILITY_ZONE` environment variable.",
				Optional: true,
			},
		},
	}
}

// Resources returns the managed resources offered by this provider.
func (p *NobusProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Networking.
		NewNetworkResource,
		providernetworking.NewStandaloneNetworkResource,
		providernetworking.NewSubnetResource,
		providernetworking.NewRouterResource,
		providernetworking.NewRouterInterfaceResource,
		providernetworking.NewPortResource,
		providernetworking.NewPortAddressPairsResource,
		providernetworking.NewFloatingIPResource,
		providernetworking.NewSecurityGroupResource,
		providernetworking.NewSecurityGroupRuleResource,
		providernetworking.NewTrunkResource,
		providernetworking.NewQosPolicyResource,
		providernetworking.NewFirewallGroupResource,

		// Block storage.
		NewVolumeResource,
		NewVolumeAttachmentResource,
		providerstorage.NewVolumeSnapshotResource,
		providerstorage.NewVolumeTypeResource,
		providerstorage.NewVolumeTransferResource,
		providerstorage.NewVolumeTransferAcceptResource,

		// File/object storage (FOS).
		providerfos.NewContainerResource,
		providerfos.NewObjectResource,

		// Managed Kubernetes.
		providerkubernetes.NewClusterResource,
		providerkubernetes.NewMigrationResource,

		// Managed databases.
		providerdatabase.NewDatabaseResource,
		providerdatabase.NewActionResource,
		providerdatabase.NewExistingResource,

		// Load balancing as a service (LBaaS).
		providerlbaas.NewLoadBalancerResource,
		providerlbaas.NewListenerResource,
		providerlbaas.NewPoolResource,
		providerlbaas.NewMemberResource,
		providerlbaas.NewHealthMonitorResource,
		providerlbaas.NewL7PolicyResource,
		providerlbaas.NewL7RuleResource,
		providerlbaas.NewSecurityGroupRuleResource,
		providerlbaas.NewFailoverResource,
		providerlbaas.NewProvisionResource,

		// Compute.
		NewInstanceResource,
		NewInstanceFromVolumeResource,
		NewKeypairResource,
		NewInstanceActionResource,
	}
}

// DataSources returns the data sources offered by this provider.
func (p *NobusProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Networking.
		providerdatasources.NewCollectionDataSource("networks"),
		providerdatasources.NewCollectionDataSource("subnets"),
		providerdatasources.NewCollectionDataSource("ports"),
		providerdatasources.NewCollectionDataSource("routers"),
		providerdatasources.NewCollectionDataSource("floating_ips"),
		func() datasource.DataSource { return &providerdatasources.InstanceNetworkInterfacesDataSource{} },
		func() datasource.DataSource { return &providerdatasources.InstanceDiagnosticsDataSource{} },
		func() datasource.DataSource { return &providerdatasources.InstanceSecurityGroupsDataSource{} },
		func() datasource.DataSource { return &providerdatasources.InstanceTagsDataSource{} },
		func() datasource.DataSource { return &providerdatasources.InstanceTopologyDataSource{} },
		providerdatasources.NewCollectionDataSource("security_groups"),
		providerdatasources.NewCollectionDataSource("security_group_rules"),
		providerdatasources.NewCollectionDataSource("trunks"),
		providerdatasources.NewCollectionDataSource("qos_policies"),
		providerdatasources.NewCollectionDataSource("firewall_groups"),

		// Block storage.
		providerdatasources.NewCollectionDataSource("volumes"),
		providerdatasources.NewCollectionDataSource("volume_snapshots"),
		providerdatasources.NewCollectionDataSource("volume_backups"),
		providerdatasources.NewCollectionDataSource("volume_transfers"),
		providerdatasources.NewCollectionDataSource("volume_availability_zones"),
		providerstorage.NewVolumeBackupDataSource,

		// File/object storage (FOS).
		providerfos.NewAccountDataSource,
		providerfos.NewAuthDetailsDataSource,
		providerfos.NewContainersDataSource,
		providerfos.NewObjectsDataSource,
		providerfos.NewObjectContentDataSource,

		// Managed Kubernetes.
		providerkubernetes.NewClustersDataSource,
		providerkubernetes.NewBastionDataSource,
		providerkubernetes.NewLoadBalancerLoginDataSource,
		providerkubernetes.NewMonitoringDataSource,
		providerkubernetes.NewNodesDataSource,
		providerkubernetes.NewTaskStatusDataSource,

		// Managed databases.
		providerdatabase.NewDatabasesDataSource,
		providerdatabase.NewCredentialsDataSource,
		providerdatabase.NewCreationStatusDataSource,
		providerdatabase.NewStatusDataSource,
		providerdatabase.NewZabbixCredentialsDataSource,
		providerdatabase.NewLogsDataSource,

		// Load balancing as a service (LBaaS).
		providerlbaas.NewLoadBalancersDataSource,
		providerlbaas.NewListenersDataSource,
		providerlbaas.NewPoolsDataSource,
		providerlbaas.NewMembersDataSource,
		providerlbaas.NewHealthMonitorsDataSource,
		providerlbaas.NewL7PoliciesDataSource,
		providerlbaas.NewL7RulesDataSource,
		providerlbaas.NewTaskDataSource,
		providerlbaas.NewDeploymentStatusDataSource,
		providerlbaas.NewLoadBalancerStatusDataSource,
		providerlbaas.NewLoadBalancerStatsDataSource,
		providerlbaas.NewListenerStatsDataSource,
	}
}

// New returns a function that returns a NobusProvider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NobusProvider{
			version: version,
		}
	}
}
