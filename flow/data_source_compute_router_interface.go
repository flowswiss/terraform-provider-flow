package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.DataSourceType = (*computeRouterInterfaceDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeRouterInterfaceDataSource)(nil)
)

type computeRouterInterfaceDataSourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	RouterID  types.Int64  `tfsdk:"router_id"`
	NetworkID types.Int64  `tfsdk:"network_id"`
	PrivateIP types.String `tfsdk:"private_ip"`
}

func (c *computeRouterInterfaceDataSourceData) FromEntity(routerID int, routerInterface compute.RouterInterface) {
	c.ID = types.Int64{Value: int64(routerInterface.ID)}
	c.RouterID = types.Int64{Value: int64(routerID)}
	c.PrivateIP = types.String{Value: routerInterface.PrivateIP}
	c.NetworkID = types.Int64{Value: int64(routerInterface.Network.ID)}
}

type computeRouterInterfaceDataSourceType struct{}

func (c computeRouterInterfaceDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the router interface",
				Computed:            true,
			},
			"router_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the router",
				Required:            true,
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
			},
			"private_ip": {
				Type:                types.StringType,
				MarkdownDescription: "private IP address of the router interface",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeRouterInterfaceDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeRouterInterfaceDataSource{
		client: prov.client,
	}, diagnostics
}

type computeRouterInterfaceDataSource struct {
	client goclient.Client
}

func (c computeRouterInterfaceDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeRouterInterfaceDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(config.RouterID.Value)
	list, err := compute.NewRouterInterfaceService(c.client, routerID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list router interfaces: %s", err))
		return
	}

	for _, routerInterface := range list.Items {
		if routerInterface.Network.ID == int(config.NetworkID.Value) {
			var state computeRouterInterfaceDataSourceData
			state.FromEntity(routerID, routerInterface)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", "requested router interface could not be found")
}
