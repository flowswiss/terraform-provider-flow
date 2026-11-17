package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeRouterRouteResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeRouterRouteResource)(nil)
	_ resource.ResourceWithImportState = (*computeRouterRouteResource)(nil)
)

type computeRouterRouteResourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	RouterID    types.Int64  `tfsdk:"router_id"`
	Destination types.String `tfsdk:"destination"`
	NextHop     types.String `tfsdk:"next_hop"`
}

func (c *computeRouterRouteResourceData) FromEntity(routerID int, route compute.Route) {
	c.ID = types.Int64Value(int64(route.ID))
	c.RouterID = types.Int64Value(int64(routerID))
	c.Destination = types.StringValue(route.Destination)
	c.NextHop = types.StringValue(route.NextHop)
}

func (c computeRouterRouteResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_router_route.<name> <router_id>:<id>`",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the route",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"router_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"destination": schema.StringAttribute{
				MarkdownDescription: "IP destination range of the route",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"next_hop": schema.StringAttribute{
				MarkdownDescription: "IP address of the next hop",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeRouterRouteResource() resource.Resource {
	return &computeRouterRouteResource{}
}

func (c *computeRouterRouteResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_router_route"
}

func (c *computeRouterRouteResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeRouterRouteResource struct {
	client flowClient
}

func (c computeRouterRouteResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeRouterRouteResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(config.RouterID.ValueInt64())
	create := compute.RouteCreateReq{
		RouterID:    uint(routerID),
		Destination: config.Destination.ValueString(),
		NextHop:     config.NextHop.ValueString(),
	}

	var route compute.Route
	err := retryCreate(ctx, "create route", func() (err error) {
		route, err = c.client.Compute.Route.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create route: %s", err))
		return
	}

	var state computeRouterRouteResourceData
	state.FromEntity(routerID, route)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeRouterRouteResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeRouterRouteResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(state.RouterID.ValueInt64())

	list, err := c.client.Compute.Route.List(ctx, compute.RouteListReq{
		RouterID: uint(routerID),
		Cursor:   core.CursorAll,
	})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("router %d", routerID))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list routes: %s", err))
		return
	}

	for _, route := range list.Items {
		if route.ID == int(state.ID.ValueInt64()) {
			state.FromEntity(routerID, route)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	removeGone(ctx, response, fmt.Sprintf("route %d", state.ID.ValueInt64()))
}

func (c computeRouterRouteResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a route is not supported")
}

func (c computeRouterRouteResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeRouterRouteResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete route", func() error {
		return c.client.Compute.Route.Delete(ctx, compute.RouteDeleteReq{
			RouterID: uint(state.RouterID.ValueInt64()),
			RouteID:  uint(state.ID.ValueInt64()),
		})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete route: %s", err))
		return
	}
}

func (c computeRouterRouteResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("router_id"), path.Root("id"))
}
