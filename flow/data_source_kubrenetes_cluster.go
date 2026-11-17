package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/core"
	"github.com/flowswiss/goclient/v2/kubernetes"
	"github.com/flowswiss/terraform-provider-flow/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*kubernetesClusterDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kubernetesClusterDataSource)(nil)
)

type kubernetesClusterDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	LocationID      types.Int64 `tfsdk:"location_id"`
	NetworkID       types.Int64 `tfsdk:"network_id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	PublicAddress types.String `tfsdk:"public_address"`
	DNSName       types.String `tfsdk:"dns_name"`

	VersionID types.Int64 `tfsdk:"version_id"`

	NodeCount     types.Int64 `tfsdk:"node_count"`
	NodeProductID types.Int64 `tfsdk:"node_product_id"`
}

func (k *kubernetesClusterDataSourceData) FromEntity(cluster kubernetes.Cluster) {
	k.ID = types.Int64Value(int64(cluster.ID))
	k.Name = types.StringValue(cluster.Name)

	k.LocationID = types.Int64Value(int64(cluster.Location.ID))
	k.NetworkID = types.Int64Value(int64(cluster.Network.ID))
	k.SecurityGroupID = types.Int64Value(int64(cluster.SecurityGroup.ID))

	if cluster.PublicAddress == "" {
		k.PublicAddress = types.StringNull()
	} else {
		k.PublicAddress = types.StringValue(cluster.PublicAddress)
	}

	k.DNSName = types.StringValue(cluster.DNSName)

	k.VersionID = types.Int64Value(int64(cluster.Version.ID))

	k.NodeCount = types.Int64Value(int64(cluster.NodeCount.Expected.Worker))
	k.NodeProductID = types.Int64Value(int64(cluster.ExpectedPreset.Worker.ID))
}

func (k kubernetesClusterDataSourceData) AppliesTo(cluster kubernetes.Cluster) bool {
	if !k.ID.IsNull() && k.ID.ValueInt64() != int64(cluster.ID) {
		return false
	}

	if !k.Name.IsNull() && k.Name.ValueString() != cluster.Name {
		return false
	}

	if !k.LocationID.IsNull() && k.LocationID.ValueInt64() != int64(cluster.Location.ID) {
		return false
	}

	if !k.NetworkID.IsNull() && k.NetworkID.ValueInt64() != int64(cluster.Network.ID) {
		return false
	}

	if !k.SecurityGroupID.IsNull() && k.SecurityGroupID.ValueInt64() != int64(cluster.SecurityGroup.ID) {
		return false
	}

	if !k.PublicAddress.IsNull() && k.PublicAddress.ValueString() != cluster.PublicAddress {
		return false
	}

	if !k.DNSName.IsNull() && k.DNSName.ValueString() != cluster.DNSName {
		return false
	}

	return true
}

func (k kubernetesClusterDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
			"security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Optional:            true,
				Computed:            true,
			},
			"public_address": schema.StringAttribute{
				MarkdownDescription: "public address of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "DNS name of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"version_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the kubernetes version",
				Computed:            true,
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "number of nodes in the cluster",
				Computed:            true,
			},
			"node_product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the node product",
				Computed:            true,
			},
		},
	}
}

func newKubernetesClusterDataSource() datasource.DataSource {
	return &kubernetesClusterDataSource{}
}

func (k *kubernetesClusterDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_cluster"
}

func (k *kubernetesClusterDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.client = client
}

type kubernetesClusterDataSource struct {
	client flowClient
}

func (k kubernetesClusterDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config kubernetesClusterDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	clusters, err := k.client.Kubernetes.Cluster.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list clusters: %s", err))
		return
	}

	cluster, err := filter.FindOne(config, clusters.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find cluster: %s", err))
		return
	}

	var state kubernetesClusterDataSourceData
	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
