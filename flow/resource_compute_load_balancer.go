package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType            = (*computeLoadBalancerResourceType)(nil)
	_ tfsdk.Resource                = (*computeLoadBalancerResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeLoadBalancerResource)(nil)
)

type computeLoadBalancerResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	PrivateIP  types.String `tfsdk:"private_ip"`
}

func (c *computeLoadBalancerResourceData) FromEntity(loadBalancer compute.LoadBalancer) {
	c.ID = types.Int64{Value: int64(loadBalancer.ID)}
	c.Name = types.String{Value: loadBalancer.Name}
	c.LocationID = types.Int64{Value: int64(loadBalancer.Location.ID)}

	if len(loadBalancer.Networks) != 0 {
		network := loadBalancer.Networks[0]
		c.NetworkID = types.Int64{Value: int64(network.ID)}
		c.PrivateIP = types.String{Value: network.Interfaces[0].PrivateIP}
	}
}

type computeLoadBalancerResourceType struct{}

func (c computeLoadBalancerResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer",
				Required:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the initial network",
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"private_ip": {
				Type:                types.StringType,
				MarkdownDescription: "initial private ip of the load balancer",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeLoadBalancerResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerResource{
		loadBalancerService: compute.NewLoadBalancerService(prov.client),
		orderService:        common.NewOrderService(prov.client),
	}, diagnostics
}

type computeLoadBalancerResource struct {
	loadBalancerService compute.LoadBalancerService
	orderService        common.OrderService
}

func (c computeLoadBalancerResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeLoadBalancerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.LoadBalancerCreate{
		Name:             config.Name.Value,
		LocationID:       int(config.LocationID.Value),
		AttachExternalIP: false,
		NetworkID:        int(config.NetworkID.Value),
		PrivateIP:        config.PrivateIP.Value,
	}

	ordering, err := c.loadBalancerService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create load balancer: %s", err))
		return
	}

	order, err := c.orderService.WaitUntilProcessed(ctx, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer creation: %s", err))
		return
	}

	loadBalancer, err := c.loadBalancerService.Get(ctx, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer: %s", err))
		return
	}

	err = c.loadBalancerService.WaitUntilMutable(ctx, loadBalancer.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
		return
	}

	var state computeLoadBalancerResourceData
	state.FromEntity(loadBalancer)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancer, err := c.loadBalancerService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer: %s", err))
		return
	}

	state.FromEntity(loadBalancer)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeLoadBalancerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	update := compute.LoadBalancerUpdate{
		Name: config.Name.Value,
	}

	loadBalancer, err := c.loadBalancerService.Update(ctx, int(state.ID.Value), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update load balancer: %s", err))
		return
	}

	state.FromEntity(loadBalancer)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := c.loadBalancerService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete load balancer: %s", err))
		return
	}
}

func (c computeLoadBalancerResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
