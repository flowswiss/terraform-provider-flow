package flow

import (
	"context"
	"fmt"
	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/kubernetes"
	"github.com/flowswiss/terraform-provider-flow/filter"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.DataSourceType = (*kubernetesClusterDataSourceType)(nil)
	_ tfsdk.DataSource     = (*kubernetesClusterDataSource)(nil)
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
	k.ID = types.Int64{Value: int64(cluster.ID)}
	k.Name = types.String{Value: cluster.Name}

	k.LocationID = types.Int64{Value: int64(cluster.Location.ID)}
	k.NetworkID = types.Int64{Value: int64(cluster.Network.ID)}
	k.SecurityGroupID = types.Int64{Value: int64(cluster.SecurityGroup.ID)}

	if cluster.PublicAddress == "" {
		k.PublicAddress = types.String{Null: true}
	} else {
		k.PublicAddress = types.String{Value: cluster.PublicAddress}
	}

	k.DNSName = types.String{Value: cluster.DNSName}

	k.VersionID = types.Int64{Value: int64(cluster.Version.ID)}

	k.NodeCount = types.Int64{Value: int64(cluster.NodeCount.Expected.Worker)}
	k.NodeProductID = types.Int64{Value: int64(cluster.ExpectedPreset.Worker.ID)}
}

func (k kubernetesClusterDataSourceData) AppliesTo(cluster kubernetes.Cluster) bool {
	if !k.ID.Null && k.ID.Value != int64(cluster.ID) {
		return false
	}

	if !k.Name.Null && k.Name.Value != cluster.Name {
		return false
	}

	if !k.LocationID.Null && k.LocationID.Value != int64(cluster.Location.ID) {
		return false
	}

	if !k.NetworkID.Null && k.NetworkID.Value != int64(cluster.Network.ID) {
		return false
	}

	if !k.SecurityGroupID.Null && k.SecurityGroupID.Value != int64(cluster.SecurityGroup.ID) {
		return false
	}

	if !k.PublicAddress.Null && k.PublicAddress.Value != cluster.PublicAddress {
		return false
	}

	if !k.DNSName.Null && k.DNSName.Value != cluster.DNSName {
		return false
	}

	return true
}

type kubernetesClusterDataSourceType struct{}

func (k kubernetesClusterDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
			"security_group_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the security group",
				Optional:            true,
				Computed:            true,
			},
			"public_address": {
				Type:                types.StringType,
				MarkdownDescription: "public address of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"dns_name": {
				Type:                types.StringType,
				MarkdownDescription: "DNS name of the cluster",
				Optional:            true,
				Computed:            true,
			},
			"version_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the kubernetes version",
				Computed:            true,
			},
			"node_count": {
				Type:                types.Int64Type,
				MarkdownDescription: "number of nodes in the cluster",
				Computed:            true,
			},
			"node_product_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the node product",
				Computed:            true,
			},
		},
	}, nil
}

func (k kubernetesClusterDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return kubernetesClusterDataSource{
		clusterService: kubernetes.NewClusterService(prov.client),
	}, diagnostics
}

type kubernetesClusterDataSource struct {
	clusterService kubernetes.ClusterService
}

func (k kubernetesClusterDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config kubernetesClusterDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	clusters, err := k.clusterService.List(ctx, goclient.Cursor{NoFilter: 1})
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
