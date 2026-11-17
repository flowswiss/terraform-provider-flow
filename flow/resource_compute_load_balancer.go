package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/common"
	"github.com/flowswiss/goclient/v2/compute"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeLoadBalancerResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeLoadBalancerResource)(nil)
	_ resource.ResourceWithImportState = (*computeLoadBalancerResource)(nil)
)

type computeLoadBalancerResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	PrivateIP  types.String `tfsdk:"private_ip"`
	PublicIP   types.String `tfsdk:"public_ip"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (c *computeLoadBalancerResourceData) FromEntity(loadBalancer compute.LoadBalancer) {
	c.ID = types.Int64Value(int64(loadBalancer.ID))
	c.Name = types.StringValue(loadBalancer.Name)
	c.LocationID = types.Int64Value(int64(loadBalancer.Location.ID))

	c.PublicIP = types.StringNull()

	if len(loadBalancer.Networks) != 0 {
		network := loadBalancer.Networks[0]
		c.NetworkID = types.Int64Value(int64(network.ID))
		if len(network.Interfaces) != 0 {
			c.PrivateIP = types.StringValue(network.Interfaces[0].PrivateIP)

			if publicIP := network.Interfaces[0].PublicIP; publicIP != "" {
				c.PublicIP = types.StringValue(publicIP)
			}
		}
	}
}

func (c computeLoadBalancerResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the initial network (the organisation's default network when omitted)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"private_ip": schema.StringAttribute{
				MarkdownDescription: "initial private ip of the load balancer",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public ip of the load balancer, null while it has none. Attach one with `flow_compute_elastic_ip_load_balancer_attachment`",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: timeoutDescription("bounds the whole create; unset, the order wait and the wait for the load balancer to become mutable are bounded at 10m each"),
				Delete:            true,
				DeleteDescription: timeoutDescription("bounds the whole delete; unset, the load balancer is given 10m to disappear"),
			}),
		},
	}
}

func newComputeLoadBalancerResource() resource.Resource {
	return &computeLoadBalancerResource{}
}

func (c *computeLoadBalancerResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer"
}

func (c *computeLoadBalancerResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeLoadBalancerResource struct {
	client flowClient
}

func (c computeLoadBalancerResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeLoadBalancerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Create, &response.Diagnostics)
	defer cancel()

	create := compute.LoadBalancerCreateReq{
		Name:       config.Name.ValueString(),
		LocationID: int(config.LocationID.ValueInt64()),
		NetworkID:  int(config.NetworkID.ValueInt64()),
		PrivateIP:  config.PrivateIP.ValueString(),
	}

	var ordering common.Ordering
	err := retryCreate(ctx, "create load balancer", func() (err error) {
		ordering, err = c.client.Compute.LoadBalancer.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create load balancer: %s", err))
		return
	}

	order, err := waitForOrder(ctx, c.client.Common.Order, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer creation: %s", err))
		return
	}

	loadBalancer, err := waitForLoadBalancerMutable(ctx, c.client.Compute.LoadBalancer, uint(order.Product.ID))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
		// Return only if the load balancer was never fetched. Otherwise it exists
		// and is saved below with the error, so Terraform replaces it on the next
		// apply instead of forgetting it.
		if loadBalancer.ID == 0 {
			return
		}
	}

	var state computeLoadBalancerResourceData
	state.FromEntity(loadBalancer)
	state.Timeouts = config.Timeouts

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancer, err := c.client.Compute.LoadBalancer.Get(ctx, compute.LoadBalancerGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("load balancer %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer: %s", err))
		return
	}

	state.FromEntity(loadBalancer)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
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

	update := compute.LoadBalancerUpdateReq{
		ID:   uint(state.ID.ValueInt64()),
		Name: nonZero(config.Name.ValueString()),
	}

	var loadBalancer compute.LoadBalancer
	err := retry(ctx, "update load balancer", func() (err error) {
		loadBalancer, err = c.client.Compute.LoadBalancer.Update(ctx, update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update load balancer: %s", err))
		return
	}

	state.FromEntity(loadBalancer)
	state.Timeouts = config.Timeouts

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeLoadBalancerResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeLoadBalancerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, state.Timeouts.Delete, &response.Diagnostics)
	defer cancel()

	loadBalancerID := uint(state.ID.ValueInt64())

	err := retryDelete(ctx, "delete load balancer", func() error {
		return c.client.Compute.LoadBalancer.Delete(ctx, compute.LoadBalancerDeleteReq{ID: loadBalancerID})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete load balancer: %s", err))
		return
	}

	err = waitForGone(ctx, goneTimeout, fmt.Sprintf("load balancer %d", loadBalancerID), func(ctx context.Context) error {
		_, err := c.client.Compute.LoadBalancer.Get(ctx, compute.LoadBalancerGetReq{ID: loadBalancerID})
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer deletion: %s", err))
		return
	}
}

func (c computeLoadBalancerResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func waitForLoadBalancerMutable(ctx context.Context, service *compute.LoadBalancerService, loadBalancerID uint) (loadBalancer compute.LoadBalancer, err error) {
	err = waitFor(ctx, loadBalancerTimeout, defaultWaitInterval, fmt.Sprintf("load balancer %d to be mutable", loadBalancerID), func(ctx context.Context) (bool, error) {
		got, err := service.Get(ctx, compute.LoadBalancerGetReq{ID: loadBalancerID})
		if err != nil {
			if isNotFound(err) {
				return false, stopWaiting(err)
			}
			return false, err
		}
		loadBalancer = got

		switch loadBalancer.Status.ID {
		case compute.LoadBalancerStatusWorking:
			return false, nil
		case compute.LoadBalancerStatusError:
			return false, fmt.Errorf("load balancer %d is in error state", loadBalancerID)
		default:
			return true, nil
		}
	})

	return loadBalancer, err
}
