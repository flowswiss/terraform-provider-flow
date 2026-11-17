package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ datasource.DataSource              = (*computeRouterInterfaceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeRouterInterfaceDataSource)(nil)
)

type computeRouterInterfaceDataSourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	RouterID  types.Int64  `tfsdk:"router_id"`
	NetworkID types.Int64  `tfsdk:"network_id"`
	PrivateIP types.String `tfsdk:"private_ip"`
}

func (c *computeRouterInterfaceDataSourceData) FromEntity(routerID int, routerInterface compute.RouterInterface) {
	c.ID = types.Int64Value(int64(routerInterface.ID))
	c.RouterID = types.Int64Value(int64(routerID))
	c.PrivateIP = types.StringValue(routerInterface.PrivateIP)
	c.NetworkID = types.Int64Value(int64(routerInterface.Network.ID))
}

func (c computeRouterInterfaceDataSourceData) AppliesTo(routerInterface compute.RouterInterface) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(routerInterface.ID) {
		return false
	}

	if !c.NetworkID.IsNull() && c.NetworkID.ValueInt64() != int64(routerInterface.Network.ID) {
		return false
	}

	if !c.PrivateIP.IsNull() && c.PrivateIP.ValueString() != routerInterface.PrivateIP {
		return false
	}

	return true
}

func (c computeRouterInterfaceDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router interface",
				Optional:            true,
				Computed:            true,
			},
			"router_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router",
				Required:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
			"private_ip": schema.StringAttribute{
				MarkdownDescription: "private IP address of the router interface",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeRouterInterfaceDataSource() datasource.DataSource {
	return &computeRouterInterfaceDataSource{}
}

func (c *computeRouterInterfaceDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_router_interface"
}

func (c *computeRouterInterfaceDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeRouterInterfaceDataSource struct {
	client flowClient
}

func (c computeRouterInterfaceDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeRouterInterfaceDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(config.RouterID.ValueInt64())
	list, err := c.client.Compute.RouterInterface.List(ctx, compute.RouterInterfaceListReq{RouterID: uint(routerID), Cursor: core.CursorAll})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list router interfaces: %s", err))
		return
	}

	routerInterface, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find router interface: %s", err))
		return
	}

	var state computeRouterInterfaceDataSourceData
	state.FromEntity(routerID, routerInterface)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
