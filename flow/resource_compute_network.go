package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeNetworkResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeNetworkResource)(nil)
	_ resource.ResourceWithImportState = (*computeNetworkResource)(nil)
)

type computeNetworkResourceAllocationPool struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type computeNetworkResourceData struct {
	ID                types.Int64                           `tfsdk:"id"`
	Name              types.String                          `tfsdk:"name"`
	CIDR              types.String                          `tfsdk:"cidr"`
	LocationID        types.Int64                           `tfsdk:"location_id"`
	DomainNameServers []types.String                        `tfsdk:"domain_name_servers"`
	AllocationPool    *computeNetworkResourceAllocationPool `tfsdk:"allocation_pool"`
	GatewayIP         types.String                          `tfsdk:"gateway_ip"`
}

func (c *computeNetworkResourceData) FromEntity(network compute.Network) {
	c.ID = types.Int64Value(int64(network.ID))
	c.Name = types.StringValue(network.Name)
	c.CIDR = types.StringValue(network.CIDR)
	c.LocationID = types.Int64Value(int64(network.Location.ID))
	c.GatewayIP = types.StringValue(network.GatewayIP)

	c.AllocationPool = &computeNetworkResourceAllocationPool{
		Start: types.StringValue(network.AllocationPoolStart),
		End:   types.StringValue(network.AllocationPoolEnd),
	}

	c.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		c.DomainNameServers[idx] = types.StringValue(domainNameServer)
	}
}

func (c computeNetworkResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the network",
				Required:            true,
			},
			"cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR of the network",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"domain_name_servers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "list of domain name servers",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"allocation_pool": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"start": schema.StringAttribute{
						MarkdownDescription: "start of the allocation pool",
						Required:            true,
					},
					"end": schema.StringAttribute{
						MarkdownDescription: "end of the allocation pool",
						Required:            true,
					},
				},
				MarkdownDescription: "allocation pool",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway_ip": schema.StringAttribute{
				MarkdownDescription: "gateway IP of the network",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func newComputeNetworkResource() resource.Resource {
	return &computeNetworkResource{}
}

func (c *computeNetworkResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_network"
}

func (c *computeNetworkResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeNetworkResource struct {
	client flowClient
}

func (c computeNetworkResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeNetworkResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.NetworkCreateReq{
		Name:       nonZero(config.Name.ValueString()),
		LocationID: nonZero(int(config.LocationID.ValueInt64())),
		CIDR:       nonZero(config.CIDR.ValueString()),
		DomainNameServers: []string{
			"1.1.1.1", "8.8.8.8",
		},
		GatewayIP: nonZero(config.GatewayIP.ValueString()),
	}

	if len(config.DomainNameServers) != 0 {
		create.DomainNameServers = make([]string, len(config.DomainNameServers))
		for idx, domainNameServer := range config.DomainNameServers {
			create.DomainNameServers[idx] = domainNameServer.ValueString()
		}
	}

	if config.AllocationPool != nil {
		create.AllocationPoolStart = nonZero(config.AllocationPool.Start.ValueString())
		create.AllocationPoolEnd = nonZero(config.AllocationPool.End.ValueString())
	}

	var network compute.Network
	err := retryCreate(ctx, "create network", func() (err error) {
		network, err = c.client.Compute.Network.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create network: %s", err))
		return
	}

	var state computeNetworkResourceData
	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	network, err := c.client.Compute.Network.Get(ctx, compute.NetworkGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("network %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeNetworkResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	update := compute.NetworkUpdateReq{
		ID:        uint(state.ID.ValueInt64()),
		Name:      nonZero(config.Name.ValueString()),
		GatewayIP: nonZero(config.GatewayIP.ValueString()),
	}

	if len(config.DomainNameServers) != 0 {
		update.DomainNameServers = make([]string, len(config.DomainNameServers))
		for idx, domainNameServer := range config.DomainNameServers {
			update.DomainNameServers[idx] = domainNameServer.ValueString()
		}
	}

	if config.AllocationPool != nil {
		update.AllocationPoolStart = nonZero(config.AllocationPool.Start.ValueString())
		update.AllocationPoolEnd = nonZero(config.AllocationPool.End.ValueString())
	}

	var network compute.Network
	err := retry(ctx, "update network", func() (err error) {
		network, err = c.client.Compute.Network.Update(ctx, update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete network", func() error {
		return c.client.Compute.Network.Delete(ctx, compute.NetworkDeleteReq{ID: uint(state.ID.ValueInt64())})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete network: %s", err))
		return
	}
}

func (c computeNetworkResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
