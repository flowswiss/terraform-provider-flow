package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*macBareMetalNetworkDataSourceType)(nil)
	_ tfsdk.DataSource     = (*macBareMetalNetworkDataSource)(nil)
)

type macBareMetalNetworkDataSourceAllocationPool struct {
	Start types.String `tfsdk:"start"`
	End   types.String `tfsdk:"end"`
}

type macBareMetalNetworkDataSourceData struct {
	ID                types.Int64                                  `tfsdk:"id"`
	Name              types.String                                 `tfsdk:"name"`
	CIDR              types.String                                 `tfsdk:"cidr"`
	LocationID        types.Int64                                  `tfsdk:"location_id"`
	DomainNameServers []types.String                               `tfsdk:"domain_name_servers"`
	AllocationPool    *macBareMetalNetworkDataSourceAllocationPool `tfsdk:"allocation_pool"`
	GatewayIP         types.String                                 `tfsdk:"gateway_ip"`
}

func (c *macBareMetalNetworkDataSourceData) FromEntity(network macbaremetal.Network) {
	c.ID = types.Int64{Value: int64(network.ID)}
	c.Name = types.String{Value: network.Name}
	c.CIDR = types.String{Value: network.Subnet}
	c.LocationID = types.Int64{Value: int64(network.Location.ID)}
	c.GatewayIP = types.String{Value: network.GatewayIP}

	c.AllocationPool = &macBareMetalNetworkDataSourceAllocationPool{
		Start: types.String{Value: network.AllocationPoolStart},
		End:   types.String{Value: network.AllocationPoolEnd},
	}

	c.DomainNameServers = make([]types.String, len(network.DomainNameServers))
	for idx, domainNameServer := range network.DomainNameServers {
		c.DomainNameServers[idx] = types.String{Value: domainNameServer}
	}
}

func (c macBareMetalNetworkDataSourceData) AppliesTo(network macbaremetal.Network) bool {
	if !c.ID.Null && network.ID != int(c.ID.Value) {
		return false
	}

	if !c.Name.Null && network.Name != c.Name.Value {
		return false
	}

	return true
}

type macBareMetalNetworkDataSourceType struct{}

func (c macBareMetalNetworkDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the network",
				Optional:            true,
				Computed:            true,
			},
			"cidr": {
				Type:                types.StringType,
				MarkdownDescription: "CIDR of the network",
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"domain_name_servers": {
				Type: types.ListType{
					ElemType: types.StringType,
				},
				MarkdownDescription: "list of domain name servers",
				Computed:            true,
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
			},
			"gateway_ip": {
				Type:                types.StringType,
				MarkdownDescription: "gateway IP of the network",
				Computed:            true,
			},
		},
	}, nil
}

func (c macBareMetalNetworkDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalNetworkDataSource{
		networkService: macbaremetal.NewNetworkService(prov.client),
	}, diagnostics
}

type macBareMetalNetworkDataSource struct {
	networkService macbaremetal.NetworkService
}

func (c macBareMetalNetworkDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config macBareMetalNetworkDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.networkService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list networks: %s", err))
		return
	}

	network, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find network: %s", err))
		return
	}

	var state macBareMetalNetworkDataSourceData
	state.FromEntity(network)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
