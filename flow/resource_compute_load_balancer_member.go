package flow

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ resource.Resource                = (*computeLoadBalancerMemberResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeLoadBalancerMemberResource)(nil)
	_ resource.ResourceWithImportState = (*computeLoadBalancerMemberResource)(nil)
)

type computeLoadBalancerMemberResourceData struct {
	ID             types.Int64 `tfsdk:"id"`
	PoolID         types.Int64 `tfsdk:"pool_id"`
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`

	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
	Status  types.String `tfsdk:"status"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (c *computeLoadBalancerMemberResourceData) FromEntity(loadBalancerID, poolID int, member compute.LoadBalancerMember) {
	c.ID = types.Int64Value(int64(member.ID))
	c.PoolID = types.Int64Value(int64(poolID))
	c.LoadBalancerID = types.Int64Value(int64(loadBalancerID))

	c.Name = types.StringValue(member.Name)
	c.Address = types.StringValue(member.Address)
	c.Port = types.Int64Value(int64(member.Port))
	c.Status = types.StringValue(member.Status.Key)
}

func (c computeLoadBalancerMemberResourceData) AppliesTo(member compute.LoadBalancerMember) bool {
	return c.ID.ValueInt64() == int64(member.ID)
}

func (c computeLoadBalancerMemberResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_load_balancer_member.<name> <load_balancer_id>:<pool_id>:<id>`",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer member",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"pool_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer pool",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"load_balancer_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer member",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "IP address of the load balancer member",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "port of the load balancer member",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "current status of the load balancer member, as a stable key (`active`, `disabled`, `working`, `degraded`, `error`)",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: timeoutDescription("bounds the whole create; unset, the load balancer is given 10m to become mutable again"),
				Delete:            true,
				DeleteDescription: timeoutDescription("bounds the whole delete; unset, the load balancer is given 10m to become mutable again"),
			}),
		},
	}
}

func newComputeLoadBalancerMemberResource() resource.Resource {
	return &computeLoadBalancerMemberResource{}
}

func (c *computeLoadBalancerMemberResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer_member"
}

func (c *computeLoadBalancerMemberResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeLoadBalancerMemberResource struct {
	client flowClient
}

func (c computeLoadBalancerMemberResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeLoadBalancerMemberResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Create, &response.Diagnostics)
	defer cancel()

	loadBalancerID := int(config.LoadBalancerID.ValueInt64())
	poolID := int(config.PoolID.ValueInt64())

	create := compute.LoadBalancerMemberCreateReq{
		LoadBalancerID:     uint(loadBalancerID),
		LoadBalancerPoolID: uint(poolID),
		Name:               config.Name.ValueString(),
		Address:            config.Address.ValueString(),
		Port:               int(config.Port.ValueInt64()),
	}

	var member compute.LoadBalancerMember
	err := retryCreate(ctx, "create load balancer member", func() (err error) {
		member, err = c.client.Compute.LoadBalancerMember.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create load balancer member: %s", err))
		return
	}

	var state computeLoadBalancerMemberResourceData
	state.FromEntity(loadBalancerID, poolID, member)
	state.Timeouts = config.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)

	_, err = waitForLoadBalancerMutable(ctx, c.client.Compute.LoadBalancer, uint(loadBalancerID))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
		return
	}
}

func (c computeLoadBalancerMemberResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeLoadBalancerMemberResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.ValueInt64())
	poolID := int(state.PoolID.ValueInt64())

	list, err := c.client.Compute.LoadBalancerMember.List(ctx, compute.LoadBalancerMemberListReq{
		LoadBalancerID:     uint(loadBalancerID),
		LoadBalancerPoolID: uint(poolID),
		Cursor:             core.CursorAll,
	})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("load balancer %d pool %d", loadBalancerID, poolID))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer members: %s", err))
		return
	}

	member, err := filter.FindOne(state, list.Items)
	if err != nil {
		if errors.Is(err, filter.ErrNoResults) {
			removeGone(ctx, response, fmt.Sprintf("load balancer member %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to find load balancer member: %s", err))
		return
	}

	state.FromEntity(loadBalancerID, poolID, member)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerMemberResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a load balancer member is not supported")
}

func (c computeLoadBalancerMemberResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeLoadBalancerMemberResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, state.Timeouts.Delete, &response.Diagnostics)
	defer cancel()

	loadBalancerID := uint(state.LoadBalancerID.ValueInt64())

	err := retryDelete(ctx, "delete load balancer member", func() error {
		return c.client.Compute.LoadBalancerMember.Delete(ctx, compute.LoadBalancerMemberDeleteReq{
			LoadBalancerID:       loadBalancerID,
			LoadBalancerPoolID:   uint(state.PoolID.ValueInt64()),
			LoadBalancerMemberID: uint(state.ID.ValueInt64()),
		})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete load balancer member: %s", err))
		return
	}

	_, err = waitForLoadBalancerMutable(ctx, c.client.Compute.LoadBalancer, loadBalancerID)
	if err != nil && !isNotFound(err) {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for load balancer to be mutable: %s", err))
		return
	}
}

func (c computeLoadBalancerMemberResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("load_balancer_id"), path.Root("pool_id"), path.Root("id"))
}
