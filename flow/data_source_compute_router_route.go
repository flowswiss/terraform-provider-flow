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
	_ datasource.DataSource              = (*computeRouterRouteDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeRouterRouteDataSource)(nil)
)

type computeRouterRouteDataSourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	RouterID    types.Int64  `tfsdk:"router_id"`
	Destination types.String `tfsdk:"destination"`
	NextHop     types.String `tfsdk:"next_hop"`
}

func (c *computeRouterRouteDataSourceData) FromEntity(routerID int, route compute.Route) {
	c.ID = types.Int64Value(int64(route.ID))
	c.RouterID = types.Int64Value(int64(routerID))
	c.Destination = types.StringValue(route.Destination)
	c.NextHop = types.StringValue(route.NextHop)
}

func (c computeRouterRouteDataSourceData) AppliesTo(route compute.Route) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(route.ID) {
		return false
	}

	if !c.Destination.IsNull() && c.Destination.ValueString() != route.Destination {
		return false
	}

	if !c.NextHop.IsNull() && c.NextHop.ValueString() != route.NextHop {
		return false
	}

	return true
}

func (c computeRouterRouteDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the route",
				Optional:            true,
				Computed:            true,
			},
			"router_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router",
				Required:            true,
			},
			"destination": schema.StringAttribute{
				MarkdownDescription: "IP destination range of the route",
				Optional:            true,
				Computed:            true,
			},
			"next_hop": schema.StringAttribute{
				MarkdownDescription: "IP address of the next hop",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeRouterRouteDataSource() datasource.DataSource {
	return &computeRouterRouteDataSource{}
}

func (c *computeRouterRouteDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_router_route"
}

func (c *computeRouterRouteDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeRouterRouteDataSource struct {
	client flowClient
}

func (c computeRouterRouteDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeRouterRouteDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(config.RouterID.ValueInt64())
	list, err := c.client.Compute.Route.List(ctx, compute.RouteListReq{RouterID: uint(routerID), Cursor: core.CursorAll})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list routes: %s", err))
		return
	}

	route, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find route: %s", err))
		return
	}

	var state computeRouterRouteDataSourceData
	state.FromEntity(routerID, route)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
