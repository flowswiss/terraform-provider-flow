package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ tfsdk.ResourceType            = (*computeNetworkResourceType)(nil)
	_ tfsdk.Resource                = (*computeNetworkResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeNetworkResource)(nil)
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
	c.ID = types.Int64{Value: int64(network.ID)}
	c.Name = types.String{Value: network.Name}
	c.CIDR = types.String{Value: network.CIDR}
	c.LocationID = types.Int64{Value: int64(network.Location.ID)}
	c.GatewayIP = types.String{Value: network.GatewayIP}

	c.AllocationPool = &computeNetworkResourceAllocationPool{
		Start: types.String{Value: network.AllocationPoolStart},
		End:   types.String{Value: network.AllocationPoolEnd},
	}

	c.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		c.DomainNameServers[idx] = types.String{Value: domainNameServer}
	}
}

type computeNetworkResourceType struct{}

func (c computeNetworkResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the network",
				Required:            true,
			},
			"cidr": {
				Type:                types.StringType,
				MarkdownDescription: "CIDR of the network",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"domain_name_servers": {
				Type: types.ListType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "list of domain name servers",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"allocation_pool": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"start": {
						Type:                types.StringType,
						MarkdownDescription: "start of the allocation pool",
						Required:            true,
					},
					"end": {
						Type:                types.StringType,
						MarkdownDescription: "end of the allocation pool",
						Required:            true,
					},
				}),
				MarkdownDescription: "allocation pool",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"gateway_ip": {
				Type:                types.StringType,
				MarkdownDescription: "gateway IP of the network",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
		},
	}, nil
}

func (c computeNetworkResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeNetworkResource{
		networkService: compute.NewNetworkService(prov.client),
	}, diagnostics
}

type computeNetworkResource struct {
	networkService compute.NetworkService
}

func (c computeNetworkResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeNetworkResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.NetworkCreate{
		Name:       config.Name.Value,
		LocationID: int(config.LocationID.Value),
		CIDR:       config.CIDR.Value,
		DomainNameServers: []string{
			"1.1.1.1", "8.8.8.8",
		},
		GatewayIP: config.GatewayIP.Value,
	}

	if len(config.DomainNameServers) != 0 {
		create.DomainNameServers = make([]string, len(config.DomainNameServers))
		for idx, domainNameServer := range config.DomainNameServers {
			create.DomainNameServers[idx] = domainNameServer.Value
		}
	}

	if config.AllocationPool != nil {
		create.AllocationPoolStart = config.AllocationPool.Start.Value
		create.AllocationPoolEnd = config.AllocationPool.End.Value
	}

	network, err := c.networkService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create network: %s", err))
		return
	}

	var state computeNetworkResourceData
	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	network, err := c.networkService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
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

	update := compute.NetworkUpdate{
		Name:      config.Name.Value,
		GatewayIP: config.GatewayIP.Value,
	}

	if len(config.DomainNameServers) != 0 {
		update.DomainNameServers = make([]string, len(config.DomainNameServers))
		for idx, domainNameServer := range config.DomainNameServers {
			update.DomainNameServers[idx] = domainNameServer.Value
		}
	}

	if config.AllocationPool != nil {
		update.AllocationPoolStart = config.AllocationPool.Start.Value
		update.AllocationPoolEnd = config.AllocationPool.End.Value
	}

	network, err := c.networkService.Update(ctx, int(state.ID.Value), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := c.networkService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete network: %s", err))
		return
	}
}

func (c computeNetworkResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), request, response)
}
