package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*kubernetesKubeConfigDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kubernetesKubeConfigDataSource)(nil)
)

type kubernetesKubeConfigDataSourceData struct {
	ClusterID  types.Int64  `tfsdk:"cluster_id"`
	KubeConfig types.String `tfsdk:"kube_config"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (k *kubernetesKubeConfigDataSourceData) FromEntity(clusterID int, kubeConfig kubernetes.ClusterKubeConfig) {
	k.ClusterID = types.Int64Value(int64(clusterID))
	k.KubeConfig = types.StringValue(kubeConfig.KubeConfig)
}

func (k kubernetesKubeConfigDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Required:            true,
			},
			"kube_config": schema.StringAttribute{
				MarkdownDescription: "kube config of the cluster",
				Computed:            true,
				Sensitive:           true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockWithOpts(ctx, timeouts.Opts{
				ReadDescription: timeoutDescription("bounds the whole read; unset, the cluster is given 20m to become ready"),
			}),
		},
	}
}

func newKubernetesKubeConfigDataSource() datasource.DataSource {
	return &kubernetesKubeConfigDataSource{}
}

func (k *kubernetesKubeConfigDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_kube_config"
}

func (k *kubernetesKubeConfigDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.client = client
}

type kubernetesKubeConfigDataSource struct {
	client flowClient
}

func (k kubernetesKubeConfigDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config kubernetesKubeConfigDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Read, &response.Diagnostics)
	defer cancel()

	clusterID := int(config.ClusterID.ValueInt64())

	cluster, err := k.client.Kubernetes.Cluster.Get(ctx, kubernetes.ClusterGetReq{ID: uint(clusterID)})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}
	if cluster.Locked || cluster.Status.ID != compute.ClusterStatusHealthy {
		if _, err := waitForClusterReady(ctx, k.client.Kubernetes.Cluster, clusterID); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be ready: %s", err))
			return
		}
	}

	kubeConfig, err := k.client.Kubernetes.Cluster.GetKubeConfig(ctx, kubernetes.ClusterGetReq{ID: uint(clusterID)})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get kube config: %s", err))
		return
	}

	var state kubernetesKubeConfigDataSourceData
	state.FromEntity(int(config.ClusterID.ValueInt64()), kubeConfig)
	state.Timeouts = config.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
