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
	_ datasource.DataSource              = (*kubernetesVolumeDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kubernetesVolumeDataSource)(nil)
)

type kubernetesVolumeDataSourceData struct {
	ID        types.Int64 `tfsdk:"id"`
	ClusterID types.Int64 `tfsdk:"cluster_id"`

	Name         types.String `tfsdk:"name"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Size         types.Int64  `tfsdk:"size"`
	LocationID   types.Int64  `tfsdk:"location_id"`
	Status       types.String `tfsdk:"status"`
}

func (k *kubernetesVolumeDataSourceData) FromEntity(clusterID int, volume kubernetes.Volume) {
	k.ID = types.Int64Value(int64(volume.ID))
	k.ClusterID = types.Int64Value(int64(clusterID))

	k.Name = types.StringValue(volume.Name)
	k.SerialNumber = types.StringValue(volume.SerialNumber)
	k.Size = types.Int64Value(int64(volume.Size))
	k.LocationID = types.Int64Value(int64(volume.Location.ID))
	k.Status = types.StringValue(volume.Status.Key)
}

func (k kubernetesVolumeDataSourceData) AppliesTo(volume kubernetes.Volume) bool {
	if !k.ID.IsNull() && k.ID.ValueInt64() != int64(volume.ID) {
		return false
	}

	if !k.Name.IsNull() && k.Name.ValueString() != volume.Name {
		return false
	}

	if !k.SerialNumber.IsNull() && k.SerialNumber.ValueString() != volume.SerialNumber {
		return false
	}

	return true
}

func (k kubernetesVolumeDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "a volume that a persistent volume claim inside the cluster created",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the volume",
				Optional:            true,
				Computed:            true,
			},
			"cluster_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Required:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the volume",
				Optional:            true,
				Computed:            true,
			},
			"serial_number": schema.StringAttribute{
				MarkdownDescription: "serial number of the volume",
				Optional:            true,
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "size of the volume in GiB",
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "current status of the volume, as a stable key",
				Computed:            true,
			},
		},
	}
}

func newKubernetesVolumeDataSource() datasource.DataSource {
	return &kubernetesVolumeDataSource{}
}

func (k *kubernetesVolumeDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_volume"
}

func (k *kubernetesVolumeDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.client = client
}

type kubernetesVolumeDataSource struct {
	client flowClient
}

func (k kubernetesVolumeDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config kubernetesVolumeDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	clusterID := int(config.ClusterID.ValueInt64())

	list, err := k.client.Kubernetes.Volume.List(ctx, kubernetes.VolumeListReq{ClusterID: uint(clusterID), Cursor: core.CursorAll})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list cluster volumes: %s", err))
		return
	}

	volume, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find cluster volume: %s", err))
		return
	}

	var state kubernetesVolumeDataSourceData
	state.FromEntity(clusterID, volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
