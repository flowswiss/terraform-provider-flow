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
	_ datasource.DataSource              = (*computeLoadBalancerMemberDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeLoadBalancerMemberDataSource)(nil)
)

type computeLoadBalancerMemberDataSourceData struct {
	ID             types.Int64 `tfsdk:"id"`
	PoolID         types.Int64 `tfsdk:"pool_id"`
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`

	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
	Status  types.String `tfsdk:"status"`
}

func (c *computeLoadBalancerMemberDataSourceData) FromEntity(loadBalancerID, poolID int, member compute.LoadBalancerMember) {
	c.ID = types.Int64Value(int64(member.ID))
	c.PoolID = types.Int64Value(int64(poolID))
	c.LoadBalancerID = types.Int64Value(int64(loadBalancerID))

	c.Name = types.StringValue(member.Name)
	c.Address = types.StringValue(member.Address)
	c.Port = types.Int64Value(int64(member.Port))
	c.Status = types.StringValue(member.Status.Key)
}

func (c computeLoadBalancerMemberDataSourceData) AppliesTo(member compute.LoadBalancerMember) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(member.ID) {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != member.Name {
		return false
	}

	if !c.Address.IsNull() && c.Address.ValueString() != member.Address {
		return false
	}

	if !c.Port.IsNull() && c.Port.ValueInt64() != int64(member.Port) {
		return false
	}

	return true
}

func (c computeLoadBalancerMemberDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
			"pool_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer pool",
				Required:            true,
			},
			"load_balancer_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "IP address of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "port of the load balancer member",
				Optional:            true,
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "current status of the load balancer member, as a stable key (`active`, `disabled`, `working`, `degraded`, `error`)",
				Computed:            true,
			},
		},
	}
}

func newComputeLoadBalancerMemberDataSource() datasource.DataSource {
	return &computeLoadBalancerMemberDataSource{}
}

func (c *computeLoadBalancerMemberDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer_member"
}

func (c *computeLoadBalancerMemberDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeLoadBalancerMemberDataSource struct {
	client flowClient
}

func (c computeLoadBalancerMemberDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeLoadBalancerMemberDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(config.LoadBalancerID.ValueInt64())
	poolID := int(config.PoolID.ValueInt64())

	list, err := c.client.Compute.LoadBalancerMember.List(ctx, compute.LoadBalancerMemberListReq{
		LoadBalancerID:     uint(loadBalancerID),
		LoadBalancerPoolID: uint(poolID),
		Cursor:             core.CursorAll,
	})
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
