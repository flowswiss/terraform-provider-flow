package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType            = (*macBareMetalNetworkResourceType)(nil)
	_ tfsdk.Resource                = (*macBareMetalNetworkResource)(nil)
	_ tfsdk.ResourceWithImportState = (*macBareMetalNetworkResource)(nil)
)

type macBareMetalNetworkResourceAllocationPool struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type macBareMetalNetworkResourceData struct {
	ID                types.Int64                                `tfsdk:"id"`
	Name              types.String                               `tfsdk:"name"`
	CIDR              types.String                               `tfsdk:"cidr"`
	LocationID        types.Int64                                `tfsdk:"location_id"`
	DomainName        types.String                               `tfsdk:"domain_name"`
	DomainNameServers []types.String                             `tfsdk:"domain_name_servers"`
	AllocationPool    *macBareMetalNetworkResourceAllocationPool `tfsdk:"allocation_pool"`
	GatewayIP         types.String                               `tfsdk:"gateway_ip"`
}

func (r *macBareMetalNetworkResourceData) FromEntity(network macbaremetal.Network) {
	r.ID = types.Int64{Value: int64(network.ID)}
	r.Name = types.String{Value: network.Name}
	r.CIDR = types.String{Value: network.Subnet}
	r.LocationID = types.Int64{Value: int64(network.Location.ID)}
	r.GatewayIP = types.String{Value: network.GatewayIP}

	r.AllocationPool = &macBareMetalNetworkResourceAllocationPool{
		Start: types.String{Value: network.AllocationPoolStart},
		End:   types.String{Value: network.AllocationPoolEnd},
	}

	r.DomainName = types.String{Value: network.DomainName}
	r.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		r.DomainNameServers[idx] = types.String{Value: domainNameServer}
	}
}

type macBareMetalNetworkResourceType struct{}

func (r macBareMetalNetworkResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
				Computed:            true,
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
			"domain_name": {
				Type:                types.StringType,
				MarkdownDescription: "domain name of the network",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
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
						Computed:            true,
					},
					"end": {
						Type:                types.StringType,
						MarkdownDescription: "end of the allocation pool",
						Computed:            true,
					},
				}),
				MarkdownDescription: "allocation pool",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"gateway_ip": {
				Type:                types.StringType,
				MarkdownDescription: "gateway IP of the network",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
		},
	}, nil
}

func (r macBareMetalNetworkResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalNetworkResource{
		networkService: macbaremetal.NewNetworkService(prov.client),
	}, diagnostics
}

type macBareMetalNetworkResource struct {
	networkService macbaremetal.NetworkService
}

func (r macBareMetalNetworkResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config macBareMetalNetworkResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := macbaremetal.NetworkCreate{
		Name:       config.Name.Value,
		LocationID: int(config.LocationID.Value),
	}

	network, err := r.networkService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create network: %s", err))
		return
	}

	// the api does not allow to set these properties on creation,
	// so we need to set afterwards using an update.
	if len(config.DomainNameServers) != 0 || !config.DomainName.Null {
		update := macbaremetal.NetworkUpdate{
			DomainName:        config.DomainName.Value,
			DomainNameServers: nil,
		}

		for _, domainNameServer := range config.DomainNameServers {
			update.DomainNameServers = append(update.DomainNameServers, domainNameServer.Value)
		}

		network, err = r.networkService.Update(ctx, network.ID, update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network: %s", err))
			return
		}
	}

	var state macBareMetalNetworkResourceData
	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalNetworkResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state macBareMetalNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	network, err := r.networkService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalNetworkResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state macBareMetalNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config macBareMetalNetworkResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	update := macbaremetal.NetworkUpdate{
		Name:       config.Name.Value,
		DomainName: config.DomainName.Value,
	}

	if len(config.DomainNameServers) != 0 {
		update.DomainNameServers = make([]string, len(config.DomainNameServers))
		for idx, domainNameServer := range config.DomainNameServers {
			update.DomainNameServers[idx] = domainNameServer.Value
		}
	}

	network, err := r.networkService.Update(ctx, int(state.ID.Value), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network: %s", err))
		return
	}

	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalNetworkResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state macBareMetalNetworkResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.networkService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete network: %s", err))
		return
	}
}

func (r macBareMetalNetworkResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
