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
	_ datasource.DataSource              = (*kubernetesNodeDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kubernetesNodeDataSource)(nil)
)

type kubernetesNodeDataSourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	ClusterID types.Int64  `tfsdk:"cluster_id"`
	Name      types.String `tfsdk:"name"`
	Role      types.String `tfsdk:"role"`

	Roles     []types.String `tfsdk:"roles"`
	ProductID types.Int64    `tfsdk:"product_id"`
	NetworkID types.Int64    `tfsdk:"network_id"`
	PrivateIP types.String   `tfsdk:"private_ip"`
	PublicIP  types.String   `tfsdk:"public_ip"`
	Status    types.String   `tfsdk:"status"`
}

func (k *kubernetesNodeDataSourceData) FromEntity(clusterID int, node kubernetes.Node) {
	k.ID = types.Int64Value(int64(node.ID))
	k.ClusterID = types.Int64Value(int64(clusterID))
	k.Name = types.StringValue(node.Name)

	k.Roles = make([]types.String, len(node.Roles))
	for idx, role := range node.Roles {
		k.Roles[idx] = types.StringValue(role.Key)
	}

	k.ProductID = types.Int64Value(int64(node.Product.ID))
	k.NetworkID = types.Int64Value(int64(node.Network.ID))
	k.Status = types.StringValue(node.Status.Key)

	k.PrivateIP = types.StringNull()
	k.PublicIP = types.StringNull()

	if len(node.Network.Interfaces) != 0 {
		k.PrivateIP = types.StringValue(node.Network.Interfaces[0].PrivateIP)

		if publicIP := node.Network.Interfaces[0].PublicIP; publicIP != "" {
			k.PublicIP = types.StringValue(publicIP)
		}
	}
}

func (k kubernetesNodeDataSourceData) AppliesTo(node kubernetes.Node) bool {
	if !k.ID.IsNull() && k.ID.ValueInt64() != int64(node.ID) {
		return false
	}

	if !k.Name.IsNull() && k.Name.ValueString() != node.Name {
		return false
	}

	if !k.Role.IsNull() && !hasRole(node, k.Role.ValueString()) {
		return false
	}

	return true
}

func hasRole(node kubernetes.Node, key string) bool {
	for _, role := range node.Roles {
		if role.Key == key {
			return true
		}
	}

	return false
}

func (k kubernetesNodeDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "a node of a cluster. the platform creates and names the nodes, so look one up by its role or name",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the node",
				Optional:            true,
				Computed:            true,
			},
			"cluster_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the node",
				Optional:            true,
				Computed:            true,
			},
			"role": schema.StringAttribute{
				MarkdownDescription: "only match a node that carries this role, for example `control-plane` or `worker`. a role held by more than one node needs a second filter",
				Optional:            true,
			},

			"roles": schema.ListAttribute{
				MarkdownDescription: "roles of the node, as stable keys",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the node product",
				Computed:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network the node is attached to",
				Computed:            true,
			},
			"private_ip": schema.StringAttribute{
				MarkdownDescription: "private IP address of the node",
				Computed:            true,
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public IP address of the node, unset if it has none",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "current status of the node, as a stable key",
				Computed:            true,
			},
		},
	}
}

func newKubernetesNodeDataSource() datasource.DataSource {
	return &kubernetesNodeDataSource{}
}

func (k *kubernetesNodeDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_node"
}

func (k *kubernetesNodeDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.client = client
}

type kubernetesNodeDataSource struct {
	client flowClient
}

func (k kubernetesNodeDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config kubernetesNodeDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	clusterID := int(config.ClusterID.ValueInt64())

	list, err := k.client.Kubernetes.Node.List(ctx, kubernetes.NodeListReq{ClusterID: uint(clusterID), Cursor: core.CursorAll})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list cluster nodes: %s", err))
		return
	}

	node, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find cluster node: %s", err))
		return
	}

	var state kubernetesNodeDataSourceData
	state.FromEntity(clusterID, node)
	state.Role = config.Role

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
