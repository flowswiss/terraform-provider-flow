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
	_ tfsdk.DataSourceType = (*computeRouterRouteDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeRouterRouteDataSource)(nil)
)

type computeRouterRouteDataSourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	RouterID    types.Int64  `tfsdk:"router_id"`
	Destination types.String `tfsdk:"destination"`
	NextHop     types.String `tfsdk:"next_hop"`
}

func (c *computeRouterRouteDataSourceData) FromEntity(routerID int, route compute.Route) {
	c.ID = types.Int64{Value: int64(route.ID)}
	c.RouterID = types.Int64{Value: int64(routerID)}
	c.Destination = types.String{Value: route.Destination}
	c.NextHop = types.String{Value: route.NextHop}
}

type computeRouterRouteDataSourceType struct{}

func (c computeRouterRouteDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the route",
				Required:            true,
			},
			"router_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the router",
				Required:            true,
			},
			"destination": {
				Type:                types.StringType,
				MarkdownDescription: "IP destination range of the route",
				Computed:            true,
			},
			"next_hop": {
				Type:                types.StringType,
				MarkdownDescription: "IP address of the next hop",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeRouterRouteDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeRouterRouteDataSource{
		client: prov.client,
	}, diagnostics
}

type computeRouterRouteDataSource struct {
	client goclient.Client
}

func (c computeRouterRouteDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeRouterRouteDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routeID := int(config.ID.Value)
	routerID := int(config.RouterID.Value)

	list, err := compute.NewRouteService(c.client, routerID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list routes: %s", err))
		return
	}

	for _, route := range list.Items {
		if route.ID == routeID {
			var state computeRouterRouteDataSourceData
			state.FromEntity(routerID, route)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", "requested route could not be found")
}
