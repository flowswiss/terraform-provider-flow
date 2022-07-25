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
	_ tfsdk.DataSourceType = (*computeLoadBalancerMemberDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeLoadBalancerMemberDataSource)(nil)
)

type computeLoadBalancerMemberDataSourceData struct {
	ID             types.Int64 `tfsdk:"id"`
	PoolID         types.Int64 `tfsdk:"pool_id"`
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`

	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`

	// TODO status
}

func (c *computeLoadBalancerMemberDataSourceData) FromEntity(loadBalancerID, poolID int, member compute.LoadBalancerMember) {
	c.ID = types.Int64{Value: int64(member.ID)}
	c.PoolID = types.Int64{Value: int64(poolID)}
	c.LoadBalancerID = types.Int64{Value: int64(loadBalancerID)}

	c.Name = types.String{Value: member.Name}
	c.Address = types.String{Value: member.Address}
	c.Port = types.Int64{Value: int64(member.Port)}
}

func (c computeLoadBalancerMemberDataSourceData) AppliesTo(member compute.LoadBalancerMember) bool {
	if !c.ID.Null && c.ID.Value != int64(member.ID) {
		return false
	}

	if !c.Name.Null && c.Name.Value != member.Name {
		return false
	}

	if !c.Address.Null && c.Address.Value != member.Address {
		return false
	}

	if !c.Port.Null && c.Port.Value != int64(member.Port) {
		return false
	}

	return true
}

type computeLoadBalancerMemberDataSourceType struct{}

func (c computeLoadBalancerMemberDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
			"pool_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer pool",
				Required:            true,
			},
			"load_balancer_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
			},

			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
			"address": {
				Type:                types.StringType,
				MarkdownDescription: "IP address of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
			"port": {
				Type:                types.Int64Type,
				MarkdownDescription: "port of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeLoadBalancerMemberDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerMemberDataSource{
		loadBalancerService: compute.NewLoadBalancerService(prov.client),
	}, diagnostics
}

type computeLoadBalancerMemberDataSource struct {
	loadBalancerService compute.LoadBalancerService
}

func (c computeLoadBalancerMemberDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeLoadBalancerMemberDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(config.LoadBalancerID.Value)
	poolID := int(config.PoolID.Value)

	list, err := c.loadBalancerService.Pools(loadBalancerID).Members(poolID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer members: %s", err))
		return
	}

	member, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer member: %s", err))
		return
	}

	var state computeLoadBalancerMemberDataSourceData
	state.FromEntity(loadBalancerID, poolID, member)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
