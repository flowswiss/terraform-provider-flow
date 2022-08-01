package flow

import (
	"context"
	"fmt"
	"github.com/flowswiss/goclient/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.DataSourceType = (*kubernetesKubeConfigDataSourceType)(nil)
	_ tfsdk.DataSource     = (*kubernetesKubeConfigDataSource)(nil)
)

type kubernetesKubeConfigDataSourceData struct {
	ClusterID  types.Int64  `tfsdk:"cluster_id"`
	KubeConfig types.String `tfsdk:"kube_config"`
}

func (k *kubernetesKubeConfigDataSourceData) FromEntity(clusterID int, kubeConfig kubernetes.ClusterKubeConfig) {
	k.ClusterID = types.Int64{Value: int64(clusterID)}
	k.KubeConfig = types.String{Value: kubeConfig.KubeConfig}
}

type kubernetesKubeConfigDataSourceType struct{}

func (k kubernetesKubeConfigDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"cluster_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the cluster",
				Required:            true,
			},
			"kube_config": {
				Type:                types.StringType,
				MarkdownDescription: "kube config of the cluster",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}, nil
}

func (k kubernetesKubeConfigDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return kubernetesKubeConfigDataSource{
		clusterService: kubernetes.NewClusterService(prov.client),
	}, diagnostics
}

type kubernetesKubeConfigDataSource struct {
	clusterService kubernetes.ClusterService
}

func (k kubernetesKubeConfigDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config kubernetesKubeConfigDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	kubeConfig, err := k.clusterService.GetKubeConfig(ctx, int(config.ClusterID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}

	var state kubernetesKubeConfigDataSourceData
	state.FromEntity(int(config.ClusterID.Value), kubeConfig)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
