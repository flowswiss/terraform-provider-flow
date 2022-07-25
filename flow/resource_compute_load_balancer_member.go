package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.ResourceType = (*computeLoadBalancerMemberResourceType)(nil)
	_ tfsdk.Resource     = (*computeLoadBalancerMemberResource)(nil)
)

type computeLoadBalancerMemberResourceData struct {
	ID             types.Int64 `tfsdk:"id"`
	PoolID         types.Int64 `tfsdk:"pool_id"`
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`

	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`

	// TODO status
}

func (c *computeLoadBalancerMemberResourceData) FromEntity(loadBalancerID, poolID int, member compute.LoadBalancerMember) {
	c.ID = types.Int64{Value: int64(member.ID)}
	c.PoolID = types.Int64{Value: int64(poolID)}
	c.LoadBalancerID = types.Int64{Value: int64(loadBalancerID)}

	c.Name = types.String{Value: member.Name}
	c.Address = types.String{Value: member.Address}
	c.Port = types.Int64{Value: int64(member.Port)}
}

func (c computeLoadBalancerMemberResourceData) AppliesTo(member compute.LoadBalancerMember) bool {
	return c.ID.Value == int64(member.ID)
}

type computeLoadBalancerMemberResourceType struct{}

func (c computeLoadBalancerMemberResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer member",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"pool_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer pool",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"load_balancer_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},

			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer member",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"address": {
				Type:                types.StringType,
				MarkdownDescription: "IP address of the load balancer member",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"port": {
				Type:                types.Int64Type,
				MarkdownDescription: "port of the load balancer member",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeLoadBalancerMemberResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerMemberResource{
		loadBalancerService: compute.NewLoadBalancerService(prov.client),
	}, diagnostics
}

type computeLoadBalancerMemberResource struct {
	loadBalancerService compute.LoadBalancerService
}

func (c computeLoadBalancerMemberResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeLoadBalancerMemberResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(config.LoadBalancerID.Value)
	poolID := int(config.PoolID.Value)

	create := compute.LoadBalancerMemberCreate{
		Name:    config.Name.Value,
		Address: config.Address.Value,
		Port:    int(config.Port.Value),
	}

	member, err := c.loadBalancerService.Pools(loadBalancerID).Members(poolID).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create load balancer member: %s", err))
		return
	}

	err = c.loadBalancerService.WaitUntilMutable(ctx, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to wait until load balancer is mutable: %s", err))
		return
	}

	var state computeLoadBalancerMemberResourceData
	state.FromEntity(loadBalancerID, poolID, member)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerMemberResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeLoadBalancerMemberResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.Value)
	poolID := int(state.PoolID.Value)

	list, err := c.loadBalancerService.Pools(loadBalancerID).Members(poolID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer members: %s", err))
		return
	}

	member, err := filter.FindOne(state, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer member: %s", err))
		return
	}

	state.FromEntity(loadBalancerID, poolID, member)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerMemberResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a load balancer member is not supported")
}

func (c computeLoadBalancerMemberResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeLoadBalancerMemberResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.Value)
	poolID := int(state.PoolID.Value)
	memberID := int(state.ID.Value)

	err := c.loadBalancerService.Pools(loadBalancerID).Members(poolID).Delete(ctx, memberID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete load balancer member: %s", err))
		return
	}

	err = c.loadBalancerService.WaitUntilMutable(ctx, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to wait until load balancer is mutable: %s", err))
		return
	}
}
