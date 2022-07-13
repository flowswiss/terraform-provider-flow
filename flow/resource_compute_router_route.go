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
	_ tfsdk.ResourceType = (*computeRouterRouteResourceType)(nil)
	_ tfsdk.Resource     = (*computeRouterRouteResource)(nil)
)

type computeRouterRouteResourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	RouterID    types.Int64  `tfsdk:"router_id"`
	Destination types.String `tfsdk:"destination"`
	NextHop     types.String `tfsdk:"next_hop"`
}

func (c *computeRouterRouteResourceData) FromEntity(routerID int, route compute.Route) {
	c.ID = types.Int64{Value: int64(route.ID)}
	c.RouterID = types.Int64{Value: int64(routerID)}
	c.Destination = types.String{Value: route.Destination}
	c.NextHop = types.String{Value: route.NextHop}
}

type computeRouterRouteResourceType struct{}

func (c computeRouterRouteResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the route",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"router_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the router",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"destination": {
				Type:                types.StringType,
				MarkdownDescription: "IP destination range of the route",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"next_hop": {
				Type:                types.StringType,
				MarkdownDescription: "IP address of the next hop",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeRouterRouteResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeRouterRouteResource{
		client: prov.client,
	}, diagnostics
}

type computeRouterRouteResource struct {
	client goclient.Client
}

func (c computeRouterRouteResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeRouterRouteResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(config.RouterID.Value)
	create := compute.RouteCreate{
		Destination: config.Destination.Value,
		NextHop:     config.NextHop.Value,
	}

	route, err := compute.NewRouteService(c.client, routerID).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create route: %s", err))
		return
	}

	var state computeRouterRouteResourceData
	state.FromEntity(routerID, route)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeRouterRouteResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeRouterRouteResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(state.RouterID.Value)

	list, err := compute.NewRouteService(c.client, routerID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list routes: %s", err))
		return
	}

	for _, route := range list.Items {
		if route.ID == int(state.ID.Value) {
			state.FromEntity(routerID, route)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("route with id %d not found", state.ID.Value))
}

func (c computeRouterRouteResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a route is not supported")
}

func (c computeRouterRouteResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeRouterRouteResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(state.RouterID.Value)
	err := compute.NewRouteService(c.client, routerID).Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete route: %s", err))
		return
	}
}
