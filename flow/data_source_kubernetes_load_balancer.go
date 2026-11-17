package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/core"
	"github.com/flowswiss/goclient/v2/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ datasource.DataSource              = (*kubernetesLoadBalancerDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kubernetesLoadBalancerDataSource)(nil)
)

type kubernetesLoadBalancerDataSourceData struct {
	ID        types.Int64 `tfsdk:"id"`
	ClusterID types.Int64 `tfsdk:"cluster_id"`

	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	PrivateIP  types.String `tfsdk:"private_ip"`
	PublicIP   types.String `tfsdk:"public_ip"`
	Status     types.String `tfsdk:"status"`
}

func (k *kubernetesLoadBalancerDataSourceData) FromEntity(clusterID int, loadBalancer kubernetes.LoadBalancer) {
	k.ID = types.Int64Value(int64(loadBalancer.ID))
	k.ClusterID = types.Int64Value(int64(clusterID))

	k.Name = types.StringValue(loadBalancer.Name)
	k.LocationID = types.Int64Value(int64(loadBalancer.Location.ID))
	k.Status = types.StringValue(loadBalancer.Status.Key)

	k.NetworkID = types.Int64Null()
	k.PrivateIP = types.StringNull()
	k.PublicIP = types.StringNull()

	if len(loadBalancer.Networks) != 0 {
		network := loadBalancer.Networks[0]
		k.NetworkID = types.Int64Value(int64(network.ID))
		if len(network.Interfaces) != 0 {
			k.PrivateIP = types.StringValue(network.Interfaces[0].PrivateIP)

			if publicIP := network.Interfaces[0].PublicIP; publicIP != "" {
				k.PublicIP = types.StringValue(publicIP)
			}
		}
	}
}

func (k kubernetesLoadBalancerDataSourceData) AppliesTo(loadBalancer kubernetes.LoadBalancer) bool {
	if !k.ID.IsNull() && k.ID.ValueInt64() != int64(loadBalancer.ID) {
		return false
	}

	if !k.Name.IsNull() && k.Name.ValueString() != loadBalancer.Name {
		return false
	}

	return true
}

func (k kubernetesLoadBalancerDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "a load balancer that a service of type `LoadBalancer` inside the cluster created",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer",
				Optional:            true,
				Computed:            true,
			},
			"cluster_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Required:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer",
				Optional:            true,
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network the load balancer is attached to",
				Computed:            true,
			},
			"private_ip": schema.StringAttribute{
				MarkdownDescription: "private IP address of the load balancer",
				Computed:            true,
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public IP address of the load balancer, unset if it has none",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "current status of the load balancer, as a stable key",
				Computed:            true,
			},
		},
	}
}

func newKubernetesLoadBalancerDataSource() datasource.DataSource {
	return &kubernetesLoadBalancerDataSource{}
}

func (k *kubernetesLoadBalancerDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_load_balancer"
}

func (k *kubernetesLoadBalancerDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.client = client
}

type kubernetesLoadBalancerDataSource struct {
	client flowClient
}

func (k kubernetesLoadBalancerDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config kubernetesLoadBalancerDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	clusterID := int(config.ClusterID.ValueInt64())

	list, err := k.client.Kubernetes.LoadBalancer.List(ctx, kubernetes.LoadBalancerListReq{ClusterID: uint(clusterID), Cursor: core.CursorAll})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list cluster load balancers: %s", err))
		return
	}

	loadBalancer, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find cluster load balancer: %s", err))
		return
	}

	var state kubernetesLoadBalancerDataSourceData
	state.FromEntity(clusterID, loadBalancer)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
