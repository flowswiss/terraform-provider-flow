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
	_ tfsdk.ResourceType = (*computeRouterInterfaceResourceType)(nil)
	_ tfsdk.Resource     = (*computeRouterInterfaceResource)(nil)
)

type computeRouterInterfaceResourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	RouterID  types.Int64  `tfsdk:"router_id"`
	NetworkID types.Int64  `tfsdk:"network_id"`
	PrivateIP types.String `tfsdk:"private_ip"`
}

func (c *computeRouterInterfaceResourceData) FromEntity(routerID int, routerInterface compute.RouterInterface) {
	c.ID = types.Int64{Value: int64(routerInterface.ID)}
	c.RouterID = types.Int64{Value: int64(routerID)}
	c.PrivateIP = types.String{Value: routerInterface.PrivateIP}
	c.NetworkID = types.Int64{Value: int64(routerInterface.Network.ID)}
}

type computeRouterInterfaceResourceType struct{}

func (c computeRouterInterfaceResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the router interface",
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
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"private_ip": {
				Type:                types.StringType,
				MarkdownDescription: "private IP address of the router interface",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeRouterInterfaceResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeRouterInterfaceResource{
		client: prov.client,
	}, diagnostics
}

type computeRouterInterfaceResource struct {
	client goclient.Client
}

func (c computeRouterInterfaceResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeRouterInterfaceResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(config.RouterID.Value)
	create := compute.RouterInterfaceCreate{
		NetworkID: int(config.NetworkID.Value),
		PrivateIP: config.PrivateIP.Value,
	}

	routerInterface, err := compute.NewRouterInterfaceService(c.client, routerID).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create router interface: %s", err))
		return
	}

	var state computeRouterInterfaceResourceData
	state.FromEntity(routerID, routerInterface)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeRouterInterfaceResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeRouterInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(state.RouterID.Value)
	list, err := compute.NewRouterInterfaceService(c.client, routerID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list router interfaces: %s", err))
		return
	}

	for _, routerInterface := range list.Items {
		if routerInterface.ID == int(state.ID.Value) {
			state.FromEntity(routerID, routerInterface)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", "router interface could not be found")
}

func (c computeRouterInterfaceResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a router interface is not supported")
}

func (c computeRouterInterfaceResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeRouterInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(state.RouterID.Value)
	err := compute.NewRouterInterfaceService(c.client, routerID).Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete router interface: %s", err))
		return
	}
}
